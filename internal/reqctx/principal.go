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
	"slices"
	"strconv"
	"strings"
)

// Principal is the authenticated caller of the current request.
//
// Issuer and Subject form the caller's stable identity; ClientID is the
// machine-caller fallback when Subject is absent. Scopes are reserved for a
// verifier that actually proves them. A service whose credential proves more
// attaches its own type under its own context key rather than re-reading the
// credential.
type Principal struct {
	// Issuer is the verified namespace in which Subject and ClientID are unique.
	Issuer string
	// Subject is correlatable identity data, not a credential and not
	// automatically safe to log.
	Subject string
	// ClientID is the stable machine identity when the token has no subject.
	ClientID string
	Scopes   []string
}

func (p Principal) HasScope(scope string) bool {
	return slices.Contains(p.Scopes, scope)
}

// CallerIdentity returns the collision-free opaque identity a feature uses to
// scope caller-owned state. Subject is authoritative when present; client ID is
// the supported machine-caller fallback. The result is correlatable and must
// not be logged.
func (p Principal) CallerIdentity() (string, bool) {
	if strings.TrimSpace(p.Issuer) == "" {
		return "", false
	}
	kind, value := "subject", p.Subject
	if strings.TrimSpace(value) == "" {
		kind, value = "client", p.ClientID
	}
	if strings.TrimSpace(value) == "" {
		return "", false
	}
	return kind + ":" + strconv.Itoa(len(p.Issuer)) + ":" + p.Issuer + ":" + strconv.Itoa(len(value)) + ":" + value, true
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
