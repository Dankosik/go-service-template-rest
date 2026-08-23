// profile:inbound-webhooks-standard:start
package postgresinboundwebhook

import (
	"encoding/base64"
	"strings"
	"testing"
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
	parsedEndpoints, err := ParseEndpointManifest(endpoints)
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
	if _, ok := trust.Lookup("orders"); !ok {
		t.Fatal("exact orders lookup failed")
	}
	if _, ok := trust.Lookup("ORDERS"); ok {
		t.Fatal("lookup folded case")
	}
	if formatted := parsedEndpoints.IDs(); strings.Contains(strings.Join(formatted, ","), string(keyA)) {
		t.Fatal("endpoint IDs leaked key bytes")
	}

	if _, err := ParseSecretManifest(`{"entries":[{"endpoint_id":"orders","key_reference":"a","secret":"whsec_` + base64.StdEncoding.EncodeToString(keyA) + `"},{"endpoint_id":"other","key_reference":"b","secret":"whsec_` + base64.StdEncoding.EncodeToString(keyA) + `"}]}`); err == nil {
		t.Fatal("cross-endpoint secret reuse accepted")
	}
	if _, err := ParseEndpointManifest(`{"endpoints":[{"endpoint_id":"orders","active_key_reference":"same","predecessor_key_reference":"same"}]}`); err == nil {
		t.Fatal("equal rotation keys accepted")
	}
}

// profile:inbound-webhooks-standard:end
