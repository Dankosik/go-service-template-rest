package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
)

const (
	startupConfigCompatibilityStage  = "startup.config.compatibility"
	startupConfigCompatibilityReason = "startup_compatibility"
)

func failedConfigStage(report config.LoadReport) string {
	stage := strings.TrimSpace(report.FailedStage)
	if stage == "" {
		return config.StageLoadDefaults
	}
	return stage
}

func bootstrapConfigStage(
	startupCtx context.Context,
	loadOptions config.LoadOptions,
) (config.Config, config.LoadReport, error) {
	slog.Info(
		"config_load_started",
		startupLogArgs(
			startupCtx,
			"config_loader",
			"load",
			"started",
			"config.strict", loadOptions.Strict,
			"config.file", loadOptions.ConfigPath,
			"config.overlay_count", len(loadOptions.ConfigOverlays),
		)...,
	)

	cfg, configReport, err := config.LoadDetailedWithContext(startupCtx, loadOptions)
	if err != nil {
		failedStage := failedConfigStage(configReport)
		errorType := config.ErrorType(err)
		slog.Error(
			"config_load_failed",
			startupLogArgs(
				startupCtx,
				"config_loader",
				"load",
				"error",
				"stage", failedStage,
				"error.type", errorType,
			)...,
		)
		return config.Config{}, config.LoadReport{}, fmt.Errorf("load config (%s): %w", errorType, err)
	}

	if err := validateStartupBudgetCompatibility(cfg); err != nil {
		errorType := startupConfigCompatibilityReason
		slog.Error(
			"config_load_failed",
			startupLogArgs(
				startupCtx,
				"config_loader",
				"startup_compatibility",
				"error",
				"stage", startupConfigCompatibilityStage,
				"error.type", errorType,
			)...,
		)
		return config.Config{}, config.LoadReport{}, fmt.Errorf("load config (%s): %w", errorType, err)
	}

	return cfg, configReport, nil
}

func validateStartupBudgetCompatibility(cfg config.Config) error {
	if cfg.Postgres.Enabled {
		if err := validateStartupTimeoutBudget("postgres.connect_timeout", cfg.Postgres.ConnectTimeout, postgresProbeBudget); err != nil {
			return err
		}
		if err := validateStartupTimeoutBudget("postgres.healthcheck_timeout", cfg.Postgres.HealthcheckTimeout, postgresProbeBudget); err != nil {
			return err
		}
	}
	if err := validateStartupReadinessHeadroom(cfg); err != nil {
		return err
	}
	return nil
}

func validateStartupTimeoutBudget(name string, value time.Duration, budget time.Duration) error {
	if value <= budget {
		return nil
	}
	return fmt.Errorf(
		"%w: %s must be <= startup probe budget %s",
		config.ErrValidate,
		name,
		budget,
	)
}

func validateStartupReadinessHeadroom(cfg config.Config) error {
	if !cfg.Postgres.Enabled {
		return nil
	}

	required := cfg.Postgres.HealthcheckTimeout + startupReadinessHeadroom
	if cfg.HTTP.ReadinessTimeout >= required {
		return nil
	}
	return fmt.Errorf(
		"%w: http.readiness_timeout must be >= postgres.healthcheck_timeout readiness budget plus startup headroom (%s + %s = %s)",
		config.ErrValidate,
		cfg.Postgres.HealthcheckTimeout,
		startupReadinessHeadroom,
		required,
	)
}
