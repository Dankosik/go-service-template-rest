package telemetry

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsHandlerExposesRuntimeCollectors(t *testing.T) {
	t.Parallel()

	metricsText := collectMetricsText(t, New())
	for _, name := range []string{"go_gc_duration_seconds", "process_cpu_seconds_total"} {
		if !strings.Contains(metricsText, name) {
			t.Fatalf("metrics output does not contain %q", name)
		}
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
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		resp := httptest.NewRecorder()
		metrics.Handler().ServeHTTP(resp, req)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("zero-value metrics handler status = %d, want %d", resp.Code, http.StatusNotFound)
		}
	}
}

func collectMetricsText(t *testing.T, metrics *Metrics) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	resp := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("metrics handler status = %d, want %d", resp.Code, http.StatusOK)
	}

	return resp.Body.String()
}
