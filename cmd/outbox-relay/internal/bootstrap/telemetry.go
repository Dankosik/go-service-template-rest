package bootstrap

import (
	"context"
	"log/slog"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

// setupTelemetry installs both signals independently and returns the flush.
//
// Neither failure is fatal, and neither blocks the other — the same reasoning and
// shape as bootstrapTelemetryStage in cmd/service/internal/bootstrap. Metrics fall
// back to the no-op provider [telemetry.Metrics.MeterProvider] already returns, so
// the outbox instruments built from it record into nothing rather than panicking.
func setupTelemetry(
	ctx context.Context,
	cfg config.Config,
	metrics *telemetry.Metrics,
	log *slog.Logger,
) func(context.Context) {
	instanceID := telemetry.ResolveInstanceID(cfg.App.InstanceID)
	metricsResult, metricsErr := telemetry.SetupMetrics(ctx, metrics, runtimeopts.Metrics(cfg, instanceID))
	if metricsErr != nil {
		log.WarnContext(ctx, "outbox_metrics_degraded", "reason", telemetry.FailureReason(metricsErr))
	}
	_, tracingShutdown, tracingErr := telemetry.SetupTracing(ctx, runtimeopts.Tracing(cfg, instanceID))
	if tracingErr != nil {
		log.WarnContext(ctx, "outbox_tracing_degraded", "reason", telemetry.FailureReason(tracingErr))
	}
	if metricsResult.ExportErr != nil {
		log.WarnContext(ctx, "outbox_metrics_export_degraded", "reason", telemetry.FailureReason(metricsResult.ExportErr))
	}
	return func(shutdownCtx context.Context) {
		if tracingShutdown != nil {
			_ = tracingShutdown(shutdownCtx)
		}
		if metricsResult.Shutdown != nil {
			_ = metricsResult.Shutdown(shutdownCtx)
		}
	}
}
