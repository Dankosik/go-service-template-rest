package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/example/go-service-template-rest/internal/config"
	"go.opentelemetry.io/otel/trace"
)

const (
	telemetryFailureReasonSetupError       = "setup_error"
	telemetryFailureReasonDeadlineExceeded = "deadline_exceeded"
	telemetryFailureReasonCanceled         = "canceled"
)

func startupLogArgs(ctx context.Context, component, operation, outcome string, extra ...any) []any {
	args := make([]any, 0, 6+len(extra))
	args = append(args,
		"component", component,
		"operation", operation,
		"outcome", outcome,
	)

	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		args = append(args,
			"trace_id", spanCtx.TraceID().String(),
			"span_id", spanCtx.SpanID().String(),
		)
	}

	args = append(args, extra...)
	return args
}

func telemetryInitFailureReason(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return telemetryFailureReasonDeadlineExceeded
	case errors.Is(err, context.Canceled):
		return telemetryFailureReasonCanceled
	default:
		return telemetryFailureReasonSetupError
	}
}

func bootstrapLoggerStage(cfg config.Config) *slog.Logger {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.Log.Level}))
	log = log.With(
		"service.name", cfg.Observability.OTel.ServiceName,
		"service.version", cfg.App.Version,
		"deployment.environment.name", cfg.App.Env,
	)
	slog.SetDefault(log)
	return log
}

func bootstrapReportStage(
	bootstrapCtx context.Context,
	log *slog.Logger,
	cfg config.Config,
	loadOptions config.LoadOptions,
	configReport config.LoadReport,
	telemetryInitErr error,
) {
	log.Info(
		"config_validated",
		startupLogArgs(
			bootstrapCtx,
			"config_validator",
			"validate",
			"success",
			"load.duration_ms", configReport.LoadDuration.Milliseconds(),
			"validate.duration_ms", configReport.ValidateDuration.Milliseconds(),
		)...,
	)

	if telemetryInitErr != nil {
		// The cause belongs in the record: telemetry setup errors name
		// configuration keys and environment variables, never secret values,
		// and an operator cannot act on a bare reason class.
		log.Warn(
			"startup_dependency_degraded",
			startupLogArgs(
				bootstrapCtx,
				startupLogComponentStartupProbes,
				startupOperationTelemetryInit,
				"degraded",
				"dependency", startupDependencyTelemetry,
				"mode", startupDependencyModeFeatureOff,
				"reason", telemetryInitFailureReason(telemetryInitErr),
				"err", telemetryInitErr,
			)...,
		)
	}

	log.Info(
		"startup config summary",
		startupLogArgs(
			bootstrapCtx,
			"config_loader",
			"startup_summary",
			"success",
			"config.file", loadOptions.ConfigPath,
			"config.overlay_count", len(loadOptions.ConfigOverlays),
			"app.env", cfg.App.Env,
			"http.addr", cfg.HTTP.Addr,
			"metrics.addr", cfg.Observability.Metrics.Addr,
			"postgres.enabled", cfg.Postgres.Enabled,
		)...,
	)
}
