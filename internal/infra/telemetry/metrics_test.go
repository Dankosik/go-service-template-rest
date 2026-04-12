package telemetry

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNormalizeTelemetryFailureReason(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "setup error", input: TelemetryFailureReasonSetupError, want: TelemetryFailureReasonSetupError},
		{name: "deadline exceeded", input: TelemetryFailureReasonDeadlineExceeded, want: TelemetryFailureReasonDeadlineExceeded},
		{name: "canceled upper", input: "CANCELED", want: TelemetryFailureReasonCanceled},
		{name: "unknown", input: "dns_failure", want: TelemetryFailureReasonOther},
		{name: "empty", input: "", want: TelemetryFailureReasonOther},
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

func TestNormalizeStartupRejectionReason(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "config load", input: "load", want: StartupRejectionReasonConfigLoad},
		{name: "config parse", input: "CONFIG_PARSE", want: StartupRejectionReasonConfigParse},
		{name: "config validate", input: "validate", want: StartupRejectionReasonConfigValidate},
		{name: "strict unknown key", input: "strict_unknown_key", want: StartupRejectionReasonConfigStrictUnknownKey},
		{name: "secret policy", input: "secret_policy", want: StartupRejectionReasonConfigSecretPolicy},
		{name: "policy violation", input: StartupRejectionReasonPolicyViolation, want: StartupRejectionReasonPolicyViolation},
		{name: "dependency init", input: StartupRejectionReasonDependencyInit, want: StartupRejectionReasonDependencyInit},
		{name: "startup error", input: StartupRejectionReasonStartupError, want: StartupRejectionReasonStartupError},
		{name: "unknown", input: "dns_failure", want: StartupRejectionReasonOther},
		{name: "empty", input: "", want: StartupRejectionReasonOther},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeStartupRejectionReason(tc.input)
			if got != tc.want {
				t.Fatalf("normalizeStartupRejectionReason(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCoreMetricsHandlerExposesExpectedSeries(t *testing.T) {
	m := New()

	m.ObserveHTTPRequest(http.MethodGet, "/ping", http.StatusOK)
	m.IncConfigValidationFailure("validate")
	m.IncStartupRejection(StartupRejectionReasonDependencyInit)
	m.IncStartupRejection("dns_failure")
	m.IncTelemetryInitFailure(TelemetryFailureReasonSetupError)
	m.IncConfigStartupOutcome("ready")
	m.MarkStartupDependencyReady("telemetry", "optional_fail_open")

	metricsText := collectMetricsText(t, m)

	expected := []string{
		`http_requests_total`,
		`config_validation_failures_total`,
		`startup_rejections_total`,
		`telemetry_init_failure_total`,
		`config_startup_outcome_total`,
		`startup_dependency_status`,
		`route="/ping"`,
		`config_validation_failures_total{reason="validate"} 1`,
		`startup_rejections_total{reason="` + StartupRejectionReasonDependencyInit + `"} 1`,
		`startup_rejections_total{reason="` + StartupRejectionReasonOther + `"} 1`,
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

func TestMetricsNilAndZeroValueMethodsAreNoops(t *testing.T) {
	for _, m := range []*Metrics{nil, &Metrics{}} {
		m.ObserveHTTPRequest(http.MethodGet, "/ping", http.StatusOK)
		m.ObserveHTTPRequestDuration(http.MethodGet, "/ping", http.StatusOK, time.Millisecond)
		m.ObserveConfigLoadDuration("load", "ok", time.Millisecond)
		m.IncConfigValidationFailure("dependency_init")
		m.IncStartupRejection(StartupRejectionReasonDependencyInit)
		m.AddConfigUnknownKeyWarnings(1)
		m.IncTelemetryInitFailure(TelemetryFailureReasonSetupError)
		m.IncConfigStartupOutcome("ready")
		m.MarkStartupDependencyReady("telemetry", "optional_fail_open")

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		resp := httptest.NewRecorder()
		m.Handler().ServeHTTP(resp, req)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("zero-value metrics handler status = %d, want %d", resp.Code, http.StatusNotFound)
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
