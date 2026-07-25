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

func TestBootstrapTelemetryStageConfiguresExporter(t *testing.T) {
	telemetrytest.ClearAmbientExporterEnv(t)
	telemetrytest.RestoreGlobals(t)

	cleanup, err := bootstrapTelemetryStage(
		context.Background(),
		telemetryStageTestConfig("http://127.0.0.1:4318"),
		telemetry.New(),
		slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)),
	)
	if err != nil {
		t.Fatalf("bootstrapTelemetryStage() error = %v", err)
	}
	t.Cleanup(func() { cleanup(context.Background()) })
}

func TestBootstrapTelemetryStageRejectsAmbientExporterEnv(t *testing.T) {
	telemetrytest.ClearAmbientExporterEnv(t)
	telemetrytest.RestoreGlobals(t)
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "authorization=Bearer secret-value")

	cleanup, err := bootstrapTelemetryStage(
		context.Background(),
		telemetryStageTestConfig("http://127.0.0.1:4318"),
		telemetry.New(),
		slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)),
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

func TestReportIgnoredAmbientOTLPEnvWarnsWhenExporterUnconfigured(t *testing.T) {
	telemetrytest.ClearAmbientExporterEnv(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://injected-collector.example:4318")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "authorization=Bearer secret-value")

	var buf bytes.Buffer
	reportIgnoredAmbientOTLPEnv(
		context.Background(),
		slog.New(slog.NewJSONHandler(&buf, nil)),
		telemetry.TraceExporterConfig{},
	)

	logged := buf.String()
	if !strings.Contains(logged, "telemetry_ambient_env_ignored") {
		t.Fatalf("log = %q, want ambient env warning", logged)
	}
	for _, want := range []string{
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_HEADERS",
		"observability.otel.exporter.otlp_endpoint",
	} {
		if !strings.Contains(logged, want) {
			t.Fatalf("log = %q, want to contain %q", logged, want)
		}
	}
	for _, leaked := range []string{"Bearer", "secret-value", "injected-collector"} {
		if strings.Contains(logged, leaked) {
			t.Fatalf("log = %q, leaked %q", logged, leaked)
		}
	}
}

// An injected endpoint no longer disables this service's own trace export, so
// the operator needs to learn that their collector is not the destination. The
// warning names the variable and the config key that won.
func TestReportIgnoredAmbientOTLPEnvWarnsOnOverriddenEndpointWhenConfigured(t *testing.T) {
	telemetrytest.ClearAmbientExporterEnv(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://injected-collector.example:4318")

	var buf bytes.Buffer
	reportIgnoredAmbientOTLPEnv(
		context.Background(),
		slog.New(slog.NewJSONHandler(&buf, nil)),
		telemetry.TraceExporterConfig{OTLPEndpoint: "http://127.0.0.1:4318"},
	)

	logged := buf.String()
	for _, want := range []string{
		"telemetry_ambient_env_ignored",
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		startupDependencyModeConfigured,
		"observability.otel.exporter.otlp_endpoint",
	} {
		if !strings.Contains(logged, want) {
			t.Fatalf("log = %q, want %q", logged, want)
		}
	}
}

// A credential or trust variable fails exporter setup and is reported as
// degraded telemetry. Listing it here as merely "ignored" would contradict that
// record, so this path stays silent for it.
func TestReportIgnoredAmbientOTLPEnvSilentOnConflictWhenConfigured(t *testing.T) {
	telemetrytest.ClearAmbientExporterEnv(t)
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "authorization=Bearer secret-value")

	var buf bytes.Buffer
	reportIgnoredAmbientOTLPEnv(
		context.Background(),
		slog.New(slog.NewJSONHandler(&buf, nil)),
		telemetry.TraceExporterConfig{OTLPEndpoint: "http://127.0.0.1:4318"},
	)

	if buf.Len() != 0 {
		t.Fatalf("log = %q, want no warning for a variable that fails exporter setup", buf.String())
	}
}

func TestReportIgnoredAmbientOTLPEnvSilentWithoutAmbientEnv(t *testing.T) {
	telemetrytest.ClearAmbientExporterEnv(t)

	var buf bytes.Buffer
	reportIgnoredAmbientOTLPEnv(
		context.Background(),
		slog.New(slog.NewJSONHandler(&buf, nil)),
		telemetry.TraceExporterConfig{},
	)

	if buf.Len() != 0 {
		t.Fatalf("log = %q, want no warning without ambient env", buf.String())
	}
}

// The startup summary is the one line an operator already reads. Trace-export
// state belongs there so "this service exports no traces" does not depend on
// correlating a separate warning a log filter may drop.
func TestBootstrapReportStageRecordsTraceExporterState(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name         string
		otlpEndpoint string
		initErr      error
		want         string
	}{
		{name: "active", otlpEndpoint: "http://127.0.0.1:4318", want: `"tracing.exporter":"active"`},
		{name: "disabled", otlpEndpoint: "", want: `"tracing.exporter":"disabled"`},
		{
			name:         "degraded",
			otlpEndpoint: "http://127.0.0.1:4318",
			initErr:      errors.New("setup tracing: boom"),
			want:         `"tracing.exporter":"degraded"`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			bootstrapReportStage(
				context.Background(),
				slog.New(slog.NewJSONHandler(&buf, nil)),
				telemetryStageTestConfig(tt.otlpEndpoint),
				config.LoadOptions{},
				config.LoadReport{},
				tt.initErr,
			)

			if !strings.Contains(buf.String(), tt.want) {
				t.Fatalf("log = %q, want %s", buf.String(), tt.want)
			}
		})
	}
}

func TestBootstrapReportStageLogsTelemetryFailureCause(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	bootstrapReportStage(
		context.Background(),
		slog.New(slog.NewJSONHandler(&buf, nil)),
		telemetryStageTestConfig(""),
		config.LoadOptions{},
		config.LoadReport{},
		errors.New("unsupported ambient otel exporter environment (OTEL_EXPORTER_OTLP_ENDPOINT)"),
	)

	logged := buf.String()
	if !strings.Contains(logged, "startup_dependency_degraded") {
		t.Fatalf("log = %q, want degraded warning", logged)
	}
	// A bare reason class is not actionable; the operator needs the cause.
	if !strings.Contains(logged, "OTEL_EXPORTER_OTLP_ENDPOINT") {
		t.Fatalf("log = %q, want the telemetry failure cause", logged)
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
