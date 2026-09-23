package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

const (
	startupDependencyTelemetry      = "telemetry"
	startupDependencyModeFeatureOff = "feature_off"
	startupDependencyModeConfigured = "configured"

	startupOperationTelemetryInit  = "telemetry_init"
	startupOperationTelemetryFlush = "telemetry_flush"
)

// telemetryStage is what the startup path needs back from telemetry setup.
//
// The tracing error is kept apart from anything metrics reported, because the
// startup summary names each signal's own outcome and one joined error made a
// mistyped metrics endpoint report the trace exporter as degraded. Metrics
// degradation is reported where it happens, by reportMetricExporterState.
type telemetryStage struct {
	flush           func(context.Context)
	tracingEndpoint telemetry.TraceExporterEndpoint
	tracingErr      error
}

// bootstrapTelemetryStage installs both signals, independently.
//
// Independence is the point. Returning on the first metrics failure would let one
// unusable OTLP metrics endpoint leave the tracer provider unset — costing
// traces, the meter provider that reports whether traces are exported at all,
// and every log record's trace_id and span_id, which logctx reads off the span
// context that provider produces. The service would still start and report
// healthy, leaving one warning at boot as the only artifact.
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
	telemetry.InstallErrorHandler(startupCtx, log)

	// Resolved once, then shared by both signals: a second resolution could pick a
	// different fallback identifier and split one replica's traces from its metrics.
	instanceID := telemetry.ResolveInstanceID(cfg.App.InstanceID)

	metricsCtx, metricsCancel := withStageBudget(startupCtx, startupTelemetryBudget)
	metricsResult, metricsErr := telemetry.SetupMetrics(metricsCtx, metrics, runtimeopts.Metrics(cfg, instanceID))
	metricsCancel()
	reportMetricExporterState(startupCtx, log, metricsResult, metricsErr)

	tracingCtx, tracingCancel := withStageBudget(startupCtx, startupTelemetryBudget)
	tracingEndpoint, tracingShutdown, tracingErr := telemetry.SetupTracing(tracingCtx, runtimeopts.Tracing(cfg, instanceID))
	tracingCancel()
	// Reporting follows setup because only setup knows which setting supplied
	// the endpoint, and the additional-variable record must not name the source
	// that was honored.
	reportAdditionalAmbientOTLPEnv(startupCtx, log, tracingEndpoint, metricsResult.Endpoint)
	recordTraceExporterInitialization(startupCtx, log, metrics, tracingEndpoint, tracingErr)

	return telemetryStage{
		flush:           newTelemetryFlush(log, tracingShutdown, metricsResult.Shutdown),
		tracingEndpoint: tracingEndpoint,
		tracingErr:      tracingErr,
	}
}

// Trace-export outcomes reported by traceExporterState.
const (
	traceExporterDegraded    = "degraded"
	traceExporterDisabled    = "disabled"
	traceExporterInitialized = "initialized"
)

// traceExporterState names the trace-export outcome in the one line an operator
// already reads at startup. Without it, "this service exports no traces" is
// only recoverable by correlating a separate warning that a log filter may drop.
func traceExporterState(tracingEndpoint telemetry.TraceExporterEndpoint, tracingInitErr error) string {
	switch {
	case tracingInitErr != nil:
		return traceExporterDegraded
	case !tracingEndpoint.Configured():
		return traceExporterDisabled
	default:
		return traceExporterInitialized
	}
}

// recordTraceExporterInitialization publishes trace-export initialization as a metric so an
// operator can alert on it. A startup warning is only visible to whoever reads
// the boot log; a service that answers every request while exporting no traces
// needs a signal that survives to a dashboard.
//
// initialized compares against traceExporterState rather than restating the
// condition, so the startup log line and this metric cannot drift apart.
func recordTraceExporterInitialization(
	ctx context.Context,
	log *slog.Logger,
	metrics *telemetry.Metrics,
	tracingEndpoint telemetry.TraceExporterEndpoint,
	tracingInitErr error,
) {
	initialized := traceExporterState(tracingEndpoint, tracingInitErr) == traceExporterInitialized
	if err := metrics.RecordTraceExporterInitialization(ctx, initialized); err != nil {
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

// newTelemetryFlush builds the flush, which takes its bound from the context
// it is called with.
//
// It deliberately derives no deadline of its own. The flush is the last teardown
// stage, so what it may spend is whatever the process grace period has left —
// a number only the caller holding the shutdown budget knows. A fixed deadline
// here would let the total teardown grow past the platform's grace period and
// get this stage killed for it.
func newTelemetryFlush(log *slog.Logger, shutdowns ...func(context.Context) error) func(context.Context) {
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
				"telemetry_flush_failed",
				startupLogArgs(
					startupLogComponentShutdown,
					startupOperationTelemetryFlush,
					"error",
					"error.type", "telemetry_flush",
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

// reportAdditionalAmbientOTLPEnv names standard OTEL_EXPORTER_OTLP_* variables
// present in addition to the endpoint source. Credential and trust conflicts
// rejected during setup are excluded; the remaining tuning variables stay under
// the official SDK's documented environment behavior.
//
// Conflicting credential and trust variables are excluded when this service
// named the endpoint: that case fails exporter setup and is already reported as
// degraded telemetry.
//
// Both signals are taken because the claim is about the process rather than
// about traces. A variable that supplied the metrics endpoint changed something,
// and reporting it here while reportMetricExporterState names its initialized
// endpoint_source would put two contradicting lines in one startup log. Two
// per-signal records would each be true and reproduce the same contradiction,
// so there is one record and it excludes what either signal honored.
func reportAdditionalAmbientOTLPEnv(
	ctx context.Context,
	log *slog.Logger,
	tracingEndpoint telemetry.TraceExporterEndpoint,
	metricsEndpoint telemetry.ExporterEndpoint,
) {
	honored := []string{tracingEndpoint.Source, metricsEndpoint.Source}
	additional := slices.DeleteFunc(telemetry.AmbientOTLPExporterEnv(), func(name string) bool {
		return slices.Contains(honored, name)
	})
	additional = withoutConflicting(additional, tracingEndpoint.ConfiguredByService, telemetry.ConflictingTraceExporterEnv)
	additional = withoutConflicting(
		additional, metricsEndpoint.ConfiguredByService, telemetry.ConflictingMetricExporterEnv,
	)
	if len(additional) == 0 {
		return
	}

	mode := startupDependencyModeFeatureOff
	if tracingEndpoint.Configured() || metricsEndpoint.Configured() {
		mode = startupDependencyModeConfigured
	}

	log.WarnContext(
		ctx,
		"telemetry_ambient_env_present",
		startupLogArgs(
			startupLogComponentStartupProbes,
			startupOperationTelemetryInit,
			"degraded",
			"dependency", startupDependencyTelemetry,
			"mode", mode,
			"reason", "ambient_exporter_env_present",
			"env.present", strings.Join(additional, ", "),
			// The shared exporter root, which is what an operator sets to own
			// both destinations rather than one signal's override.
			"config.key", telemetry.SharedOTLPExporterConfigKey,
		)...,
	)
}

// withoutConflicting drops the variables a signal rejects rather than ignores.
//
// A conflict only exists when this service named that signal's endpoint itself;
// when the platform named it, the platform owns the credentials with it and
// nothing was refused.
func withoutConflicting(
	additional []string,
	configuredByService bool,
	conflicting func() []string,
) []string {
	if !configuredByService {
		return additional
	}
	rejected := conflicting()
	return slices.DeleteFunc(additional, func(name string) bool {
		return slices.Contains(rejected, name)
	})
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
				"reason", telemetry.FailureReason(setupErr),
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
				"reason", telemetry.FailureReason(result.ExportErr),
				"err", result.ExportErr,
			)...,
		)
	case !result.PushInitialized():
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
