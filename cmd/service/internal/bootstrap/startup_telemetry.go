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
)

func bootstrapTelemetryStage(
	startupCtx context.Context,
	cfg config.Config,
	metrics *telemetry.Metrics,
	log *slog.Logger,
) (func(context.Context), telemetry.TraceExporterEndpoint, error) {
	metricsCtx, metricsCancel := withStageBudget(startupCtx, startupTelemetryBudget)
	metricsShutdown, metricEndpoint, telemetryInitErr := telemetry.SetupMetrics(metricsCtx, metrics, telemetry.MetricsConfig{
		ServiceName:    cfg.Observability.OTel.ServiceName,
		ServiceVersion: cfg.App.Version,
		DeploymentEnv:  cfg.App.Env,
		Exporter:       metricExporterConfig(cfg),
	})
	metricsCancel()
	if telemetryInitErr != nil {
		return func(context.Context) {}, telemetry.TraceExporterEndpoint{}, fmt.Errorf("setup metrics: %w", telemetryInitErr)
	}
	reportMetricExporterState(startupCtx, log, metricEndpoint)
	cleanup := newTelemetryCleanup(log, metricsShutdown)

	exporterCfg := traceExporterConfig(cfg)
	telemetryCtx, telemetryCancel := withStageBudget(startupCtx, startupTelemetryBudget)
	traceEndpoint, tracingShutdown, telemetryInitErr := telemetry.SetupTracing(telemetryCtx, telemetry.TracingConfig{
		ServiceName:      cfg.Observability.OTel.ServiceName,
		ServiceVersion:   cfg.App.Version,
		DeploymentEnv:    cfg.App.Env,
		TracesSampler:    cfg.Observability.OTel.TracesSampler,
		TracesSamplerArg: cfg.Observability.OTel.TracesSamplerArg,
		Exporter:         exporterCfg,
	})
	telemetryCancel()
	// Reporting follows setup because only setup knows which setting supplied
	// the endpoint, and "ignored" must not name the variable that was honored.
	reportIgnoredAmbientOTLPEnv(startupCtx, log, traceEndpoint)
	recordTraceExporterState(startupCtx, log, metrics, traceEndpoint, telemetryInitErr)
	if telemetryInitErr != nil {
		return cleanup, traceEndpoint, fmt.Errorf("setup tracing: %w", telemetryInitErr)
	}

	return newTelemetryCleanup(log, tracingShutdown, metricsShutdown), traceEndpoint, nil
}

// recordTraceExporterState publishes trace-export state as a metric so an
// operator can alert on it. A startup warning is only visible to whoever reads
// the boot log; a service that answers every request while exporting no traces
// needs a signal that survives to a dashboard.
func recordTraceExporterState(
	ctx context.Context,
	log *slog.Logger,
	metrics *telemetry.Metrics,
	endpoint telemetry.TraceExporterEndpoint,
	telemetryInitErr error,
) {
	active := telemetryInitErr == nil && endpoint.Configured()
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

// reportIgnoredAmbientOTLPEnv warns about the standard OTEL_EXPORTER_OTLP_*
// variables a platform injected that this service did not act on. The endpoint
// variables are honored when this service names no endpoint of its own, so the
// one that supplied the endpoint is not reported: an operator who sees a
// variable listed here can trust that it changed nothing.
//
// Conflicting credential and trust variables are excluded when this service
// named the endpoint: that case fails exporter setup and is already reported as
// degraded telemetry, so listing it here as merely "ignored" would contradict
// that record.
func reportIgnoredAmbientOTLPEnv(ctx context.Context, log *slog.Logger, endpoint telemetry.TraceExporterEndpoint) {
	ignored := telemetry.AmbientOTLPExporterEnv()
	ignored = slices.DeleteFunc(ignored, func(name string) bool {
		return name == endpoint.Source
	})
	if endpoint.Source == telemetry.TraceExporterConfigKey {
		conflicting := telemetry.ConflictingTraceExporterEnv()
		ignored = slices.DeleteFunc(ignored, func(name string) bool {
			return slices.Contains(conflicting, name)
		})
	}
	if len(ignored) == 0 {
		return
	}

	mode := startupDependencyModeFeatureOff
	if endpoint.Configured() {
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
			"config.key", telemetry.TraceExporterConfigKey,
		)...,
	)
}

func traceExporterConfig(cfg config.Config) telemetry.TraceExporterConfig {
	return telemetry.TraceExporterConfig{
		OTLPEndpoint: cfg.Observability.OTel.Exporter.OTLPEndpoint,
		OTLPHeaders:  cfg.Observability.OTel.Exporter.OTLPHeaders,
	}
}

func metricExporterConfig(cfg config.Config) telemetry.MetricExporterConfig {
	return telemetry.MetricExporterConfig{
		OTLPEndpoint:       cfg.Observability.OTel.Exporter.OTLPMetricsEndpoint,
		SharedOTLPEndpoint: cfg.Observability.OTel.Exporter.OTLPEndpoint,
		OTLPHeaders:        cfg.Observability.OTel.Exporter.OTLPHeaders,
	}
}

// reportMetricExporterState names the metric-export destination in the startup
// log, because the Prometheus endpoint always exists and therefore proves
// nothing: a service reachable only by a collector needs the operator to see
// whether anything is being pushed, and where.
func reportMetricExporterState(ctx context.Context, log *slog.Logger, endpoint telemetry.ExporterEndpoint) {
	if !endpoint.Configured() {
		log.Info(
			"metrics_exporter_scrape_only",
			startupLogArgs(
				ctx,
				startupLogComponentStartupProbes,
				startupOperationTelemetryInit,
				"success",
				"dependency", startupDependencyTelemetry,
				"metrics.export", "scrape_only",
			)...,
		)
		return
	}

	log.Info(
		"metrics_exporter_configured",
		startupLogArgs(
			ctx,
			startupLogComponentStartupProbes,
			startupOperationTelemetryInit,
			"success",
			"dependency", startupDependencyTelemetry,
			"metrics.export", "otlp",
			"metrics.endpoint_source", endpoint.Source,
		)...,
	)
}
