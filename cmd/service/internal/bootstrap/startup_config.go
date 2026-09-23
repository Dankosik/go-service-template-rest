package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/example/go-service-template-rest/internal/config"
)

const (
	startupConfigCompatibilityStage  = "startup.config.compatibility"
	startupConfigCompatibilityReason = "startup_compatibility"
)

func bootstrapConfigStage(
	startupCtx context.Context,
	loadOptions config.LoadOptions,
) (config.Config, config.LoadReport, error) {
	slog.InfoContext(
		startupCtx,
		"config_load_started",
		startupLogArgs(
			"config_loader",
			"load",
			"started",
			"config.file", loadOptions.ConfigPath,
			"config.overlay_count", len(loadOptions.ConfigOverlays),
		)...,
	)

	cfg, configReport, err := config.Load(startupCtx, loadOptions)
	if err != nil {
		errorType := config.ErrorType(err)
		slog.ErrorContext(
			startupCtx,
			"config_load_failed",
			startupLogArgs(
				"config_loader",
				"load",
				"error",
				"stage", configReport.FailedStage,
				"error.type", errorType,
			)...,
		)
		return config.Config{}, config.LoadReport{}, fmt.Errorf("load config (%s): %w", errorType, err)
	}

	// The dependency profile owns startup-budget compatibility; a profile
	// with no dependencies has nothing to check.
	if err := errors.Join(
		validateShutdownGraceBudget(cfg),
		validateStartupBudgetCompatibility(cfg),
	); err != nil {
		errorType := startupConfigCompatibilityReason
		slog.ErrorContext(
			startupCtx,
			"config_load_failed",
			startupLogArgs(
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
