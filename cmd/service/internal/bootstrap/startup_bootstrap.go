package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type startupBootstrap struct {
	cfg              config.Config
	log              *slog.Logger
	tracer           trace.Tracer
	bootstrapSpan    trace.Span
	networkPolicy    networkPolicy
	telemetryCleanup func(context.Context)
}

func bootstrapRuntime(
	startupCtx context.Context,
	loadOptions config.LoadOptions,
	metrics *telemetry.Metrics,
	deployTelemetry *deployTelemetryRecorder,
	startupLifecycleStartedAt time.Time,
) (result startupBootstrap, err error) {
	telemetryCleanup := func(context.Context) {}
	defer func() {
		if err != nil {
			telemetryCleanup(startupCtx)
		}
	}()

	cfg, configReport, err := bootstrapConfigStage(
		startupCtx,
		loadOptions,
		metrics,
		deployTelemetry,
		startupLifecycleStartedAt,
	)
	if err != nil {
		return startupBootstrap{}, err
	}

	log := bootstrapLoggerStage(cfg, deployTelemetry)
	telemetryCleanup, telemetryInitErr := bootstrapTelemetryStage(startupCtx, cfg, metrics, log)
	tracer, bootstrapCtx, bootstrapSpan := bootstrapTraceStage(startupCtx)
	spanOwnedByCaller := false
	defer func() {
		if !spanOwnedByCaller {
			bootstrapSpan.End()
		}
	}()

	bootstrapReportStage(bootstrapCtx, tracer, log, deployTelemetry, cfg, loadOptions, configReport, telemetryInitErr)
	netPolicy, err := bootstrapNetworkPolicyStage(
		bootstrapCtx,
		bootstrapSpan,
		metrics,
		log,
		deployTelemetry,
		startupLifecycleStartedAt,
	)
	if err != nil {
		return startupBootstrap{}, err
	}

	result = startupBootstrap{
		cfg:              cfg,
		log:              log,
		tracer:           tracer,
		bootstrapSpan:    bootstrapSpan,
		networkPolicy:    netPolicy,
		telemetryCleanup: telemetryCleanup,
	}
	spanOwnedByCaller = true
	return result, nil
}

func startupBootstrapContext(startupCtx context.Context, bootstrapSpan trace.Span) context.Context {
	return trace.ContextWithSpan(startupCtx, bootstrapSpan)
}

func bootstrapConfigStage(
	startupCtx context.Context,
	loadOptions config.LoadOptions,
	metrics *telemetry.Metrics,
	deployTelemetry *deployTelemetryRecorder,
	startupLifecycleStartedAt time.Time,
) (config.Config, config.LoadReport, error) {
	slog.Info(
		"config_load_started",
		startupLogArgs(
			startupCtx,
			"config_loader",
			"load",
			"started",
			"config.strict", loadOptions.Strict,
			"config.file", loadOptions.ConfigPath,
			"config.overlay_count", len(loadOptions.ConfigOverlays),
		)...,
	)

	cfg, configReport, err := config.LoadDetailedWithContext(startupCtx, loadOptions)
	if err != nil {
		failedStage, failedDuration := failedStageDetails(configReport)
		errorType := config.ErrorType(err)
		metrics.ObserveConfigLoadDuration(failedStage, "error", failedDuration)
		metrics.IncConfigValidationFailure(errorType)
		metrics.IncConfigStartupOutcome("rejected")
		slog.Error(
			"config_load_failed",
			startupLogArgs(
				startupCtx,
				"config_loader",
				"load",
				"error",
				"stage", failedStage,
				"error.type", errorType,
			)...,
		)
		recordAdmissionFailure(startupCtx, deployTelemetry, "startup_error", "startup")
		return config.Config{}, config.LoadReport{}, fmt.Errorf("load config (%s): %w", errorType, err)
	}

	recordConfigSuccessMetrics(metrics, configReport)
	if len(configReport.UnknownKeyWarnings) > 0 {
		metrics.AddConfigUnknownKeyWarnings(len(configReport.UnknownKeyWarnings))
	}

	return cfg, configReport, nil
}

func bootstrapLoggerStage(cfg config.Config, deployTelemetry *deployTelemetryRecorder) *slog.Logger {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.Log.Level}))
	log = log.With(
		"service.name", cfg.Observability.OTel.ServiceName,
		"service.version", cfg.App.Version,
		"deployment.environment.name", cfg.App.Env,
	)
	slog.SetDefault(log)
	deployTelemetry.SetLogger(log)
	deployTelemetry.SetEnvironment(cfg.App.Env)
	return log
}

func bootstrapTelemetryStage(
	startupCtx context.Context,
	cfg config.Config,
	metrics *telemetry.Metrics,
	log *slog.Logger,
) (func(context.Context), error) {
	metrics.SetStartupDependencyStatus("telemetry", "optional_fail_open", false)
	telemetryCtx, telemetryCancel := withStageBudget(startupCtx, startupTelemetryBudget)
	tracingShutdown, telemetryInitErr := telemetry.SetupTracing(telemetryCtx, telemetry.TracingConfig{
		ServiceName:      cfg.Observability.OTel.ServiceName,
		ServiceVersion:   cfg.App.Version,
		DeploymentEnv:    cfg.App.Env,
		TracesSampler:    cfg.Observability.OTel.TracesSampler,
		TracesSamplerArg: cfg.Observability.OTel.TracesSamplerArg,
		Exporter: telemetry.TraceExporterConfig{
			OTLPEndpoint:       cfg.Observability.OTel.Exporter.OTLPEndpoint,
			OTLPTracesEndpoint: cfg.Observability.OTel.Exporter.OTLPTracesEndpoint,
			OTLPHeaders:        cfg.Observability.OTel.Exporter.OTLPHeaders,
			OTLPProtocol:       cfg.Observability.OTel.Exporter.OTLPProtocol,
		},
	})
	telemetryCancel()
	if telemetryInitErr != nil {
		metrics.IncTelemetryInitFailure(telemetryInitFailureReason(telemetryInitErr))
		metrics.SetStartupDependencyStatus("telemetry", "feature_off", false)
		return func(context.Context) {}, telemetryInitErr
	}

	metrics.SetStartupDependencyStatus("telemetry", "optional_fail_open", true)
	return func(shutdownBaseCtx context.Context) {
		log.Info(
			"telemetry_flush_started",
			startupLogArgs(
				shutdownBaseCtx,
				"shutdown",
				"telemetry_flush",
				"started",
			)...,
		)
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(shutdownBaseCtx), telemetryShutdownTimeout)
		defer cancel()
		if shutdownErr := tracingShutdown(shutdownCtx); shutdownErr != nil {
			log.Error(
				"tracing shutdown failed",
				startupLogArgs(
					shutdownBaseCtx,
					"startup_probes",
					"telemetry_shutdown",
					"error",
					"error.type", "dependency_init",
					"err", shutdownErr,
				)...,
			)
			return
		}
		log.Info(
			"telemetry_flush_completed",
			startupLogArgs(
				shutdownBaseCtx,
				"shutdown",
				"telemetry_flush",
				"success",
			)...,
		)
	}, nil
}

func bootstrapTraceStage(startupCtx context.Context) (trace.Tracer, context.Context, trace.Span) {
	tracer := otel.Tracer("service.startup")
	bootstrapCtx, bootstrapSpan := tracer.Start(startupCtx, "config.bootstrap")
	return tracer, bootstrapCtx, bootstrapSpan
}

func bootstrapReportStage(
	bootstrapCtx context.Context,
	tracer trace.Tracer,
	log *slog.Logger,
	deployTelemetry *deployTelemetryRecorder,
	cfg config.Config,
	loadOptions config.LoadOptions,
	configReport config.LoadReport,
	telemetryInitErr error,
) {
	recordConfigStageSpan(tracer, bootstrapCtx, config.StageLoadDefaults, configReport.LoadDefaultsDuration, "success", "")
	recordConfigStageSpan(tracer, bootstrapCtx, config.StageLoadFile, configReport.LoadFileDuration, "success", "")
	recordConfigStageSpan(tracer, bootstrapCtx, config.StageLoadEnv, configReport.LoadEnvDuration, "success", "")
	recordConfigStageSpan(tracer, bootstrapCtx, config.StageParse, configReport.ParseDuration, "success", "")
	recordConfigStageSpan(tracer, bootstrapCtx, config.StageValidate, configReport.ValidateDuration, "success", "")

	log.Info(
		"config_validated",
		startupLogArgs(
			bootstrapCtx,
			"config_validator",
			"validate",
			"success",
			"load.duration_ms", configReport.LoadDuration.Milliseconds(),
			"validate.duration_ms", configReport.ValidateDuration.Milliseconds(),
			"unknown_key_warnings", len(configReport.UnknownKeyWarnings),
		)...,
	)

	for _, key := range configReport.UnknownKeyWarnings {
		log.Warn(
			"unknown config key ignored (strict mode disabled)",
			startupLogArgs(
				bootstrapCtx,
				"config_validator",
				"unknown_key",
				"warning",
				"key", key,
			)...,
		)
	}
	if telemetryInitErr != nil {
		log.Warn(
			"startup_dependency_degraded",
			startupLogArgs(
				bootstrapCtx,
				"startup_probes",
				"telemetry_init",
				"degraded",
				"dependency", "telemetry",
				"mode", "feature_off",
				"reason", telemetryInitFailureReason(telemetryInitErr),
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
			"config.strict", loadOptions.Strict,
			"config.file", loadOptions.ConfigPath,
			"config.overlay_count", len(loadOptions.ConfigOverlays),
			"app.env", cfg.App.Env,
			"http.addr", cfg.HTTP.Addr,
			"postgres.enabled", cfg.Postgres.Enabled,
			"redis.enabled", cfg.Redis.Enabled,
			"redis.mode", cfg.Redis.Mode,
			"mongo.enabled", cfg.Mongo.Enabled,
		)...,
	)
}

func bootstrapNetworkPolicyStage(
	bootstrapCtx context.Context,
	bootstrapSpan trace.Span,
	metrics *telemetry.Metrics,
	log *slog.Logger,
	deployTelemetry *deployTelemetryRecorder,
	startupLifecycleStartedAt time.Time,
) (networkPolicy, error) {
	netPolicy, netPolicyErr := loadNetworkPolicyFromEnv()
	if netPolicyErr != nil {
		policyClass, reasonClass := networkPolicyErrorLabels(netPolicyErr)
		if policyClass == "egress" {
			deployTelemetry.RecordNetworkEgressPolicyViolation(bootstrapCtx, reasonClass, "deny")
		} else {
			deployTelemetry.RecordNetworkIngressPolicyViolation(bootstrapCtx, reasonClass, "deny")
		}
		return networkPolicy{}, rejectStartupForPolicyViolation(
			bootstrapCtx,
			bootstrapSpan,
			metrics,
			log,
			deployTelemetry,
			startupLifecycleStartedAt,
			"network_policy",
			fmt.Errorf("%w: invalid network policy configuration: %w", config.ErrDependencyInit, netPolicyErr),
		)
	}

	if ingressErr := netPolicy.EnforceIngress(bootstrapCtx, deployTelemetry); ingressErr != nil {
		return networkPolicy{}, rejectStartupForPolicyViolation(
			bootstrapCtx,
			bootstrapSpan,
			metrics,
			log,
			deployTelemetry,
			startupLifecycleStartedAt,
			"ingress_policy",
			ingressErr,
		)
	}

	if egressErr := netPolicy.EmitEgressExceptionState(bootstrapCtx, deployTelemetry); egressErr != nil {
		return networkPolicy{}, rejectStartupForPolicyViolation(
			bootstrapCtx,
			bootstrapSpan,
			metrics,
			log,
			deployTelemetry,
			startupLifecycleStartedAt,
			"egress_exception",
			egressErr,
		)
	}

	return netPolicy, nil
}
