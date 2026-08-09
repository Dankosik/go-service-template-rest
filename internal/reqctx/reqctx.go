// Package reqctx owns the request-scoped values a handler reads and the
// transport adapter writes.
//
// It is a leaf on purpose. internal/infra/http produces these values while
// serving a request and a feature package consumes them while handling one, and
// depguard forbids a feature package from importing a concrete infra adapter —
// so neither may import the other and both need the same answer.
package reqctx

import (
	"context"
	"crypto/rand"
	"net/http"
	"slices"
	"strings"
)

// MaxRequestIDLength is the shared wire limit for a correlation identifier.
const MaxRequestIDLength = 128

// RequestIDHeader and RequestIDMetadataKey are one wire name in the two
// spellings the transports require: net/http canonicalizes header keys, gRPC
// requires lowercase metadata keys. This package owns both because every
// transport adapter must agree on them and none may import another;
// TestRequestIDWireNamesAreOneName proves the two equal.
const (
	RequestIDHeader      = "X-Request-ID"
	RequestIDMetadataKey = "x-request-id"
)

// Principal is the authenticated caller of the current request.
//
// Issuer and Subject form the caller's stable identity. Scopes are reserved for
// a verifier that actually proves them. A service whose credential proves more
// attaches its own type under its own context key rather than re-reading the
// credential.
type Principal struct {
	// Issuer is the verified namespace in which Subject is unique.
	Issuer string
	// Subject is correlatable identity data, not a credential and not
	// automatically safe to log.
	Subject  string
	ClientID string
	Scopes   []string
}

func (p Principal) HasScope(scope string) bool {
	return slices.Contains(p.Scopes, scope)
}

type principalContextKey struct{}

// ContextWithPrincipal returns ctx carrying the authenticated caller.
//
// Scopes are cloned because the value is shared with every handler that reads
// the context: without the copy, a handler that sorted or appended to the slice
// would edit the authorization decision every other reader sees.
func ContextWithPrincipal(ctx context.Context, principal Principal) context.Context {
	principal.Scopes = slices.Clone(principal.Scopes)
	return context.WithValue(ctx, principalContextKey{}, principal)
}

// PrincipalFromContext returns the authenticated caller, and false when the
// request reached the handler without one.
//
// On a secured operation, false is a wiring defect rather than an anonymous
// caller: a declared security requirement fails closed before any handler runs,
// so a handler that observes false should answer 500 and never 401.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	if !ok {
		return Principal{}, false
	}
	principal.Scopes = slices.Clone(principal.Scopes)
	return principal, true
}

// SetPrincipal publishes principal to every later reader of r, mutating r in
// place.
//
// The mutation is the only thing that works at this seam: the OpenAPI request
// validator builds its validation input around the same *http.Request it later
// hands to the next handler, so a returned copy would be discarded and the
// principal with it. internal/infra/http/middleware_timeout.go mutates the same
// request for the same reason.
func SetPrincipal(r *http.Request, principal Principal) {
	if r == nil {
		return
	}
	*r = *r.WithContext(ContextWithPrincipal(r.Context(), principal))
}

type requestIDContextKey struct{}

// ContextWithRequestID returns ctx carrying the correlation identifier.
// Generating and validating the value belongs to the transport adapter.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

// RequestID returns the correlation identifier, or the empty string when the
// request did not travel through the correlation middleware. Trace and span
// identifiers are reachable through go.opentelemetry.io/otel/trace; this is the
// one correlation value that is not.
func RequestID(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func ContextWithAcceptedRequestID(ctx context.Context, candidate string) (context.Context, string) {
	requestID := strings.TrimSpace(candidate)
	if !ValidRequestID(requestID) {
		requestID = rand.Text()
	}
	return ContextWithRequestID(ctx, requestID), requestID
}

// ValidRequestID reports whether value is safe to carry in logs and response
// metadata. The alphabet is transport-neutral and unchanged from the original
// HTTP contract.
func ValidRequestID(value string) bool {
	if value == "" || len(value) > MaxRequestIDLength {
		return false
	}
	for index := range len(value) {
		character := value[index]
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') {
			continue
		}
		switch character {
		case '.', '_', '~', '-':
			continue
		default:
			return false
		}
	}
	return true
}
