package bootstrap

import (
	"context"
	"log/slog"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

// setupTelemetry installs both signals and applies this binary's answer to the
// one outcome the two background binaries answer differently.
//
// A metrics provider that could not be built degrades the relay instead of
// stopping it, which is the opposite of cmd/worker: the outbox instruments fall
// back to the no-op provider [telemetry.Metrics.MeterProvider] already returns
// and record into nothing rather than panicking, and a relay that stopped
// publishing for it would turn an observability outage into a delivery one.
// runtimeopts.InstallTelemetry owns everything above that choice.
func setupTelemetry(
	ctx context.Context,
	cfg config.Config,
	metrics *telemetry.Metrics,
	log *slog.Logger,
) runtimeopts.TelemetryFlush {
	flush, metricsErr := runtimeopts.InstallTelemetry(ctx, cfg, metrics, log, "outbox")
	if metricsErr != nil {
		log.WarnContext(ctx, "outbox_metrics_degraded", "reason", telemetry.FailureReason(metricsErr))
	}
	return flush
}
