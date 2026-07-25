package bootstrap

import (
	"context"
	"log/slog"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel/trace"
)

type startupBootstrap struct {
	cfg              config.Config
	log              *slog.Logger
	tracer           trace.Tracer
	bootstrapSpan    trace.Span
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
	telemetryCleanup, telemetryInitErr := bootstrapTelemetryStage(startupCtx, cfg, metrics, log)
	tracer, bootstrapCtx, bootstrapSpan := bootstrapTraceStage(startupCtx)
	spanOwnedByCaller := false
	defer func() {
		if !spanOwnedByCaller {
			bootstrapSpan.End()
		}
	}()

	bootstrapReportStage(bootstrapCtx, log, cfg, loadOptions, configReport, telemetryInitErr)

	result = startupBootstrap{
		cfg:              cfg,
		log:              log,
		tracer:           tracer,
		bootstrapSpan:    bootstrapSpan,
		telemetryCleanup: telemetryCleanup,
	}
	spanOwnedByCaller = true
	return result, nil
}

func startupBootstrapContext(startupCtx context.Context, bootstrapSpan trace.Span) context.Context {
	return trace.ContextWithSpan(startupCtx, bootstrapSpan)
}
