package natsjs

import (
	"errors"
	"testing"
)

const (
	testMaxPayloadBytes  = 256 << 10
	testMaxPending       = 64
	testMaxDeliveryBytes = testMaxPayloadBytes + HeaderLimitBytes
)

func testConfig() Config {
	return Config{MaxPayloadBytes: testMaxPayloadBytes}
}

func testWorkerConfig() WorkerConfig {
	return DefaultWorkerConfig("events-worker", "events.>", "dead.events", 8, testMaxPayloadBytes)
}

func TestConfigValidation(t *testing.T) {
	valid := testConfig()
	valid.URLs = []string{"tls://nats.example:4222"}
	valid.CredentialsFile = "/run/secrets/nats.creds"
	valid.Stream = "EVENTS"
	if err := ValidateConfig(valid); err != nil {
		t.Fatalf("ValidateConfig(valid) error = %v", err)
	}

	cases := map[string]func(*Config){
		"missing URLs":        func(cfg *Config) { cfg.URLs = nil },
		"URL userinfo":        func(cfg *Config) { cfg.URLs = []string{"tls://user@nats.example:4222"} },
		"plaintext":           func(cfg *Config) { cfg.URLs = []string{"nats://nats.example:4222"} },
		"missing credentials": func(cfg *Config) { cfg.CredentialsFile = "" },
		"invalid stream":      func(cfg *Config) { cfg.Stream = "EVENTS.BAD" },
		"zero payload":        func(cfg *Config) { cfg.MaxPayloadBytes = 0 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := valid
			mutate(&cfg)
			if err := ValidateConfig(cfg); !errors.Is(err, ErrRejected) {
				t.Fatalf("ValidateConfig() error = %v, want ErrRejected", err)
			}
		})
	}

	plaintext := valid
	plaintext.URLs = []string{"nats://127.0.0.1:4222"}
	plaintext.CredentialsFile = ""
	plaintext.AllowPlaintext = true
	plaintext.AllowUnauthenticated = true
	if err := ValidateConfig(plaintext); err != nil {
		t.Fatalf("ValidateConfig(explicit local escape hatches) error = %v", err)
	}
}
