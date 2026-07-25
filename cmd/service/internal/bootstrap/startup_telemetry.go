package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

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
	reportIgnoredAmbientOTLPEnv(startupCtx, log, exporterCfg)
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
	recordTraceExporterState(startupCtx, log, metrics, exporterCfg, telemetryInitErr)
	if telemetryInitErr != nil {
		return cleanup, fmt.Errorf("setup tracing: %w", telemetryInitErr)
	}

	return newTelemetryCleanup(log, tracingShutdown, metricsShutdown), nil
}

// recordTraceExporterState publishes trace-export state as a metric so an
// operator can alert on it. A startup warning is only visible to whoever reads
// the boot log; a service that answers every request while exporting no traces
// needs a signal that survives to a dashboard.
func recordTraceExporterState(
	ctx context.Context,
	log *slog.Logger,
	metrics *telemetry.Metrics,
	exporterCfg telemetry.TraceExporterConfig,
	telemetryInitErr error,
) {
	active := telemetryInitErr == nil && strings.TrimSpace(exporterCfg.OTLPEndpoint) != ""
	if err := metrics.RecordTraceExporterState(ctx, active); err != nil {
		log.Warn(
			"telemetry_state_metric_unavailable",
			startupLogArgs(
				ctx,
				startupLogComponentStartupProbes,
				startupOperationTelemetryInit,
				"degraded",
				"dependency", startupDependencyTelemetry,
				"err", err,
			)...,
		)
	}
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

// reportIgnoredAmbientOTLPEnv warns when a platform injected the standard
// OTEL_EXPORTER_OTLP_* variables. This service reads its exporter settings from
// observability.otel.exporter.* only, so an injected collector endpoint looks
// effective when it is not — and without the warning the deployment looks
// healthy while no trace is ever exported, which is the failure an operator
// finds during an incident.
//
// Conflicting credential and trust variables are excluded when this service has
// its own exporter: that case fails exporter setup and is already reported as
// degraded telemetry, so listing it here as merely "ignored" would contradict
// that record.
func reportIgnoredAmbientOTLPEnv(ctx context.Context, log *slog.Logger, exporterCfg telemetry.TraceExporterConfig) {
	configured := strings.TrimSpace(exporterCfg.OTLPEndpoint) != ""
	ignored := telemetry.AmbientOTLPExporterEnv()
	if configured {
		conflicting := telemetry.ConflictingTraceExporterEnv()
		ignored = slices.DeleteFunc(ignored, func(name string) bool {
			return slices.Contains(conflicting, name)
		})
	}
	if len(ignored) == 0 {
		return
	}

	mode := startupDependencyModeFeatureOff
	if configured {
		mode = startupDependencyModeConfigured
	}

	log.Warn(
		"telemetry_ambient_env_ignored",
		startupLogArgs(
			ctx,
			startupLogComponentStartupProbes,
			startupOperationTelemetryInit,
			"degraded",
			"dependency", startupDependencyTelemetry,
			"mode", mode,
			"reason", "ambient_exporter_env_ignored",
			"env.ignored", strings.Join(ignored, ", "),
			"config.key", "observability.otel.exporter.otlp_endpoint",
		)...,
	)
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
