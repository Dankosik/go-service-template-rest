package bootstrap

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type startupSpanController struct {
	span    trace.Span
	cleanup func(context.Context)
	once    sync.Once
}

func newStartupSpanController(span trace.Span, cleanup func(context.Context)) *startupSpanController {
	if cleanup == nil {
		cleanup = func(context.Context) {}
	}
	return &startupSpanController{
		span:    span,
		cleanup: cleanup,
	}
}

func (c *startupSpanController) MarkReady() {
	if c == nil {
		return
	}
	c.end(attribute.String("result", "success"))
}

func (c *startupSpanController) Close(ctx context.Context) {
	if c == nil {
		return
	}
	c.end()
	c.cleanup(ctx)
}

func (c *startupSpanController) end(attrs ...attribute.KeyValue) {
	c.once.Do(func() {
		if c.span == nil {
			return
		}
		if len(attrs) > 0 {
			c.span.SetAttributes(attrs...)
		}
		c.span.End()
	})
}
