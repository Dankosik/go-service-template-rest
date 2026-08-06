package bootstrap

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/observability/logctx"
)

const (
	telemetryFailureReasonSetupError       = "setup_error"
	telemetryFailureReasonDeadlineExceeded = "deadline_exceeded"
	telemetryFailureReasonCanceled         = "canceled"
)

// startupLogArgs builds the stage attributes every startup and shutdown record
// carries. Trace correlation is deliberately absent: the logger installed by
// bootstrapLoggerStage adds it from the context passed to the logging call, so
// adding it here too would duplicate the keys on every record.
func startupLogArgs(component, operation, outcome string, extra ...any) []any {
	args := make([]any, 0, 6+len(extra))
	args = append(args,
		"component", component,
		"operation", operation,
		"outcome", outcome,
	)

	return append(args, extra...)
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

// newProcessLogger builds the logger every record in this process goes through.
//
// The logctx decorator is the reason this is one function rather than two
// literals: it publishes request and trace correlation from the context a record
// was logged with, so a service's own handlers get it without adding the
// attributes. A logger built without it looks identical and silently drops the
// only keys that join an application error to its trace. The writer is a
// parameter so a test can prove the decorator is installed.
func newProcessLogger(out io.Writer, level slog.Level) *slog.Logger {
	return slog.New(logctx.New(slog.NewJSONHandler(out, &slog.HandlerOptions{Level: level})))
}

func bootstrapLoggerStage(cfg config.Config) *slog.Logger {
	log := newProcessLogger(os.Stdout, cfg.Log.Level).With(
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
	traceEndpoint telemetry.TraceExporterEndpoint,
	telemetryInitErr error,
) {
	log.InfoContext(
		bootstrapCtx,
		"config_validated",
		startupLogArgs(
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
		log.WarnContext(
			bootstrapCtx,
			"startup_dependency_degraded",
			startupLogArgs(
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

	log.InfoContext(
		bootstrapCtx,
		"startup config summary",
		startupLogArgs(
			"config_loader",
			"startup_summary",
			"success",
			"config.file", loadOptions.ConfigPath,
			"config.overlay_count", len(loadOptions.ConfigOverlays),
			"app.env", cfg.App.Env,
			"http.addr", cfg.HTTP.Addr,
			// profile:grpc:start
			"grpc.enabled", cfg.GRPC.Server.Enabled,
			"grpc.addr", cfg.GRPC.Server.Addr,
			"grpc.transport_security", cfg.GRPC.Server.TransportSecurity,
			// profile:grpc:end
			"metrics.addr", cfg.Observability.Metrics.Addr,
			"tracing.exporter", traceExporterState(traceEndpoint, telemetryInitErr),
			// The endpoint can come from this service's own configuration or
			// from a platform-injected variable, and an operator debugging where
			// traces went needs to know which without reading another line.
			"tracing.endpoint_source", traceEndpoint.Source,
			// profile:database-postgres:start
			"postgres.enabled", cfg.Postgres.Enabled,
			// profile:database-postgres:end
		)...,
	)
}

// traceExporterState names the trace-export outcome in the one line an operator
// already reads at startup. Without it, "this service exports no traces" is
// only recoverable by correlating a separate warning that a log filter may drop.
func traceExporterState(traceEndpoint telemetry.TraceExporterEndpoint, telemetryInitErr error) string {
	switch {
	case telemetryInitErr != nil:
		return "degraded"
	case !traceEndpoint.Configured():
		return "disabled"
	default:
		return "active"
	}
}
