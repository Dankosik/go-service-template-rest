package natsjs

import (
	"errors"
	"testing"
	"time"
)

func TestConfigValidation(t *testing.T) {
	valid := DefaultConfig()
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

func TestWorkerAdmissionBound(t *testing.T) {
	cfg := DefaultWorkerConfig()
	cfg.Consumer = "events-worker"
	cfg.FilterSubject = "events.>"
	cfg.DeadLetterSubject = "dead.events"
	if err := ValidateWorkerConfig(cfg, DefaultMaxPayloadBytes); err != nil {
		t.Fatalf("ValidateWorkerConfig(valid) error = %v", err)
	}

	cfg.MaxConcurrency = ResidentDeliveryLimit/cfg.MaxDeliveryBytes + 1
	if err := ValidateWorkerConfig(cfg, DefaultMaxPayloadBytes); !errors.Is(err, ErrRejected) {
		t.Fatalf("ValidateWorkerConfig(over resident bound) error = %v, want ErrRejected", err)
	}

	cfg = DefaultWorkerConfig()
	cfg.Consumer = "events-worker"
	cfg.FilterSubject = "events.>"
	cfg.DeadLetterSubject = "events.dead"
	if err := ValidateWorkerConfig(cfg, DefaultMaxPayloadBytes); !errors.Is(err, ErrRejected) {
		t.Fatalf("ValidateWorkerConfig(overlapping DLQ) error = %v, want ErrRejected", err)
	}

	cfg.DeadLetterSubject = "dead.events"
	cfg.RetryDelays = []time.Duration{0}
	if err := ValidateWorkerConfig(cfg, DefaultMaxPayloadBytes); !errors.Is(err, ErrRejected) {
		t.Fatalf("ValidateWorkerConfig(zero retry) error = %v, want ErrRejected", err)
	}
}
