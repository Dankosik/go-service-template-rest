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
		{name: "config load", input: StartupRejectionReasonConfigLoad, want: StartupRejectionReasonConfigLoad},
		{name: "config parse", input: "CONFIG_PARSE", want: StartupRejectionReasonConfigParse},
		{name: "config validate", input: StartupRejectionReasonConfigValidate, want: StartupRejectionReasonConfigValidate},
		{name: "config startup compatibility", input: StartupRejectionReasonConfigStartupCompatibility, want: StartupRejectionReasonConfigStartupCompatibility},
		{name: "strict unknown key", input: StartupRejectionReasonConfigStrictUnknownKey, want: StartupRejectionReasonConfigStrictUnknownKey},
		{name: "secret policy", input: StartupRejectionReasonConfigSecretPolicy, want: StartupRejectionReasonConfigSecretPolicy},
		{name: "config load taxonomy alias rejected", input: "load", want: StartupRejectionReasonOther},
		{name: "config validate taxonomy alias rejected", input: "validate", want: StartupRejectionReasonOther},
		{name: "startup compatibility taxonomy alias rejected", input: "startup_compatibility", want: StartupRejectionReasonOther},
		{name: "strict unknown key taxonomy alias rejected", input: "strict_unknown_key", want: StartupRejectionReasonOther},
		{name: "secret policy taxonomy alias rejected", input: "secret_policy", want: StartupRejectionReasonOther},
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

func TestNormalizeConfigLoadResult(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "success", input: ConfigLoadResultSuccess, want: ConfigLoadResultSuccess},
		{name: "error upper", input: "ERROR", want: ConfigLoadResultError},
		{name: "success with whitespace", input: " success ", want: ConfigLoadResultSuccess},
		{name: "unknown", input: "partial", want: ConfigLoadResultOther},
		{name: "empty", input: "", want: ConfigLoadResultOther},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeConfigLoadResult(tc.input)
			if got != tc.want {
				t.Fatalf("normalizeConfigLoadResult(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeConfigLoadStage(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "load defaults", input: ConfigLoadStageLoadDefaults, want: ConfigLoadStageLoadDefaults},
		{name: "load file", input: ConfigLoadStageLoadFile, want: ConfigLoadStageLoadFile},
		{name: "load env", input: ConfigLoadStageLoadEnv, want: ConfigLoadStageLoadEnv},
		{name: "parse", input: ConfigLoadStageParse, want: ConfigLoadStageParse},
		{name: "validate", input: ConfigLoadStageValidate, want: ConfigLoadStageValidate},
		{name: "startup compatibility", input: ConfigLoadStageStartupCompatibility, want: ConfigLoadStageStartupCompatibility},
		{name: "stage upper", input: "CONFIG.PARSE", want: ConfigLoadStageParse},
		{name: "stage with whitespace", input: " " + ConfigLoadStageValidate + " ", want: ConfigLoadStageValidate},
		{name: "unknown", input: "config.remote.fetch", want: ConfigLoadStageOther},
		{name: "empty", input: "", want: ConfigLoadStageOther},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeConfigLoadStage(tc.input)
			if got != tc.want {
				t.Fatalf("normalizeConfigLoadStage(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeConfigFailureReason(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "load", input: ConfigFailureReasonLoad, want: ConfigFailureReasonLoad},
		{name: "parse upper", input: "PARSE", want: ConfigFailureReasonParse},
		{name: "validate", input: ConfigFailureReasonValidate, want: ConfigFailureReasonValidate},
		{name: "strict unknown key", input: ConfigFailureReasonStrictUnknownKey, want: ConfigFailureReasonStrictUnknownKey},
		{name: "secret policy", input: ConfigFailureReasonSecretPolicy, want: ConfigFailureReasonSecretPolicy},
		{name: "unknown", input: "startup_compatibility", want: ConfigFailureReasonOther},
		{name: "empty", input: "", want: ConfigFailureReasonOther},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeConfigFailureReason(tc.input)
			if got != tc.want {
				t.Fatalf("normalizeConfigFailureReason(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeConfigStartupOutcome(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "ready", input: ConfigStartupOutcomeReady, want: ConfigStartupOutcomeReady},
		{name: "rejected upper", input: "REJECTED", want: ConfigStartupOutcomeRejected},
		{name: "ready with whitespace", input: " ready ", want: ConfigStartupOutcomeReady},
		{name: "unknown", input: "degraded", want: ConfigStartupOutcomeOther},
		{name: "empty", input: "", want: ConfigStartupOutcomeOther},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeConfigStartupOutcome(tc.input)
			if got != tc.want {
				t.Fatalf("normalizeConfigStartupOutcome(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeStartupDependency(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "postgres", input: StartupDependencyPostgres, want: StartupDependencyPostgres},
		{name: "redis", input: StartupDependencyRedis, want: StartupDependencyRedis},
		{name: "mongo", input: StartupDependencyMongo, want: StartupDependencyMongo},
		{name: "telemetry", input: StartupDependencyTelemetry, want: StartupDependencyTelemetry},
		{name: "network policy", input: StartupDependencyNetworkPolicy, want: StartupDependencyNetworkPolicy},
		{name: "ingress policy", input: StartupDependencyIngressPolicy, want: StartupDependencyIngressPolicy},
		{name: "metrics exposure", input: StartupDependencyMetricsExposure, want: StartupDependencyMetricsExposure},
		{name: "egress exception", input: StartupDependencyEgressException, want: StartupDependencyEgressException},
		{name: "other", input: StartupDependencyOther, want: StartupDependencyOther},
		{name: "dependency upper", input: "TELEMETRY", want: StartupDependencyTelemetry},
		{name: "dependency with whitespace", input: " " + StartupDependencyPostgres + " ", want: StartupDependencyPostgres},
		{name: "unknown", input: "search", want: StartupDependencyOther},
		{name: "empty", input: "", want: StartupDependencyOther},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeStartupDependency(tc.input)
			if got != tc.want {
				t.Fatalf("normalizeStartupDependency(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeStartupDependencyMode(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "disabled", input: StartupDependencyModeDisabled, want: StartupDependencyModeDisabled},
		{name: "critical fail closed", input: StartupDependencyModeCriticalFailClosed, want: StartupDependencyModeCriticalFailClosed},
		{name: "critical fail degraded", input: StartupDependencyModeCriticalFailDegraded, want: StartupDependencyModeCriticalFailDegraded},
		{name: "optional fail open", input: StartupDependencyModeOptionalFailOpen, want: StartupDependencyModeOptionalFailOpen},
		{name: "feature off", input: StartupDependencyModeFeatureOff, want: StartupDependencyModeFeatureOff},
		{name: "degraded read only or stale", input: StartupDependencyModeDegradedReadOnlyOrStale, want: StartupDependencyModeDegradedReadOnlyOrStale},
		{name: "other", input: StartupDependencyModeOther, want: StartupDependencyModeOther},
		{name: "mode upper", input: "FEATURE_OFF", want: StartupDependencyModeFeatureOff},
		{name: "mode with whitespace", input: " " + StartupDependencyModeDisabled + " ", want: StartupDependencyModeDisabled},
		{name: "unknown", input: "read_only", want: StartupDependencyModeOther},
		{name: "empty", input: "", want: StartupDependencyModeOther},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeStartupDependencyMode(tc.input)
			if got != tc.want {
				t.Fatalf("normalizeStartupDependencyMode(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCoreMetricsHandlerExposesExpectedSeries(t *testing.T) {
	m := New()

	m.ObserveHTTPRequest(http.MethodGet, "/ping", http.StatusOK)
	m.IncConfigFailure(ConfigFailureReasonValidate)
	m.IncStartupRejection(StartupRejectionReasonDependencyInit)
	m.IncStartupRejection("dns_failure")
	m.IncTelemetryInitFailure(TelemetryFailureReasonSetupError)
	m.IncConfigStartupOutcome(ConfigStartupOutcomeReady)
	m.MarkStartupDependencyReady("telemetry", "optional_fail_open")

	metricsText := collectMetricsText(t, m)

	expected := []string{
		`http_requests_total`,
		`config_failures_total`,
		`startup_rejections_total`,
		`telemetry_init_failure_total`,
		`config_startup_outcome_total`,
		`startup_dependency_status`,
		`route="/ping"`,
		`config_failures_total{reason="validate"} 1`,
		`startup_rejections_total{reason="` + StartupRejectionReasonDependencyInit + `"} 1`,
		`startup_rejections_total{reason="` + StartupRejectionReasonOther + `"} 1`,
		`outcome="` + ConfigStartupOutcomeReady + `"`,
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
		`config_validation_failures_total`,
		`network_policy_violation_total`,
	}
	for _, pattern := range removed {
		if strings.Contains(metricsText, pattern) {
			t.Fatalf("metrics output unexpectedly contains removed series %q\n%s", pattern, metricsText)
		}
	}
}

func TestConfigFailureMetricUsesBoundedReasons(t *testing.T) {
	m := New()

	m.IncConfigFailure(ConfigFailureReasonLoad)
	m.IncConfigFailure(ConfigFailureReasonParse)
	m.IncConfigFailure(ConfigFailureReasonValidate)
	m.IncConfigFailure(ConfigFailureReasonStrictUnknownKey)
	m.IncConfigFailure(ConfigFailureReasonSecretPolicy)
	m.IncConfigFailure("new_reason")

	metricsText := collectMetricsText(t, m)
	expected := []string{
		`config_failures_total{reason="` + ConfigFailureReasonLoad + `"} 1`,
		`config_failures_total{reason="` + ConfigFailureReasonParse + `"} 1`,
		`config_failures_total{reason="` + ConfigFailureReasonValidate + `"} 1`,
		`config_failures_total{reason="` + ConfigFailureReasonStrictUnknownKey + `"} 1`,
		`config_failures_total{reason="` + ConfigFailureReasonSecretPolicy + `"} 1`,
		`config_failures_total{reason="` + ConfigFailureReasonOther + `"} 1`,
	}
	for _, pattern := range expected {
		if !strings.Contains(metricsText, pattern) {
			t.Fatalf("metrics output does not contain %q\n%s", pattern, metricsText)
		}
	}
	if strings.Contains(metricsText, `reason="new_reason"`) {
		t.Fatalf("metrics output contains unbounded config failure reason:\n%s", metricsText)
	}
}

func TestConfigMetricTaxonomiesCollapseUnknownValues(t *testing.T) {
	m := New()

	m.ObserveConfigLoadDuration(ConfigLoadStageLoadFile, "unexpected-result", time.Millisecond)
	m.ObserveConfigLoadDuration("config.remote.fetch", ConfigLoadResultSuccess, time.Millisecond)
	m.IncConfigFailure("startup_compatibility")
	m.IncConfigStartupOutcome("degraded")

	metricsText := collectMetricsText(t, m)
	expected := []string{
		`config_load_duration_seconds_count{result="` + ConfigLoadResultOther + `",stage="` + ConfigLoadStageLoadFile + `"} 1`,
		`config_load_duration_seconds_count{result="` + ConfigLoadResultSuccess + `",stage="` + ConfigLoadStageOther + `"} 1`,
		`config_failures_total{reason="` + ConfigFailureReasonOther + `"} 1`,
		`config_startup_outcome_total{outcome="` + ConfigStartupOutcomeOther + `"} 1`,
	}
	for _, pattern := range expected {
		if !strings.Contains(metricsText, pattern) {
			t.Fatalf("metrics output does not contain %q\n%s", pattern, metricsText)
		}
	}

	for _, rawLabel := range []string{`result="unexpected-result"`, `stage="config.remote.fetch"`, `reason="startup_compatibility"`, `outcome="degraded"`} {
		if strings.Contains(metricsText, rawLabel) {
			t.Fatalf("metrics output contains unbounded label %q:\n%s", rawLabel, metricsText)
		}
	}
}

func TestStartupDependencyStatusMetricUsesBoundedLabels(t *testing.T) {
	m := New()

	m.MarkStartupDependencyReady(StartupDependencyTelemetry, StartupDependencyModeOptionalFailOpen)
	m.MarkStartupDependencyBlocked("search", "read_only")

	metricsText := collectMetricsText(t, m)
	expected := []string{
		`startup_dependency_status{dep="` + StartupDependencyTelemetry + `",mode="` + StartupDependencyModeOptionalFailOpen + `"} 1`,
		`startup_dependency_status{dep="` + StartupDependencyOther + `",mode="` + StartupDependencyModeOther + `"} 0`,
	}
	for _, pattern := range expected {
		if !strings.Contains(metricsText, pattern) {
			t.Fatalf("metrics output does not contain %q\n%s", pattern, metricsText)
		}
	}

	for _, rawLabel := range []string{`dep="search"`, `mode="read_only"`} {
		if strings.Contains(metricsText, rawLabel) {
			t.Fatalf("metrics output contains unbounded startup dependency label %q:\n%s", rawLabel, metricsText)
		}
	}
}

func TestMetricsNilAndZeroValueMethodsAreNoops(t *testing.T) {
	for _, m := range []*Metrics{nil, &Metrics{}} {
		m.ObserveHTTPRequest(http.MethodGet, "/ping", http.StatusOK)
		m.ObserveHTTPRequestDuration(http.MethodGet, "/ping", http.StatusOK, time.Millisecond)
		m.ObserveConfigLoadDuration("load", "ok", time.Millisecond)
		m.IncConfigFailure("dependency_init")
		m.IncStartupRejection(StartupRejectionReasonDependencyInit)
		m.AddConfigUnknownKeyWarnings(1)
		m.IncTelemetryInitFailure(TelemetryFailureReasonSetupError)
		m.IncConfigStartupOutcome(ConfigStartupOutcomeReady)
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
