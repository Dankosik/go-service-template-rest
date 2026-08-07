package authntrust_test

import (
	"testing"

	"github.com/example/go-service-template-rest/internal/authntrust"
)

// TestValidProviderURL holds one case per term in the predicate, so loosening
// any single term fails here rather than in whichever caller reached the value
// first.
func TestValidProviderURL(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name  string
		raw   string
		valid bool
	}{
		{name: "canonical", raw: "https://issuer.example.com", valid: true},
		{name: "path", raw: "https://issuer.example.com/realms/main", valid: true},
		{name: "port", raw: "https://issuer.example.com:8443", valid: true},
		{name: "uppercase scheme", raw: "HTTPS://issuer.example.com", valid: true},
		{name: "surrounding space", raw: "  https://issuer.example.com  ", valid: true},

		{name: "plaintext scheme", raw: "http://issuer.example.com"},
		{name: "relative", raw: "/realms/main"},
		{name: "opaque", raw: "https:issuer.example.com"},
		{name: "no host", raw: "https://"},
		{name: "user info", raw: "https://user:secret@issuer.example.com"},
		{name: "query", raw: "https://issuer.example.com?next=a"},
		{name: "forced query", raw: "https://issuer.example.com?"},
		{name: "fragment", raw: "https://issuer.example.com#frag"},
		{name: "unparseable escape", raw: "https://issuer.example.com/%zz"},
		{name: "empty", raw: ""},
		{name: "blank", raw: "   "},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := authntrust.ValidProviderURL(testCase.raw); got != testCase.valid {
				t.Fatalf("ValidProviderURL(%q) = %v, want %v", testCase.raw, got, testCase.valid)
			}
		})
	}
}
