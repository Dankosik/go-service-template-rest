package postgreswebhook

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestWebhookStaticSecretManifest(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	raw := `{"revision":12,"entries":[{"owner_scope":"owner-a","destination_id":"dest-01","key_reference":"key-new","secret":"whsec_` + encoded + `"}]}`
	manifest, err := ParseSecretManifest(raw)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Revision() != 12 {
		t.Fatalf("revision = %d", manifest.Revision())
	}
	key, err := manifest.Resolve("owner-a", "dest-01", "key-new")
	if err != nil || string(key.Bytes) != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("Resolve() = %q, %v", key.Bytes, err)
	}
	if _, err := manifest.Resolve("owner-b", "dest-01", "key-new"); err == nil {
		t.Fatal("cross-owner resolve succeeded")
	}
	for _, invalid := range []string{`{}`, raw + ` {}`, strings.Replace(raw, "\"revision\":12", "\"revision\":0", 1), strings.Replace(raw, "\"entries\"", "\"unknown\"", 1)} {
		if _, err := ParseSecretManifest(invalid); err == nil {
			t.Fatalf("ParseSecretManifest(%q) succeeded", invalid)
		}
	}
	if _, err := ParseSecretManifest(strings.Repeat("x", MaxSecretManifestBytes+1)); err == nil {
		t.Fatal("oversized manifest was accepted")
	}
	for _, duplicate := range []string{
		`{"revision":1,"revision":2,"entries":[]}`,
		`{"revision":1,"entries":[{"owner_scope":"owner-a","owner_scope":"owner-b","destination_id":"dest-a","key_reference":"key-a","secret":"whsec_` + encoded + `"}]}`,
	} {
		if _, err := ParseSecretManifest(duplicate); err == nil {
			t.Fatal("duplicate JSON field was accepted")
		}
	}
}
