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

func TestValidEndpointID(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name  string
		value string
		want  bool
	}{
		{name: "allowed ASCII token", value: "orders_2026-aZ", want: true},
		{name: "empty", value: "", want: false},
		{name: "too long", value: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", want: false},
		{name: "non ASCII", value: "orders-é", want: false},
		{name: "invalid ASCII punctuation", value: "orders:2026", want: false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := ValidEndpointID(testCase.value); got != testCase.want {
				t.Fatalf("ValidEndpointID(%q) = %t, want %t", testCase.value, got, testCase.want)
			}
		})
	}
}

// profile:inbound-webhooks-standard:end
