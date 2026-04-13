package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	if got := telemetryInitFailureReason(context.DeadlineExceeded); got != telemetry.TelemetryFailureReasonDeadlineExceeded {
		t.Fatalf("got %q", got)
	}
	if got := telemetryInitFailureReason(context.Canceled); got != telemetry.TelemetryFailureReasonCanceled {
		t.Fatalf("got %q", got)
	}
	if got := telemetryInitFailureReason(errors.New("x")); got != telemetry.TelemetryFailureReasonSetupError {
		t.Fatalf("got %q", got)
	}
}

func TestBootstrapTelemetryStageAdmitsAllowedExporterTarget(t *testing.T) {
	restoreGlobalTelemetry(t)
	t.Setenv(envNetworkEgressAllowedSchemes, "http")

	metrics := telemetry.New()
	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))

	cleanup, err := bootstrapTelemetryStage(
		context.Background(),
		telemetryStageTestConfig("127.0.0.1:4318"),
		metrics,
		logger,
		loadNetworkPolicy(),
	)
	if err != nil {
		t.Fatalf("bootstrapTelemetryStage() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		cleanup(context.Background())
	})

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `startup_dependency_status{dep="telemetry",mode="optional_fail_open"} 1`) {
		t.Fatalf("metrics output missing ready telemetry status:\n%s", metricsText)
	}
	if strings.Contains(metricsText, `telemetry_init_failure_total{`) {
		t.Fatalf("metrics output contains telemetry init failure:\n%s", metricsText)
	}
}

func TestBootstrapTelemetryStageDeniesExporterTargetFailOpen(t *testing.T) {
	restoreGlobalTelemetry(t)
	t.Setenv(envNetworkEgressAllowedSchemes, "http")

	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))

	cleanup, err := bootstrapTelemetryStage(
		context.Background(),
		telemetryStageTestConfig("public-otel.example.com:4318"),
		metrics,
		logger,
		loadNetworkPolicy(),
	)
	cleanup(context.Background())
	if err == nil {
		t.Fatal("bootstrapTelemetryStage() error = nil, want policy denial")
	}
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("bootstrapTelemetryStage() error = %v, want wrapped %v", err, errDependencyInit)
	}
	if !strings.Contains(err.Error(), "telemetry egress target denied") {
		t.Fatalf("bootstrapTelemetryStage() error = %v, want telemetry egress context", err)
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `telemetry_init_failure_total{reason="setup_error"} 1`) {
		t.Fatalf("metrics output missing telemetry init failure:\n%s", metricsText)
	}
	if !strings.Contains(metricsText, `startup_dependency_status{dep="telemetry",mode="feature_off"} 0`) {
		t.Fatalf("metrics output missing feature_off telemetry status:\n%s", metricsText)
	}
	if strings.Contains(metricsText, `startup_rejections_total{`) {
		t.Fatalf("metrics output contains startup rejection for optional telemetry denial:\n%s", metricsText)
	}
}

func TestBootstrapTelemetryStageRejectsAmbientExporterEnvFailOpen(t *testing.T) {
	restoreGlobalTelemetry(t)
	t.Setenv(envNetworkEgressAllowedSchemes, "http")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "authorization=Bearer secret-value")

	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))

	cleanup, err := bootstrapTelemetryStage(
		context.Background(),
		telemetryStageTestConfig("127.0.0.1:4318"),
		metrics,
		logger,
		loadNetworkPolicy(),
	)
	cleanup(context.Background())
	if err == nil {
		t.Fatal("bootstrapTelemetryStage() error = nil, want ambient env rejection")
	}
	if !strings.Contains(err.Error(), "unsupported ambient otel exporter environment") {
		t.Fatalf("bootstrapTelemetryStage() error = %v, want ambient env context", err)
	}
	for _, leaked := range []string{"Bearer", "secret-value"} {
		if strings.Contains(err.Error(), leaked) {
			t.Fatalf("bootstrapTelemetryStage() error = %v, leaked %q", err, leaked)
		}
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `telemetry_init_failure_total{reason="setup_error"} 1`) {
		t.Fatalf("metrics output missing telemetry init failure:\n%s", metricsText)
	}
	if !strings.Contains(metricsText, `startup_dependency_status{dep="telemetry",mode="feature_off"} 0`) {
		t.Fatalf("metrics output missing feature_off telemetry status:\n%s", metricsText)
	}
	if strings.Contains(metricsText, `startup_dependency_status{dep="telemetry",mode="optional_fail_open"} 1`) {
		t.Fatalf("metrics output marked telemetry ready:\n%s", metricsText)
	}
}

func TestBootstrapTelemetryStageLeavesInvalidNetworkPolicyStartupCritical(t *testing.T) {
	restoreGlobalTelemetry(t)
	t.Setenv(envNetworkEgressAllowedSchemes, "1bad")
	netPolicyResult := loadNetworkPolicy()
	if netPolicyResult.err == nil {
		t.Fatal("loadNetworkPolicy() error = nil, want invalid policy error")
	}

	metrics := telemetry.New()
	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))

	cleanup, err := bootstrapTelemetryStage(
		context.Background(),
		telemetryStageTestConfig("127.0.0.1:4318"),
		metrics,
		logger,
		netPolicyResult,
	)
	cleanup(context.Background())
	if err != nil {
		t.Fatalf("bootstrapTelemetryStage() error = %v, want nil for policy-stage ownership", err)
	}
	metricsText := collectServiceMetricsText(t, metrics)
	if strings.Contains(metricsText, `telemetry_init_failure_total{`) {
		t.Fatalf("metrics output contains telemetry init failure for startup-critical policy error:\n%s", metricsText)
	}

	ctx, span := otel.Tracer("test").Start(context.Background(), "invalid-network-policy")
	_, networkErr := bootstrapNetworkPolicyStage(ctx, span, metrics, logger, netPolicyResult, config.Config{})
	span.End()
	if networkErr == nil {
		t.Fatal("bootstrapNetworkPolicyStage() error = nil, want invalid network policy rejection")
	}
	if !errors.Is(networkErr, errDependencyInit) {
		t.Fatalf("bootstrapNetworkPolicyStage() error = %v, want wrapped %v", networkErr, errDependencyInit)
	}

	metricsText = collectServiceMetricsText(t, metrics)
	assertStartupRejectionMetric(t, metricsText, telemetry.StartupRejectionReasonPolicyViolation)
}

func TestBootstrapTelemetryStageAdmitTelemetryExporterTargetUsesNamedOutcomes(t *testing.T) {
	t.Run("unconfigured", func(t *testing.T) {
		got, err := admitTelemetryExporterTarget(telemetry.TraceExporterConfig{}, loadNetworkPolicy())
		if err != nil {
			t.Fatalf("admitTelemetryExporterTarget() error = %v, want nil", err)
		}
		if got != telemetryExporterTargetUnconfigured {
			t.Fatalf("admitTelemetryExporterTarget() = %v, want %v", got, telemetryExporterTargetUnconfigured)
		}
	})

	t.Run("allowed", func(t *testing.T) {
		t.Setenv(envNetworkEgressAllowedSchemes, "http")
		got, err := admitTelemetryExporterTarget(
			traceExporterConfig(telemetryStageTestConfig("127.0.0.1:4318")),
			loadNetworkPolicy(),
		)
		if err != nil {
			t.Fatalf("admitTelemetryExporterTarget() error = %v, want nil", err)
		}
		if got != telemetryExporterTargetAllowed {
			t.Fatalf("admitTelemetryExporterTarget() = %v, want %v", got, telemetryExporterTargetAllowed)
		}
	})

	t.Run("deferred to network policy", func(t *testing.T) {
		t.Setenv(envNetworkEgressAllowedSchemes, "1bad")
		netPolicyResult := loadNetworkPolicy()
		if netPolicyResult.err == nil {
			t.Fatal("loadNetworkPolicy() error = nil, want invalid policy")
		}
		got, err := admitTelemetryExporterTarget(
			traceExporterConfig(telemetryStageTestConfig("127.0.0.1:4318")),
			netPolicyResult,
		)
		if err != nil {
			t.Fatalf("admitTelemetryExporterTarget() error = %v, want nil", err)
		}
		if got != telemetryExporterTargetDeferredToNetworkPolicy {
			t.Fatalf("admitTelemetryExporterTarget() = %v, want %v", got, telemetryExporterTargetDeferredToNetworkPolicy)
		}
	})
}

func TestBootstrapStagesUseOnceLoadedNetworkPolicyResult(t *testing.T) {
	restoreGlobalTelemetry(t)
	t.Setenv(envNetworkEgressAllowedSchemes, "http")
	netPolicyResult := loadNetworkPolicy()
	if netPolicyResult.err != nil {
		t.Fatalf("loadNetworkPolicy() error = %v", netPolicyResult.err)
	}
	t.Setenv(envNetworkEgressAllowedSchemes, "1bad")

	metrics := telemetry.New()
	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))

	cleanup, err := bootstrapTelemetryStage(
		context.Background(),
		telemetryStageTestConfig("127.0.0.1:4318"),
		metrics,
		logger,
		netPolicyResult,
	)
	if err != nil {
		t.Fatalf("bootstrapTelemetryStage() error = %v, want nil from loaded policy", err)
	}
	cleanup(context.Background())

	ctx, span := otel.Tracer("test").Start(context.Background(), "loaded-network-policy")
	_, err = bootstrapNetworkPolicyStage(ctx, span, metrics, logger, netPolicyResult, config.Config{})
	span.End()
	if err != nil {
		t.Fatalf("bootstrapNetworkPolicyStage() error = %v, want nil from loaded policy", err)
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

func restoreGlobalTelemetry(t *testing.T) {
	t.Helper()

	clearAmbientTraceExporterEnv(t)

	previousTracerProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(previousTracerProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})
}

func clearAmbientTraceExporterEnv(t *testing.T) {
	t.Helper()

	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "OTEL_EXPORTER_OTLP_") {
			t.Setenv(name, "")
		}
	}
	for _, name := range []string{
		"HTTP_PROXY",
		"HTTPS_PROXY",
		"NO_PROXY",
		"http_proxy",
		"https_proxy",
		"no_proxy",
	} {
		t.Setenv(name, "")
	}
}

func telemetryStageTestConfig(otlpEndpoint string) config.Config {
	return config.Config{
		App: config.AppConfig{
			Env:     "local",
			Version: "test",
		},
		Observability: config.ObservabilityConfig{
			OTel: config.OTelConfig{
				ServiceName:      "test-service",
				TracesSampler:    "always_off",
				TracesSamplerArg: 0,
				Exporter: config.OTelExporterConfig{
					OTLPEndpoint: otlpEndpoint,
					OTLPProtocol: "http/protobuf",
				},
			},
		},
	}
}

func TestRecordConfigHelpers(t *testing.T) {
	t.Parallel()

	report := config.LoadReport{
		LoadDefaultsDuration: 10 * time.Millisecond,
		LoadFileDuration:     11 * time.Millisecond,
		LoadEnvDuration:      12 * time.Millisecond,
		ParseDuration:        13 * time.Millisecond,
		ValidateDuration:     14 * time.Millisecond,
	}
	wantStages := []string{
		config.StageLoadDefaults,
		config.StageLoadFile,
		config.StageLoadEnv,
		config.StageParse,
		config.StageValidate,
	}
	stageDurations := configLoadStageDurations(report)
	if len(stageDurations) != len(wantStages) {
		t.Fatalf("configLoadStageDurations() len = %d, want %d", len(stageDurations), len(wantStages))
	}
	for i, wantStage := range wantStages {
		if stageDurations[i].stage != wantStage {
			t.Fatalf("configLoadStageDurations()[%d].stage = %q, want %q", i, stageDurations[i].stage, wantStage)
		}
	}

	metrics := telemetry.New()
	recordConfigSuccessMetrics(metrics, report)
	metricsText := collectServiceMetricsText(t, metrics)
	for _, stage := range wantStages {
		pattern := `config_load_duration_seconds_count{result="success",stage="` + stage + `"}`
		if !strings.Contains(metricsText, pattern) {
			t.Fatalf("metrics output missing stage count %q:\n%s", stage, metricsText)
		}
	}

	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(spanRecorder)
	tracer := provider.Tracer("test")
	for _, stage := range stageDurations {
		recordConfigStageSpan(tracer, context.Background(), stage.stage, stage.duration, "success", "")
	}
	recordConfigStageSpan(tracer, context.Background(), "cfg.zero", 0, "success", "")
	_ = provider.Shutdown(context.Background())
	spans := spanRecorder.Ended()
	if len(spans) != len(wantStages) {
		t.Fatalf("ended spans len = %d, want %d", len(spans), len(wantStages))
	}
	seenSpans := make(map[string]struct{}, len(spans))
	for _, span := range spans {
		seenSpans[span.Name()] = struct{}{}
	}
	for _, stage := range wantStages {
		if _, ok := seenSpans[stage]; !ok {
			t.Fatalf("missing config stage span %q; spans=%v", stage, seenSpans)
		}
	}
}

func TestBootstrapConfigStageRecordsConfigFailureAndStartupRejection(t *testing.T) {
	t.Setenv("APP__APP__ENV", "local")

	metrics := telemetry.New()
	missingConfig := filepath.Join(t.TempDir(), "missing.yaml")

	_, _, err := bootstrapConfigStage(context.Background(), config.LoadOptions{ConfigPath: missingConfig}, metrics)
	if err == nil {
		t.Fatal("bootstrapConfigStage() error = nil, want non-nil")
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `config_failures_total{reason="load"} 1`) {
		t.Fatalf("metrics output missing config load failure:\n%s", metricsText)
	}
	assertStartupRejectionMetric(t, metricsText, telemetry.StartupRejectionReasonConfigLoad)
}

func TestStartupRejectionReasonForConfigErrorType(t *testing.T) {
	testCases := []struct {
		name      string
		errorType string
		want      string
	}{
		{name: "load", errorType: "load", want: telemetry.StartupRejectionReasonConfigLoad},
		{name: "parse", errorType: "parse", want: telemetry.StartupRejectionReasonConfigParse},
		{name: "validate", errorType: "validate", want: telemetry.StartupRejectionReasonConfigValidate},
		{name: "strict unknown key", errorType: "strict_unknown_key", want: telemetry.StartupRejectionReasonConfigStrictUnknownKey},
		{name: "secret policy", errorType: "secret_policy", want: telemetry.StartupRejectionReasonConfigSecretPolicy},
		{name: "unknown", errorType: "new_config_reason", want: telemetry.StartupRejectionReasonOther},
		{name: "empty", errorType: "", want: telemetry.StartupRejectionReasonOther},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := startupRejectionReasonForConfigErrorType(tc.errorType); got != tc.want {
				t.Fatalf("startupRejectionReasonForConfigErrorType(%q) = %q, want %q", tc.errorType, got, tc.want)
			}
		})
	}
}

func TestBootstrapNetworkPolicyStageRejectsPublicIngressForRootMetrics(t *testing.T) {
	now := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	t.Setenv(envNetworkPublicIngressEnabled, "true")
	setValidIngressExceptionEnv(t, now, map[string]string{
		"ID":     "ex-ingress-metrics-bootstrap",
		"REASON": "temporary-public-api",
	})

	netPolicyResult := loadNetworkPolicy()
	if netPolicyResult.err != nil {
		t.Fatalf("loadNetworkPolicy() error = %v", netPolicyResult.err)
	}
	netPolicyResult.policy.now = func() time.Time { return now }

	metrics := telemetry.New()
	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))

	ctx, span := otel.Tracer("test").Start(context.Background(), "metrics-exposure-policy")
	_, err := bootstrapNetworkPolicyStage(ctx, span, metrics, logger, netPolicyResult, config.Config{
		App:  config.AppConfig{Env: "prod"},
		HTTP: config.HTTPConfig{Addr: ":8080"},
	})
	span.End()
	if err == nil {
		t.Fatal("bootstrapNetworkPolicyStage() error = nil, want metrics exposure rejection")
	}
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("bootstrapNetworkPolicyStage() error = %v, want wrapped %v", err, errDependencyInit)
	}
	if !strings.Contains(err.Error(), "operational metrics") {
		t.Fatalf("bootstrapNetworkPolicyStage() error = %v, want operational metrics detail", err)
	}

	metricsText := collectServiceMetricsText(t, metrics)
	assertStartupRejectionMetric(t, metricsText, telemetry.StartupRejectionReasonPolicyViolation)
	assertConfigFailureMetricAbsent(t, metricsText, telemetry.StartupRejectionReasonPolicyViolation)
	if !strings.Contains(logBuffer.String(), `"dependency":"metrics_exposure"`) {
		t.Fatalf("bootstrapNetworkPolicyStage() log = %q, want metrics exposure dependency", logBuffer.String())
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
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("err = %v, want wrapped %v", err, errDependencyInit)
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `config_startup_outcome_total{outcome="rejected"}`) {
		t.Fatalf("metrics output missing rejected startup outcome:\n%s", metricsText)
	}
	assertStartupRejectionMetric(t, metricsText, telemetry.StartupRejectionReasonPolicyViolation)
	assertConfigFailureMetricAbsent(t, metricsText, telemetry.StartupRejectionReasonPolicyViolation)
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

func TestRejectStartupForPolicyViolationDoesNotDuplicateDependencyInitSentinel(t *testing.T) {
	t.Parallel()

	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
	cause := fmt.Errorf("%w: invalid network policy configuration: %w", errDependencyInit, errors.New("RFC3339 parse failed"))

	ctx, span := otel.Tracer("test").Start(context.Background(), "policy-idempotent")
	err := rejectStartupForPolicyViolation(
		ctx,
		span,
		metrics,
		logger,
		startupDependencyNetworkPolicy,
		cause,
	)
	span.End()
	if err == nil {
		t.Fatal("rejectStartupForPolicyViolation() error = nil, want non-nil")
	}
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("rejectStartupForPolicyViolation() error = %v, want wrapped %v", err, errDependencyInit)
	}
	if count := strings.Count(err.Error(), errDependencyInit.Error()); count != 1 {
		t.Fatalf("rejectStartupForPolicyViolation() error = %v, dependency init count = %d, want 1", err, count)
	}
	if !strings.Contains(err.Error(), "RFC3339 parse failed") {
		t.Fatalf("rejectStartupForPolicyViolation() error = %v, want original config detail", err)
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
		startupDependencyProbeLabels{
			dependency: " Redis ",
			operation:  " redis_probe ",
			probeStage: " startup.probe.redis ",
		},
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
	_, err := bootstrapNetworkPolicyStage(ctx, span, metrics, logger, loadNetworkPolicy(), config.Config{})
	span.End()
	if err == nil {
		t.Fatal("bootstrapNetworkPolicyStage() error = nil, want non-nil")
	}
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("bootstrapNetworkPolicyStage() error = %v, want wrapped %v", err, errDependencyInit)
	}
	if !strings.Contains(err.Error(), "RFC3339") {
		t.Fatalf("bootstrapNetworkPolicyStage() error = %v, want original parse detail", err)
	}
	logLine := logBuffer.String()
	if !strings.Contains(logLine, `"policy.class":"ingress"`) {
		t.Fatalf("bootstrapNetworkPolicyStage() log = %q, want policy class", logLine)
	}
	if !strings.Contains(logLine, `"reason.class":"invalid_configuration"`) {
		t.Fatalf("bootstrapNetworkPolicyStage() log = %q, want reason class", logLine)
	}
}

func TestBootstrapNetworkPolicyStageRequiresExplicitIngressDeclarationForNonLocalWildcardBind(t *testing.T) {
	t.Setenv(envNetworkPublicIngressEnabled, "")

	metrics := telemetry.New()
	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))

	ctx, span := otel.Tracer("test").Start(context.Background(), "network-policy-stage")
	_, err := bootstrapNetworkPolicyStage(ctx, span, metrics, logger, loadNetworkPolicy(), config.Config{
		App:  config.AppConfig{Env: "prod"},
		HTTP: config.HTTPConfig{Addr: ":8080"},
	})
	span.End()
	if err == nil {
		t.Fatal("bootstrapNetworkPolicyStage() error = nil, want non-nil")
	}
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("bootstrapNetworkPolicyStage() error = %v, want wrapped %v", err, errDependencyInit)
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

func assertStartupRejectionMetric(t *testing.T, metricsText string, reason string) {
	t.Helper()

	pattern := `startup_rejections_total{reason="` + reason + `"} 1`
	if !strings.Contains(metricsText, pattern) {
		t.Fatalf("metrics output missing startup rejection %q:\n%s", reason, metricsText)
	}
}

func assertConfigFailureMetricAbsent(t *testing.T, metricsText string, reason string) {
	t.Helper()

	pattern := `config_failures_total{reason="` + reason + `"}`
	if strings.Contains(metricsText, pattern) {
		t.Fatalf("metrics output unexpectedly contains config failure %q:\n%s", reason, metricsText)
	}
}
