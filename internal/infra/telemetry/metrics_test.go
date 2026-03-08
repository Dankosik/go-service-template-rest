package telemetry

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeTelemetryFailureReason(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "setup error", input: "setup_error", want: "setup_error"},
		{name: "deadline exceeded", input: "deadline_exceeded", want: "deadline_exceeded"},
		{name: "canceled upper", input: "CANCELED", want: "canceled"},
		{name: "unknown", input: "dns_failure", want: "other"},
		{name: "empty", input: "", want: "other"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeTelemetryFailureReason(tc.input)
			if got != tc.want {
				t.Fatalf("normalizeTelemetryFailureReason(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCoreMetricsHandlerExposesExpectedSeries(t *testing.T) {
	m := New()

	m.ObserveHTTPRequest(http.MethodGet, "/ping", http.StatusOK)
	m.IncConfigValidationFailure("dependency_init")
	m.IncTelemetryInitFailure("setup_error")
	m.IncConfigStartupOutcome("ready")
	m.SetStartupDependencyStatus("telemetry", "optional_fail_open", true)

	metricsText := collectMetricsText(t, m)

	expected := []string{
		`http_requests_total`,
		`config_validation_failures_total`,
		`telemetry_init_failure_total`,
		`config_startup_outcome_total`,
		`startup_dependency_status`,
		`route="/ping"`,
		`reason="dependency_init"`,
		`outcome="ready"`,
		`dep="telemetry"`,
		`mode="optional_fail_open"`,
	}
	for _, pattern := range expected {
		if !strings.Contains(metricsText, pattern) {
			t.Fatalf("metrics output does not contain %q\n%s", pattern, metricsText)
		}
	}

	removed := []string{
		`deploy_health_admission_total`,
		`rollback_execution_total`,
		`config_drift_detected_total`,
		`network_policy_violation_total`,
	}
	for _, pattern := range removed {
		if strings.Contains(metricsText, pattern) {
			t.Fatalf("metrics output unexpectedly contains removed series %q\n%s", pattern, metricsText)
		}
	}
}

func collectMetricsText(t *testing.T, m *Metrics) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	resp := httptest.NewRecorder()
	m.Handler().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("metrics handler status = %d, want %d", resp.Code, http.StatusOK)
	}

	return resp.Body.String()
}
