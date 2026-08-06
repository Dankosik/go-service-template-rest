package config

import (
	"errors"
	"reflect"
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
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			enableOutbox(t)
			t.Setenv(test.key, test.value)
			_, _, err := LoadDetailed(LoadOptions{})
			if !errors.Is(err, ErrValidate) || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("LoadDetailed() error = %v, want %s validation", err, test.want)
			}
		})
	}
}

// Every OutboxConfig field must be rejected when it carries an unusable value.
// validateOutbox checks fields by listing them, so a field added to the struct
// and to bootstrap's relayConfig mapper but forgotten in those lists reaches the
// relay unvalidated: nothing rejects it at load time, and ValidateRelayConfig
// only covers the fields it happens to range over.
//
// Deriving the field set by reflection rather than listing it is the whole point.
// Two tests in cmd/outbox-relay walk the same struct for the halves this one
// cannot reach, because depguard keeps postgresoutbox out of this package:
// TestRelayConfigMapsEveryOutboxField guards the mapper and
// TestRelayRejectsEveryOutboxBudget the relay's own validator. The three together
// cover every field end to end.
func TestOutboxConfigValidatesEveryField(t *testing.T) {
	// The only field with no unusable value: it is the switch that turns the
	// rest on, and TestOutboxConfigDefaultsAndPostgresRequirement covers what it
	// gates. Anything else added here needs a reason.
	unvalidatable := map[string]string{"enabled": "a switch has no invalid value"}

	for _, field := range reflect.VisibleFields(reflect.TypeFor[OutboxConfig]()) {
		key := field.Tag.Get("koanf")
		if key == "" {
			t.Fatalf("OutboxConfig.%s has no koanf tag, so no operator can set it", field.Name)
		}
		if _, skipped := unvalidatable[key]; skipped {
			continue
		}
		var unusable string
		switch field.Type {
		case reflect.TypeFor[time.Duration]():
			unusable = "0s"
		case reflect.TypeFor[int]():
			unusable = "0"
		default:
			t.Fatalf("OutboxConfig.%s is a %s; teach this test an unusable value for it", field.Name, field.Type)
		}

		t.Run(key, func(t *testing.T) {
			enableOutbox(t)
			t.Setenv("APP__OUTBOX__"+strings.ToUpper(key), unusable)
			_, _, err := LoadDetailed(LoadOptions{})
			if !errors.Is(err, ErrValidate) || !strings.Contains(err.Error(), "outbox."+key) {
				t.Fatalf("LoadDetailed(outbox.%s=%s) error = %v, want an outbox.%s validation failure",
					key, unusable, err, key)
			}
		})
	}
}

// enableOutbox resets the environment to the smallest configuration the outbox
// loads under, so each case only has to set the one value it is testing.
func enableOutbox(t *testing.T) {
	t.Helper()
	resetConfigEnv(t)
	t.Setenv("APP__POSTGRES__ENABLED", "true")
	t.Setenv("APP__POSTGRES__DSN", "postgres://app:secret@localhost/app")
	t.Setenv("APP__OUTBOX__ENABLED", "true")
}
