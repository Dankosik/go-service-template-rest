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
	valid := testWorkerConfig()
	valid.Consumer = "events-worker"
	valid.FilterSubject = "events.>"
	valid.DeadLetterSubject = "dead.events"
	if err := ValidateWorkerConfig(valid, testMaxPayloadBytes); err != nil {
		t.Fatalf("ValidateWorkerConfig(valid) error = %v", err)
	}

	cases := map[string]func(*WorkerConfig){
		"invalid consumer":        func(cfg *WorkerConfig) { cfg.Consumer = "events worker" },
		"invalid filter":          func(cfg *WorkerConfig) { cfg.FilterSubject = "events.>.invalid" },
		"invalid dead letter":     func(cfg *WorkerConfig) { cfg.DeadLetterSubject = "dead.*" },
		"overlapping dead letter": func(cfg *WorkerConfig) { cfg.DeadLetterSubject = "events.dead" },
		"zero concurrency":        func(cfg *WorkerConfig) { cfg.MaxConcurrency = 0 },
		"zero delivery bound":     func(cfg *WorkerConfig) { cfg.MaxDeliveryBytes = 0 },
		"undersized envelope":     func(cfg *WorkerConfig) { cfg.MaxDeliveryBytes = testMaxPayloadBytes + HeaderLimitBytes - 1 },
		"resident bound":          func(cfg *WorkerConfig) { cfg.MaxConcurrency = ResidentDeliveryLimit/cfg.MaxDeliveryBytes + 1 },
		"zero handler timeout":    func(cfg *WorkerConfig) { cfg.HandlerTimeout = 0 },
		"no retry delays":         func(cfg *WorkerConfig) { cfg.RetryDelays = nil },
		"too many retry delays": func(cfg *WorkerConfig) {
			cfg.RetryDelays = []time.Duration{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		},
		"zero retry delay":       func(cfg *WorkerConfig) { cfg.RetryDelays = []time.Duration{0} },
		"zero dead letter retry": func(cfg *WorkerConfig) { cfg.DeadLetterRetryDelay = 0 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := valid
			mutate(&cfg)
			if err := ValidateWorkerConfig(cfg, testMaxPayloadBytes); !errors.Is(err, ErrRejected) {
				t.Fatalf("ValidateWorkerConfig() error = %v, want ErrRejected", err)
			}
		})
	}
}
