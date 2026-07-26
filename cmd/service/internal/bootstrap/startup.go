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
			// A startup that failed has no shutdown budget yet, so the flush gets
			// its own bound here. Detached from startupCtx because a spent startup
			// budget is one of the reasons this path runs, and a flush handed an
			// already-expired context reports nothing about why.
			flushCtx, cancel := context.WithTimeout(context.WithoutCancel(startupCtx), telemetryShutdownTimeout)
			defer cancel()
			telemetryCleanup(flushCtx)
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
	stage := bootstrapTelemetryStage(startupCtx, cfg, metrics, log)
	telemetryCleanup = stage.cleanup

	bootstrapReportStage(startupCtx, log, cfg, loadOptions, configReport, stage.traceEndpoint, stage.tracingErr)

	return startupBootstrap{
		cfg:              cfg,
		log:              log,
		telemetryCleanup: telemetryCleanup,
	}, nil
}
