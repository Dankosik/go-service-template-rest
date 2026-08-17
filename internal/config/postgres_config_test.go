package //nolint:paralleltest // This test mutates process-global environment or working directory.

// The three tests below hold the budgets this section shares with HTTP. They
// live here rather than in validate_test.go because validatePostgres owns those
// rules, and because a build profile that removes Postgres removes this file
// whole instead of cutting marked blocks out of a shared one.
//nolint:paralleltest // This test mutates process-global environment or working directory.

// TestAcquireTimeoutMustLeaveQueryBudget keeps the two numbers one budget. An
// acquire budget at or above the request budget is not a bound: a caller that
// waited it out has nothing left to run a query with, which is the unbounded
// wait this setting replaced.
config

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
