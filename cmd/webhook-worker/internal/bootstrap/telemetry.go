package bootstrap

import (
	"context"
	"log/slog"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

func setupTelemetry(ctx context.Context, cfg config.Config, metrics *telemetry.Metrics, log *slog.Logger) runtimeopts.TelemetryFlush {
	flush, err := runtimeopts.InstallTelemetry(ctx, cfg, metrics, log, "webhook-worker")
	if err != nil {
		log.WarnContext(ctx, "webhook_metrics_degraded", "reason", telemetry.FailureReason(err))
	}
	return flush
}
