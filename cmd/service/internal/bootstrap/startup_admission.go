package bootstrap

import (
	"context"
	"errors"
	"sync/atomic"
)

var errStartupAdmissionPending = errors.New("startup admission is not ready")

type startupAdmissionController struct {
	ready atomic.Bool
}

func newStartupAdmissionController() *startupAdmissionController {
	return &startupAdmissionController{}
}

// The methods below carry no nil-receiver guard. The controller is constructed
// unconditionally in Run and reaches every caller through a field that is always
// set, so a nil check here is an unreachable branch on the path a readiness probe
// takes for the life of the pod — and one that would turn a wiring mistake into a
// permanent 503 instead of a panic on the first request.

func (c *startupAdmissionController) MarkReady() {
	c.ready.Store(true)
}

func (c *startupAdmissionController) Ready() bool {
	return c.ready.Load()
}

func (c *startupAdmissionController) CheckReady(context.Context) error {
	if !c.Ready() {
		return errStartupAdmissionPending
	}
	return nil
}
