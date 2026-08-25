// profile:messaging-nats-jetstream:start
package messagingconfig

import "testing"

func TestRules(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name      string
		url       string
		plaintext bool
		valid     bool
	}{
		{name: "tls", url: "tls://nats.example:4222", valid: true},
		{name: "wss", url: "wss://nats.example:443", valid: true},
		{name: "plaintext opted in", url: "nats://127.0.0.1:4222", plaintext: true, valid: true},
		{name: "missing host", url: "tls://"},
		{name: "userinfo", url: "tls://user@nats.example:4222"},
		{name: "plaintext", url: "nats://nats.example:4222"},
		{name: "unsupported", url: "amqp://broker.example:5672"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateURL(test.url, test.plaintext)
			if (err == nil) != test.valid {
				t.Fatalf("ValidateURL(%q) error = %v, valid = %v", test.url, err, test.valid)
			}
		})
	}

	if !ValidStreamOrConsumerName("EVENTS") || ValidStreamOrConsumerName("EVENTS.BAD") {
		t.Fatal("stream or consumer name rule drifted")
	}
}

// profile:messaging-nats-jetstream:end
