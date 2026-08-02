package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

func setupTelemetry(ctx context.Context, cfg config.Config, metrics *telemetry.Metrics, log *slog.Logger) (func(context.Context), error) {
	instanceID := telemetry.ResolveInstanceID(cfg.App.InstanceID)
	metricsResult, err := telemetry.SetupMetrics(ctx, metrics, telemetry.MetricsConfig{
		ServiceName: cfg.Observability.OTel.ServiceName, ServiceVersion: cfg.App.Version,
		ServiceCommit: cfg.App.Commit, ServiceInstanceID: instanceID, DeploymentEnv: cfg.App.Env,
		Exporter: telemetry.MetricExporterConfig{
			OTLPEndpoint:       cfg.Observability.OTel.Exporter.OTLPMetricsEndpoint,
			SharedOTLPEndpoint: cfg.Observability.OTel.Exporter.OTLPEndpoint,
			OTLPHeaders:        cfg.Observability.OTel.Exporter.OTLPHeaders,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("initialize worker metrics: %w", err)
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
		log.WarnContext(ctx, "worker_tracing_degraded",
			"operation", "telemetry_init", "outcome", "degraded",
			"reason", workerTelemetryFailureReason(tracingErr), "err", tracingErr,
		)
	}
	if metricsResult.ExportErr != nil {
		log.WarnContext(ctx, "worker_metrics_export_degraded",
			"operation", "telemetry_init", "outcome", "degraded",
			"reason", workerTelemetryFailureReason(metricsResult.ExportErr), "err", metricsResult.ExportErr,
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

func workerTelemetryFailureReason(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return "setup_error"
	}
}
