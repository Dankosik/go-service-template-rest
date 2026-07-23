package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel"
)

func TestFailedConfigStage(t *testing.T) {
	t.Parallel()

	if got := failedConfigStage(config.LoadReport{}); got != config.StageLoadDefaults {
		t.Fatalf("failedConfigStage() = %q, want %q", got, config.StageLoadDefaults)
	}
	if got := failedConfigStage(config.LoadReport{FailedStage: config.StageValidate}); got != config.StageValidate {
		t.Fatalf("failedConfigStage() = %q, want %q", got, config.StageValidate)
	}
}

func TestTelemetryInitFailureReason(t *testing.T) {
	t.Parallel()
	if got := telemetryInitFailureReason(context.DeadlineExceeded); got != telemetryFailureReasonDeadlineExceeded {
		t.Fatalf("got %q", got)
	}
	if got := telemetryInitFailureReason(context.Canceled); got != telemetryFailureReasonCanceled {
		t.Fatalf("got %q", got)
	}
	if got := telemetryInitFailureReason(errors.New("x")); got != telemetryFailureReasonSetupError {
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
	ctx, span := otel.Tracer("test").Start(context.Background(), "invalid-network-policy")
	_, networkErr := bootstrapNetworkPolicyStage(ctx, span, logger, netPolicyResult, config.Config{})
	span.End()
	if networkErr == nil {
		t.Fatal("bootstrapNetworkPolicyStage() error = nil, want invalid network policy rejection")
	}
	if !errors.Is(networkErr, errDependencyInit) {
		t.Fatalf("bootstrapNetworkPolicyStage() error = %v, want wrapped %v", networkErr, errDependencyInit)
	}
}

func TestBootstrapTelemetryStageAdmitTelemetryExporterTargetUsesNamedOutcomes(t *testing.T) {
	//nolint:paralleltest // Sibling subtests mutate process-wide network policy env and must stay serialized.
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
	_, err = bootstrapNetworkPolicyStage(ctx, span, logger, netPolicyResult, config.Config{})
	span.End()
	if err != nil {
		t.Fatalf("bootstrapNetworkPolicyStage() error = %v, want nil from loaded policy", err)
	}
}

//nolint:paralleltest // Installs a process-wide tracer provider for span capture.
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

func TestBootstrapConfigStageReturnsConfigLoadFailure(t *testing.T) {
	t.Setenv("APP__APP__ENV", "local")

	missingConfig := filepath.Join(t.TempDir(), "missing.yaml")

	_, _, err := bootstrapConfigStage(context.Background(), config.LoadOptions{ConfigPath: missingConfig})
	if err == nil {
		t.Fatal("bootstrapConfigStage() error = nil, want non-nil")
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

	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))

	ctx, span := otel.Tracer("test").Start(context.Background(), "metrics-exposure-policy")
	_, err := bootstrapNetworkPolicyStage(ctx, span, logger, netPolicyResult, config.Config{
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

	if !strings.Contains(logBuffer.String(), `"dependency":"metrics_exposure"`) {
		t.Fatalf("bootstrapNetworkPolicyStage() log = %q, want metrics exposure dependency", logBuffer.String())
	}
}

func TestPolicyViolationAndRollbackHelpers(t *testing.T) {
	t.Parallel()

	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))

	ctx, span := otel.Tracer("test").Start(context.Background(), "policy")
	err := rejectStartupForPolicyViolation(
		ctx,
		span,
		logger,
		"cache",
		errors.New("blocked"),
	)
	span.End()
	if err == nil {
		t.Fatal("rejectStartupForPolicyViolation() error = nil, want non-nil")
	}
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("err = %v, want wrapped %v", err, errDependencyInit)
	}

}

func TestRejectStartupForPolicyViolationLogsRootCause(t *testing.T) {
	t.Parallel()

	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))

	ctx, span := otel.Tracer("test").Start(context.Background(), "policy-log")
	rootCause := errors.New("NETWORK_INGRESS_EXCEPTION_EXPIRY must be RFC3339")
	err := rejectStartupForPolicyViolation(
		ctx,
		span,
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

	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
	cause := fmt.Errorf("%w: invalid network policy configuration: %w", errDependencyInit, errors.New("RFC3339 parse failed"))

	ctx, span := otel.Tracer("test").Start(context.Background(), "policy-idempotent")
	err := rejectStartupForPolicyViolation(
		ctx,
		span,
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

	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))
	rootCause := errors.New("cache probe connection refused")
	ctx, span := otel.Tracer("test").Start(context.Background(), "dependency-probe-log")
	runtime := dependencyProbeRuntime{
		tracer:        otel.Tracer("test"),
		bootstrapSpan: span,
		log:           logger,
	}

	recordDependencyProbeRejection(
		ctx,
		runtime,
		startupDependencyProbeLabels{
			dependency: " Cache ",
			operation:  " cache_probe ",
			probeStage: " startup.probe.cache ",
		},
		" cache ",
		rootCause,
	)
	span.End()

	logLine := logBuffer.String()
	if !strings.Contains(logLine, `"msg":"startup_blocked"`) {
		t.Fatalf("dependency probe rejection log = %q, want startup_blocked message", logLine)
	}
	if !strings.Contains(logLine, `"dependency":"cache"`) {
		t.Fatalf("dependency probe rejection log = %q, want normalized dependency", logLine)
	}
	if !strings.Contains(logLine, `"mode":"cache"`) {
		t.Fatalf("dependency probe rejection log = %q, want mode", logLine)
	}
	if !strings.Contains(logLine, `"err":"cache probe connection refused"`) {
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

	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))

	ctx, span := otel.Tracer("test").Start(context.Background(), "network-policy-stage")
	_, err := bootstrapNetworkPolicyStage(ctx, span, logger, loadNetworkPolicy(), config.Config{})
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

	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))

	ctx, span := otel.Tracer("test").Start(context.Background(), "network-policy-stage")
	_, err := bootstrapNetworkPolicyStage(ctx, span, logger, loadNetworkPolicy(), config.Config{
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
