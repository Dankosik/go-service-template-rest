package httpx

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/example/go-service-template-rest/internal/problem"
	"golang.org/x/time/rate"
)

// RateLimiter decides whether one caller may proceed right now.
//
// It is a seam rather than a policy. MaxInFlight bounds how many requests run at
// once, which protects the process but not its other callers: one client opening
// three hundred concurrent connections takes every in-flight slot, and every
// other tenant is shed for capacity that one caller took. Telling them apart
// needs an identity, and which identity — API key, tenant header, source network
// — is a decision only the service can make.
//
// retryAfter is what the caller is told to wait, and is only meaningful when
// allowed is false.
type RateLimiter interface {
	Allow(ctx context.Context, key string) (allowed bool, retryAfter time.Duration)
}

// RateLimitKeyFunc reports which bucket a request is charged against. An empty
// key means the request is not limited.
//
// It runs in the root middleware chain, ahead of the OpenAPI request validator,
// because limiting a caller after their body has been schema-validated means the
// expensive half of the request was already done for them. That placement is also
// why the authenticated principal is not available here: the validator is what
// resolves it. A service that would rather limit per resolved identity installs
// RateLimit inside its own chain and keys on reqctx.PrincipalFromContext.
type RateLimitKeyFunc func(*http.Request) string

// HeaderRateLimitKey charges a request to the value of one header, hashed.
//
// The hash is not decoration. The header that identifies a caller before
// authentication is usually the credential itself, and a credential used as a map
// key is a credential one heap dump or one debug log away from disclosure. A
// digest identifies the same caller and discloses nothing.
//
// The client address is deliberately not offered. Behind any proxy, RemoteAddr is
// the proxy, so limiting by it throttles the whole fleet as one caller, and
// X-Forwarded-For is attacker-controlled unless the trusted proxy topology is
// known — which a template cannot know. A service that does know its edge writes
// the four-line key function for it.
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
// shipped default: the limit that suits one deployment is wrong for the next, and
// a template that picked one would have every generated service inherit it.
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

// retryAfterSeconds rounds a delay up to whole seconds, which is the only form
// this repository sends. A sub-second budget must not be advertised as "retry
// immediately": that is how a client that thinks it is being polite gets limited
// harder.
func retryAfterSeconds(delay time.Duration) int {
	seconds := int(math.Ceil(delay.Seconds()))
	return max(seconds, 1)
}

// KeyedRateLimiter is one token bucket per key, held in a bounded map.
//
// It is per instance, not per fleet, and that is the property to understand
// before using it: N replicas admit up to N times the configured rate, and sizing
// it as limit/replicas breaks under a rolling deploy. It is the right shape when
// the limit exists to stop one caller monopolizing one instance — the failure
// MaxInFlight leaves open — and the wrong shape for a contractual quota, which
// needs shared state behind this same RateLimiter interface.
//
// The key map is bounded because keys are caller-controlled: an unbounded
// map[string]*rate.Limiter keyed on anything an attacker can vary is a memory
// leak with a rate limiter attached, and that is the version most services write.
type KeyedRateLimiter struct {
	limit rate.Limit
	burst int
	// maxKeys bounds one generation. Two are live at a time, so the real ceiling
	// is twice this.
	maxKeys int

	mu sync.Mutex
	// current and previous are generations rather than an LRU. When current
	// fills, it becomes previous and a fresh map takes over, so eviction is one
	// pointer swap instead of per-entry bookkeeping on the hot path. A key still
	// in use is promoted out of previous on its next request, so an active caller
	// keeps its bucket; an idle one is dropped a generation later.
	current  map[string]*rate.Limiter
	previous map[string]*rate.Limiter
}

const (
	// defaultRateLimitMaxKeys bounds the tracked callers when none is named. It
	// is large enough not to evict an ordinary tenant set and small enough that
	// two generations of it stay a rounding error against a service's heap.
	defaultRateLimitMaxKeys = 8192

	// rateLimitInitialKeys caps the preallocation, so a large maxKeys does not
	// reserve the whole map up front for a service that has three callers.
	rateLimitInitialKeys = 256
)

// NewKeyedRateLimiter builds a per-key token bucket admitting perSecond requests
// with room for burst of them arriving at once, tracking at most maxKeys callers
// per generation. A non-positive maxKeys uses defaultRateLimitMaxKeys.
//
// Unusable settings are an error rather than a nil limiter. Returning nil here
// looked convenient — the caller could pass the result straight into
// RouterConfig.RateLimit and get no middleware — but a nil *KeyedRateLimiter
// stored in a RateLimiter interface is a non-nil interface holding a nil pointer,
// so the middleware would install itself and dereference nothing on the first
// request that carried a key. A service that wants no limiting leaves the
// field unset.
func NewKeyedRateLimiter(perSecond float64, burst, maxKeys int) (*KeyedRateLimiter, error) {
	if perSecond <= 0 {
		return nil, fmt.Errorf("keyed rate limiter: requests per second must be > 0")
	}
	if burst <= 0 {
		return nil, fmt.Errorf("keyed rate limiter: burst must be > 0")
	}
	if maxKeys <= 0 {
		maxKeys = defaultRateLimitMaxKeys
	}

	return &KeyedRateLimiter{
		limit:   rate.Limit(perSecond),
		burst:   burst,
		maxKeys: maxKeys,
		current: make(map[string]*rate.Limiter, min(maxKeys, rateLimitInitialKeys)),
	}, nil
}

// Allow charges one request to key.
func (l *KeyedRateLimiter) Allow(_ context.Context, key string) (bool, time.Duration) {
	limiter := l.bucket(key)

	// Reserve rather than Allow, so a rejected caller can be told how long to
	// wait. Cancel returns the token the reservation took, which keeps a
	// rejection from consuming budget the caller never got to use.
	reservation := limiter.Reserve()
	if !reservation.OK() {
		return false, time.Second
	}
	if delay := reservation.Delay(); delay > 0 {
		reservation.Cancel()
		return false, delay
	}
	return true, 0
}

func (l *KeyedRateLimiter) bucket(key string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	if limiter, ok := l.current[key]; ok {
		return limiter
	}
	if limiter, ok := l.previous[key]; ok {
		// Promoted, so a caller that is still active does not lose its bucket —
		// and therefore its accumulated debt — when a generation rolls over.
		if len(l.current) >= l.maxKeys {
			l.previous = l.current
			l.current = make(map[string]*rate.Limiter, min(l.maxKeys, rateLimitInitialKeys))
		}
		l.current[key] = limiter
		return limiter
	}

	if len(l.current) >= l.maxKeys {
		l.previous = l.current
		l.current = make(map[string]*rate.Limiter, min(l.maxKeys, rateLimitInitialKeys))
	}
	limiter := rate.NewLimiter(l.limit, l.burst)
	l.current[key] = limiter
	return limiter
}
