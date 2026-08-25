// profile:inbound-webhooks-standard:start
package manifest

import "testing"

func TestParseEndpoints(t *testing.T) {
	t.Parallel()

	parsed, err := ParseEndpoints(`{"endpoints":[{"endpoint_id":"orders","active_key_reference":"key-v1"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if endpoint, ok := parsed.Lookup("orders"); !ok || endpoint.ActiveKeyReference != "key-v1" {
		t.Fatalf("Lookup(orders) = %+v, %v", endpoint, ok)
	}
	for _, raw := range []string{
		`{"endpoints":[{"endpoint_id":"bad id","active_key_reference":"key-v1"}]}`,
		`{"endpoints":[{"endpoint_id":"orders","endpoint_id":"other","active_key_reference":"key-v1"}]}`,
		`{"endpoints":[{"endpoint_id":"orders","active_key_reference":"key-v1","unknown":true}]}`,
		`{"endpoints":[{"endpoint_id":"orders","active_key_reference":"same","predecessor_key_reference":"same"}]}`,
		`{"endpoints":[{"endpoint_id":"orders","active_key_reference":"key-v1"},{"endpoint_id":"orders","active_key_reference":"key-v2"}]}`,
	} {
		if _, err := ParseEndpoints(raw); err == nil {
			t.Fatalf("ParseEndpoints(%s) succeeded", raw)
		}
	}
}

// profile:inbound-webhooks-standard:end
