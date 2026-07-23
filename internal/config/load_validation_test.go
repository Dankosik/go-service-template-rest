package config

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestStrictUnknownKeyRejects(t *testing.T) {
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
unknown:
  field: value
`)

	_, _, err := LoadDetailed(LoadOptions{
		ConfigPath: configPath,
		Strict:     true,
	})
	if err == nil {
		t.Fatalf("LoadDetailed() expected strict unknown key error")
	}
	if !errors.Is(err, ErrStrictUnknownKey) {
		t.Fatalf("error = %v, want ErrStrictUnknownKey", err)
	}
	if got := ErrorType(err); got != "strict_unknown_key" {
		t.Fatalf("ErrorType(error) = %q, want strict_unknown_key", got)
	}
}

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
func TestPermissiveUnknownKeyAllows(t *testing.T) {
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
unknown:
  field: value
`)

	_, report, err := LoadDetailed(LoadOptions{
		ConfigPath: configPath,
		Strict:     false,
	})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if !slices.Contains(report.UnknownKeyWarnings, "unknown.field") {
		t.Fatalf("UnknownKeyWarnings = %v, want unknown.field", report.UnknownKeyWarnings)
	}
}

func TestPermissiveUnknownKeyWarningsPreservedOnValidationError(t *testing.T) {
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
unknown:
  field: value
`)
	t.Setenv("APP__HTTP__ADDR", "")

	_, report, err := LoadDetailed(LoadOptions{
		ConfigPath: configPath,
		Strict:     false,
	})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !slices.Contains(report.UnknownKeyWarnings, "unknown.field") {
		t.Fatalf("UnknownKeyWarnings = %v, want unknown.field", report.UnknownKeyWarnings)
	}
}

func TestStrictUnknownKeyRejectsScalarSectionKeys(t *testing.T) {
	for _, tc := range []struct {
		name    string
		envKey  string
		wantKey string
	}{
		{name: "root section", envKey: "APP__HTTP", wantKey: "http"},
		{name: "nested section", envKey: "APP__OBSERVABILITY__OTEL", wantKey: "observability.otel"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv(tc.envKey, "oops")

			_, report, err := LoadDetailed(LoadOptions{Strict: true})
			if err == nil {
				t.Fatalf("LoadDetailed() expected strict unknown key error")
			}
			if !errors.Is(err, ErrStrictUnknownKey) {
				t.Fatalf("error = %v, want ErrStrictUnknownKey", err)
			}
			if !strings.Contains(err.Error(), tc.wantKey) {
				t.Fatalf("error = %v, want unknown section key %q", err, tc.wantKey)
			}
			if report.FailedStage != StageValidate {
				t.Fatalf("FailedStage = %q, want %q", report.FailedStage, StageValidate)
			}
		})
	}
}

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
func TestPermissiveUnknownKeyWarnsAndIgnoresScalarSectionKey(t *testing.T) {
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
http: oops
`)

	cfg, report, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if !slices.Contains(report.UnknownKeyWarnings, "http") {
		t.Fatalf("UnknownKeyWarnings = %v, want http", report.UnknownKeyWarnings)
	}
	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("HTTP.Addr = %q, want default :8080 after ignored section scalar", cfg.HTTP.Addr)
	}
}

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
func TestRemovedObservabilityKeysRejectInStrictMode(t *testing.T) {
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
observability:
  metrics:
    enabled: true
    path: /internal/metrics
  grafana:
    enabled: true
    cloud_otlp_endpoint: "https://example.invalid"
`)

	_, _, err := LoadDetailed(LoadOptions{
		ConfigPath: configPath,
		Strict:     true,
	})
	if err == nil {
		t.Fatalf("LoadDetailed() expected strict unknown key error")
	}
	if !errors.Is(err, ErrStrictUnknownKey) {
		t.Fatalf("error = %v, want ErrStrictUnknownKey", err)
	}
}

func TestRequiredIfEnabledPostgresSecretPolicy(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__POSTGRES__ENABLED", "true")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected secret policy error")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestTST003RequiredIfEnabledContracts(t *testing.T) {
	t.Run("postgres_enabled_without_dsn_rejected", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("APP__POSTGRES__ENABLED", "true")

		_, _, err := LoadDetailed(LoadOptions{})
		if err == nil {
			t.Fatalf("LoadDetailed() expected secret policy error")
		}
		if !errors.Is(err, ErrSecretPolicy) {
			t.Fatalf("error = %v, want ErrSecretPolicy", err)
		}
	})

	t.Run("postgres_enabled_with_dsn_allowed", func(t *testing.T) {
		resetConfigEnv(t)
		dsn := "postgres://app:app@localhost:5432/app?sslmode=disable"
		t.Setenv("APP__POSTGRES__ENABLED", "true")
		t.Setenv("APP__POSTGRES__DSN", dsn)

		cfg, _, err := LoadDetailed(LoadOptions{})
		if err != nil {
			t.Fatalf("LoadDetailed() error = %v", err)
		}
		if !cfg.Postgres.Enabled {
			t.Fatalf("Postgres.Enabled = false, want true")
		}
		if cfg.Postgres.DSN != dsn {
			t.Fatalf("Postgres.DSN = %q, want %q", cfg.Postgres.DSN, dsn)
		}
	})
}

func TestValidatePostgresReadinessBudget(t *testing.T) {
	t.Parallel()

	disabled := Config{
		HTTP:     HTTPConfig{ReadinessTimeout: time.Second},
		Postgres: PostgresConfig{HealthcheckTimeout: 10 * time.Second},
	}
	if err := validatePostgresReadinessBudget(disabled); err != nil {
		t.Fatalf("validatePostgresReadinessBudget() error = %v, want nil for disabled postgres", err)
	}

	tooSmall := Config{
		HTTP: HTTPConfig{ReadinessTimeout: time.Second},
		Postgres: PostgresConfig{
			Enabled:            true,
			HealthcheckTimeout: 2 * time.Second,
		},
	}
	err := validatePostgresReadinessBudget(tooSmall)
	if err == nil {
		t.Fatal("validatePostgresReadinessBudget() error = nil, want budget error")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "postgres.healthcheck_timeout") {
		t.Fatalf("error = %v, want readiness probe budget name", err)
	}

	tooSmall.HTTP.ReadinessTimeout = 2 * time.Second
	if err := validatePostgresReadinessBudget(tooSmall); err != nil {
		t.Fatalf("validatePostgresReadinessBudget() error = %v, want nil when budget fits", err)
	}
}

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
