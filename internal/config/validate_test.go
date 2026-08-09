package config

import (
	"errors"
	"strings"
	"testing"
	"time"
)

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
func TestReadDurationParsesDefaultDurations(t *testing.T) {
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

func TestPprofRequiresDiagnosticsListener(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP__OBSERVABILITY__PPROF__ENABLED", "true")
	t.Setenv("APP__OBSERVABILITY__METRICS__ADDR", "")

	_, _, err := LoadDetailed(LoadOptions{})
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "observability.metrics.addr") {
		t.Fatalf("error = %q, want it to name observability.metrics.addr", err.Error())
	}
}

// The tests below cross two sections, so they belong to validateCrossSectionBudgets
// here rather than to either section's own file.

// profile:database-postgres:start
func TestStatementTimeoutMustFitRequestBudget(t *testing.T) {
	for _, tc := range []struct {
		name             string
		statementTimeout string
		requestTimeout   string
		wantErr          bool
	}{
		{name: "defaults are coherent"},
		{name: "equal budgets allowed", statementTimeout: "8s", requestTimeout: "8s"},
		{name: "smaller statement budget allowed", statementTimeout: "3s", requestTimeout: "8s"},
		{name: "statement budget outlives request", statementTimeout: "9s", requestTimeout: "8s", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			if tc.statementTimeout != "" {
				t.Setenv("APP__POSTGRES__STATEMENT_TIMEOUT", tc.statementTimeout)
			}
			if tc.requestTimeout != "" {
				t.Setenv("APP__HTTP__REQUEST_TIMEOUT", tc.requestTimeout)
			}
			t.Setenv("APP__POSTGRES__ENABLED", "true")
			t.Setenv("APP__POSTGRES__DSN", "postgres://app:app@127.0.0.1:5432/app?sslmode=disable")

			_, _, err := LoadDetailed(LoadOptions{})
			if tc.wantErr {
				if !errors.Is(err, ErrValidate) {
					t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
				}
				if !strings.Contains(err.Error(), "postgres.statement_timeout") {
					t.Fatalf("error = %q, want it to name postgres.statement_timeout", err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadDetailed() error = %v", err)
			}
		})
	}
}

func TestHTTPAdmissionAndPoolCapacityAreIndependent(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP__POSTGRES__ENABLED", "true")
	t.Setenv("APP__POSTGRES__DSN", "postgres://app:app@127.0.0.1:5432/app?sslmode=disable")
	t.Setenv("APP__POSTGRES__MAX_OPEN_CONNS", "25")
	t.Setenv("APP__HTTP__MAX_IN_FLIGHT", "10")

	if _, _, err := LoadDetailed(LoadOptions{}); err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
}

// TestAcquireTimeoutMustLeaveQueryBudget keeps the two numbers one budget. An
// acquire budget at or above the request budget is not a bound: a caller that
// waited it out has nothing left to run a query with, which is the unbounded
// wait this setting replaced.
func TestAcquireTimeoutMustLeaveQueryBudget(t *testing.T) {
	for _, tc := range []struct {
		name           string
		acquireTimeout string
		requestTimeout string
		wantErr        bool
	}{
		{name: "defaults are coherent"},
		{name: "small slice of the budget allowed", acquireTimeout: "1s", requestTimeout: "8s"},
		{name: "equal budgets rejected", acquireTimeout: "8s", requestTimeout: "8s", wantErr: true},
		{name: "acquire budget outlives request", acquireTimeout: "9s", requestTimeout: "8s", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			if tc.acquireTimeout != "" {
				t.Setenv("APP__POSTGRES__ACQUIRE_TIMEOUT", tc.acquireTimeout)
			}
			if tc.requestTimeout != "" {
				t.Setenv("APP__HTTP__REQUEST_TIMEOUT", tc.requestTimeout)
			}
			t.Setenv("APP__POSTGRES__ENABLED", "true")
			t.Setenv("APP__POSTGRES__DSN", "postgres://app:app@127.0.0.1:5432/app?sslmode=disable")

			_, _, err := LoadDetailed(LoadOptions{})
			if tc.wantErr {
				if !errors.Is(err, ErrValidate) {
					t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
				}
				if !strings.Contains(err.Error(), "postgres.acquire_timeout") {
					t.Fatalf("error = %q, want it to name postgres.acquire_timeout", err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadDetailed() error = %v", err)
			}
		})
	}
}

// profile:database-postgres:end
