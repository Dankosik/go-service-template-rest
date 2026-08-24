package bootstrap

import (
	"context"
	"log/slog"
	"os"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

const (
	startupLogComponentStartupProbes = "startup_probes"
	startupLogComponentShutdown      = "shutdown"
)

// startupLogArgs builds the stage attributes every startup and shutdown record
// carries. Trace correlation is deliberately absent: the logger installed by
// bootstrapLoggerStage adds it from the context passed to the logging call, so
// adding it here too would duplicate the keys on every record.
func startupLogArgs(component, operation, outcome string, extra ...any) []any {
	args := []any{
		"component", component,
		"operation", operation,
		"outcome", outcome,
	}

	return append(args, extra...)
}

// logProcessExit writes the last record of the process, through the default
// logger rather than a captured one: it runs after the deferred teardown that
// may have replaced what the bootstrap logger wrote to.
func logProcessExit(ctx context.Context, runErr error) {
	if runErr != nil {
		slog.ErrorContext(
			ctx,
			"process_exit",
			startupLogArgs("lifecycle", "process_exit", "error", "err", runErr)...,
		)
		return
	}
	slog.InfoContext(
		ctx,
		"process_exit",
		startupLogArgs("lifecycle", "process_exit", "success")...,
	)
}

func bootstrapLoggerStage(cfg config.Config) *slog.Logger {
	log := runtimeopts.Logger(os.Stdout, cfg)
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
				"reason", telemetry.FailureReason(telemetryInitErr),
				"err", telemetryInitErr,
			)...,
		)
	}

	log.InfoContext(
		bootstrapCtx,
		"startup_config_summary",
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
