package natsjs

import (
	"errors"
	"testing"
	"time"
)

const (
	testMaxPayloadBytes = 256 << 10
	testMaxPending      = 64
)

func testConfig() Config {
	return Config{
		MinStreamReplicas: 1, MinStreamRetention: 24 * time.Hour,
		MaxPayloadBytes: testMaxPayloadBytes, MaxPendingPublishes: testMaxPending,
	}
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
		"zero replicas":       func(cfg *Config) { cfg.MinStreamReplicas = 0 },
		"too many replicas":   func(cfg *Config) { cfg.MinStreamReplicas = 6 },
		"zero retention":      func(cfg *Config) { cfg.MinStreamRetention = 0 },
		"zero payload":        func(cfg *Config) { cfg.MaxPayloadBytes = 0 },
		"zero pending":        func(cfg *Config) { cfg.MaxPendingPublishes = 0 },
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
