package telemetry

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNormalizeFieldGroupLabel(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "http field", input: "http.addr", want: "http"},
		{name: "observability field", input: "observability.otel.service_name", want: "observability"},
		{name: "redis group", input: "redis", want: "redis"},
		{name: "unknown group", input: "custom.group.field", want: "other"},
		{name: "empty", input: "", want: "other"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeFieldGroupLabel(tc.input)
			if got != tc.want {
				t.Fatalf("normalizeFieldGroupLabel(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

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

func TestDeployRollbackAndDriftMetrics(t *testing.T) {
	m := New()

	m.ObserveDeployHealthAdmission("Production", "success", "ready", 2*time.Second)
	m.IncDeployHealthProbeFailure("Production", "readiness")

	m.ObserveRollbackExecution("Production", "runtime_error", "failure", 3*time.Second)
	m.IncRollbackPostcheck("Production", "/health/ready", "failure")

	m.IncConfigDriftDetected("Production", "ci")
	m.SetConfigDriftOpen("Production", true)
	m.ObserveConfigDriftReconcile("Production", "success", 10*time.Second)
	m.SetConfigDriftOpen("Production", false)

	m.IncNetworkPolicyViolation("Production", "ingress", "missing_exception")
	m.SetNetworkExceptionActive("Production", "ingress", true)

	metricsText := collectMetricsText(t, m)

	expected := []string{
		`deploy_health_admission_total`,
		`deploy_health_admission_duration_seconds`,
		`deploy_health_probe_failures_total`,
		`rollback_execution_total`,
		`rollback_recovery_duration_seconds`,
		`rollback_postcheck_total`,
		`config_drift_detected_total`,
		`config_drift_open`,
		`config_drift_reconcile_duration_seconds`,
		`network_policy_violation_total`,
		`network_exception_active`,
		`environment="production"`,
		`reason_class="ready"`,
		`trigger="runtime_error"`,
		`source="ci"`,
		`policy_class="ingress"`,
	}
	for _, pattern := range expected {
		if !strings.Contains(metricsText, pattern) {
			t.Fatalf("metrics output does not contain %q\n%s", pattern, metricsText)
		}
	}
}

func TestTelemetryLabelNormalizers(t *testing.T) {
	if got := normalizeEnvironmentLabel(""); got != "unknown" {
		t.Fatalf("normalizeEnvironmentLabel(\"\") = %q, want %q", got, "unknown")
	}
	if got := normalizeResultLabel("unhandled"); got != "other" {
		t.Fatalf("normalizeResultLabel(\"unhandled\") = %q, want %q", got, "other")
	}
	if got := normalizeReasonClassLabel("bad"); got != "other" {
		t.Fatalf("normalizeReasonClassLabel(\"bad\") = %q, want %q", got, "other")
	}
	if got := normalizeProbeTypeLabel("redis"); got != "redis" {
		t.Fatalf("normalizeProbeTypeLabel(\"redis\") = %q, want %q", got, "redis")
	}
	if got := normalizeTriggerLabel("manual"); got != "manual" {
		t.Fatalf("normalizeTriggerLabel(\"manual\") = %q, want %q", got, "manual")
	}
	if got := normalizeEndpointLabel("/x"); got != "other" {
		t.Fatalf("normalizeEndpointLabel(\"/x\") = %q, want %q", got, "other")
	}
	if got := normalizeDriftSourceLabel("runtime"); got != "runtime" {
		t.Fatalf("normalizeDriftSourceLabel(\"runtime\") = %q, want %q", got, "runtime")
	}
	if got := normalizePolicyClassLabel("egress"); got != "egress" {
		t.Fatalf("normalizePolicyClassLabel(\"egress\") = %q, want %q", got, "egress")
	}
	if got := normalizeNetworkReasonClassLabel("scheme_denied"); got != "scheme_denied" {
		t.Fatalf("normalizeNetworkReasonClassLabel(\"scheme_denied\") = %q, want %q", got, "scheme_denied")
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
