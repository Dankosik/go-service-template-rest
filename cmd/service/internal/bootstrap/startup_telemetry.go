package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

// telemetryStage is what the startup path needs back from telemetry setup.
//
// The tracing error is kept apart from anything metrics reported, because the
// startup summary names each signal's own outcome and one joined error made a
// mistyped metrics endpoint report the trace exporter as degraded. Metrics
// degradation is reported where it happens, by reportMetricExporterState.
type telemetryStage struct {
	cleanup       func(context.Context)
	traceEndpoint telemetry.TraceExporterEndpoint
	tracingErr    error
}

// bootstrapTelemetryStage installs both signals, independently.
//
// Independence is the point. The version this replaced returned on the first
// metrics failure, so SetupTracing never ran: one unusable OTLP metrics endpoint
// left the tracer provider unset, which cost traces, cost the meter provider that
// records whether traces are being exported at all, and silently stopped every log
// record carrying trace_id and span_id — logctx reads those off the span context
// the provider produces. The service started anyway and reported healthy, so the
// only artifact was one warning at boot.
//
// Neither failure is fatal. A service that cannot export telemetry still serves
// its contract, and taking it down for that would trade an observability outage
// for a real one.
func bootstrapTelemetryStage(
	startupCtx context.Context,
	cfg config.Config,
	metrics *telemetry.Metrics,
	log *slog.Logger,
) telemetryStage {
	// Resolved once, then shared by both signals: a second resolution could pick a
	// different fallback identifier and split one replica's traces from its metrics.
	instanceID := telemetry.ResolveInstanceID(cfg.App.InstanceID)

	metricsCtx, metricsCancel := withStageBudget(startupCtx, startupTelemetryBudget)
	metricsResult, metricsErr := telemetry.SetupMetrics(metricsCtx, metrics, telemetry.MetricsConfig{
		ServiceName:       cfg.Observability.OTel.ServiceName,
		ServiceVersion:    cfg.App.Version,
		ServiceCommit:     cfg.App.Commit,
		ServiceInstanceID: instanceID,
		DeploymentEnv:     cfg.App.Env,
		Exporter:          metricExporterConfig(cfg),
	})
	metricsCancel()
	reportMetricExporterState(startupCtx, log, metricsResult, metricsErr)

	exporterCfg := traceExporterConfig(cfg)
	telemetryCtx, telemetryCancel := withStageBudget(startupCtx, startupTelemetryBudget)
	traceEndpoint, tracingShutdown, tracingErr := telemetry.SetupTracing(telemetryCtx, telemetry.TracingConfig{
		ServiceName:       cfg.Observability.OTel.ServiceName,
		ServiceVersion:    cfg.App.Version,
		ServiceCommit:     cfg.App.Commit,
		ServiceInstanceID: instanceID,
		DeploymentEnv:     cfg.App.Env,
		TracesSampler:     cfg.Observability.OTel.TracesSampler,
		TracesSamplerArg:  cfg.Observability.OTel.TracesSamplerArg,
		Exporter:          exporterCfg,
	})
	telemetryCancel()
	// Reporting follows setup because only setup knows which setting supplied
	// the endpoint, and "ignored" must not name the variable that was honored.
	reportIgnoredAmbientOTLPEnv(startupCtx, log, traceEndpoint)
	recordTraceExporterState(startupCtx, log, metrics, traceEndpoint, tracingErr)

	return telemetryStage{
		cleanup:       newTelemetryCleanup(log, tracingShutdown, metricsResult.Shutdown),
		traceEndpoint: traceEndpoint,
		tracingErr:    tracingErr,
	}
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
		log.WarnContext(
			ctx,
			"telemetry_state_metric_unavailable",
			startupLogArgs(
				startupLogComponentStartupProbes,
				startupOperationTelemetryInit,
				"degraded",
				"dependency", startupDependencyTelemetry,
				"err", err,
			)...,
		)
	}
}

// newTelemetryCleanup builds the flush, which takes its bound from the context
// it is called with.
//
// It deliberately derives no deadline of its own. The flush is the last teardown
// stage, so what it may spend is whatever the process grace period has left —
// a number only the caller holding the shutdown budget knows. Re-deriving a fixed
// five seconds here is how the total teardown grew past the platform's grace
// period and got this stage killed for it.
func newTelemetryCleanup(log *slog.Logger, shutdowns ...func(context.Context) error) func(context.Context) {
	return func(shutdownCtx context.Context) {
		log.InfoContext(
			shutdownCtx,
			"telemetry_flush_started",
			startupLogArgs(
				startupLogComponentShutdown,
				startupOperationTelemetryFlush,
				"started",
			)...,
		)

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
			log.ErrorContext(
				shutdownCtx,
				"telemetry shutdown failed",
				startupLogArgs(
					startupLogComponentShutdown,
					startupOperationTelemetryFlush,
					"error",
					"error.type", "dependency_init",
					"err", shutdownErr,
				)...,
			)
			return
		}
		log.InfoContext(
			shutdownCtx,
			"telemetry_flush_completed",
			startupLogArgs(
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

	log.WarnContext(
		ctx,
		"telemetry_ambient_env_ignored",
		startupLogArgs(
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
//
// setupErr is the failure that leaves no meter provider at all; result.ExportErr
// is the narrower one where scrape still works and push does not. They are
// reported apart because the remedies differ, and because "no metrics" and "no
// pushed metrics" look identical on a dashboard that only ever scraped.
func reportMetricExporterState(
	ctx context.Context,
	log *slog.Logger,
	result telemetry.MetricsResult,
	setupErr error,
) {
	switch {
	case setupErr != nil:
		log.ErrorContext(
			ctx,
			"metrics_exporter_unavailable",
			startupLogArgs(
				startupLogComponentStartupProbes,
				startupOperationTelemetryInit,
				"error",
				"dependency", startupDependencyTelemetry,
				"metrics.export", "none",
				"reason", telemetryInitFailureReason(setupErr),
				"err", setupErr,
			)...,
		)
	case result.ExportErr != nil:
		log.WarnContext(
			ctx,
			"metrics_exporter_degraded",
			startupLogArgs(
				startupLogComponentStartupProbes,
				startupOperationTelemetryInit,
				"degraded",
				"dependency", startupDependencyTelemetry,
				// Scrape survives an unusable collector, and saying so is what
				// keeps an operator from chasing a total metrics outage.
				"metrics.export", "scrape_only",
				"reason", telemetryInitFailureReason(result.ExportErr),
				"err", result.ExportErr,
			)...,
		)
	case !result.PushConfigured():
		log.InfoContext(
			ctx,
			"metrics_exporter_scrape_only",
			startupLogArgs(
				startupLogComponentStartupProbes,
				startupOperationTelemetryInit,
				"success",
				"dependency", startupDependencyTelemetry,
				"metrics.export", "scrape_only",
			)...,
		)
	default:
		log.InfoContext(
			ctx,
			"metrics_exporter_configured",
			startupLogArgs(
				startupLogComponentStartupProbes,
				startupOperationTelemetryInit,
				"success",
				"dependency", startupDependencyTelemetry,
				"metrics.export", "otlp",
				"metrics.endpoint_source", result.Endpoint.Source,
			)...,
		)
	}
}
