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
) (result startupBootstrap, err error) {
	telemetryCleanup := func(context.Context) {}
	defer func() {
		if err != nil {
			telemetryCleanup(startupCtx)
		}
	}()

	cfg, configReport, err := bootstrapConfigStage(
		startupCtx,
		loadOptions,
	)
	if err != nil {
		return startupBootstrap{}, err
	}

	log := bootstrapLoggerStage(cfg)
	telemetryCleanup, traceEndpoint, telemetryInitErr := bootstrapTelemetryStage(startupCtx, cfg, metrics, log)

	bootstrapReportStage(startupCtx, log, cfg, loadOptions, configReport, traceEndpoint, telemetryInitErr)

	return startupBootstrap{
		cfg:              cfg,
		log:              log,
		telemetryCleanup: telemetryCleanup,
	}, nil
}
