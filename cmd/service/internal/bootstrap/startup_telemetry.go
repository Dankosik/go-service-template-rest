package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func bootstrapTelemetryStage(
	startupCtx context.Context,
	cfg config.Config,
	metrics *telemetry.Metrics,
	log *slog.Logger,
) (func(context.Context), error) {
	metricsCtx, metricsCancel := withStageBudget(startupCtx, startupTelemetryBudget)
	metricsShutdown, telemetryInitErr := telemetry.SetupMetrics(metricsCtx, metrics, telemetry.MetricsConfig{
		ServiceName:    cfg.Observability.OTel.ServiceName,
		ServiceVersion: cfg.App.Version,
		DeploymentEnv:  cfg.App.Env,
	})
	metricsCancel()
	if telemetryInitErr != nil {
		return func(context.Context) {}, fmt.Errorf("setup metrics: %w", telemetryInitErr)
	}
	cleanup := newTelemetryCleanup(log, metricsShutdown)

	exporterCfg := traceExporterConfig(cfg)
	telemetryCtx, telemetryCancel := withStageBudget(startupCtx, startupTelemetryBudget)
	tracingShutdown, telemetryInitErr := telemetry.SetupTracing(telemetryCtx, telemetry.TracingConfig{
		ServiceName:      cfg.Observability.OTel.ServiceName,
		ServiceVersion:   cfg.App.Version,
		DeploymentEnv:    cfg.App.Env,
		TracesSampler:    cfg.Observability.OTel.TracesSampler,
		TracesSamplerArg: cfg.Observability.OTel.TracesSamplerArg,
		Exporter:         exporterCfg,
	})
	telemetryCancel()
	if telemetryInitErr != nil {
		return cleanup, fmt.Errorf("setup tracing: %w", telemetryInitErr)
	}

	return newTelemetryCleanup(log, tracingShutdown, metricsShutdown), nil
}

func newTelemetryCleanup(log *slog.Logger, shutdowns ...func(context.Context) error) func(context.Context) {
	return func(shutdownBaseCtx context.Context) {
		log.Info(
			"telemetry_flush_started",
			startupLogArgs(
				shutdownBaseCtx,
				startupLogComponentShutdown,
				startupOperationTelemetryFlush,
				"started",
			)...,
		)
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(shutdownBaseCtx), telemetryShutdownTimeout)
		defer cancel()

		var shutdownErrors []error
		for _, shutdown := range shutdowns {
			if shutdown == nil {
				continue
			}
			if shutdownErr := shutdown(shutdownCtx); shutdownErr != nil {
				shutdownErrors = append(shutdownErrors, shutdownErr)
			}
		}
		if shutdownErr := errors.Join(shutdownErrors...); shutdownErr != nil {
			log.Error(
				"telemetry shutdown failed",
				startupLogArgs(
					shutdownBaseCtx,
					startupLogComponentShutdown,
					startupOperationTelemetryFlush,
					"error",
					"error.type", "dependency_init",
					"err", shutdownErr,
				)...,
			)
			return
		}
		log.Info(
			"telemetry_flush_completed",
			startupLogArgs(
				shutdownBaseCtx,
				startupLogComponentShutdown,
				startupOperationTelemetryFlush,
				"success",
			)...,
		)
	}
}

func traceExporterConfig(cfg config.Config) telemetry.TraceExporterConfig {
	return telemetry.TraceExporterConfig{
		OTLPEndpoint: cfg.Observability.OTel.Exporter.OTLPEndpoint,
		OTLPHeaders:  cfg.Observability.OTel.Exporter.OTLPHeaders,
	}
}

// bootstrapTraceStage transfers bootstrapSpan ownership to bootstrapRuntime.
// bootstrapRuntime ends it on setup failure; startupSpanController closes it after successful startup.
//
//nolint:spancheck // bootstrapSpan intentionally outlives this helper and is returned to its lifecycle owner.
func bootstrapTraceStage(startupCtx context.Context) (trace.Tracer, context.Context, trace.Span) {
	tracer := otel.Tracer("service.startup")
	bootstrapCtx, bootstrapSpan := tracer.Start(startupCtx, "config.bootstrap")
	return tracer, bootstrapCtx, bootstrapSpan
}
