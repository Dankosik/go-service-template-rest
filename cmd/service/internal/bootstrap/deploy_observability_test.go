package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestDeployTelemetryRecorderRecordAdmissionEmitsLogAndMetrics(t *testing.T) {
	spanRecorder := installTestTracerProvider(t)

	t.Setenv("ROLLOUT_ID", "rollout-123")
	t.Setenv("DEPLOYMENT_ID", "deploy-123")
	t.Setenv("CI_RUN_ID", "ci-123")
	t.Setenv("COMMIT_SHA", "abc123")

	recorder, metrics, logBuffer := newBufferedDeployTelemetryRecorder("production")
	recorder.admissionStarted = time.Now().Add(-1500 * time.Millisecond)

	recorder.RecordAdmission(context.Background(), "success", "ready", "readiness")
	recorder.RecordAdmission(context.Background(), "failure", "startup_error", "startup")

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `deploy_health_admission_total{environment="production",reason_class="ready",result="success"} 1`) {
		t.Fatalf("metrics output does not contain successful deploy admission:\n%s", metricsText)
	}
	if strings.Contains(metricsText, `deploy_health_admission_total{environment="production",reason_class="startup_error",result="failure"}`) {
		t.Fatalf("metrics output contains duplicate admission recording:\n%s", metricsText)
	}

	entries := parseJSONLogEntries(t, logBuffer)
	if len(entries) != 1 {
		t.Fatalf("log entries len = %d, want %d", len(entries), 1)
	}
	entry := entries[0]
	if got := jsonFieldString(entry, "msg"); got != "deploy_health_check" {
		t.Fatalf("msg = %q, want %q", got, "deploy_health_check")
	}
	if got := jsonFieldString(entry, "result"); got != "success" {
		t.Fatalf("result = %q, want %q", got, "success")
	}
	if got := jsonFieldString(entry, "rollout_id"); got != "rollout-123" {
		t.Fatalf("rollout_id = %q, want %q", got, "rollout-123")
	}
	if got := jsonFieldString(entry, "deployment_id"); got != "deploy-123" {
		t.Fatalf("deployment_id = %q, want %q", got, "deploy-123")
	}
	if got := jsonFieldString(entry, "ci_run_id"); got != "ci-123" {
		t.Fatalf("ci_run_id = %q, want %q", got, "ci-123")
	}
	if got := jsonFieldString(entry, "commit_sha"); got != "abc123" {
		t.Fatalf("commit_sha = %q, want %q", got, "abc123")
	}

	admissionSpan := assertSingleSpanByName(t, spanRecorder.Ended(), "deploy.health.admission")
	assertSpanStringAttribute(t, admissionSpan, "environment", "production")
	assertSpanStringAttribute(t, admissionSpan, "result", "success")
	assertSpanStringAttribute(t, admissionSpan, "reason_class", "ready")
	assertSpanStringAttribute(t, admissionSpan, "rollout_id", "rollout-123")
	assertSpanStringAttribute(t, admissionSpan, "deployment_id", "deploy-123")
	assertSpanStringAttribute(t, admissionSpan, "ci_run_id", "ci-123")
	assertSpanStringAttribute(t, admissionSpan, "commit_sha", "abc123")
}

func TestDeployTelemetryRecorderRecordAdmissionAllowsSuccessAfterFailure(t *testing.T) {
	recorder, metrics, logBuffer := newBufferedDeployTelemetryRecorder("production")

	recorder.RecordAdmission(context.Background(), "failure", "startup_error", "startup")
	recorder.RecordAdmission(context.Background(), "success", "ready", "readiness")

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `deploy_health_admission_total{environment="production",reason_class="startup_error",result="failure"} 1`) {
		t.Fatalf("metrics output does not contain failure admission:\n%s", metricsText)
	}
	if !strings.Contains(metricsText, `deploy_health_admission_total{environment="production",reason_class="ready",result="success"} 1`) {
		t.Fatalf("metrics output does not contain success admission after failure:\n%s", metricsText)
	}

	entries := parseJSONLogEntries(t, logBuffer)
	if len(entries) != 2 {
		t.Fatalf("log entries len = %d, want %d", len(entries), 2)
	}
	if got := jsonFieldString(entries[0], "result"); got != "failure" {
		t.Fatalf("first result = %q, want %q", got, "failure")
	}
	if got := jsonFieldString(entries[1], "result"); got != "success" {
		t.Fatalf("second result = %q, want %q", got, "success")
	}
}

func TestDeployTelemetryRecorderRecordRollbackIncludesCorrelation(t *testing.T) {
	spanRecorder := installTestTracerProvider(t)

	t.Setenv("ROLLOUT_ID", "rollout-rollback")
	t.Setenv("DEPLOYMENT_ID", "deploy-rollback")
	t.Setenv("CI_RUN_ID", "ci-rollback")
	t.Setenv("COMMIT_SHA", "rollbacksha")
	t.Setenv("ROLLBACK_ID", "rb-rollback")
	t.Setenv("ROLLBACK_OWNER", "platform")
	t.Setenv("RAILWAY_PREVIOUS_DEPLOYMENT_ID", "rev-fallback")

	recorder, metrics, logBuffer := newBufferedDeployTelemetryRecorder("production")

	recorder.RecordRollback(context.Background(), "admission_failed", "failure", "", 2*time.Second)
	recorder.RecordRollbackPostcheck("/health/ready", "failure")

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `rollback_execution_total{environment="production",result="failure",trigger="admission_failed"} 1`) {
		t.Fatalf("metrics output does not contain rollback execution:\n%s", metricsText)
	}
	if !strings.Contains(metricsText, `rollback_postcheck_total{endpoint="/health/ready",environment="production",result="failure"} 1`) {
		t.Fatalf("metrics output does not contain rollback postcheck:\n%s", metricsText)
	}

	entries := parseJSONLogEntries(t, logBuffer)
	if len(entries) != 1 {
		t.Fatalf("log entries len = %d, want %d", len(entries), 1)
	}
	entry := entries[0]
	if got := jsonFieldString(entry, "msg"); got != "rollback_execution" {
		t.Fatalf("msg = %q, want %q", got, "rollback_execution")
	}
	if got := jsonFieldString(entry, "trigger"); got != "admission_failed" {
		t.Fatalf("trigger = %q, want %q", got, "admission_failed")
	}
	if got := jsonFieldString(entry, "rollout_id"); got != "rollout-rollback" {
		t.Fatalf("rollout_id = %q, want %q", got, "rollout-rollback")
	}
	if got := jsonFieldString(entry, "rollback_id"); got != "rb-rollback" {
		t.Fatalf("rollback_id = %q, want %q", got, "rb-rollback")
	}
	if got := jsonFieldString(entry, "owner"); got != "platform" {
		t.Fatalf("owner = %q, want %q", got, "platform")
	}
	if got := jsonFieldString(entry, "previous_revision"); got != "rev-fallback" {
		t.Fatalf("previous_revision = %q, want %q", got, "rev-fallback")
	}

	rollbackSpan := assertSingleSpanByName(t, spanRecorder.Ended(), "deploy.rollback.execute")
	assertSpanStringAttribute(t, rollbackSpan, "environment", "production")
	assertSpanStringAttribute(t, rollbackSpan, "trigger", "admission_failed")
	assertSpanStringAttribute(t, rollbackSpan, "result", "failure")
	assertSpanStringAttribute(t, rollbackSpan, "owner", "platform")
	assertSpanStringAttribute(t, rollbackSpan, "previous_revision", "rev-fallback")
	assertSpanStringAttribute(t, rollbackSpan, "rollout_id", "rollout-rollback")
	assertSpanStringAttribute(t, rollbackSpan, "deployment_id", "deploy-rollback")
	assertSpanStringAttribute(t, rollbackSpan, "ci_run_id", "ci-rollback")
	assertSpanStringAttribute(t, rollbackSpan, "commit_sha", "rollbacksha")
	assertSpanStringAttribute(t, rollbackSpan, "rollback_id", "rb-rollback")
}

func TestDeployTelemetryRecorderRecordConfigDriftLifecycle(t *testing.T) {
	recorder, metrics, logBuffer := newBufferedDeployTelemetryRecorder("production")

	recorder.RecordConfigDriftDetected(context.Background(), "runtime", "drift-1", "cfg-1")
	recorder.RecordConfigDriftReconciled(context.Background(), "success", "drift-1", "cfg-1", 3*time.Second)

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `config_drift_detected_total{environment="production",source="runtime"} 1`) {
		t.Fatalf("metrics output does not contain drift detected counter:\n%s", metricsText)
	}
	if !strings.Contains(metricsText, `config_drift_open{environment="production"} 0`) {
		t.Fatalf("metrics output does not contain closed drift gauge state:\n%s", metricsText)
	}
	if !strings.Contains(metricsText, `config_drift_reconcile_duration_seconds_count{environment="production",result="success"} 1`) {
		t.Fatalf("metrics output does not contain drift reconcile duration:\n%s", metricsText)
	}

	entries := parseJSONLogEntries(t, logBuffer)
	if len(entries) != 2 {
		t.Fatalf("log entries len = %d, want %d", len(entries), 2)
	}
	if got := jsonFieldString(entries[0], "msg"); got != "config_drift_detected" {
		t.Fatalf("first msg = %q, want %q", got, "config_drift_detected")
	}
	if got := jsonFieldString(entries[1], "msg"); got != "config_drift_reconciled" {
		t.Fatalf("second msg = %q, want %q", got, "config_drift_reconciled")
	}
}

func TestDeployTelemetryRecorderNetworkPolicySignals(t *testing.T) {
	recorder, metrics, logBuffer := newBufferedDeployTelemetryRecorder("production")

	recorder.RecordNetworkExceptionStateChange(context.Background(), "ingress", "active", "allow", "ex-1")
	recorder.RecordNetworkEgressPolicyViolation(context.Background(), "public_target_denied", "deny")

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `network_exception_active{environment="production",policy_class="ingress"} 1`) {
		t.Fatalf("metrics output does not contain active ingress exception gauge:\n%s", metricsText)
	}
	if !strings.Contains(metricsText, `network_policy_violation_total{environment="production",policy_class="egress",reason_class="public_target_denied"} 1`) {
		t.Fatalf("metrics output does not contain egress violation counter:\n%s", metricsText)
	}

	entries := parseJSONLogEntries(t, logBuffer)
	if len(entries) != 2 {
		t.Fatalf("log entries len = %d, want %d", len(entries), 2)
	}
	if got := jsonFieldString(entries[0], "msg"); got != "network_exception_state_change" {
		t.Fatalf("first msg = %q, want %q", got, "network_exception_state_change")
	}
	if got := jsonFieldString(entries[1], "msg"); got != "network_egress_policy_violation" {
		t.Fatalf("second msg = %q, want %q", got, "network_egress_policy_violation")
	}
}

func newBufferedDeployTelemetryRecorder(environment string) (*deployTelemetryRecorder, *telemetry.Metrics, *bytes.Buffer) {
	metrics := telemetry.New()
	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))

	recorder := newDeployTelemetryRecorder(logger, metrics, environment)
	recorder.SetEnvironment(environment)

	return recorder, metrics, logBuffer
}

func parseJSONLogEntries(t *testing.T, logBuffer *bytes.Buffer) []map[string]any {
	t.Helper()

	trimmed := strings.TrimSpace(logBuffer.String())
	if trimmed == "" {
		return nil
	}

	lines := strings.Split(trimmed, "\n")
	entries := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		entry := map[string]any{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("json.Unmarshal(log line) error = %v, line = %q", err, line)
		}
		entries = append(entries, entry)
	}

	return entries
}

func jsonFieldString(entry map[string]any, key string) string {
	value, ok := entry[key]
	if !ok {
		return ""
	}
	str, _ := value.(string)
	return str
}

func installTestTracerProvider(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()

	previousProvider := otel.GetTracerProvider()
	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider()
	tracerProvider.RegisterSpanProcessor(spanRecorder)
	otel.SetTracerProvider(tracerProvider)

	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer shutdownCancel()
		if err := tracerProvider.Shutdown(shutdownCtx); err != nil {
			t.Errorf("tracerProvider.Shutdown() error = %v", err)
		}
	})

	return spanRecorder
}

func assertSingleSpanByName(t *testing.T, spans []sdktrace.ReadOnlySpan, name string) sdktrace.ReadOnlySpan {
	t.Helper()

	var matched []sdktrace.ReadOnlySpan
	for _, span := range spans {
		if span.Name() == name {
			matched = append(matched, span)
		}
	}
	if len(matched) != 1 {
		names := make([]string, 0, len(spans))
		for _, span := range spans {
			names = append(names, span.Name())
		}
		t.Fatalf("span %q count = %d, want 1 (all spans: %v)", name, len(matched), names)
	}
	return matched[0]
}

func assertSpanStringAttribute(t *testing.T, span sdktrace.ReadOnlySpan, key, want string) {
	t.Helper()

	for _, attr := range span.Attributes() {
		if string(attr.Key) != key {
			continue
		}
		if got := attr.Value.AsString(); got != want {
			t.Fatalf("span %q attribute %q = %q, want %q", span.Name(), key, got, want)
		}
		return
	}

	t.Fatalf("span %q missing attribute %q", span.Name(), key)
}
