package config

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestOutboxConfigDefaultsAndPostgresRequirement(t *testing.T) {
	resetConfigEnv(t)
	disabled, _, err := LoadDetailed(LoadOptions{})
	if err != nil || disabled.Outbox.Enabled {
		t.Fatalf("default disabled outbox = %+v, %v", disabled.Outbox, err)
	}

	resetConfigEnv(t)
	t.Setenv("APP__POSTGRES__ENABLED", "true")
	t.Setenv("APP__POSTGRES__DSN", "postgres://app:secret@localhost/app")
	t.Setenv("APP__OUTBOX__ENABLED", "true")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if !cfg.Outbox.Enabled || cfg.Outbox.MaxAttempts != 10 || cfg.Outbox.CleanupBatchSize != 1000 {
		t.Fatalf("outbox defaults = %+v", cfg.Outbox)
	}

	t.Setenv("APP__POSTGRES__MAX_OPEN_CONNS", "1")
	t.Setenv("APP__POSTGRES__MIN_IDLE_CONNS", "0")
	_, _, err = LoadDetailed(LoadOptions{})
	if !errors.Is(err, ErrValidate) || !strings.Contains(err.Error(), "postgres.max_open_conns >= 2") {
		t.Fatalf("single-connection outbox error = %v", err)
	}
	t.Setenv("APP__POSTGRES__MAX_OPEN_CONNS", "25")
	t.Setenv("APP__POSTGRES__MIN_IDLE_CONNS", "2")

	t.Setenv("APP__POSTGRES__ENABLED", "false")
	_, _, err = LoadDetailed(LoadOptions{})
	if !errors.Is(err, ErrValidate) || !strings.Contains(err.Error(), "outbox.enabled requires postgres.enabled") {
		t.Fatalf("invalid database/outbox combination error = %v", err)
	}
}

func TestOutboxConfigRejectsIncoherentBudgets(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		want  string
	}{
		{name: "lease total", key: "APP__OUTBOX__LEASE_DURATION", value: "20s", want: "outbox.lease_duration"},
		{name: "attempts below", key: "APP__OUTBOX__MAX_ATTEMPTS", value: "0", want: "outbox.max_attempts"},
		{name: "attempts above", key: "APP__OUTBOX__MAX_ATTEMPTS", value: "101", want: "outbox.max_attempts"},
		{name: "retry order", key: "APP__OUTBOX__RETRY_MAX", value: "500ms", want: "outbox.retry_max"},
		{name: "cleanup batch below", key: "APP__OUTBOX__CLEANUP_BATCH_SIZE", value: "0", want: "outbox.cleanup_batch_size"},
		{name: "cleanup batch above", key: "APP__OUTBOX__CLEANUP_BATCH_SIZE", value: "10001", want: "outbox.cleanup_batch_size"},
		{
			name: "lease sum overflow", key: "APP__OUTBOX__PUBLISH_TIMEOUT",
			value: time.Duration(1<<63 - 1).String(), want: "outbox.lease_duration",
		},
	}
	for _, key := range []string{
		"POLL_INTERVAL", "PUBLISH_TIMEOUT", "LEASE_DURATION", "RETRY_BASE", "RETRY_MAX",
		"OBSERVATION_INTERVAL", "CLEANUP_INTERVAL", "PUBLISHED_RETENTION", "DRAIN_TIMEOUT",
	} {
		tests = append(tests, struct {
			name  string
			key   string
			value string
			want  string
		}{
			name:  "positive " + strings.ToLower(key),
			key:   "APP__OUTBOX__" + key,
			value: "0s",
			want:  "outbox." + strings.ToLower(key),
		})
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv("APP__POSTGRES__ENABLED", "true")
			t.Setenv("APP__POSTGRES__DSN", "postgres://app:secret@localhost/app")
			t.Setenv("APP__OUTBOX__ENABLED", "true")
			t.Setenv(test.key, test.value)
			_, _, err := LoadDetailed(LoadOptions{})
			if !errors.Is(err, ErrValidate) || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("LoadDetailed() error = %v, want %s validation", err, test.want)
			}
		})
	}
}
