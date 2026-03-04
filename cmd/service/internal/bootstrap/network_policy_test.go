package bootstrap

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

func TestLoadNetworkPolicyFromEnvRequiresExceptionMetadata(t *testing.T) {
	t.Setenv(envNetworkPublicIngressEnabled, "true")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ACTIVE", "true")

	_, err := loadNetworkPolicyFromEnv()
	if err == nil {
		t.Fatal("loadNetworkPolicyFromEnv() error = nil, want non-nil")
	}
	policyClass, reasonClass := networkPolicyErrorLabels(err)
	if policyClass != "ingress" {
		t.Fatalf("policyClass = %q, want %q", policyClass, "ingress")
	}
	if reasonClass != "missing_metadata" {
		t.Fatalf("reasonClass = %q, want %q", reasonClass, "missing_metadata")
	}
}

func TestNetworkPolicyEnforceIngressFailClosedWithoutException(t *testing.T) {
	t.Setenv(envNetworkPublicIngressEnabled, "true")

	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}

	recorder, metrics := newTestDeployTelemetryRecorder()
	err = policy.EnforceIngress(context.Background(), recorder)
	if err == nil {
		t.Fatal("EnforceIngress() error = nil, want non-nil")
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `network_policy_violation_total`) {
		t.Fatalf("metrics output does not contain network policy violations:\n%s", metricsText)
	}
	if !strings.Contains(metricsText, `policy_class="ingress"`) {
		t.Fatalf("metrics output does not contain ingress policy class:\n%s", metricsText)
	}
	if !strings.Contains(metricsText, `reason_class="missing_exception"`) {
		t.Fatalf("metrics output does not contain missing_exception reason:\n%s", metricsText)
	}
}

func TestNetworkPolicyEnforceIngressAllowsActiveException(t *testing.T) {
	now := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	t.Setenv(envNetworkPublicIngressEnabled, "true")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ACTIVE", "true")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ID", "ex-ingress-1")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_OWNER", "platform")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_REASON", "temporary-load-test")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_SCOPE", "example.internal")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_EXPIRY", now.Add(2*time.Hour).Format(time.RFC3339))
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ROLLBACK_PLAN", "disable-public-ingress")

	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}
	policy.now = func() time.Time { return now }

	recorder, metrics := newTestDeployTelemetryRecorder()
	if err := policy.EnforceIngress(context.Background(), recorder); err != nil {
		t.Fatalf("EnforceIngress() error = %v, want nil", err)
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `network_exception_active{environment="test",policy_class="ingress"} 1`) {
		t.Fatalf("metrics output does not contain active ingress exception gauge:\n%s", metricsText)
	}
}

func TestNetworkPolicyEnforceIngressRejectsExpiredException(t *testing.T) {
	now := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	t.Setenv(envNetworkPublicIngressEnabled, "true")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ACTIVE", "true")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ID", "ex-ingress-expired")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_OWNER", "platform")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_REASON", "temporary-diagnostic")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_SCOPE", "example.internal")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_EXPIRY", now.Add(-5*time.Minute).Format(time.RFC3339))
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ROLLBACK_PLAN", "disable-public-ingress")

	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}
	policy.now = func() time.Time { return now }

	recorder, metrics := newTestDeployTelemetryRecorder()
	err = policy.EnforceIngress(context.Background(), recorder)
	if err == nil {
		t.Fatal("EnforceIngress() error = nil, want non-nil")
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `policy_class="ingress"`) {
		t.Fatalf("metrics output does not contain ingress policy class:\n%s", metricsText)
	}
	if !strings.Contains(metricsText, `reason_class="expired_exception"`) {
		t.Fatalf("metrics output does not contain expired_exception reason:\n%s", metricsText)
	}
}

func TestNetworkPolicyEnforceEgressTargetDeniesPublicHost(t *testing.T) {
	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}

	recorder, metrics := newTestDeployTelemetryRecorder()
	err = policy.EnforceEgressTarget(context.Background(), recorder, "api.example.com:443", "tcp")
	if err == nil {
		t.Fatal("EnforceEgressTarget() error = nil, want non-nil")
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `policy_class="egress"`) {
		t.Fatalf("metrics output does not contain egress policy class:\n%s", metricsText)
	}
	if !strings.Contains(metricsText, `reason_class="public_target_denied"`) {
		t.Fatalf("metrics output does not contain public_target_denied reason:\n%s", metricsText)
	}
}

func TestNetworkPolicyEnforceEgressTargetAllowsPrivateAndAllowlistedHosts(t *testing.T) {
	t.Setenv(envNetworkEgressAllowlist, "api.example.com,*.allowed.example")
	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}

	recorder, _ := newTestDeployTelemetryRecorder()

	privateTargetErr := policy.EnforceEgressTarget(context.Background(), recorder, "10.0.0.12:5432", "tcp")
	if privateTargetErr != nil {
		t.Fatalf("EnforceEgressTarget(private) error = %v, want nil", privateTargetErr)
	}
	allowlistedErr := policy.EnforceEgressTarget(context.Background(), recorder, "api.example.com:443", "tcp")
	if allowlistedErr != nil {
		t.Fatalf("EnforceEgressTarget(allowlisted exact) error = %v, want nil", allowlistedErr)
	}
	allowlistedSuffixErr := policy.EnforceEgressTarget(context.Background(), recorder, "service.allowed.example:443", "tcp")
	if allowlistedSuffixErr != nil {
		t.Fatalf("EnforceEgressTarget(allowlisted suffix) error = %v, want nil", allowlistedSuffixErr)
	}
}

func TestNetworkPolicyEnforceEgressTargetDeniesSchemeOutsideAllowlist(t *testing.T) {
	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}

	recorder, metrics := newTestDeployTelemetryRecorder()
	err = policy.EnforceEgressTarget(context.Background(), recorder, "10.0.0.12:5432", "udp")
	if err == nil {
		t.Fatal("EnforceEgressTarget() error = nil, want non-nil")
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `reason_class="scheme_denied"`) {
		t.Fatalf("metrics output does not contain scheme_denied reason:\n%s", metricsText)
	}
}

func TestNetworkPolicyEmitEgressExceptionStateRejectsExpiredException(t *testing.T) {
	now := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	t.Setenv("NETWORK_EGRESS_EXCEPTION_ACTIVE", "true")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_ID", "ex-egress-expired")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_OWNER", "platform")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_REASON", "temporary-upstream-debug")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_SCOPE", "api.example.com")
	t.Setenv("NETWORK_EGRESS_EXCEPTION_EXPIRY", now.Add(-5*time.Minute).Format(time.RFC3339))
	t.Setenv("NETWORK_EGRESS_EXCEPTION_ROLLBACK_PLAN", "disable-egress-exception")

	policy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		t.Fatalf("loadNetworkPolicyFromEnv() error = %v", err)
	}
	policy.now = func() time.Time { return now }

	recorder, metrics := newTestDeployTelemetryRecorder()
	err = policy.EmitEgressExceptionState(context.Background(), recorder)
	if err == nil {
		t.Fatal("EmitEgressExceptionState() error = nil, want non-nil")
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `policy_class="egress"`) {
		t.Fatalf("metrics output does not contain egress policy class:\n%s", metricsText)
	}
	if !strings.Contains(metricsText, `reason_class="expired_exception"`) {
		t.Fatalf("metrics output does not contain expired_exception reason:\n%s", metricsText)
	}
}

func newTestDeployTelemetryRecorder() (*deployTelemetryRecorder, *telemetry.Metrics) {
	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	recorder := newDeployTelemetryRecorder(logger, metrics, "test")
	recorder.SetEnvironment("test")
	return recorder, metrics
}

func collectServiceMetricsText(t *testing.T, metrics *telemetry.Metrics) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	resp := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("metrics handler status = %d, want %d", resp.Code, http.StatusOK)
	}
	return resp.Body.String()
}
