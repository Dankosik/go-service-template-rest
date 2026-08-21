package oidcjwt

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/go-service-template-rest/internal/reqctx"
)

func TestHTTPPublishesPrincipalWithoutForwardedTransportPolicy(t *testing.T) {
	t.Parallel()
	key := loadTestRSAKey(t, testSigningKey)
	verifier := newTestVerifier(t, key)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/private", http.NoBody)
	request.Header.Set("Authorization", "Bearer "+signToken(t, key, "key-1", "JWT", validClaims(testNow)))

	principal, err := verifier.ResolveHTTP(t.Context(), bearerAuthInput(request))
	if err != nil {
		t.Fatalf("ResolveHTTP() error = %v", err)
	}
	if principal.Subject != "subject-1" || principal.ClientID != "client-1" {
		t.Fatalf("principal = %+v", principal)
	}
	if request.Header.Get("Authorization") != "" {
		t.Fatal("Authorization reached the handler-visible request")
	}
	ctx := reqctx.ContextWithPrincipal(request.Context(), principal)
	if stored, ok := reqctx.PrincipalFromContext(ctx); !ok ||
		stored.Issuer != principal.Issuer || stored.Subject != principal.Subject || stored.ClientID != principal.ClientID {
		t.Fatalf("PrincipalFromContext() = %+v, %v", stored, ok)
	}
}
