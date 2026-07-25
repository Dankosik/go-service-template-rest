package config

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// profile:database-postgres:start
func TestPostgresDurationBounds(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__POSTGRES__CONNECT_TIMEOUT", "50ms")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for connect timeout")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
}

// profile:database-postgres:end

func TestMetricsAddressValidation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{name: "disabled", addr: ""},
		{name: "loopback", addr: "127.0.0.1:9090"},
		{name: "ipv6 loopback", addr: "[::1]:9090"},
		{name: "missing port", addr: "127.0.0.1", wantErr: true},
		{name: "zero port", addr: "127.0.0.1:0", wantErr: true},
		{name: "non numeric port", addr: "127.0.0.1:http", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv("APP__OBSERVABILITY__METRICS__ADDR", tc.addr)

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

// profile:database-postgres:start
func TestPostgresMaxOpenConnsMustStayWithinRange(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__POSTGRES__MAX_OPEN_CONNS", "501")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for postgres max open conns")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "postgres.max_open_conns must be in range") {
		t.Fatalf("error = %v, want postgres max open conns range policy", err)
	}
}

// profile:database-postgres:end

func TestShutdownTimeoutCanBeTunedWhenDrainBudgetIsValid(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__HTTP__SHUTDOWN_TIMEOUT", "45s")
	t.Setenv("APP__HTTP__READINESS_PROPAGATION_DELAY", "20s")
	t.Setenv("APP__HTTP__WRITE_TIMEOUT", "10s")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v, want nil for tuned shutdown timeout", err)
	}
	if cfg.HTTP.ShutdownTimeout != 45*time.Second {
		t.Fatalf("HTTP.ShutdownTimeout = %s, want 45s", cfg.HTTP.ShutdownTimeout)
	}
}

func TestShutdownTimeoutMustStayWithinRange(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__HTTP__SHUTDOWN_TIMEOUT", "500ms")
	t.Setenv("APP__HTTP__READINESS_PROPAGATION_DELAY", "0s")
	t.Setenv("APP__HTTP__WRITE_TIMEOUT", "100ms")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for shutdown timeout range")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "http.shutdown_timeout must be in range") {
		t.Fatalf("error = %v, want shutdown timeout range policy", err)
	}
}

func TestHTTPShutdownBudgetMustLeaveWriteDrainTime(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__HTTP__READINESS_PROPAGATION_DELAY", "25s")
	t.Setenv("APP__HTTP__WRITE_TIMEOUT", "10s")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for write timeout beyond drain budget")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "http.write_timeout must be <= effective drain budget") {
		t.Fatalf("error = %v, want explicit drain budget policy", err)
	}
}

func TestReadinessTimeoutMustNotExceedWriteTimeout(t *testing.T) {
	t.Run("greater readiness timeout rejects", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("APP__HTTP__READINESS_TIMEOUT", "6s")
		t.Setenv("APP__HTTP__REQUEST_TIMEOUT", "5s")
		t.Setenv("APP__HTTP__WRITE_TIMEOUT", "5s")

		_, _, err := LoadDetailed(LoadOptions{})
		if err == nil {
			t.Fatalf("LoadDetailed() expected validation error for readiness timeout beyond write timeout")
		}
		if !errors.Is(err, ErrValidate) {
			t.Fatalf("error = %v, want ErrValidate", err)
		}
		if !strings.Contains(err.Error(), "http.readiness_timeout must be <= http.write_timeout") {
			t.Fatalf("error = %v, want readiness/write timeout compatibility policy", err)
		}
	})

	for _, tc := range []struct {
		name             string
		readinessTimeout string
		writeTimeout     string
	}{
		{name: "equal timeout allows", readinessTimeout: "5s", writeTimeout: "5s"},
		{name: "lower readiness timeout allows", readinessTimeout: "4s", writeTimeout: "5s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv("APP__HTTP__READINESS_TIMEOUT", tc.readinessTimeout)
			t.Setenv("APP__HTTP__REQUEST_TIMEOUT", tc.writeTimeout)
			t.Setenv("APP__HTTP__WRITE_TIMEOUT", tc.writeTimeout)

			_, _, err := LoadDetailed(LoadOptions{})
			if err != nil {
				t.Fatalf("LoadDetailed() error = %v", err)
			}
		})
	}
}

// The readiness/health-check relationship is owned by
// bootstrap.validateStartupBudgetCompatibility, which enforces it with the
// startup headroom this package cannot see. It is proved there.

// profile:database-postgres:start
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

// profile:database-postgres:end

// profile:database-postgres:start
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

// profile:database-postgres:end

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

// The request budget must expire while the connection can still carry the 504
// that reports it, so it may not outlast the response write deadline.
func TestRequestTimeoutMustNotExceedWriteTimeout(t *testing.T) {
	t.Run("greater request timeout rejects", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("APP__HTTP__REQUEST_TIMEOUT", "6s")
		t.Setenv("APP__HTTP__WRITE_TIMEOUT", "5s")

		_, _, err := LoadDetailed(LoadOptions{})
		if err == nil {
			t.Fatalf("LoadDetailed() expected validation error for request timeout beyond write timeout")
		}
		if !errors.Is(err, ErrValidate) {
			t.Fatalf("error = %v, want ErrValidate", err)
		}
		if !strings.Contains(err.Error(), "http.request_timeout must be <= http.write_timeout") {
			t.Fatalf("error = %v, want request/write timeout compatibility policy", err)
		}
	})

	for _, tc := range []struct {
		name           string
		requestTimeout string
		writeTimeout   string
	}{
		{name: "equal timeout allows", requestTimeout: "5s", writeTimeout: "5s"},
		{name: "lower request timeout allows", requestTimeout: "4s", writeTimeout: "5s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv("APP__HTTP__READINESS_TIMEOUT", "1s")
			t.Setenv("APP__HTTP__REQUEST_TIMEOUT", tc.requestTimeout)
			t.Setenv("APP__HTTP__WRITE_TIMEOUT", tc.writeTimeout)

			_, _, err := LoadDetailed(LoadOptions{})
			if err != nil {
				t.Fatalf("LoadDetailed() error = %v", err)
			}
		})
	}
}

func TestRequestTimeoutRejectsOutOfRangeValues(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
	}{
		{name: "below lower bound", value: "50ms"},
		{name: "above upper bound", value: "11m"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv("APP__HTTP__REQUEST_TIMEOUT", tc.value)

			_, _, err := LoadDetailed(LoadOptions{})
			if err == nil {
				t.Fatalf("LoadDetailed() expected validation error for http.request_timeout = %q", tc.value)
			}
			if !errors.Is(err, ErrValidate) {
				t.Fatalf("error = %v, want ErrValidate", err)
			}
			if !strings.Contains(err.Error(), "http.request_timeout must be in range") {
				t.Fatalf("error = %v, want request timeout range policy", err)
			}
		})
	}
}
