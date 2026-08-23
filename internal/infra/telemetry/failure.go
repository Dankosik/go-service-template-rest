package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel"
)

// Failure reasons a composition root records when telemetry setup degrades. They
// are a closed set because they become a metric label and a log field an operator
// alerts on.
const (
	FailureReasonSetupError       = "setup_error"
	FailureReasonDeadlineExceeded = "deadline_exceeded"
	FailureReasonCanceled         = "canceled"
)

// FailureReason classifies why setting up a signal failed, as the bounded label
// a startup record carries.
//
// The two context reasons are separated from everything else because they are the
// only ones that say the process, not the collector, ran out of budget: a spent
// startup deadline and a shutdown that arrived first need different operator
// action from an exporter that could not be built.
func FailureReason(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return FailureReasonDeadlineExceeded
	case errors.Is(err, context.Canceled):
		return FailureReasonCanceled
	default:
		return FailureReasonSetupError
	}
}

// InstallErrorHandler routes asynchronous OpenTelemetry SDK failures through the
// process logger without publishing exporter error text, which can contain an
// endpoint, certificate path, or other deployment detail.
func InstallErrorHandler(ctx context.Context, log *slog.Logger) {
	if ctx == nil {
		ctx = context.Background()
	}
	if log == nil {
		log = slog.Default()
	}
	otel.SetErrorHandler(newSDKErrorHandler(context.WithoutCancel(ctx), log))
}

//nolint:ireturn // OpenTelemetry defines the process error handler as an interface.
func newSDKErrorHandler(ctx context.Context, log *slog.Logger) otel.ErrorHandler {
	return otel.ErrorHandlerFunc(func(err error) {
		if err == nil {
			return
		}
		log.ErrorContext(
			ctx,
			"otel_sdk_error",
			"component", "telemetry",
			"operation", "sdk_export",
			"outcome", "error",
			"reason", "sdk_error",
			"error.type", fmt.Sprintf("%T", err),
		)
	})
}
