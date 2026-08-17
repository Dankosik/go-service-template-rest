package config

import (
	"errors"
	"strings"
	"testing"
)

func TestUnknownKeyRejects(t *testing.T) {
	t.Parallel()
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
unknown:
  field: value
`)

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
	if err == nil {
		t.Fatal("LoadDetailed() expected unknown key error")
	}
	if !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("error = %v, want ErrUnknownKey", err)
	}
	if got := ErrorType(err); got != "unknown_key" {
		t.Fatalf("ErrorType(error) = %q, want unknown_key", got)
	}
}

func TestOverlayUnknownKeyRejects(t *testing.T) {
	t.Parallel()
	resetConfigEnv(t)

	overlayPath := writeTempConfig(t, `
unknown:
  field: value
`)

	_, _, err := LoadDetailed(LoadOptions{
		ConfigOverlays: []string{overlayPath},
	})
	if err == nil {
		t.Fatal("LoadDetailed() expected unknown overlay key error")
	}
	if !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("error = %v, want ErrUnknownKey", err)
	}
}

func TestUnknownKeyRejectsScalarSectionKeys(t *testing.T) {
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

			_, report, err := LoadDetailed(LoadOptions{})
			if err == nil {
				t.Fatal("LoadDetailed() expected unknown key error")
			}
			if !errors.Is(err, ErrUnknownKey) {
				t.Fatalf("error = %v, want ErrUnknownKey", err)
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
func TestScalarSectionRejects(t *testing.T) {
	t.Parallel()
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
http: oops
`)

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
	if err == nil {
		t.Fatal("LoadDetailed() error = nil, want unknown key error")
	}
	if !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("error = %v, want ErrUnknownKey", err)
	}
}

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
//nolint:paralleltest // This test mutates process-global environment or working directory.

// profile:database-postgres:start
func TestRemovedObservabilityKeysReject(t *testing.T) {
	t.Parallel()
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

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
	if err == nil {
		t.Fatal("LoadDetailed() expected unknown key error")
	}
	if !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("error = %v, want ErrUnknownKey", err)
	}
}

func TestRequiredIfEnabledPostgresSecretPolicy(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__POSTGRES__ENABLED", "true")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatal("LoadDetailed() expected secret policy error")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

// profile:database-postgres:end
// profile:database-postgres:start
//
//nolint:paralleltest // This test mutates process-global environment or working directory.
func TestTST003RequiredIfEnabledContracts(t *testing.T) {
	t.Run("postgres_enabled_without_dsn_rejected", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("APP__POSTGRES__ENABLED", "true")

		_, _, err := LoadDetailed(LoadOptions{})
		if err == nil {
			t.Fatal("LoadDetailed() expected secret policy error")
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
			t.Fatal("Postgres.Enabled = false, want true")
		}
		if cfg.Postgres.DSN != dsn {
			t.Fatalf("Postgres.DSN = %q, want %q", cfg.Postgres.DSN, dsn)
		}
	})
}

// profile:database-postgres:end

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
