package oauthintrospection

import (
	"os"
	"strings"
	"testing"
)

func TestGuideDocumentsIntrospectionProfile(t *testing.T) {
	t.Parallel()
	body, err := os.ReadFile("../../../docs/authentication.md")
	if err != nil {
		t.Fatalf("read authentication guide: %v", err)
	}
	for _, required := range []string{
		"oidc-introspection",
		"APP__AUTHN__INTROSPECTION_ENDPOINT",
		"APP__AUTHN__INTROSPECTION_CLIENT_SECRET",
	} {
		if !strings.Contains(string(body), required) {
			t.Fatalf("authentication guide does not document %q", required)
		}
	}
}
