package bootstrap

import (
	"context"
	"log/slog"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

type startupBootstrap struct {
	cfg              config.Config
	log              *slog.Logger
	telemetryCleanup func(context.Context)
}

func bootstrapRuntime(
	startupCtx context.Context,
	loadOptions config.LoadOptions,
	metrics *telemetry.Metrics,
) (startupBootstrap, error) {
	cfg, configReport, err := bootstrapConfigStage(
		startupCtx,
		loadOptions,
	)
	if err != nil {
		return startupBootstrap{}, err
	}

	log := bootstrapLoggerStage(cfg)
	stage := bootstrapTelemetryStage(startupCtx, cfg, metrics, log)

	bootstrapReportStage(startupCtx, log, cfg, loadOptions, configReport, stage.traceEndpoint, stage.tracingErr)

	return startupBootstrap{
		cfg:              cfg,
		log:              log,
		telemetryCleanup: stage.cleanup,
	}, nil
}
