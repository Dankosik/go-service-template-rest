package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

// setupTelemetry installs both signals and returns the flush. Metrics are fatal
// and tracing is not: a worker with no meter cannot report what it consumed, so
// nothing would notice it stopped consuming, while a worker with no exporter for
// spans still records every count an alert is built on.
func setupTelemetry(ctx context.Context, cfg config.Config, metrics *telemetry.Metrics, log *slog.Logger) (func(context.Context), error) {
	instanceID := telemetry.ResolveInstanceID(cfg.App.InstanceID)
	metricsResult, err := telemetry.SetupMetrics(ctx, metrics, runtimeopts.Metrics(cfg, instanceID))
	if err != nil {
		return nil, fmt.Errorf("initialize worker metrics: %w", err)
	}
	_, tracingShutdown, tracingErr := telemetry.SetupTracing(ctx, runtimeopts.Tracing(cfg, instanceID))
	if tracingErr != nil {
		log.WarnContext(ctx, "worker_tracing_degraded",
			"operation", "telemetry_init", "outcome", "degraded",
			"reason", telemetry.FailureReason(tracingErr), "err", tracingErr,
		)
	}
	if metricsResult.ExportErr != nil {
		log.WarnContext(ctx, "worker_metrics_export_degraded",
			"operation", "telemetry_init", "outcome", "degraded",
			"reason", telemetry.FailureReason(metricsResult.ExportErr), "err", metricsResult.ExportErr,
		)
	}
	return func(shutdownCtx context.Context) {
		if tracingShutdown != nil {
			_ = tracingShutdown(shutdownCtx)
		}
		if metricsResult.Shutdown != nil {
			_ = metricsResult.Shutdown(shutdownCtx)
		}
	}, nil
}
