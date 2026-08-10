package httpx

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/problem"
)

// RateLimiter decides whether one caller may proceed right now.
//
// It is a seam rather than a policy. MaxInFlight protects the process but not its
// other callers: one client opening three hundred concurrent connections takes
// every in-flight slot. Telling callers apart needs an identity, and which one —
// API key, tenant header, source network — only the service can decide.
//
// retryAfter is only meaningful when allowed is false.
type RateLimiter interface {
	Allow(ctx context.Context, key string) (allowed bool, retryAfter time.Duration)
}

// RateLimitKeyFunc reports which bucket a request is charged against. An empty
// key means the request is not limited.
//
// It runs ahead of the OpenAPI request validator, so a limited caller does not
// get the expensive half of the request done for them. That placement is also why
// the authenticated principal is not available here — the validator resolves it.
// A service that would rather limit per identity installs RateLimit inside its
// own chain.
type RateLimitKeyFunc func(*http.Request) string

// HeaderRateLimitKey charges a request to the value of one header, hashed.
//
// The header identifying a caller before authentication is usually the credential
// itself, and a credential used as a map key is one heap dump away from
// disclosure; a digest identifies the same caller and discloses nothing.
//
// The client address is deliberately not offered: behind a proxy RemoteAddr
// throttles the whole fleet as one caller, and X-Forwarded-For is
// attacker-controlled unless the trusted proxy topology is known.
func HeaderRateLimitKey(name string) RateLimitKeyFunc {
	return func(r *http.Request) string {
		if r == nil {
			return ""
		}
		value := strings.TrimSpace(r.Header.Get(name))
		if value == "" {
			return ""
		}
		digest := sha256.Sum256([]byte(value))
		return hex.EncodeToString(digest[:])
	}
}

// RateLimit rejects a caller that is over its budget with 429 and a Retry-After.
//
// A nil limiter leaves the middleware out of the chain entirely, which is the
// shipped default: the limit that suits one deployment is wrong for the next.
//
// Platform probe routes are exempt for the same reason they are exempt from
// shedding: rate limiting a readiness probe evicts the instance.
func RateLimit(limiter RateLimiter, key RateLimitKeyFunc, next http.Handler) http.Handler {
	if limiter == nil || key == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isHealthProbeRequest(r) {
			next.ServeHTTP(w, r)
			return
		}
		bucket := key(r)
		if bucket == "" {
			next.ServeHTTP(w, r)
			return
		}
		allowed, retryAfter := limiter.Allow(r.Context(), bucket)
		if allowed {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds(retryAfter)))
		writeProblem(w, r, problemResponse{
			code:   problem.CodeTooManyRequests,
			detail: "too many requests for this caller",
		})
	})
}
