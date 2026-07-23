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

func TestConfigReadinessProbeRequiredPolicyHelpers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		cfg          Config
		wantPostgres bool
	}{
		{
			name: "disabled dependencies ignore readiness flags",
			cfg: Config{
				FeatureFlags: FeatureFlagsConfig{
					PostgresReadinessProbe: true,
				},
			},
		},
		{
			name: "enabled postgres without readiness flag",
			cfg: Config{
				Postgres: PostgresConfig{Enabled: true},
			},
		},
		{
			name: "enabled postgres with readiness flag",
			cfg: Config{
				Postgres: PostgresConfig{Enabled: true},
				FeatureFlags: FeatureFlagsConfig{
					PostgresReadinessProbe: true,
				},
			},
			wantPostgres: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.cfg.PostgresReadinessProbeRequired(); got != tc.wantPostgres {
				t.Fatalf("PostgresReadinessProbeRequired() = %v, want %v", got, tc.wantPostgres)
			}
		})
	}
}

func TestConfigReadinessProbeBudgetsUseRequiredRuntimeProbes(t *testing.T) {
	t.Parallel()

	cfg := Config{
		HTTP: HTTPConfig{
			ReadinessTimeout: 10 * time.Second,
		},
		Postgres: PostgresConfig{
			Enabled:            true,
			HealthcheckTimeout: 2 * time.Second,
		},
		FeatureFlags: FeatureFlagsConfig{
			PostgresReadinessProbe: true,
		},
	}

	budgets := cfg.ReadinessProbeBudgets()
	want := []ReadinessProbeBudget{
		{ConfigKey: "postgres.healthcheck_timeout", Budget: 2 * time.Second},
	}
	if len(budgets) != len(want) {
		t.Fatalf("ReadinessProbeBudgets() len = %d, want %d", len(budgets), len(want))
	}
	for i := range want {
		if budgets[i] != want[i] {
			t.Fatalf("ReadinessProbeBudgets()[%d] = %+v, want %+v", i, budgets[i], want[i])
		}
	}

	budgets[0].Budget = time.Nanosecond
	if got := cfg.ReadinessProbeBudgets()[0].Budget; got != 2*time.Second {
		t.Fatalf("ReadinessProbeBudgets() returned aliased slice; first budget = %s, want 2s", got)
	}
}

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
