package oidcjwt

import (
	"os"
	"strings"
	"testing"
)

func TestGuideDocumentsBothTokenProfiles(t *testing.T) {
	t.Parallel()
	body, err := os.ReadFile("../../../docs/authentication.md")
	if err != nil {
		t.Fatalf("read authentication guide: %v", err)
	}
	for _, required := range []string{"resource-server", "rfc9068", "APP__AUTHN__ISSUER", "APP__AUTHN__AUDIENCE"} {
		if !strings.Contains(string(body), required) {
			t.Fatalf("authentication guide does not document %q", required)
		}
	}
}
