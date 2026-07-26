package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

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

	cfg, configReport, err := config.LoadDetailedWithContext(startupCtx, loadOptions)
	if err != nil {
		failedStage := failedConfigStage(configReport)
		errorType := config.ErrorType(err)
		slog.ErrorContext(
			startupCtx,
			"config_load_failed",
			startupLogArgs(
				"config_loader",
				"load",
				"error",
				"stage", failedStage,
				"error.type", errorType,
			)...,
		)
		return config.Config{}, config.LoadReport{}, fmt.Errorf("load config (%s): %w", errorType, err)
	}

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

// validateStartupBudgetCompatibility is implemented by the dependency stage,
// because every rule it enforces relates a configured dependency budget to the
// startup stage that runs it. A profile with no dependencies has nothing to
// check. See startup_dependencies.go.
