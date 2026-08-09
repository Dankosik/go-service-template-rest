package config

import (
	"errors"
	"strings"
	"testing"
)

func TestPostgresDurationBounds(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__POSTGRES__CONNECT_TIMEOUT", "50ms")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatal("LoadDetailed() expected validation error for connect timeout")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
}

func TestPostgresMaxOpenConnsMustStayWithinRange(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__POSTGRES__MAX_OPEN_CONNS", "501")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatal("LoadDetailed() expected validation error for postgres max open conns")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "postgres.max_open_conns must be in range") {
		t.Fatalf("error = %v, want postgres max open conns range policy", err)
	}
}

func TestMigrationTimeoutsMustFitOverallBudget(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__POSTGRES__MIGRATION_TIMEOUT", "30s")
	t.Setenv("APP__POSTGRES__MIGRATION_STATEMENT_TIMEOUT", "31s")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatal("LoadDetailed() error = nil, want migration budget error")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "postgres.migration_statement_timeout") {
		t.Fatalf("error = %v, want statement budget name", err)
	}
}

func TestMigrationLockTimeoutMustLeaveCleanupReserve(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__POSTGRES__MIGRATION_TIMEOUT", "30s")
	t.Setenv("APP__POSTGRES__MIGRATION_STATEMENT_TIMEOUT", "20s")
	t.Setenv("APP__POSTGRES__MIGRATION_LOCK_TIMEOUT", "30s")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatal("LoadDetailed() error = nil, want cleanup reserve error")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "reserve cleanup time") {
		t.Fatalf("error = %v, want cleanup reserve context", err)
	}
}

func TestPostgresDSNParseIsAdapterOwned(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__POSTGRES__ENABLED", "true")
	t.Setenv("APP__POSTGRES__DSN", "postgres://%zz")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v, want nil because driver-specific parsing is adapter-owned", err)
	}
	if cfg.Postgres.DSN != "postgres://%zz" {
		t.Fatalf("Postgres.DSN = %q, want raw invalid DSN preserved for adapter-owned parsing", cfg.Postgres.DSN)
	}
}

func TestMinIdleConnsMustFitPool(t *testing.T) {
	for _, tc := range []struct {
		name         string
		minIdleConns string
		maxOpenConns string
		wantErr      bool
	}{
		{name: "defaults are coherent"},
		{name: "no warm floor allowed", minIdleConns: "0", maxOpenConns: "25"},
		{name: "whole pool warm allowed", minIdleConns: "25", maxOpenConns: "25"},
		{name: "warm floor above pool ceiling", minIdleConns: "26", maxOpenConns: "25", wantErr: true},
		{name: "negative warm floor", minIdleConns: "-1", maxOpenConns: "25", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			if tc.minIdleConns != "" {
				t.Setenv("APP__POSTGRES__MIN_IDLE_CONNS", tc.minIdleConns)
			}
			if tc.maxOpenConns != "" {
				t.Setenv("APP__POSTGRES__MAX_OPEN_CONNS", tc.maxOpenConns)
			}
			t.Setenv("APP__POSTGRES__ENABLED", "true")
			t.Setenv("APP__POSTGRES__DSN", "postgres://app:app@127.0.0.1:5432/app?sslmode=disable")

			_, _, err := LoadDetailed(LoadOptions{})
			if tc.wantErr {
				if !errors.Is(err, ErrValidate) {
					t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
				}
				if !strings.Contains(err.Error(), "postgres.min_idle_conns") {
					t.Fatalf("error = %q, want it to name postgres.min_idle_conns", err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadDetailed() error = %v", err)
			}
		})
	}
}
