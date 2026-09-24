// profile:inbound-webhooks-standard:start
package postgresinboundwebhook

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	inboundmanifest "github.com/example/go-service-template-rest/internal/inboundwebhook/manifest"
)

func TestEndpointManifestSecurityBoundary(t *testing.T) {
	t.Parallel()

	keyA := []byte("0123456789abcdef0123456789abcdef")
	keyB := []byte("fedcba9876543210fedcba9876543210")
	pred := []byte("abcdef0123456789abcdef0123456789")
	endpoints := `{"endpoints":[{"endpoint_id":"orders","active_key_reference":"key-v1","predecessor_key_reference":"key-v0"},{"endpoint_id":"Orders","active_key_reference":"key-v2"}]}`
	secrets := `{"entries":[` +
		`{"endpoint_id":"orders","key_reference":"key-v1","secret":"whsec_` + base64.StdEncoding.EncodeToString(keyA) + `"},` +
		`{"endpoint_id":"orders","key_reference":"key-v0","secret":"whsec_` + base64.StdEncoding.EncodeToString(pred) + `"},` +
		`{"endpoint_id":"Orders","key_reference":"key-v2","secret":"whsec_` + base64.StdEncoding.EncodeToString(keyB) + `"}` +
		`]}`
	parsedEndpoints, err := inboundmanifest.ParseEndpoints(endpoints)
	if err != nil {
		t.Fatal(err)
	}
	parsedSecrets, err := ParseSecretManifest(secrets)
	if err != nil {
		t.Fatal(err)
	}
	trust, err := BindSecrets(parsedEndpoints, parsedSecrets)
	if err != nil {
		t.Fatal(err)
	}
	endpoint, ok := trust.Lookup("orders")
	if !ok || endpoint.ActiveKeyReference != "key-v1" || endpoint.PredecessorKeyReference != "key-v0" {
		t.Fatal("exact orders lookup failed")
	}
	if _, ok := trust.Lookup("ORDERS"); ok {
		t.Fatal("lookup folded case")
	}
	if formatted := parsedEndpoints.IDs(); strings.Contains(strings.Join(formatted, ","), string(keyA)) {
		t.Fatal("endpoint IDs leaked key bytes")
	}
	parsedSecrets.secrets["orders"]["key-v1"][0] ^= 0xff
	parsedSecrets.secrets["orders"]["key-v0"][0] ^= 0xff
	bound, ok := trust.secretsFor("orders")
	if !ok || !bytes.Equal(bound.active, keyA) || !bytes.Equal(bound.predecessor, pred) {
		t.Fatal("bound secrets changed with source manifest")
	}

	if _, err := ParseSecretManifest(`{"entries":[{"endpoint_id":"orders","key_reference":"a","secret":"whsec_` + base64.StdEncoding.EncodeToString(keyA) + `"},{"endpoint_id":"other","key_reference":"b","secret":"whsec_` + base64.StdEncoding.EncodeToString(keyA) + `"}]}`); err == nil {
		t.Fatal("cross-endpoint secret reuse accepted")
	}
	if _, err := inboundmanifest.ParseEndpoints(`{"endpoints":[{"endpoint_id":"orders","active_key_reference":"same","predecessor_key_reference":"same"}]}`); err == nil {
		t.Fatal("equal rotation keys accepted")
	}
}

func TestSecretManifestRejectsInvalidBindings(t *testing.T) {
	t.Parallel()

	empty, err := ParseSecretManifest("")
	if err != nil || len(empty.secrets) != 0 {
		t.Fatalf("ParseSecretManifest(empty) = %v, %v, want empty manifest", empty, err)
	}
	secret := "whsec_" + base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	entry := `{"endpoint_id":"orders","key_reference":"key-v1","secret":"` + secret + `"}`
	for _, tc := range []struct {
		name string
		raw  string
	}{
		{name: "invalid encoding", raw: `{"entries":[{"endpoint_id":"orders","key_reference":"key-v1","secret":"whsec_***"}]}`},
		{name: "duplicate binding", raw: `{"entries":[` + entry + `,` + entry + `]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseSecretManifest(tc.raw)
			if err == nil {
				t.Fatal("ParseSecretManifest() accepted an invalid binding")
			}
		})
	}
}

// profile:inbound-webhooks-standard:end
