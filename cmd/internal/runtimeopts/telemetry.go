package runtimeopts

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

// TelemetryFlush releases both signal providers. It takes its bound from the
// context it is called with, because what the flush may spend is whatever the
// caller's shutdown budget has left — a number only that caller holds.
type TelemetryFlush func(context.Context)

// InstallTelemetry installs both OpenTelemetry signals for a background binary
// and returns the flush its shutdown owes them.
//
// The two signals are installed independently: a metrics provider that could not
// be built must not cost the process its tracer, because logctx reads request
// correlation off the span context that provider produces. That is also why the
// flush is returned even alongside an error — a caller that refuses to start
// still has a tracer to release, and returning nil there is how the exporter
// goroutine outlived the process that gave up on it.
//
// component prefixes the degraded-signal records. It is a parameter rather than
// a constant because those names are what an operator's alert matches on, and
// the binaries already publish their own.
//
// The metrics error is returned rather than logged because the two callers
// answer it differently, and that difference is a policy each of them owns: a
// worker with no meter cannot report what it consumed, so nothing would notice
// it stopped consuming, while the relay keeps publishing against the no-op
// provider [telemetry.Metrics] already returns. The operator-facing wording is
// still built here, so a caller applies its policy rather than restating what
// failed. Everything below is degradation both report the same way, so it is
// reported here.
//
// cmd/service does not use this. Its telemetry stage carries per-signal startup
// budgets, the additional ambient-environment report, and trace-exporter initialization
// metric, none of which a background binary publishes; see
// bootstrapTelemetryStage in cmd/service/internal/bootstrap.
func InstallTelemetry(
	ctx context.Context,
	cfg config.Config,
	metrics *telemetry.Metrics,
	log *slog.Logger,
	component string,
) (TelemetryFlush, error) {
	telemetry.InstallErrorHandler(ctx, log)

	// Resolved once and shared by both signals, so a span and a metric from the
	// same replica carry the same resource identity.
	instanceID := telemetry.ResolveInstanceID(cfg.App.InstanceID)

	metricsResult, metricsErr := telemetry.SetupMetrics(ctx, metrics, Metrics(cfg, instanceID))
	if metricsErr != nil {
		metricsErr = fmt.Errorf("initialize %s metrics: %w", component, metricsErr)
	}
	_, tracingShutdown, tracingErr := telemetry.SetupTracing(ctx, Tracing(cfg, instanceID))

	if tracingErr != nil {
		logDegradedSignal(ctx, log, component+"_tracing_degraded", tracingErr)
	}
	// Distinct from a metrics provider that could not be built: the provider
	// exists and records, and only the export leg is degraded.
	if metricsResult.ExportErr != nil {
		logDegradedSignal(ctx, log, component+"_metrics_export_degraded", metricsResult.ExportErr)
	}

	return func(shutdownCtx context.Context) {
		if tracingShutdown != nil {
			_ = tracingShutdown(shutdownCtx)
		}
		if metricsResult.Shutdown != nil {
			_ = metricsResult.Shutdown(shutdownCtx)
		}
	}, metricsErr
}

func logDegradedSignal(ctx context.Context, log *slog.Logger, event string, err error) {
	log.WarnContext(ctx, event,
		"operation", "telemetry_init", "outcome", "degraded",
		"reason", telemetry.FailureReason(err), "err", err,
	)
}
