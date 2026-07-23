package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

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

	bootstrapReportStage(bootstrapCtx, log, cfg, loadOptions, configReport, telemetryInitErr)
	netPolicy, err := bootstrapNetworkPolicyStage(
		bootstrapCtx,
		bootstrapSpan,
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
		failedStage := failedConfigStage(configReport)
		errorType := config.ErrorType(err)
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

	if err := validateStartupBudgetCompatibility(cfg); err != nil {
		errorType := startupConfigCompatibilityReason
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
	metricsCtx, metricsCancel := withStageBudget(startupCtx, startupTelemetryBudget)
	metricsShutdown, telemetryInitErr := telemetry.SetupMetrics(metricsCtx, metrics, telemetry.MetricsConfig{
		ServiceName:    cfg.Observability.OTel.ServiceName,
		ServiceVersion: cfg.App.Version,
		DeploymentEnv:  cfg.App.Env,
	})
	metricsCancel()
	if telemetryInitErr != nil {
		return func(context.Context) {}, fmt.Errorf("setup metrics: %w", telemetryInitErr)
	}
	cleanup := newTelemetryCleanup(log, metricsShutdown)

	exporterCfg := traceExporterConfig(cfg)
	skipTracing, telemetryInitErr := admitTelemetryExporterTarget(exporterCfg, netPolicyResult)
	if telemetryInitErr != nil {
		return cleanup, fmt.Errorf("setup tracing: %w", telemetryInitErr)
	}
	if skipTracing {
		return cleanup, nil
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
		return cleanup, fmt.Errorf("setup tracing: %w", telemetryInitErr)
	}

	return newTelemetryCleanup(log, tracingShutdown, metricsShutdown), nil
}

func newTelemetryCleanup(log *slog.Logger, shutdowns ...func(context.Context) error) func(context.Context) {
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

		var shutdownErrors []error
		for _, shutdown := range shutdowns {
			if shutdown == nil {
				continue
			}
			if shutdownErr := shutdown(shutdownCtx); shutdownErr != nil {
				shutdownErrors = append(shutdownErrors, shutdownErr)
			}
		}
		if shutdownErr := errors.Join(shutdownErrors...); shutdownErr != nil {
			log.Error(
				"telemetry shutdown failed",
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
	}
}

func traceExporterConfig(cfg config.Config) telemetry.TraceExporterConfig {
	return telemetry.TraceExporterConfig{
		OTLPEndpoint: cfg.Observability.OTel.Exporter.OTLPEndpoint,
		OTLPHeaders:  cfg.Observability.OTel.Exporter.OTLPHeaders,
	}
}

// admitTelemetryExporterTarget reports whether optional tracing setup must be
// skipped: an invalid network policy defers the exporter fail-open to the later
// mandatory policy enforcement stage instead of blocking startup here.
func admitTelemetryExporterTarget(cfg telemetry.TraceExporterConfig, netPolicyResult networkPolicyLoadResult) (bool, error) {
	target, err := telemetry.DescribeTraceExporterTarget(cfg)
	if err != nil {
		return false, fmt.Errorf("describe trace exporter target: %w", err)
	}
	if !target.Configured {
		return false, nil
	}

	if netPolicyResult.err != nil {
		//nolint:nilerr // Invalid network policy defers optional telemetry admission fail-open.
		return true, nil
	}
	if err := netPolicyResult.policy.EnforceEgressTarget(target.Target, target.Scheme); err != nil {
		return false, fmt.Errorf("telemetry egress target denied: %w", err)
	}
	return false, nil
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
		)...,
	)
}

func bootstrapNetworkPolicyStage(
	bootstrapCtx context.Context,
	bootstrapSpan trace.Span,
	log *slog.Logger,
	netPolicyResult networkPolicyLoadResult,
	cfg config.Config,
) (networkPolicy, error) {
	if netPolicyResult.err != nil {
		policyClass, reasonClass := networkPolicyErrorLabels(netPolicyResult.err)
		return networkPolicy{}, rejectStartupForPolicyViolation(
			bootstrapCtx,
			bootstrapSpan,
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
			log,
			startupDependencyIngressPolicy,
			ingressErr,
		)
	}

	if metricsExposureErr := netPolicy.ValidateOperationalMetricsExposure(); metricsExposureErr != nil {
		return networkPolicy{}, rejectStartupForPolicyViolation(
			bootstrapCtx,
			bootstrapSpan,
			log,
			startupDependencyMetricsExposure,
			metricsExposureErr,
		)
	}

	if egressErr := netPolicy.ValidateEgressExceptionState(); egressErr != nil {
		return networkPolicy{}, rejectStartupForPolicyViolation(
			bootstrapCtx,
			bootstrapSpan,
			log,
			startupDependencyEgressException,
			egressErr,
		)
	}

	return netPolicy, nil
}
