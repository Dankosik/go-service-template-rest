package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestSetupTracingUsesConfigResourceAttributesOnly(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.name=env-service,service.version=env-version,deployment.environment.name=env,env.only=true")
	t.Setenv("OTEL_SERVICE_NAME", "env-service-name")

	previousTracerProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(previousTracerProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	shutdown, err := SetupTracing(context.Background(), TracingConfig{
		ServiceName:      " config-service ",
		ServiceVersion:   " config-version ",
		DeploymentEnv:    " config-env ",
		TracesSampler:    "always_on",
		TracesSamplerArg: 0.1,
	})
	if err != nil {
		t.Fatalf("SetupTracing() error = %v", err)
	}
	for _, field := range otel.GetTextMapPropagator().Fields() {
		if field == "baggage" {
			t.Fatalf("global propagator fields include unowned baggage: %v", otel.GetTextMapPropagator().Fields())
		}
	}
	t.Cleanup(func() {
		if err := shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown tracing: %v", err)
		}
	})

	provider, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	if !ok {
		t.Fatalf("global tracer provider = %T, want *sdktrace.TracerProvider", otel.GetTracerProvider())
	}
	recorder := tracetest.NewSpanRecorder()
	provider.RegisterSpanProcessor(recorder)
	t.Cleanup(func() {
		provider.UnregisterSpanProcessor(recorder)
	})

	_, span := otel.Tracer("telemetry-test").Start(context.Background(), "resource-test")
	span.End()

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("ended spans len = %d, want 1", len(spans))
	}
	attrs := resourceAttributes(spans[0])
	for key, want := range map[string]string{
		"service.name":                "config-service",
		"service.version":             "config-version",
		"deployment.environment.name": "config-env",
	} {
		got, ok := attrs[key]
		if !ok || got != want {
			t.Fatalf("resource attribute %q = %q (present %v), want %q; attrs=%v", key, got, ok, want, attrs)
		}
	}
	if _, ok := attrs["env.only"]; ok {
		t.Fatalf("resource attribute env.only was read from OTEL_RESOURCE_ATTRIBUTES: %v", attrs)
	}
}

//nolint:paralleltest // Mutates the process-wide OpenTelemetry provider and propagator.
func TestSetupTracingDoesNotApplyResourceIdentityFallbacks(t *testing.T) {
	previousTracerProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(previousTracerProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	shutdown, err := SetupTracing(context.Background(), TracingConfig{
		TracesSampler:    "always_on",
		TracesSamplerArg: 0.1,
	})
	if err != nil {
		t.Fatalf("SetupTracing() error = %v", err)
	}
	t.Cleanup(func() {
		if err := shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown tracing: %v", err)
		}
	})

	provider, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	if !ok {
		t.Fatalf("global tracer provider = %T, want *sdktrace.TracerProvider", otel.GetTracerProvider())
	}
	recorder := tracetest.NewSpanRecorder()
	provider.RegisterSpanProcessor(recorder)
	t.Cleanup(func() {
		provider.UnregisterSpanProcessor(recorder)
	})

	_, span := otel.Tracer("telemetry-test").Start(context.Background(), "resource-test")
	span.End()

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("ended spans len = %d, want 1", len(spans))
	}
	attrs := resourceAttributes(spans[0])
	for key, fallback := range map[string]string{
		"service.name":                "service",
		"service.version":             "dev",
		"deployment.environment.name": "unknown",
	} {
		if got := attrs[key]; got == fallback {
			t.Fatalf("resource attribute %q used fallback %q; attrs=%v", key, fallback, attrs)
		}
	}
}

func TestSetupTracingSerializesResourceEnvSuppression(t *testing.T) {
	const (
		resourceAttrs = "service.name=env-service,service.version=env-version,deployment.environment.name=env,env.only=true"
		serviceName   = "env-service-name"
		setupCount    = 16
	)
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", resourceAttrs)
	t.Setenv("OTEL_SERVICE_NAME", serviceName)

	previousTracerProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(previousTracerProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	shutdowns := make(chan func(context.Context) error, setupCount)
	errs := make(chan error, setupCount)
	var wg sync.WaitGroup
	for i := range setupCount {
		wg.Go(func() {
			shutdown, err := SetupTracing(context.Background(), TracingConfig{
				ServiceName:      fmt.Sprintf("config-service-%d", i),
				ServiceVersion:   "config-version",
				DeploymentEnv:    "config-env",
				TracesSampler:    "always_off",
				TracesSamplerArg: 0,
			})
			if err != nil {
				errs <- err
				return
			}
			shutdowns <- shutdown
		})
	}
	wg.Wait()
	close(errs)
	close(shutdowns)

	for err := range errs {
		if err != nil {
			t.Fatalf("SetupTracing() concurrent error = %v", err)
		}
	}
	for shutdown := range shutdowns {
		if err := shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown tracing: %v", err)
		}
	}
	if got := os.Getenv("OTEL_RESOURCE_ATTRIBUTES"); got != resourceAttrs {
		t.Fatalf("OTEL_RESOURCE_ATTRIBUTES = %q, want %q", got, resourceAttrs)
	}
	if got := os.Getenv("OTEL_SERVICE_NAME"); got != serviceName {
		t.Fatalf("OTEL_SERVICE_NAME = %q, want %q", got, serviceName)
	}
}

func TestSetupTracingRejectsAmbientOTLPExporterEnv(t *testing.T) {
	typedCollector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(typedCollector.Close)

	envCollector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(envCollector.Close)

	tests := []struct {
		name      string
		envName   string
		envValue  string
		forbidden []string
	}{
		{
			name:     "insecure downgrade",
			envName:  "OTEL_EXPORTER_OTLP_INSECURE",
			envValue: "true",
		},
		{
			name:      "header injection",
			envName:   "OTEL_EXPORTER_OTLP_HEADERS",
			envValue:  "authorization=Bearer secret-value",
			forbidden: []string{"Bearer", "secret-value"},
		},
		{
			name:      "generic endpoint retarget",
			envName:   "OTEL_EXPORTER_OTLP_ENDPOINT",
			envValue:  envCollector.URL + "/env",
			forbidden: []string{envCollector.URL, "/env"},
		},
		{
			name:      "trace endpoint retarget",
			envName:   "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
			envValue:  envCollector.URL + "/trace-env",
			forbidden: []string{envCollector.URL, "/trace-env"},
		},
		{
			name:      "certificate path",
			envName:   "OTEL_EXPORTER_OTLP_CERTIFICATE",
			envValue:  "/tmp/secret-ca.pem",
			forbidden: []string{"/tmp/secret-ca.pem", "secret-ca"},
		},
		{
			name:     "timeout",
			envName:  "OTEL_EXPORTER_OTLP_TIMEOUT",
			envValue: "15000",
		},
		{
			name:     "compression",
			envName:  "OTEL_EXPORTER_OTLP_TRACES_COMPRESSION",
			envValue: "gzip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restoreGlobalTelemetry(t)
			t.Setenv(tt.envName, tt.envValue)

			err := setupTracingForEnvPolicyTest(t, TraceExporterConfig{
				OTLPEndpoint: typedCollector.URL,
				OTLPProtocol: "http/protobuf",
			})
			requireAmbientExporterEnvError(t, err, "unsupported ambient otel exporter environment")
			requireErrorDoesNotContain(t, err, tt.envValue)
			for _, forbidden := range tt.forbidden {
				requireErrorDoesNotContain(t, err, forbidden)
			}
		})
	}
}

func TestSetupTracingDoesNotEnableExporterFromAmbientOTLPEndpointEnv(t *testing.T) {
	restoreGlobalTelemetry(t)

	requests := make(chan string, 1)
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(collector.Close)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", collector.URL)

	shutdown, err := SetupTracing(context.Background(), envPolicyTracingConfig(TraceExporterConfig{
		OTLPProtocol: "http/protobuf",
	}))
	if err != nil {
		t.Fatalf("SetupTracing() error = %v", err)
	}
	t.Cleanup(func() {
		if err := shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown tracing: %v", err)
		}
	})

	provider, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	if !ok {
		t.Fatalf("global tracer provider = %T, want *sdktrace.TracerProvider", otel.GetTracerProvider())
	}
	_, span := provider.Tracer("telemetry-test").Start(context.Background(), "ambient-env-disabled")
	span.End()
	if err := provider.ForceFlush(context.Background()); err != nil {
		t.Fatalf("force flush trace provider: %v", err)
	}
	assertNoCollectorRequest(t, requests, "ambient endpoint collector")
}

func TestSetupTracingRejectsAmbientOTLPProxyEnv(t *testing.T) {
	tests := []struct {
		name     string
		envName  string
		endpoint string
	}{
		{
			name:     "http proxy",
			envName:  "HTTP_PROXY",
			endpoint: "http://127.0.0.1:4318",
		},
		{
			name:     "https proxy",
			envName:  "HTTPS_PROXY",
			endpoint: "https://otel.example.com:4318",
		},
		{
			name:     "no proxy",
			envName:  "NO_PROXY",
			endpoint: "http://127.0.0.1:4318",
		},
		{
			name:     "lowercase http proxy",
			envName:  "http_proxy",
			endpoint: "http://127.0.0.1:4318",
		},
		{
			name:     "lowercase https proxy",
			envName:  "https_proxy",
			endpoint: "https://otel.example.com:4318",
		},
		{
			name:     "lowercase no proxy",
			envName:  "no_proxy",
			endpoint: "http://127.0.0.1:4318",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restoreGlobalTelemetry(t)
			t.Setenv(tt.envName, "http://proxy.example.com:8080")

			err := setupTracingForEnvPolicyTest(t, TraceExporterConfig{
				OTLPEndpoint: tt.endpoint,
				OTLPProtocol: "http/protobuf",
			})
			requireAmbientExporterEnvError(t, err, "unsupported ambient otlp proxy environment")
			requireErrorDoesNotContain(t, err, "proxy.example.com")
		})
	}
}

//nolint:tparallel // Top-level t.Parallel would make the t.Setenv subtest invalid.
