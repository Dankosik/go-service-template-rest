package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
)

func TestFailedConfigStage(t *testing.T) {
	t.Parallel()

	if got := failedConfigStage(config.LoadReport{}); got != config.StageLoadDefaults {
		t.Fatalf("failedConfigStage() = %q, want %q", got, config.StageLoadDefaults)
	}
	if got := failedConfigStage(config.LoadReport{FailedStage: config.StageValidate}); got != config.StageValidate {
		t.Fatalf("failedConfigStage() = %q, want %q", got, config.StageValidate)
	}
}

func TestBootstrapConfigStageReturnsConfigLoadFailure(t *testing.T) {
	t.Setenv("APP__APP__ENV", "local")

	missingConfig := filepath.Join(t.TempDir(), "missing.yaml")

	_, _, err := bootstrapConfigStage(context.Background(), config.LoadOptions{ConfigPath: missingConfig})
	if err == nil {
		t.Fatal("bootstrapConfigStage() error = nil, want non-nil")
	}
}

func TestValidateStartupBudgetCompatibilityRejectsDependencyTimeoutsAboveProbeBudgets(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		cfg     config.Config
		wantKey string
	}{
		{
			name: "postgres connect timeout",
			cfg: config.Config{
				Postgres: config.PostgresConfig{
					Enabled:        true,
					ConnectTimeout: postgresProbeBudget + time.Nanosecond,
				},
			},
			wantKey: "postgres.connect_timeout",
		},
		{
			name: "postgres healthcheck timeout",
			cfg: config.Config{
				Postgres: config.PostgresConfig{
					Enabled:            true,
					ConnectTimeout:     postgresProbeBudget,
					HealthcheckTimeout: postgresProbeBudget + time.Nanosecond,
				},
			},
			wantKey: "postgres.healthcheck_timeout",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateStartupBudgetCompatibility(tc.cfg)
			if err == nil {
				t.Fatal("validateStartupBudgetCompatibility() error = nil, want validation error")
			}
			if !errors.Is(err, config.ErrValidate) {
				t.Fatalf("error = %v, want ErrValidate", err)
			}
			if !strings.Contains(err.Error(), tc.wantKey) {
				t.Fatalf("error = %v, want key %q", err, tc.wantKey)
			}
		})
	}
}

func TestValidateStartupBudgetCompatibilityIgnoresDisabledDependencies(t *testing.T) {
	t.Parallel()

	err := validateStartupBudgetCompatibility(config.Config{
		HTTP: config.HTTPConfig{ReadinessTimeout: time.Second},
		Postgres: config.PostgresConfig{
			ConnectTimeout:     postgresProbeBudget + time.Second,
			HealthcheckTimeout: postgresProbeBudget + time.Second,
		},
	})
	if err != nil {
		t.Fatalf("validateStartupBudgetCompatibility() error = %v, want nil for disabled dependencies", err)
	}
}

func TestValidateStartupBudgetCompatibilityRequiresReadinessHeadroom(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		HTTP: config.HTTPConfig{
			ReadinessTimeout: time.Second,
		},
		Postgres: config.PostgresConfig{
			Enabled:            true,
			HealthcheckTimeout: time.Second,
		},
	}

	err := validateStartupBudgetCompatibility(cfg)
	if err == nil {
		t.Fatal("validateStartupBudgetCompatibility() error = nil, want readiness headroom validation error")
	}
	if !errors.Is(err, config.ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "startup headroom") {
		t.Fatalf("error = %v, want startup headroom context", err)
	}
	if !strings.Contains(err.Error(), "postgres.healthcheck_timeout") {
		t.Fatalf("error = %v, want readiness probe name", err)
	}

	cfg.HTTP.ReadinessTimeout = time.Second + startupReadinessHeadroom
	if err := validateStartupBudgetCompatibility(cfg); err != nil {
		t.Fatalf("validateStartupBudgetCompatibility() error = %v, want nil when headroom is included", err)
	}
}

func TestValidateStartupBudgetCompatibilityAllowsDefaultPostgresReadiness(t *testing.T) {
	resetBootstrapConfigEnv(t)
	t.Setenv("APP__POSTGRES__ENABLED", "true")
	t.Setenv("APP__POSTGRES__DSN", "postgres://user:pass@localhost:5432/app?sslmode=disable")

	cfg, _, err := config.LoadDetailed(config.LoadOptions{})
	if err != nil {
		t.Fatalf("config.LoadDetailed() error = %v", err)
	}
	if cfg.HTTP.ReadinessTimeout != 4*time.Second {
		t.Fatalf("HTTP.ReadinessTimeout = %s, want 4s default", cfg.HTTP.ReadinessTimeout)
	}
	if cfg.Postgres.HealthcheckTimeout != 3*time.Second {
		t.Fatalf("Postgres.HealthcheckTimeout = %s, want 3s default", cfg.Postgres.HealthcheckTimeout)
	}

	if err := validateStartupBudgetCompatibility(cfg); err != nil {
		t.Fatalf("validateStartupBudgetCompatibility() error = %v, want nil for default Postgres readiness headroom", err)
	}
}

func TestBootstrapConfigStageReturnsStartupCompatibilityFailure(t *testing.T) {
	resetBootstrapConfigEnv(t)
	t.Setenv("APP__POSTGRES__ENABLED", "true")
	t.Setenv("APP__POSTGRES__DSN", "postgres://user:pass@localhost:5432/app?sslmode=disable")
	t.Setenv("APP__POSTGRES__CONNECT_TIMEOUT", "6s")

	_, _, err := bootstrapConfigStage(context.Background(), config.LoadOptions{})
	if err == nil {
		t.Fatal("bootstrapConfigStage() error = nil, want startup compatibility validation error")
	}
	if !errors.Is(err, config.ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
}

func resetBootstrapConfigEnv(t *testing.T) {
	t.Helper()

	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if !strings.HasPrefix(key, "APP__") && key != "APP_CONFIG_ALLOWED_ROOTS" {
			continue
		}
		t.Setenv(key, value)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("os.Unsetenv(%q) error = %v", key, err)
		}
	}
}
