package bootstrap

import (
	"context"
	"errors"
	"sync/atomic"
)

var errStartupAdmissionPending = errors.New("startup admission is not ready")

type startupAdmissionController struct {
	ready       atomic.Bool
	startupSpan *startupSpanController
}

func newStartupAdmissionController(startupSpan *startupSpanController) *startupAdmissionController {
	return &startupAdmissionController{
		startupSpan: startupSpan,
	}
}

func (c *startupAdmissionController) MarkReady() {
	if c == nil {
		return
	}

	if c.ready.CompareAndSwap(false, true) && c.startupSpan != nil {
		c.startupSpan.MarkReady()
	}
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
