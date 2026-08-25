package httpx

import (
	"log/slog"
	"net/http"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
)

func TestOTelServerNameBoundsAuthorityLabels(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		configured string
		want       string
	}{
		{configured: "orders-api", want: "orders-api:0"},
		{configured: " orders-api ", want: "orders-api:0"},
		{want: "service:0"},
	} {
		if got := otelServerName(tt.configured); got != tt.want {
			t.Fatalf("otelServerName(%q) = %q, want %q", tt.configured, got, tt.want)
		}
	}
}

//nolint:paralleltest // Installs a process-wide tracer provider for span capture.
func TestHardenUsesCallerTraceFilter(t *testing.T) {
	recorder := telemetrytest.InstallSpanRecorder(t)

	handler, err := Harden(
		slog.New(slog.DiscardHandler),
		telemetry.New(),
		HardenConfig{
			MaxBodyBytes:   testRouterMaxBodyBytes,
			RequestTimeout: testRouterRequestTimeout,
			OTelServerName: "custom-contract",
			TraceRequest:   func(*http.Request) bool { return true },
		},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	if err != nil {
		t.Fatalf("Harden() error = %v", err)
	}

	resp := doRequest(handler, http.MethodPost, "/webhooks/caller-owned")
	if resp.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusNoContent)
	}
	if got := len(recorder.Ended()); got != 1 {
		t.Fatalf("ended spans = %d, want 1 from the caller-owned filter", got)
	}
}
