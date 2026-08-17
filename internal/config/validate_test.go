package config

import (
	"errors"
	"testing"
	"time"
)

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
func TestReadDurationParsesDefaultDurations(t *testing.T) {
	t.Parallel()
	resetConfigEnv(t)

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.HTTP.ReadTimeout != 5*time.Second {
		t.Fatalf("HTTP.ReadTimeout = %s, want 5s", cfg.HTTP.ReadTimeout)
	}
	// profile:database-postgres:start
	if cfg.Postgres.ConnMaxLifetime != 30*time.Minute {
		t.Fatalf("Postgres.ConnMaxLifetime = %s, want 30m", cfg.Postgres.ConnMaxLifetime)
	}
	// profile:database-postgres:end
}

func TestHealthRefreshBounds(t *testing.T) {

	for _, tc := range []struct {
		name      string
		interval  string
		threshold string
		wantErr   bool
	}{
		{name: "defaults accepted"},
		{name: "interval too small", interval: "50ms", wantErr: true},
		{name: "interval too large", interval: "2m", wantErr: true},
		{name: "threshold zero", threshold: "0", wantErr: true},
		{name: "threshold too large", threshold: "101", wantErr: true},
		{name: "threshold one accepted", threshold: "1"},
	} {
		t.Run(tc.name, func(t *testing.T) {

			resetConfigEnv(t)
			if tc.interval != "" {
				t.Setenv("APP__HEALTH__REFRESH_INTERVAL", tc.interval)
			}
			if tc.threshold != "" {
				t.Setenv("APP__HEALTH__FAILURE_THRESHOLD", tc.threshold)
			}

			_, _, err := LoadDetailed(LoadOptions{})
			if tc.wantErr && !errors.Is(err, ErrValidate) {
				t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("LoadDetailed() error = %v", err)
			}
		})
	}
}

func TestRuntimeMemoryLimitRatioBounds(t *testing.T) {

	for _, tc := range []struct {
		name    string
		ratio   string
		wantErr bool
	}{
		{name: "default accepted"},
		{name: "zero disables detection", ratio: "0"},
		{name: "one accepted", ratio: "1"},
		{name: "negative", ratio: "-0.1", wantErr: true},
		{name: "above one", ratio: "1.1", wantErr: true},
		{name: "not a number", ratio: "NaN", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {

			resetConfigEnv(t)
			if tc.ratio != "" {
				t.Setenv("APP__RUNTIME__MEMORY_LIMIT_RATIO", tc.ratio)
			}

			_, _, err := LoadDetailed(LoadOptions{})
			if tc.wantErr && err == nil {
				t.Fatal("LoadDetailed() error = nil, want non-nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("LoadDetailed() error = %v", err)
			}
		})
	}
}
