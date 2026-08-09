package telemetry

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestBuildTraceSampler(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		samplerName string
		samplerArg  float64
		wantErr     bool
	}{
		{
			name:        "default sampler",
			samplerName: "",
			samplerArg:  0.1,
		},
		{
			name:        "always_on",
			samplerName: "always_on",
			samplerArg:  0.5,
		},
		{
			name:        "always_off",
			samplerName: "always_off",
			samplerArg:  0.5,
		},
		{
			name:        "traceidratio",
			samplerName: "traceidratio",
			samplerArg:  0.5,
		},
		{
			name:        "parentbased_traceidratio",
			samplerName: "parentbased_traceidratio",
			samplerArg:  0.5,
		},
		{
			name:        "unsupported sampler",
			samplerName: "unsupported",
			samplerArg:  0.5,
			wantErr:     true,
		},
		{
			name:        "nan sampler arg",
			samplerName: "traceidratio",
			samplerArg:  math.NaN(),
			wantErr:     true,
		},
		{
			name:        "negative sampler arg",
			samplerName: "traceidratio",
			samplerArg:  -0.1,
			wantErr:     true,
		},
		{
			name:        "greater than one sampler arg",
			samplerName: "traceidratio",
			samplerArg:  1.1,
			wantErr:     true,
		},
		{
			name:        "positive infinity sampler arg",
			samplerName: "traceidratio",
			samplerArg:  math.Inf(1),
			wantErr:     true,
		},
		{
			name:        "negative infinity sampler arg",
			samplerName: "traceidratio",
			samplerArg:  math.Inf(-1),
			wantErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := buildTraceSampler(tc.samplerName, tc.samplerArg)
			if (err != nil) != tc.wantErr {
				t.Fatalf("buildTraceSampler() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestExporterOptionTypeCompatibility(t *testing.T) {
	t.Parallel()

	// Guard against accidental option-type drift when upgrading OTLP exporter package.
	var _ []otlptracehttp.Option
}

func setupTracingForEnvPolicyTest(t *testing.T, exporter TraceExporterConfig) error {
	t.Helper()

	_, shutdown, err := SetupTracing(context.Background(), envPolicyTracingConfig(exporter))
	if err == nil {
		if shutdownErr := shutdown(context.Background()); shutdownErr != nil {
			t.Fatalf("shutdown tracing after unexpected setup success: %v", shutdownErr)
		}
	}
	return err
}

func envPolicyTracingConfig(exporter TraceExporterConfig) TracingConfig {
	return TracingConfig{
		Resource: ResourceConfig{
			ServiceName:    "test-service",
			ServiceVersion: "test",
			DeploymentEnv:  "local",
		},
		TracesSampler:    "always_off",
		TracesSamplerArg: 0,
		Exporter:         exporter,
	}
}

func requireAmbientExporterEnvError(t *testing.T, err error, want string) {
	t.Helper()

	if err == nil {
		t.Fatal("SetupTracing() error = nil, want ambient exporter environment rejection")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("SetupTracing() error = %v, want %q", err, want)
	}
}

func requireErrorDoesNotContain(t *testing.T, err error, forbidden string) {
	t.Helper()

	if forbidden == "" {
		return
	}
	if strings.Contains(err.Error(), forbidden) {
		t.Fatalf("error %q leaked %q", err, forbidden)
	}
}

func exportOneTestSpan(t *testing.T, options []otlptracehttp.Option) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exporter, err := otlptracehttp.New(ctx, options...)
	if err != nil {
		t.Fatalf("create OTLP trace exporter: %v", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSyncer(exporter),
	)
	_, span := provider.Tracer("telemetry-test").Start(ctx, "test-span")
	span.End()

	if err := provider.ForceFlush(ctx); err != nil {
		_ = provider.Shutdown(ctx)
		t.Fatalf("force flush trace provider: %v", err)
	}
	if err := provider.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown trace provider: %v", err)
	}
}

func assertCollectorPath(t *testing.T, paths <-chan string, want string) {
	t.Helper()

	select {
	case got := <-paths:
		if got != want {
			t.Fatalf("collector path = %q, want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("collector path was not requested")
	}
}

func assertNoCollectorRequest(t *testing.T, paths <-chan string, name string) {
	t.Helper()

	select {
	case got := <-paths:
		t.Fatalf("%s received unexpected request path %q", name, got)
	default:
	}
}

func resourceAttributes(span sdktrace.ReadOnlySpan) map[string]string {
	attrs := make(map[string]string)
	for _, attr := range span.Resource().Attributes() {
		attrs[string(attr.Key)] = attr.Value.AsString()
	}
	return attrs
}
