package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestMetricsHandlerExposesProcessCollectorOnly pins the split between the two
// runtime signals. Process-level series come from the operating system and are
// registered here, so they exist on the scrape path only. Go runtime series come
// from the OpenTelemetry instruments SetupMetrics registers on the meter provider,
// so they reach the OTLP reader too — which is why the Prometheus Go collector
// that used to be registered here is gone, and why go_gc_duration_seconds must
// not come back: two naming schemes for the same facts is what this replaced.
func TestMetricsHandlerExposesProcessCollectorOnly(t *testing.T) {
	t.Parallel()

	metricsText := collectMetricsText(t, New())
	if !strings.Contains(metricsText, "process_cpu_seconds_total") {
		t.Fatal("metrics output does not contain process_cpu_seconds_total")
	}
	if strings.Contains(metricsText, "go_gc_duration_seconds") {
		t.Fatal("metrics output contains the Prometheus Go collector, which the OTel runtime instruments replaced")
	}
	for _, removed := range []string{
		"config_load_duration_seconds",
		"config_failures_total",
		"startup_rejections_total",
		"telemetry_init_failure_total",
		"config_unknown_key_warnings_total",
		"config_startup_outcome_total",
		"startup_dependency_status",
	} {
		if strings.Contains(metricsText, removed) {
			t.Fatalf("metrics output unexpectedly contains removed startup-only series %q", removed)
		}
	}
}

func TestMetricsNilAndZeroValueHandlersReturnNotFound(t *testing.T) {
	t.Parallel()

	for _, metrics := range []*Metrics{nil, {}} {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", nil)
		resp := httptest.NewRecorder()
		metrics.Handler().ServeHTTP(resp, req)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("zero-value metrics handler status = %d, want %d", resp.Code, http.StatusNotFound)
		}
	}
}

func collectMetricsText(t *testing.T, metrics *Metrics) string {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", nil)
	resp := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("metrics handler status = %d, want %d", resp.Code, http.StatusOK)
	}

	return resp.Body.String()
}
