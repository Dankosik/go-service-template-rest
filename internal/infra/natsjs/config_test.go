package natsjs

import (
	"errors"
	"testing"
	"time"
)

const (
	testMaxPayloadBytes  = 256 << 10
	testMaxPending       = 64
	testMaxConcurrency   = 8
	testMaxDeliveryBytes = 1 << 20
)

func testConfig() Config {
	return Config{MaxPayloadBytes: testMaxPayloadBytes, MaxPendingPublishes: testMaxPending}
}

func testWorkerConfig() WorkerConfig {
	return WorkerConfig{
		MaxConcurrency:       testMaxConcurrency,
		MaxDeliveryBytes:     testMaxDeliveryBytes,
		HandlerTimeout:       30 * time.Second,
		RetryDelays:          []time.Duration{time.Second, 5 * time.Second, 30 * time.Second, 2 * time.Minute},
		DeadLetterRetryDelay: 30 * time.Second,
	}
}

func TestConfigValidation(t *testing.T) {
	valid := testConfig()
	valid.URLs = []string{"tls://nats.example:4222"}
	valid.CredentialsFile = "/run/secrets/nats.creds"
	valid.Stream = "EVENTS"
	if err := validateConfig(valid); err != nil {
		t.Fatalf("validateConfig(valid) error = %v", err)
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
			if err := validateConfig(cfg); !errors.Is(err, ErrRejected) {
				t.Fatalf("validateConfig() error = %v, want ErrRejected", err)
			}
		})
	}

	plaintext := valid
	plaintext.URLs = []string{"nats://127.0.0.1:4222"}
	plaintext.CredentialsFile = ""
	plaintext.AllowPlaintext = true
	plaintext.AllowUnauthenticated = true
	if err := validateConfig(plaintext); err != nil {
		t.Fatalf("validateConfig(explicit local escape hatches) error = %v", err)
	}
}

func TestWorkerAdmissionBound(t *testing.T) {
	cfg := testWorkerConfig()
	cfg.Consumer = "events-worker"
	cfg.FilterSubject = "events.>"
	cfg.DeadLetterSubject = "dead.events"
	if err := ValidateWorkerConfig(cfg, testMaxPayloadBytes); err != nil {
		t.Fatalf("ValidateWorkerConfig(valid) error = %v", err)
	}

	cfg.MaxConcurrency = ResidentDeliveryLimit/cfg.MaxDeliveryBytes + 1
	if err := ValidateWorkerConfig(cfg, testMaxPayloadBytes); !errors.Is(err, ErrRejected) {
		t.Fatalf("ValidateWorkerConfig(over resident bound) error = %v, want ErrRejected", err)
	}

	cfg = testWorkerConfig()
	cfg.Consumer = "events-worker"
	cfg.FilterSubject = "events.>"
	cfg.DeadLetterSubject = "events.dead"
	if err := ValidateWorkerConfig(cfg, testMaxPayloadBytes); !errors.Is(err, ErrRejected) {
		t.Fatalf("ValidateWorkerConfig(overlapping DLQ) error = %v, want ErrRejected", err)
	}

	cfg.DeadLetterSubject = "dead.events"
	cfg.RetryDelays = []time.Duration{0}
	if err := ValidateWorkerConfig(cfg, testMaxPayloadBytes); !errors.Is(err, ErrRejected) {
		t.Fatalf("ValidateWorkerConfig(zero retry) error = %v, want ErrRejected", err)
	}
}
