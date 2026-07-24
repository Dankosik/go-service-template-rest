package bootstrap

import (
	"context"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"go.opentelemetry.io/otel"
)

//nolint:paralleltest // Installs a process-wide tracer provider for span capture.
func TestStartupLogArgsIncludesTraceIDs(t *testing.T) {
	spanRecorder := telemetrytest.InstallSpanRecorder(t)
	ctx, span := otel.Tracer("test").Start(context.Background(), "startup-log-test")
	args := startupLogArgs(ctx, "c", "o", "ok", "k", "v")
	span.End()
	if len(spanRecorder.Ended()) == 0 {
		t.Fatal("expected ended span")
	}

	foundTrace := false
	foundSpan := false
	for i := 0; i < len(args)-1; i += 2 {
		k, ok := args[i].(string)
		if !ok {
			continue
		}
		if k == "trace_id" {
			v, _ := args[i+1].(string)
			foundTrace = strings.TrimSpace(v) != ""
		}
		if k == "span_id" {
			v, _ := args[i+1].(string)
			foundSpan = strings.TrimSpace(v) != ""
		}
	}
	if !foundTrace || !foundSpan {
		t.Fatalf("trace/span ids not found in args: %#v", args)
	}
}
