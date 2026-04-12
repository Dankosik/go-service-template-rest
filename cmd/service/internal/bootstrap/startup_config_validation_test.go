package bootstrap

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

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
		{
			name: "redis dial timeout",
			cfg: config.Config{
				Redis: config.RedisConfig{
					Enabled:     true,
					Mode:        config.RedisModeCache,
					DialTimeout: redisProbeBudget + time.Nanosecond,
				},
			},
			wantKey: "redis.dial_timeout",
		},
		{
			name: "mongo connect timeout",
			cfg: config.Config{
				Mongo: config.MongoConfig{
					Enabled:        true,
					ConnectTimeout: mongoProbeBudget + time.Nanosecond,
				},
			},
			wantKey: "mongo.connect_timeout",
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
		Redis: config.RedisConfig{
			Mode:        config.RedisModeCache,
			DialTimeout: redisProbeBudget + time.Second,
		},
		Mongo: config.MongoConfig{
			ConnectTimeout: mongoProbeBudget + time.Second,
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
		FeatureFlags: config.FeatureFlagsConfig{
			PostgresReadinessProbe: true,
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

func TestBootstrapConfigStageRecordsStartupCompatibilityFailureAsConfigValidation(t *testing.T) {
	resetBootstrapConfigEnv(t)
	t.Setenv("APP__POSTGRES__ENABLED", "true")
	t.Setenv("APP__POSTGRES__DSN", "postgres://user:pass@localhost:5432/app?sslmode=disable")
	t.Setenv("APP__POSTGRES__CONNECT_TIMEOUT", "6s")

	metrics := telemetry.New()
	_, _, err := bootstrapConfigStage(context.Background(), config.LoadOptions{}, metrics)
	if err == nil {
		t.Fatal("bootstrapConfigStage() error = nil, want startup compatibility validation error")
	}
	if !errors.Is(err, config.ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `config_validation_failures_total{reason="validate"} 1`) {
		t.Fatalf("metrics output missing config validation failure:\n%s", metricsText)
	}
	assertStartupRejectionMetric(t, metricsText, telemetry.StartupRejectionReasonConfigValidate)
	if strings.Contains(metricsText, `config_load_duration_seconds_count{result="success"`) {
		t.Fatalf("metrics output contains config success metrics after compatibility failure:\n%s", metricsText)
	}
}

func resetBootstrapConfigEnv(t *testing.T) {
	t.Helper()

	previousValues := make(map[string]string)
	previousSet := make(map[string]bool)
	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if !strings.HasPrefix(key, "APP__") && key != "APP_CONFIG_ALLOWED_ROOTS" {
			continue
		}
		previousValues[key] = value
		previousSet[key] = true
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("os.Unsetenv(%q) error = %v", key, err)
		}
	}
	t.Cleanup(func() {
		for key, value := range previousValues {
			if previousSet[key] {
				_ = os.Setenv(key, value)
				continue
			}
			_ = os.Unsetenv(key)
		}
	})
}
