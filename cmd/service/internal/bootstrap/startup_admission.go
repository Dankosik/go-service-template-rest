package bootstrap

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

type startupAdmissionController struct {
	ready           atomic.Bool
	readyOnce       sync.Once
	startupSpan     *startupSpanController
	metrics         *telemetry.Metrics
	deployTelemetry *deployTelemetryRecorder
}

func newStartupAdmissionController(
	startupSpan *startupSpanController,
	metrics *telemetry.Metrics,
	deployTelemetry *deployTelemetryRecorder,
) *startupAdmissionController {
	return &startupAdmissionController{
		startupSpan:     startupSpan,
		metrics:         metrics,
		deployTelemetry: deployTelemetry,
	}
}

func (c *startupAdmissionController) MarkReady(ctx context.Context) {
	if c == nil {
		return
	}

	c.readyOnce.Do(func() {
		c.ready.Store(true)
		if c.metrics != nil {
			c.metrics.IncConfigStartupOutcome("ready")
		}
		if c.startupSpan != nil {
			c.startupSpan.MarkReady()
		}
		if c.deployTelemetry != nil {
			c.deployTelemetry.RecordAdmission(ctx, "success", "ready", "readiness")
		}
	})
}

func (c *startupAdmissionController) Ready() bool {
	if c == nil {
		return false
	}
	return c.ready.Load()
}

type runtimeIngressAdmissionGuard struct {
	policy          networkPolicy
	deployTelemetry *deployTelemetryRecorder
	violationOnce   sync.Once
}

func newRuntimeIngressAdmissionGuard(policy networkPolicy, deployTelemetry *deployTelemetryRecorder) *runtimeIngressAdmissionGuard {
	return &runtimeIngressAdmissionGuard{
		policy:          policy,
		deployTelemetry: deployTelemetry,
	}
}

func (g *runtimeIngressAdmissionGuard) Check(ctx context.Context) error {
	if g == nil {
		return nil
	}

	if err := g.policy.ValidateIngressRuntime(); err != nil {
		g.violationOnce.Do(func() {
			if !g.policy.ingressException.Active {
				g.deployTelemetry.RecordNetworkExceptionStateChange(ctx, "ingress", "denied", "deny", g.policy.ingressException.ID)
				g.deployTelemetry.RecordNetworkIngressPolicyViolation(ctx, "missing_exception", "deny")
				return
			}

			g.deployTelemetry.RecordNetworkExceptionStateChange(ctx, "ingress", "expired", "deny", g.policy.ingressException.ID)
			g.deployTelemetry.RecordNetworkIngressPolicyViolation(ctx, "expired_exception", "deny")
		})
		return err
	}

	return nil
}
