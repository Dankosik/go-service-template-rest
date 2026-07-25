// Package reqctx owns the request-scoped values a handler reads and the
// transport adapter writes.
//
// It is a leaf on purpose. internal/infra/http produces these values while
// serving a request and a feature package consumes them while handling one, and
// the repository's depguard contract forbids a feature package from importing a
// concrete infra adapter — so neither may import the other and both need the
// same answer. internal/observability/otelconfig is a leaf for the same reason.
//
// Nothing here performs I/O or holds a dependency. A carrier is all a feature
// package needs; deciding what a credential proves stays with whoever validates
// it.
package reqctx

import (
	"context"
	"net/http"
	"slices"
)

// Principal is the authenticated caller of the current request.
//
// It carries the two things an authorization decision needs and nothing else: a
// stable subject to attribute the action to, and the scopes the presented
// credential was granted. A service whose credential proves more attaches its
// own type under its own context key; this is the shape that must not be
// reinvented once per handler.
type Principal struct {
	// Subject identifies the caller. It is safe to log and to attribute an
	// action to, and it must never be the credential itself.
	Subject string
	// Scopes are the permissions the presented credential was granted.
	Scopes []string
}

// HasScope reports whether the presented credential was granted scope.
func (p Principal) HasScope(scope string) bool {
	return slices.Contains(p.Scopes, scope)
}

type principalContextKey struct{}

// ContextWithPrincipal returns ctx carrying the authenticated caller.
//
// Scopes are cloned because the value is shared with every handler that reads
// the context: without the copy, a handler that sorted or appended to the slice
// would edit the authorization decision every other reader sees. One small
// allocation per authenticated request costs nothing next to decoding its body.
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
// The mutation is deliberate and is the only thing that works at this seam. The
// OpenAPI request validator builds its validation input around the same
// *http.Request it later hands to the next handler, so a returned copy would be
// discarded and the principal with it. internal/infra/http/middleware_timeout.go
// mutates the same request for the same class of reason.
//
// This lives here, rather than beside the validator wiring, so it is written
// once. An authenticator that reimplements it is how a service ends up with a
// version that silently returns a copy.
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

// RequestID returns the correlation identifier for the current request, or the
// empty string when the request did not travel through the correlation
// middleware.
//
// A handler needs this to put its own records beside the access log line for the
// same request, and to propagate correlation to a dependency. The trace and span
// identifiers are already reachable through go.opentelemetry.io/otel/trace; this
// is the one correlation value that is not.
func RequestID(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}
