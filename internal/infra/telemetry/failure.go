package telemetry

import (
	"context"
	"errors"
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
