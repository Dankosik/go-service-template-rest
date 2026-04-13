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
	)
	if err != nil {
		return startupBootstrap{}, err
	}

	log := bootstrapLoggerStage(cfg)
	netPolicyResult := loadNetworkPolicy()
	telemetryCleanup, telemetryInitErr := bootstrapTelemetryStage(startupCtx, cfg, metrics, log, netPolicyResult)
	tracer, bootstrapCtx, bootstrapSpan := bootstrapTraceStage(startupCtx)
	spanOwnedByCaller := false
	defer func() {
		if !spanOwnedByCaller {
			bootstrapSpan.End()
		}
	}()

	bootstrapReportStage(bootstrapCtx, tracer, log, cfg, loadOptions, configReport, telemetryInitErr)
	netPolicy, err := bootstrapNetworkPolicyStage(
		bootstrapCtx,
		bootstrapSpan,
		metrics,
		log,
		netPolicyResult,
		cfg,
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
		metrics.ObserveConfigLoadDuration(configLoadStageMetricLabel(failedStage), telemetry.ConfigLoadResultError, failedDuration)
		metrics.IncConfigFailure(errorType)
		metrics.IncStartupRejection(startupRejectionReasonForConfigErrorType(errorType))
		metrics.IncConfigStartupOutcome(telemetry.ConfigStartupOutcomeRejected)
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
		return config.Config{}, config.LoadReport{}, fmt.Errorf("load config (%s): %w", errorType, err)
	}

	compatibilityStarted := time.Now()
	if err := validateStartupBudgetCompatibility(cfg); err != nil {
		failedDuration := time.Since(compatibilityStarted)
		if failedDuration <= 0 {
			failedDuration = time.Millisecond
		}
		errorType := startupConfigCompatibilityReason
		metrics.ObserveConfigLoadDuration(startupConfigCompatibilityStage, telemetry.ConfigLoadResultError, failedDuration)
		metrics.IncConfigFailure(errorType)
		metrics.IncStartupRejection(telemetry.StartupRejectionReasonConfigStartupCompatibility)
		metrics.IncConfigStartupOutcome(telemetry.ConfigStartupOutcomeRejected)
		slog.Error(
			"config_load_failed",
			startupLogArgs(
				startupCtx,
				"config_loader",
				"startup_compatibility",
				"error",
				"stage", startupConfigCompatibilityStage,
				"error.type", errorType,
			)...,
		)
		return config.Config{}, config.LoadReport{}, fmt.Errorf("load config (%s): %w", errorType, err)
	}

	recordConfigSuccessMetrics(metrics, configReport)
	if len(configReport.UnknownKeyWarnings) > 0 {
		metrics.AddConfigUnknownKeyWarnings(len(configReport.UnknownKeyWarnings))
	}

	return cfg, configReport, nil
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

func bootstrapTelemetryStage(
	startupCtx context.Context,
	cfg config.Config,
	metrics *telemetry.Metrics,
	log *slog.Logger,
	netPolicyResult networkPolicyLoadResult,
) (func(context.Context), error) {
	metrics.MarkStartupDependencyBlocked(startupDependencyTelemetry, startupDependencyModeOptionalFailOpen)
	exporterCfg := traceExporterConfig(cfg)
	targetAdmission, telemetryInitErr := admitTelemetryExporterTarget(exporterCfg, netPolicyResult)
	if telemetryInitErr != nil {
		metrics.IncTelemetryInitFailure(telemetryInitFailureReason(telemetryInitErr))
		metrics.MarkStartupDependencyReady(startupDependencyTelemetry, startupDependencyModeFeatureOff)
		return func(context.Context) {}, fmt.Errorf("setup tracing: %w", telemetryInitErr)
	}
	if targetAdmission == telemetryExporterTargetDeferredToNetworkPolicy {
		metrics.MarkStartupDependencyReady(startupDependencyTelemetry, startupDependencyModeFeatureOff)
		return func(context.Context) {}, nil
	}

	telemetryCtx, telemetryCancel := withStageBudget(startupCtx, startupTelemetryBudget)
	tracingShutdown, telemetryInitErr := telemetry.SetupTracing(telemetryCtx, telemetry.TracingConfig{
		ServiceName:      cfg.Observability.OTel.ServiceName,
		ServiceVersion:   cfg.App.Version,
		DeploymentEnv:    cfg.App.Env,
		TracesSampler:    cfg.Observability.OTel.TracesSampler,
		TracesSamplerArg: cfg.Observability.OTel.TracesSamplerArg,
		Exporter:         exporterCfg,
	})
	telemetryCancel()
	if telemetryInitErr != nil {
		metrics.IncTelemetryInitFailure(telemetryInitFailureReason(telemetryInitErr))
		metrics.MarkStartupDependencyReady(startupDependencyTelemetry, startupDependencyModeFeatureOff)
		return func(context.Context) {}, fmt.Errorf("setup tracing: %w", telemetryInitErr)
	}

	metrics.MarkStartupDependencyReady(startupDependencyTelemetry, startupDependencyModeOptionalFailOpen)
	return func(shutdownBaseCtx context.Context) {
		log.Info(
			"telemetry_flush_started",
			startupLogArgs(
				shutdownBaseCtx,
				startupLogComponentShutdown,
				startupOperationTelemetryFlush,
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
					startupLogComponentShutdown,
					startupOperationTelemetryFlush,
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
				startupLogComponentShutdown,
				startupOperationTelemetryFlush,
				"success",
			)...,
		)
	}, nil
}

func traceExporterConfig(cfg config.Config) telemetry.TraceExporterConfig {
	return telemetry.TraceExporterConfig{
		OTLPEndpoint:       cfg.Observability.OTel.Exporter.OTLPEndpoint,
		OTLPTracesEndpoint: cfg.Observability.OTel.Exporter.OTLPTracesEndpoint,
		OTLPHeaders:        cfg.Observability.OTel.Exporter.OTLPHeaders,
		OTLPProtocol:       cfg.Observability.OTel.Exporter.OTLPProtocol,
	}
}

type telemetryExporterTargetAdmission int

const (
	telemetryExporterTargetUnconfigured telemetryExporterTargetAdmission = iota
	telemetryExporterTargetAllowed
	telemetryExporterTargetDeferredToNetworkPolicy
)

func admitTelemetryExporterTarget(cfg telemetry.TraceExporterConfig, netPolicyResult networkPolicyLoadResult) (telemetryExporterTargetAdmission, error) {
	target, err := telemetry.DescribeTraceExporterTarget(cfg)
	if err != nil {
		return telemetryExporterTargetUnconfigured, fmt.Errorf("describe trace exporter target: %w", err)
	}
	if !target.Configured {
		return telemetryExporterTargetUnconfigured, nil
	}

	policyErr := netPolicyResult.err
	if policyErr != nil {
		//nolint:nilerr // Invalid network policy defers optional telemetry exporter admission fail-open.
		return telemetryExporterTargetDeferredToNetworkPolicy, nil
	}
	if err := netPolicyResult.policy.EnforceEgressTarget(target.Target, target.Scheme); err != nil {
		return telemetryExporterTargetUnconfigured, fmt.Errorf("telemetry egress target denied: %w", err)
	}
	return telemetryExporterTargetAllowed, nil
}

// bootstrapTraceStage transfers bootstrapSpan ownership to bootstrapRuntime.
// bootstrapRuntime ends it on setup failure; startupSpanController closes it after successful startup.
//
//nolint:spancheck // bootstrapSpan intentionally outlives this helper and is returned to its lifecycle owner.
func bootstrapTraceStage(startupCtx context.Context) (trace.Tracer, context.Context, trace.Span) {
	tracer := otel.Tracer("service.startup")
	bootstrapCtx, bootstrapSpan := tracer.Start(startupCtx, "config.bootstrap")
	return tracer, bootstrapCtx, bootstrapSpan
}

func bootstrapReportStage(
	bootstrapCtx context.Context,
	tracer trace.Tracer,
	log *slog.Logger,
	cfg config.Config,
	loadOptions config.LoadOptions,
	configReport config.LoadReport,
	telemetryInitErr error,
) {
	for _, stage := range configLoadStageDurations(configReport) {
		recordConfigStageSpan(bootstrapCtx, tracer, stage.stage, stage.duration, "success", "")
	}

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
				startupLogComponentStartupProbes,
				startupOperationTelemetryInit,
				"degraded",
				"dependency", startupDependencyTelemetry,
				"mode", startupDependencyModeFeatureOff,
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
	netPolicyResult networkPolicyLoadResult,
	cfg config.Config,
) (networkPolicy, error) {
	if netPolicyResult.err != nil {
		policyClass, reasonClass := networkPolicyErrorLabels(netPolicyResult.err)
		return networkPolicy{}, rejectStartupForPolicyViolation(
			bootstrapCtx,
			bootstrapSpan,
			metrics,
			log,
			startupDependencyNetworkPolicy,
			fmt.Errorf("invalid network policy configuration: %w", netPolicyResult.err),
			"policy.class", policyClass,
			"reason.class", reasonClass,
		)
	}
	netPolicy := netPolicyResult.policy.withIngressExposure(cfg.App.Env, cfg.HTTP.Addr)

	if ingressErr := netPolicy.EnforceIngress(); ingressErr != nil {
		return networkPolicy{}, rejectStartupForPolicyViolation(
			bootstrapCtx,
			bootstrapSpan,
			metrics,
			log,
			startupDependencyIngressPolicy,
			ingressErr,
		)
	}

	if metricsExposureErr := netPolicy.ValidateOperationalMetricsExposure(); metricsExposureErr != nil {
		return networkPolicy{}, rejectStartupForPolicyViolation(
			bootstrapCtx,
			bootstrapSpan,
			metrics,
			log,
			startupDependencyMetricsExposure,
			metricsExposureErr,
		)
	}

	if egressErr := netPolicy.ValidateEgressExceptionState(); egressErr != nil {
		return networkPolicy{}, rejectStartupForPolicyViolation(
			bootstrapCtx,
			bootstrapSpan,
			metrics,
			log,
			startupDependencyEgressException,
			egressErr,
		)
	}

	return netPolicy, nil
}
