package authntrust_test

import (
	"testing"

	"github.com/example/go-service-template-rest/internal/authntrust"
)

// TestValidOIDCURLs holds one case per shared HTTPS rule and pins the one
// deliberate difference: a discovered JWKS endpoint may contain a query while
// a configured issuer may not.
func TestValidOIDCURLs(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name       string
		raw        string
		wantIssuer bool
		wantJWKS   bool
	}{
		{name: "canonical", raw: "https://issuer.example.com", wantIssuer: true, wantJWKS: true},
		{name: "path", raw: "https://issuer.example.com/realms/main", wantIssuer: true, wantJWKS: true},
		{name: "port", raw: "https://issuer.example.com:8443", wantIssuer: true, wantJWKS: true},
		{name: "uppercase scheme", raw: "HTTPS://issuer.example.com", wantIssuer: true, wantJWKS: true},
		{name: "query", raw: "https://issuer.example.com?tenant=a", wantJWKS: true},
		{name: "forced query", raw: "https://issuer.example.com?", wantJWKS: true},

		{name: "plaintext scheme", raw: "http://issuer.example.com"},
		{name: "relative", raw: "/realms/main"},
		{name: "opaque", raw: "https:issuer.example.com"},
		{name: "no host", raw: "https://"},
		{name: "user info", raw: "https://user:secret@issuer.example.com"}, //nolint:gosec // Test fixture verifies user-info rejection; the value is not a credential.
		{name: "fragment", raw: "https://issuer.example.com#frag"},
		{name: "surrounding space", raw: "  https://issuer.example.com  "},
		{name: "unparseable escape", raw: "https://issuer.example.com/%zz"},
		{name: "empty", raw: ""},
		{name: "blank", raw: "   "},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := authntrust.ValidIssuerURL(testCase.raw); got != testCase.wantIssuer {
				t.Errorf("ValidIssuerURL(%q) = %v, want %v", testCase.raw, got, testCase.wantIssuer)
			}
			if got := authntrust.ValidJWKSURL(testCase.raw); got != testCase.wantJWKS {
				t.Errorf("ValidJWKSURL(%q) = %v, want %v", testCase.raw, got, testCase.wantJWKS)
			}
		})
	}
}
