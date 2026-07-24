package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"go.opentelemetry.io/otel"
)

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
	telemetrytest.ClearAmbientExporterEnv(t)
	telemetrytest.RestoreGlobals(t)
	t.Setenv(envNetworkEgressAllowedSchemes, "http")

	metrics := telemetry.New()
	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, nil))

	cleanup, err := bootstrapTelemetryStage(
		context.Background(),
		telemetryStageTestConfig("http://127.0.0.1:4318"),
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
	telemetrytest.ClearAmbientExporterEnv(t)
	telemetrytest.RestoreGlobals(t)
	t.Setenv(envNetworkEgressAllowedSchemes, "http")

	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))

	cleanup, err := bootstrapTelemetryStage(
		context.Background(),
		telemetryStageTestConfig("http://public-otel.example.com:4318"),
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
	telemetrytest.ClearAmbientExporterEnv(t)
	telemetrytest.RestoreGlobals(t)
	t.Setenv(envNetworkEgressAllowedSchemes, "http")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "authorization=Bearer secret-value")

	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))

	cleanup, err := bootstrapTelemetryStage(
		context.Background(),
		telemetryStageTestConfig("http://127.0.0.1:4318"),
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
	telemetrytest.ClearAmbientExporterEnv(t)
	telemetrytest.RestoreGlobals(t)
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
		telemetryStageTestConfig("http://127.0.0.1:4318"),
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

func TestBootstrapTelemetryStageAdmitTelemetryExporterTargetSkipsOnlyOnInvalidPolicy(t *testing.T) {
	//nolint:paralleltest // Sibling subtests mutate process-wide network policy env and must stay serialized.
	t.Run("unconfigured target does not skip tracing", func(t *testing.T) {
		skip, err := admitTelemetryExporterTarget(telemetry.TraceExporterConfig{}, loadNetworkPolicy())
		if err != nil {
			t.Fatalf("admitTelemetryExporterTarget() error = %v, want nil", err)
		}
		if skip {
			t.Fatal("admitTelemetryExporterTarget() = true, want false for unconfigured target")
		}
	})

	t.Run("allowed target does not skip tracing", func(t *testing.T) {
		t.Setenv(envNetworkEgressAllowedSchemes, "http")
		skip, err := admitTelemetryExporterTarget(
			traceExporterConfig(telemetryStageTestConfig("http://127.0.0.1:4318")),
			loadNetworkPolicy(),
		)
		if err != nil {
			t.Fatalf("admitTelemetryExporterTarget() error = %v, want nil", err)
		}
		if skip {
			t.Fatal("admitTelemetryExporterTarget() = true, want false for allowed target")
		}
	})

	t.Run("invalid network policy defers and skips tracing", func(t *testing.T) {
		t.Setenv(envNetworkEgressAllowedSchemes, "1bad")
		netPolicyResult := loadNetworkPolicy()
		if netPolicyResult.err == nil {
			t.Fatal("loadNetworkPolicy() error = nil, want invalid policy")
		}
		skip, err := admitTelemetryExporterTarget(
			traceExporterConfig(telemetryStageTestConfig("http://127.0.0.1:4318")),
			netPolicyResult,
		)
		if err != nil {
			t.Fatalf("admitTelemetryExporterTarget() error = %v, want nil", err)
		}
		if !skip {
			t.Fatal("admitTelemetryExporterTarget() = false, want true when policy is invalid")
		}
	})
}

func TestBootstrapStagesUseOnceLoadedNetworkPolicyResult(t *testing.T) {
	telemetrytest.ClearAmbientExporterEnv(t)
	telemetrytest.RestoreGlobals(t)
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
		telemetryStageTestConfig("http://127.0.0.1:4318"),
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
				},
			},
		},
	}
}
