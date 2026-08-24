package postgreswebhook

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestWebhookManifests(t *testing.T) {
	endpoints, err := ParseEndpointManifest(`{"endpoints":[{"owner_scope":"orders","receiver_id":"alpha","generation":1,"url":"https://alpha.example/hooks","active_key_reference":"key-v1"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	endpoint, err := endpoints.resolve("orders", "alpha")
	if err != nil || endpoint.URL != "https://alpha.example:443/hooks" {
		t.Fatalf("Resolve() = %+v, %v", endpoint, err)
	}

	secret := "whsec_" + base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	secrets, err := ParseSecretManifest(`{"entries":[{"owner_scope":"orders","receiver_id":"alpha","key_reference":"key-v1","secret":"` + secret + `"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	key, err := secrets.resolve("orders", "alpha", "key-v1")
	if err != nil || len(key) != 32 {
		t.Fatalf("Resolve() = %+v, %v", key, err)
	}

	for _, raw := range []string{
		`{"endpoints":[{"owner_scope":"orders","owner_scope":"other","receiver_id":"alpha","generation":1,"url":"https://alpha.example","active_key_reference":"key-v1"}]}`,
		`{"endpoints":[{"owner_scope":"orders","receiver_id":"alpha","generation":1,"url":"http://alpha.example/hooks","active_key_reference":"key-v1"}]}`,
		`{"endpoints":[{"owner_scope":"orders","receiver_id":"alpha","generation":1,"url":"https://alpha.example/hooks","active_key_reference":"key-v1","unknown":true}]}`,
		`{"Endpoints":[{"owner_scope":"orders","receiver_id":"alpha","generation":1,"url":"https://alpha.example/hooks","active_key_reference":"key-v1"}]}`,
		`{"endpoints":[]} {}`,
		`{"endpoints":[{"owner_scope":"` + string([]byte{0xff}) + `","receiver_id":"alpha","generation":1,"url":"https://alpha.example/hooks","active_key_reference":"key-v1"}]}`,
	} {
		if _, err := ParseEndpointManifest(raw); err == nil {
			t.Fatalf("ParseEndpointManifest(%s) succeeded", raw)
		}
	}
	duplicateSecretField := strings.ReplaceAll(`{"entries":[{"owner_scope":"orders","receiver_id":"alpha","key_reference":"key-v1","secret":"SECRET","secret":"SECRET"}]}`, "SECRET", secret)
	if _, err := ParseSecretManifest(duplicateSecretField); err == nil {
		t.Fatal("duplicate secret field succeeded")
	}
	if _, err := ParseSecretManifest(strings.ReplaceAll(`{"entries":[{"owner_scope":"orders","receiver_id":"alpha","key_reference":"key-v1","secret":"SECRET"},{"owner_scope":"other","receiver_id":"beta","key_reference":"key-v2","secret":"SECRET"}]}`, "SECRET", secret)); err == nil {
		t.Fatal("cross-bound secret reuse succeeded")
	}
}
