package telemetry_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

// TestFailureReasonSeparatesSpentBudgetFromSetup holds the distinction the label
// exists for: an operator seeing setup_error looks at the collector, and one
// seeing a context reason looks at the process budget instead. Each case wraps
// the sentinel, because every caller reports an error some exporter already
// annotated.
func TestFailureReasonSeparatesSpentBudgetFromSetup(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name string
		err  error
		want string
	}{
		{name: "deadline", err: context.DeadlineExceeded, want: telemetry.FailureReasonDeadlineExceeded},
		{name: "canceled", err: context.Canceled, want: telemetry.FailureReasonCanceled},
		{name: "other", err: errors.New("build exporter"), want: telemetry.FailureReasonSetupError},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := telemetry.FailureReason(fmt.Errorf("setup traces: %w", testCase.err)); got != testCase.want {
				t.Errorf("FailureReason(%v) = %q, want %q", testCase.err, got, testCase.want)
			}
		})
	}
}
