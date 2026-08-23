package telemetry_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel"
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

func TestSDKErrorHandlerPublishesOnlyBoundedMetadata(t *testing.T) {
	const secret = "collector-token-secret"
	var output bytes.Buffer
	previous := otel.GetErrorHandler()
	t.Cleanup(func() { otel.SetErrorHandler(previous) })
	telemetry.InstallErrorHandler(t.Context(), slog.New(slog.NewJSONHandler(&output, nil)))
	otel.Handle(errors.New(secret))

	logged := output.String()
	for _, want := range []string{"otel_sdk_error", "sdk_export", "sdk_error", "*errors.errorString"} {
		if !strings.Contains(logged, want) {
			t.Fatalf("SDK error log %q does not contain %q", logged, want)
		}
	}
	if strings.Contains(logged, secret) {
		t.Fatalf("SDK error log leaked raw error text: %q", logged)
	}
}
