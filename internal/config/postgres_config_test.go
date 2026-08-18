package config

import (
	"errors"
	"strings"
	"testing"
)

func TestPostgresMaxOpenConnsMustStayWithinRange(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP__POSTGRES__MAX_OPEN_CONNS", "501")

	_, _, err := LoadDetailed(LoadOptions{})
	if !errors.Is(err, ErrValidate) || !strings.Contains(err.Error(), "postgres.max_open_conns must be in range") {
		t.Fatalf("LoadDetailed() error = %v, want postgres max-open-conns range error", err)
	}
}

func TestPostgresDSNParseIsAdapterOwned(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP__POSTGRES__ENABLED", "true")
	t.Setenv("APP__POSTGRES__DSN", "postgres://%zz")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v, want adapter-owned DSN parsing", err)
	}
	if cfg.Postgres.DSN != "postgres://%zz" {
		t.Fatalf("Postgres.DSN = %q, want raw invalid DSN", cfg.Postgres.DSN)
	}
}

func TestRemovedPostgresTuningKeyIsUnknown(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP__POSTGRES__ACQUIRE_TIMEOUT", "1s")

	_, _, err := LoadDetailed(LoadOptions{})
	if !errors.Is(err, ErrUnknownKey) || !strings.Contains(err.Error(), "postgres.acquire_timeout") {
		t.Fatalf("LoadDetailed() error = %v, want removed key to fail closed", err)
	}
}
