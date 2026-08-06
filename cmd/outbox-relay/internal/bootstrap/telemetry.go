package bootstrap

import (
	"context"
	"errors"
	"log/slog"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

// setupTelemetry installs both signals independently and returns the flush.
//
// Neither failure is fatal, and neither blocks the other. A relay that cannot
// export telemetry still drains its backlog durably, and refusing to start for
// that would trade an observability outage for a publication one — the same
// reasoning, and the same shape, as bootstrapTelemetryStage in
// cmd/service/internal/bootstrap/startup_telemetry.go. Metrics fall back to the
// no-op provider [telemetry.Metrics.MeterProvider] already returns, so the
// outbox instruments built from it record into nothing rather than panicking.
func setupTelemetry(
	ctx context.Context,
	cfg config.Config,
	metrics *telemetry.Metrics,
	log *slog.Logger,
) func(context.Context) {
	instanceID := telemetry.ResolveInstanceID(cfg.App.InstanceID)
	metricsResult, metricsErr := telemetry.SetupMetrics(ctx, metrics, telemetry.MetricsConfig{
		ServiceName: cfg.Observability.OTel.ServiceName, ServiceVersion: cfg.App.Version,
		ServiceCommit: cfg.App.Commit, ServiceInstanceID: instanceID, DeploymentEnv: cfg.App.Env,
		Exporter: telemetry.MetricExporterConfig{
			OTLPEndpoint:       cfg.Observability.OTel.Exporter.OTLPMetricsEndpoint,
			SharedOTLPEndpoint: cfg.Observability.OTel.Exporter.OTLPEndpoint,
			OTLPHeaders:        cfg.Observability.OTel.Exporter.OTLPHeaders,
		},
	})
	if metricsErr != nil {
		log.WarnContext(ctx, "outbox_metrics_degraded", "reason", telemetryFailureReason(metricsErr))
	}
	_, tracingShutdown, tracingErr := telemetry.SetupTracing(ctx, telemetry.TracingConfig{
		ServiceName: cfg.Observability.OTel.ServiceName, ServiceVersion: cfg.App.Version,
		ServiceCommit: cfg.App.Commit, ServiceInstanceID: instanceID, DeploymentEnv: cfg.App.Env,
		TracesSampler: cfg.Observability.OTel.TracesSampler, TracesSamplerArg: cfg.Observability.OTel.TracesSamplerArg,
		Exporter: telemetry.TraceExporterConfig{
			OTLPEndpoint: cfg.Observability.OTel.Exporter.OTLPEndpoint,
			OTLPHeaders:  cfg.Observability.OTel.Exporter.OTLPHeaders,
		},
	})
	if tracingErr != nil {
		log.WarnContext(ctx, "outbox_tracing_degraded", "reason", telemetryFailureReason(tracingErr))
	}
	if metricsResult.ExportErr != nil {
		log.WarnContext(ctx, "outbox_metrics_export_degraded", "reason", telemetryFailureReason(metricsResult.ExportErr))
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

func telemetryFailureReason(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return "setup_error"
	}
}
