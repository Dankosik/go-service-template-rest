package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestFailedStageDetails(t *testing.T) {
	t.Parallel()

	stage, dur := failedStageDetails(config.LoadReport{})
	if stage != config.StageLoadDefaults {
		t.Fatalf("stage = %q, want %q", stage, config.StageLoadDefaults)
	}
	if dur <= 0 {
		t.Fatalf("duration = %s, want > 0", dur)
	}

	stage, dur = failedStageDetails(config.LoadReport{FailedStage: config.StageValidate, FailedStageDuration: 2 * time.Second})
	if stage != config.StageValidate || dur != 2*time.Second {
		t.Fatalf("got (%q,%s), want (%q,%s)", stage, dur, config.StageValidate, 2*time.Second)
	}
}

func TestTelemetryInitFailureReason(t *testing.T) {
	t.Parallel()
	if got := telemetryInitFailureReason(context.DeadlineExceeded); got != "deadline_exceeded" {
		t.Fatalf("got %q", got)
	}
	if got := telemetryInitFailureReason(context.Canceled); got != "canceled" {
		t.Fatalf("got %q", got)
	}
	if got := telemetryInitFailureReason(errors.New("x")); got != "setup_error" {
		t.Fatalf("got %q", got)
	}
}

func TestStartupLogArgsIncludesTraceIDs(t *testing.T) {
	spanRecorder := installTestTracerProvider(t)
	ctx, span := otel.Tracer("test").Start(context.Background(), "startup-log-test")
	args := startupLogArgs(ctx, "c", "o", "ok", "k", "v")
	span.End()
	if len(spanRecorder.Ended()) == 0 {
		t.Fatal("expected ended span")
	}

	foundTrace := false
	foundSpan := false
	for i := 0; i < len(args)-1; i += 2 {
		k, ok := args[i].(string)
		if !ok {
			continue
		}
		if k == "trace_id" {
			v, _ := args[i+1].(string)
			foundTrace = strings.TrimSpace(v) != ""
		}
		if k == "span_id" {
			v, _ := args[i+1].(string)
			foundSpan = strings.TrimSpace(v) != ""
		}
	}
	if !foundTrace || !foundSpan {
		t.Fatalf("trace/span ids not found in args: %#v", args)
	}
}

func TestRecordConfigHelpers(t *testing.T) {
	t.Parallel()

	metrics := telemetry.New()
	recordConfigSuccessMetrics(metrics, config.LoadReport{
		LoadDefaultsDuration: 10 * time.Millisecond,
		LoadFileDuration:     10 * time.Millisecond,
		LoadEnvDuration:      10 * time.Millisecond,
		ParseDuration:        10 * time.Millisecond,
		ValidateDuration:     10 * time.Millisecond,
	})
	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `config_load_duration_seconds_count{result="success",stage="config.load.defaults"}`) {
		t.Fatalf("metrics output missing stage count:\n%s", metricsText)
	}

	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(spanRecorder)
	tracer := provider.Tracer("test")
	recordConfigStageSpan(tracer, context.Background(), "cfg.stage", 15*time.Millisecond, "success", "")
	recordConfigStageSpan(tracer, context.Background(), "cfg.zero", 0, "success", "")
	_ = provider.Shutdown(context.Background())
	if len(spanRecorder.Ended()) == 0 {
		t.Fatal("expected recorded config stage span")
	}
}

func TestPolicyViolationAndRollbackHelpers(t *testing.T) {
	t.Parallel()

	metrics := telemetry.New()
	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))

	ctx, span := otel.Tracer("test").Start(context.Background(), "policy")
	err := rejectStartupForPolicyViolation(
		ctx,
		span,
		metrics,
		logger,
		"redis",
		errors.New("blocked"),
	)
	span.End()
	if err == nil {
		t.Fatal("rejectStartupForPolicyViolation() error = nil, want non-nil")
	}
	if !errors.Is(err, config.ErrDependencyInit) {
		t.Fatalf("err = %v, want wrapped %v", err, config.ErrDependencyInit)
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `config_startup_outcome_total{outcome="rejected"}`) {
		t.Fatalf("metrics output missing rejected startup outcome:\n%s", metricsText)
	}
}

func TestRejectStartupForPolicyViolationLogsRootCause(t *testing.T) {
	t.Parallel()

	metrics := telemetry.New()
	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))

	ctx, span := otel.Tracer("test").Start(context.Background(), "policy-log")
	rootCause := errors.New("NETWORK_INGRESS_EXCEPTION_EXPIRY must be RFC3339")
	err := rejectStartupForPolicyViolation(
		ctx,
		span,
		metrics,
		logger,
		"network_policy",
		rootCause,
	)
	span.End()
	if err == nil {
		t.Fatal("rejectStartupForPolicyViolation() error = nil, want non-nil")
	}
	if !strings.Contains(logBuffer.String(), "RFC3339") {
		t.Fatalf("policy violation log does not contain root cause:\n%s", logBuffer.String())
	}
}

func TestRecordDependencyProbeRejectionLogsRootCause(t *testing.T) {
	t.Parallel()

	metrics := telemetry.New()
	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))
	rootCause := errors.New("redis probe connection refused")
	ctx, span := otel.Tracer("test").Start(context.Background(), "dependency-probe-log")
	runtime := dependencyProbeRuntime{
		tracer:        otel.Tracer("test"),
		bootstrapSpan: span,
		metrics:       metrics,
		log:           logger,
	}

	recordDependencyProbeRejection(
		ctx,
		runtime,
		" Redis ",
		" redis_probe ",
		" startup.probe.redis ",
		" cache ",
		rootCause,
	)
	span.End()

	logLine := logBuffer.String()
	if !strings.Contains(logLine, `"msg":"startup_blocked"`) {
		t.Fatalf("dependency probe rejection log = %q, want startup_blocked message", logLine)
	}
	if !strings.Contains(logLine, `"dependency":"redis"`) {
		t.Fatalf("dependency probe rejection log = %q, want normalized dependency", logLine)
	}
	if !strings.Contains(logLine, `"mode":"cache"`) {
		t.Fatalf("dependency probe rejection log = %q, want mode", logLine)
	}
	if !strings.Contains(logLine, `"err":"redis probe connection refused"`) {
		t.Fatalf("dependency probe rejection log = %q, want root cause err", logLine)
	}
}

func TestBootstrapNetworkPolicyStagePreservesConfigCause(t *testing.T) {
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ACTIVE", "true")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_OWNER", "platform")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_REASON", "temporary-diagnostic")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_SCOPE", "example.internal")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_EXPIRY", "not-rfc3339")
	t.Setenv("NETWORK_INGRESS_EXCEPTION_ROLLBACK_PLAN", "disable-public-ingress")

	metrics := telemetry.New()
	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))

	ctx, span := otel.Tracer("test").Start(context.Background(), "network-policy-stage")
	_, err := bootstrapNetworkPolicyStage(ctx, span, metrics, logger, config.Config{})
	span.End()
	if err == nil {
		t.Fatal("bootstrapNetworkPolicyStage() error = nil, want non-nil")
	}
	if !errors.Is(err, config.ErrDependencyInit) {
		t.Fatalf("bootstrapNetworkPolicyStage() error = %v, want wrapped %v", err, config.ErrDependencyInit)
	}
	if !strings.Contains(err.Error(), "RFC3339") {
		t.Fatalf("bootstrapNetworkPolicyStage() error = %v, want original parse detail", err)
	}
}

func TestBootstrapNetworkPolicyStageRequiresExplicitIngressDeclarationForNonLocalWildcardBind(t *testing.T) {
	t.Setenv(envNetworkPublicIngressEnabled, "")

	metrics := telemetry.New()
	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))

	ctx, span := otel.Tracer("test").Start(context.Background(), "network-policy-stage")
	_, err := bootstrapNetworkPolicyStage(ctx, span, metrics, logger, config.Config{
		App:  config.AppConfig{Env: "prod"},
		HTTP: config.HTTPConfig{Addr: ":8080"},
	})
	span.End()
	if err == nil {
		t.Fatal("bootstrapNetworkPolicyStage() error = nil, want non-nil")
	}
	if !errors.Is(err, config.ErrDependencyInit) {
		t.Fatalf("bootstrapNetworkPolicyStage() error = %v, want wrapped %v", err, config.ErrDependencyInit)
	}
	if !strings.Contains(err.Error(), envNetworkPublicIngressEnabled) {
		t.Fatalf("bootstrapNetworkPolicyStage() error = %v, want missing ingress declaration detail", err)
	}
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
