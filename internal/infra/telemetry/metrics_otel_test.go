package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"go.opentelemetry.io/otel/trace"
)

//nolint:paralleltest // Mutates the process-wide OpenTelemetry MeterProvider.
func TestSetupMetricsUsesPrivateRegistryAndConfigResource(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.name=env-service,env.only=true")
	t.Setenv("OTEL_SERVICE_NAME", "env-service")

	telemetrytest.RestoreGlobals(t)

	metrics := New()
	shutdown, err := SetupMetrics(context.Background(), metrics, MetricsConfig{
		ServiceName:    " test-service ",
		ServiceVersion: " test-version ",
		DeploymentEnv:  " test-env ",
	})
	if err != nil {
		t.Fatalf("SetupMetrics() error = %v", err)
	}
	t.Cleanup(func() {
		if err := shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown metrics: %v", err)
		}
	})

	meter := metrics.MeterProvider().Meter("metrics-test")
	counter, err := meter.Int64Counter("template.operations")
	if err != nil {
		t.Fatalf("create counter: %v", err)
	}
	counter.Add(context.Background(), 1)
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:     trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		TraceFlags: trace.FlagsSampled,
	})
	counter.Add(trace.ContextWithSpanContext(context.Background(), spanContext), 1)

	metricsText := collectMetricsText(t, metrics)
	for _, pattern := range []string{
		"template_operations_total",
		`service_name="test-service"`,
		`service_version="test-version"`,
		`deployment_environment_name="test-env"`,
	} {
		if !strings.Contains(metricsText, pattern) {
			t.Fatalf("metrics output does not contain %q\n%s", pattern, metricsText)
		}
	}
	for _, forbidden := range []string{"env-service", "env_only"} {
		if strings.Contains(metricsText, forbidden) {
			t.Fatalf("metrics output contains ambient resource value %q\n%s", forbidden, metricsText)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Accept", "application/openmetrics-text; version=1.0.0")
	resp := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(resp, req)
	if got := resp.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/openmetrics-text") {
		t.Fatalf("metrics content type = %q, want OpenMetrics", got)
	}
	for _, label := range []string{
		`trace_id="0102030405060708090a0b0c0d0e0f10"`,
		`span_id="0102030405060708"`,
	} {
		if !strings.Contains(resp.Body.String(), label) {
			t.Fatalf("OpenMetrics output does not contain exemplar %q\n%s", label, resp.Body.String())
		}
	}
}

func TestSetupMetricsRequiresRegistry(t *testing.T) {
	t.Parallel()

	for _, metrics := range []*Metrics{nil, {}} {
		if _, err := SetupMetrics(context.Background(), metrics, MetricsConfig{}); err == nil {
			t.Fatal("SetupMetrics() error = nil, want registry error")
		}
	}
}
