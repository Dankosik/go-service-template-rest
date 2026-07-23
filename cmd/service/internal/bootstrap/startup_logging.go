package bootstrap

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/trace"
)

const (
	telemetryFailureReasonSetupError       = "setup_error"
	telemetryFailureReasonDeadlineExceeded = "deadline_exceeded"
	telemetryFailureReasonCanceled         = "canceled"
)

func startupLogArgs(ctx context.Context, component, operation, outcome string, extra ...any) []any {
	args := make([]any, 0, 6+len(extra))
	args = append(args,
		"component", component,
		"operation", operation,
		"outcome", outcome,
	)

	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		args = append(args,
			"trace_id", spanCtx.TraceID().String(),
			"span_id", spanCtx.SpanID().String(),
		)
	}

	args = append(args, extra...)
	return args
}

func telemetryInitFailureReason(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return telemetryFailureReasonDeadlineExceeded
	case errors.Is(err, context.Canceled):
		return telemetryFailureReasonCanceled
	default:
		return telemetryFailureReasonSetupError
	}
}
