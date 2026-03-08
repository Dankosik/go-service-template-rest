package bootstrap

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/Dankosik/privacy-sanitization-service/internal/infra/telemetry"
)

type startupAdmissionController struct {
	ready       atomic.Bool
	readyOnce   sync.Once
	startupSpan *startupSpanController
	metrics     *telemetry.Metrics
}

func newStartupAdmissionController(
	startupSpan *startupSpanController,
	metrics *telemetry.Metrics,
) *startupAdmissionController {
	return &startupAdmissionController{
		startupSpan: startupSpan,
		metrics:     metrics,
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
	})
}

func (c *startupAdmissionController) Ready() bool {
	if c == nil {
		return false
	}
	return c.ready.Load()
}

type runtimeIngressAdmissionGuard struct {
	policy        networkPolicy
	violationOnce sync.Once
}

func newRuntimeIngressAdmissionGuard(policy networkPolicy) *runtimeIngressAdmissionGuard {
	return &runtimeIngressAdmissionGuard{
		policy: policy,
	}
}

func (g *runtimeIngressAdmissionGuard) Check(ctx context.Context) error {
	if g == nil {
		return nil
	}

	if err := g.policy.ValidateIngressRuntime(); err != nil {
		g.violationOnce.Do(func() {
			_ = ctx
		})
		return err
	}

	return nil
}
