package bootstrap

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

var errStartupAdmissionPending = errors.New("startup admission is not ready")

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

func (c *startupAdmissionController) MarkReady() {
	if c == nil {
		return
	}

	c.readyOnce.Do(func() {
		c.ready.Store(true)
		if c.metrics != nil {
			c.metrics.IncConfigStartupOutcome(telemetry.ConfigStartupOutcomeReady)
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

func (c *startupAdmissionController) CheckReady(context.Context) error {
	if c == nil || !c.Ready() {
		return errStartupAdmissionPending
	}
	return nil
}

type runtimeIngressAdmissionGuard struct {
	policy networkPolicy
}

func newRuntimeIngressAdmissionGuard(policy networkPolicy) *runtimeIngressAdmissionGuard {
	return &runtimeIngressAdmissionGuard{
		policy: policy,
	}
}

func (g *runtimeIngressAdmissionGuard) Check(context.Context) error {
	if g == nil {
		return nil
	}

	if err := g.policy.ValidateIngressRuntime(); err != nil {
		return err
	}

	return nil
}
