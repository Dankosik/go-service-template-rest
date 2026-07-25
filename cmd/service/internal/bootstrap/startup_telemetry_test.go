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

	cleanup, endpoint, err := bootstrapTelemetryStage(
		context.Background(),
		telemetryStageTestConfig("http://127.0.0.1:4318"),
		telemetry.New(),
		slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)),
	)
	if err != nil {
		t.Fatalf("bootstrapTelemetryStage() error = %v", err)
	}
	if endpoint.Source != telemetry.TraceExporterConfigKey {
		t.Fatalf("endpoint source = %q, want %q", endpoint.Source, telemetry.TraceExporterConfigKey)
	}
	t.Cleanup(func() { cleanup(context.Background()) })
}

// A platform that injects only the standard endpoint variable must still get
// traces: this is the deployment where ignoring it looks healthy and exports
// nothing.
func TestBootstrapTelemetryStageUsesAmbientEndpointEnv(t *testing.T) {
	telemetrytest.ClearAmbientExporterEnv(t)
	telemetrytest.RestoreGlobals(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:4318")

	cleanup, endpoint, err := bootstrapTelemetryStage(
		context.Background(),
		telemetryStageTestConfig(""),
		telemetry.New(),
		slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)),
	)
	if err != nil {
		t.Fatalf("bootstrapTelemetryStage() error = %v", err)
	}
	if endpoint.Source != "OTEL_EXPORTER_OTLP_ENDPOINT" {
		t.Fatalf("endpoint source = %q, want the ambient endpoint variable", endpoint.Source)
	}
	t.Cleanup(func() { cleanup(context.Background()) })
}

func TestBootstrapTelemetryStageRejectsAmbientExporterEnv(t *testing.T) {
	telemetrytest.ClearAmbientExporterEnv(t)
	telemetrytest.RestoreGlobals(t)
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "authorization=Bearer secret-value")

	cleanup, _, err := bootstrapTelemetryStage(
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

// An unconfigured exporter with ambient variables present is reachable when
// configured headers pin the destination and there is no configured endpoint:
// nothing was honored, so everything injected is reported.
func TestReportIgnoredAmbientOTLPEnvWarnsWhenExporterUnconfigured(t *testing.T) {
	telemetrytest.ClearAmbientExporterEnv(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://injected-collector.example:4318")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "authorization=Bearer secret-value")

	var buf bytes.Buffer
	reportIgnoredAmbientOTLPEnv(
		context.Background(),
		slog.New(slog.NewJSONHandler(&buf, nil)),
		telemetry.TraceExporterEndpoint{},
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
		configuredTestTraceEndpoint(),
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

// The variable that supplied the endpoint was honored, so reporting it as
// ignored would send an operator looking for a problem that does not exist.
// Everything else injected alongside it is still reported.
func TestReportIgnoredAmbientOTLPEnvSkipsTheHonoredEndpointVariable(t *testing.T) {
	telemetrytest.ClearAmbientExporterEnv(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://injected-collector.example:4318")
	t.Setenv("OTEL_EXPORTER_OTLP_TIMEOUT", "15000")

	var buf bytes.Buffer
	reportIgnoredAmbientOTLPEnv(
		context.Background(),
		slog.New(slog.NewJSONHandler(&buf, nil)),
		telemetry.TraceExporterEndpoint{
			URL:    "http://injected-collector.example:4318/v1/traces",
			Source: "OTEL_EXPORTER_OTLP_ENDPOINT",
		},
	)

	logged := buf.String()
	if !strings.Contains(logged, "OTEL_EXPORTER_OTLP_TIMEOUT") {
		t.Fatalf("log = %q, want the genuinely ignored variable", logged)
	}
	if strings.Contains(logged, "OTEL_EXPORTER_OTLP_ENDPOINT") {
		t.Fatalf("log = %q, must not report the honored endpoint variable", logged)
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
		configuredTestTraceEndpoint(),
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
		telemetry.TraceExporterEndpoint{},
	)

	if buf.Len() != 0 {
		t.Fatalf("log = %q, want no warning without ambient env", buf.String())
	}
}

func configuredTestTraceEndpoint() telemetry.TraceExporterEndpoint {
	return telemetry.TraceExporterEndpoint{
		URL:    "http://127.0.0.1:4318/v1/traces",
		Source: telemetry.TraceExporterConfigKey,
	}
}

// The startup summary is the one line an operator already reads. Trace-export
// state belongs there so "this service exports no traces" does not depend on
// correlating a separate warning a log filter may drop.
func TestBootstrapReportStageRecordsTraceExporterState(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name     string
		endpoint telemetry.TraceExporterEndpoint
		initErr  error
		want     []string
	}{
		{
			name:     "active from configuration",
			endpoint: configuredTestTraceEndpoint(),
			want: []string{
				`"tracing.exporter":"active"`,
				`"tracing.endpoint_source":"observability.otel.exporter.otlp_endpoint"`,
			},
		},
		{
			// An operator debugging where traces went needs to see that the
			// destination came from the platform, not from this service.
			name: "active from the ambient endpoint variable",
			endpoint: telemetry.TraceExporterEndpoint{
				URL:    "http://collector.example:4318/v1/traces",
				Source: "OTEL_EXPORTER_OTLP_ENDPOINT",
			},
			want: []string{
				`"tracing.exporter":"active"`,
				`"tracing.endpoint_source":"OTEL_EXPORTER_OTLP_ENDPOINT"`,
			},
		},
		{
			name:     "disabled",
			endpoint: telemetry.TraceExporterEndpoint{},
			want:     []string{`"tracing.exporter":"disabled"`},
		},
		{
			name:     "degraded",
			endpoint: configuredTestTraceEndpoint(),
			initErr:  errors.New("setup tracing: boom"),
			want:     []string{`"tracing.exporter":"degraded"`},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			bootstrapReportStage(
				context.Background(),
				slog.New(slog.NewJSONHandler(&buf, nil)),
				telemetryStageTestConfig(tt.endpoint.URL),
				config.LoadOptions{},
				config.LoadReport{},
				tt.endpoint,
				tt.initErr,
			)

			for _, want := range tt.want {
				if !strings.Contains(buf.String(), want) {
					t.Fatalf("log = %q, want %s", buf.String(), want)
				}
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
		telemetry.TraceExporterEndpoint{},
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
