package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
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
	telemetryCleanup, telemetryInitErr := bootstrapTelemetryStage(startupCtx, cfg, metrics, log)
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

func bootstrapNetworkPolicyStage(
	bootstrapCtx context.Context,
	bootstrapSpan trace.Span,
	log *slog.Logger,
	netPolicyResult networkPolicyLoadResult,
	cfg config.Config,
) (networkPolicy, error) {
	if netPolicyResult.err != nil {
		return networkPolicy{}, rejectStartupForPolicyViolation(
			bootstrapCtx,
			bootstrapSpan,
			log,
			startupDependencyNetworkPolicy,
			fmt.Errorf("invalid network policy configuration: %w", netPolicyResult.err),
			"policy.class", "ingress",
			"reason.class", "invalid_configuration",
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

	return netPolicy, nil
}
