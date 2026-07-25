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

func (c *startupAdmissionController) MarkReady() {
	if c == nil {
		return
	}

	c.ready.Store(true)
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
