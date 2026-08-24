package reqctx_test

import (
	"context"
	"testing"

	"github.com/example/go-service-template-rest/internal/reqctx"
)

func TestPrincipalRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := reqctx.ContextWithPrincipal(context.Background(), reqctx.Principal{
		Subject: "svc-checkout",
		Scopes:  []string{"articles:read", "articles:write"},
	})

	principal, ok := reqctx.PrincipalFromContext(ctx)
	if !ok {
		t.Fatal("PrincipalFromContext() reported no principal")
	}
	if principal.Subject != "svc-checkout" {
		t.Fatalf("Subject = %q, want %q", principal.Subject, "svc-checkout")
	}
	if !principal.HasScope("articles:write") {
		t.Fatalf("HasScope(articles:write) = false, scopes = %v", principal.Scopes)
	}
	if principal.HasScope("articles:delete") {
		t.Fatal("HasScope(articles:delete) = true, want false")
	}
}

func TestPrincipalFromContextWithoutValue(t *testing.T) {
	t.Parallel()

	principal, ok := reqctx.PrincipalFromContext(context.Background())
	if ok {
		t.Fatalf("PrincipalFromContext() = %+v, true; want false for an unauthenticated request", principal)
	}
	if principal.Subject != "" || principal.Scopes != nil {
		t.Fatalf("PrincipalFromContext() returned %+v, want the zero principal", principal)
	}
}

func TestPrincipalCallerIdentitySeparatesSubjectsAndClients(t *testing.T) {
	t.Parallel()

	issuer := "https://issuer.example.com"
	subject, ok := (reqctx.Principal{Issuer: issuer, Subject: "same"}).CallerIdentity()
	if !ok {
		t.Fatal("subject CallerIdentity() reported invalid principal")
	}
	client, ok := (reqctx.Principal{Issuer: issuer, ClientID: "same"}).CallerIdentity()
	if !ok {
		t.Fatal("client CallerIdentity() reported invalid principal")
	}
	otherClient, ok := (reqctx.Principal{Issuer: issuer, ClientID: "other"}).CallerIdentity()
	if !ok {
		t.Fatal("other client CallerIdentity() reported invalid principal")
	}
	if subject == client || client == otherClient {
		t.Fatalf("caller identities collided: subject=%q client=%q other_client=%q", subject, client, otherClient)
	}
	for _, invalid := range []reqctx.Principal{{Subject: "subject"}, {Issuer: issuer}} {
		if identity, valid := invalid.CallerIdentity(); valid || identity != "" {
			t.Fatalf("CallerIdentity(%+v) = %q, true; want invalid", invalid, identity)
		}
	}
}

// TestPrincipalScopesAreNotAliased is the reason the carrier clones. Every
// handler reading the context sees the same value, so a caller that sorts or
// appends to the slice it was handed would edit the authorization decision the
// next reader makes.
func TestPrincipalScopesAreNotAliased(t *testing.T) {
	t.Parallel()

	granted := []string{"articles:read"}
	ctx := reqctx.ContextWithPrincipal(context.Background(), reqctx.Principal{Subject: "svc", Scopes: granted})

	granted[0] = "articles:write"
	stored, ok := reqctx.PrincipalFromContext(ctx)
	if !ok {
		t.Fatal("PrincipalFromContext() reported no principal")
	}
	if stored.HasScope("articles:write") {
		t.Fatal("mutating the caller's slice granted a scope that was never presented")
	}

	stored.Scopes[0] = "articles:write"
	reread, ok := reqctx.PrincipalFromContext(ctx)
	if !ok {
		t.Fatal("PrincipalFromContext() reported no principal on reread")
	}
	if reread.HasScope("articles:write") {
		t.Fatal("mutating a returned slice granted a scope to every later reader")
	}
}

func TestRequestIDRoundTrip(t *testing.T) {
	t.Parallel()

	ctx, _ := reqctx.ContextWithAcceptedRequestID(context.Background(), "caller-id")

	if got := reqctx.RequestID(ctx); got != "caller-id" {
		t.Fatalf("RequestID() = %q, want %q", got, "caller-id")
	}
	if got := reqctx.RequestID(context.Background()); got != "" {
		t.Fatalf("RequestID() = %q, want empty without a value", got)
	}
}
