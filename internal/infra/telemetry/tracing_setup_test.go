package //nolint:paralleltest // This test mutates process-global environment or working directory.

// TestSetupTracingMergesAmbientResourceUnderConfig pins both halves of the merge.
// Configured attributes win, so a platform cannot silently rename this service in
// every dashboard. Attributes it does not set survive, which is how k8s.pod.name
// and container.id reach the exported resource — the version this replaced unset
// OTEL_RESOURCE_ATTRIBUTES around provider construction and discarded them, while
// protecting nothing, because resource.Merge already gives the local resource
// precedence.
telemetry

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestSetupTracingMergesAmbientResourceUnderConfig(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.name=env-service,service.version=env-version,deployment.environment.name=env,env.only=true")
	t.Setenv("OTEL_SERVICE_NAME", "env-service-name")

	telemetrytest.RestoreGlobals(t)

	_, shutdown, err := SetupTracing(context.Background(), TracingConfig{
		Resource: ResourceConfig{
			ServiceName:       " config-service ",
			ServiceVersion:    " config-version ",
			ServiceInstanceID: " config-instance ",
			DeploymentEnv:     " config-env ",
		},
		TracesSampler:    "always_on",
		TracesSamplerArg: 0.1,
		Exporter:         testTraceExporter(t),
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
		"service.instance.id":         "config-instance",
		"deployment.environment.name": "config-env",
	} {
		got, ok := attrs[key]
		if !ok || got != want {
			t.Fatalf("resource attribute %q = %q (present %v), want %q; attrs=%v", key, got, ok, want, attrs)
		}
	}
	if got, ok := attrs["env.only"]; !ok || got != "true" {
		t.Fatalf("ambient resource attribute env.only = %q (present %v), want it merged; attrs=%v", got, ok, attrs)
	}
}

func TestSetupTracingWithoutExporterDoesNotRecord(t *testing.T) {
	// "Without exporter" now includes the ambient endpoint variables, so this
	// test states that condition rather than inheriting the machine's.
	telemetrytest.ClearAmbientExporterEnv(t)
	telemetrytest.RestoreGlobals(t)

	_, shutdown, err := SetupTracing(t.Context(), TracingConfig{
		Resource: ResourceConfig{
			ServiceName:    "test-service",
			ServiceVersion: "test",
			DeploymentEnv:  "test",
		},
		TracesSampler:    "always_on",
		TracesSamplerArg: 1,
	})
	if err != nil {
		t.Fatalf("SetupTracing() error = %v", err)
	}
	t.Cleanup(func() {
		if err := shutdown(context.Background()); err != nil {
			t.Errorf("shutdown tracing: %v", err)
		}
	})

	_, span := otel.Tracer("telemetry-test").Start(t.Context(), "no-exporter")
	defer span.End()
	if span.IsRecording() {
		t.Fatal("span.IsRecording() = true without an exporter")
	}
	if !span.SpanContext().IsValid() {
		t.Fatal("span context is invalid, want trace correlation preserved")
	}
}

func BenchmarkTracingWithoutExporter(b *testing.B) {
	telemetrytest.ClearAmbientExporterEnv(b)
	previousTracerProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()
	b.Cleanup(func() {
		otel.SetTracerProvider(previousTracerProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	_, shutdown, err := SetupTracing(b.Context(), TracingConfig{
		Resource: ResourceConfig{
			ServiceName:    "benchmark-service",
			ServiceVersion: "benchmark",
			DeploymentEnv:  "benchmark",
		},
		TracesSampler:    "parentbased_traceidratio",
		TracesSamplerArg: 0.10,
	})
	if err != nil {
		b.Fatalf("SetupTracing() error = %v", err)
	}
	b.Cleanup(func() {
		if err := shutdown(context.Background()); err != nil {
			b.Errorf("shutdown tracing: %v", err)
		}
	})

	ctx := context.Background()
	tracer := otel.Tracer("telemetry-benchmark")

	b.ReportAllocs()
	for b.Loop() {
		_, span := tracer.Start(ctx, "request")
		span.End()
	}
}

//nolint:paralleltest // Mutates the process-wide OpenTelemetry provider and propagator.
//nolint:paralleltest // This test mutates process-global environment or working directory.

// TestSetupTracingIsSafeUnderConcurrentSetup keeps the provider installation
// serialized. It used to also assert that the ambient resource variables were
// restored after each call, because setup unset them around provider construction;
// that suppression is gone, so what remains to prove is that concurrent callers do
// not corrupt the globals under the race detector.
func TestSetupTracingDoesNotApplyResourceIdentityFallbacks(t *testing.T) {
	telemetrytest.RestoreGlobals(t)

	_, shutdown, err := SetupTracing(context.Background(), TracingConfig{
		TracesSampler:    "always_on",
		TracesSamplerArg: 0.1,
		Exporter:         testTraceExporter(t),
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

func testTraceExporter(t *testing.T) TraceExporterConfig {
	t.Helper()
	telemetrytest.ClearAmbientExporterEnv(t)

	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(collector.Close)
	return TraceExporterConfig{OTLPEndpoint: collector.URL}
}

func TestSetupTracingIsSafeUnderConcurrentSetup(t *testing.T) {
	const (
		resourceAttrs = "service.name=env-service,service.version=env-version,deployment.environment.name=env,env.only=true"
		serviceName   = "env-service-name"
		setupCount    = 16
	)
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", resourceAttrs)
	t.Setenv("OTEL_SERVICE_NAME", serviceName)

	telemetrytest.ClearAmbientExporterEnv(t)
	telemetrytest.RestoreGlobals(t)

	shutdowns := make(chan func(context.Context) error, setupCount)
	errs := make(chan error, setupCount)
	var wg sync.WaitGroup
	for i := range setupCount {
		wg.Go(func() {
			_, shutdown, err := SetupTracing(context.Background(), TracingConfig{
				Resource: ResourceConfig{
					ServiceName:    fmt.Sprintf("config-service-%d", i),
					ServiceVersion: "config-version",
					DeploymentEnv:  "config-env",
				},
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
}

func TestAmbientOTLPExporterEnvReportsNamesOnly(t *testing.T) {
	telemetrytest.ClearAmbientExporterEnv(t)
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "authorization=Bearer secret-value")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector.example:4318")
	t.Setenv("OTEL_EXPORTER_OTLP_TIMEOUT", "   ")

	got := AmbientOTLPExporterEnv()

	want := []string{"OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_HEADERS"}
	if !slices.Equal(got, want) {
		t.Fatalf("AmbientOTLPExporterEnv() = %v, want %v", got, want)
	}
}

func TestAmbientOTLPExporterEnvEmptyWithoutAmbientEnv(t *testing.T) {
	telemetrytest.ClearAmbientExporterEnv(t)

	if got := AmbientOTLPExporterEnv(); len(got) != 0 {
		t.Fatalf("AmbientOTLPExporterEnv() = %v, want empty", got)
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

	// Credential and trust material is not covered by any option this service
	// sets, so an injected value would reach the collector unverified. These
	// must fail exporter setup rather than be silently accepted.
	//nolint:paralleltest // This test mutates process-global environment or working directory.

	// TestSetupTracingIgnoresOverriddenAmbientOTLPExporterEnv locks in the reason
	// these variables are safe to ignore: otlptracehttp applies ambient environment
	// before explicit options, so WithEndpointURL wins for the endpoint, the URL
	// path, and the TLS scheme. A platform collector injects exactly these, and
	// treating them as conflicts would disable this service's own trace export on
	// every such deployment.
	//nolint:paralleltest // This test mutates process-global environment or working directory.

	// TestSetupTracingExportsToAmbientOTLPEndpointEnv covers the deployment this
	// service is most often put into: a platform injects the standard OpenTelemetry
	// endpoint variable and expects traces. Ignoring it would leave the service
	// reporting healthy, answering every request, and exporting nothing.
	//nolint:paralleltest // This test mutates process-global environment or working directory.

	// TestSetupTracingPrefersConfiguredEndpointOverAmbientEnv keeps this service's
	// own setting authoritative: the ambient variable is a fallback, not an
	// override, so a platform cannot redirect a service that named its collector.
	//nolint:paralleltest // This test mutates process-global environment or working directory.

	// TestSetupTracingDoesNotSendConfiguredHeadersToAmbientEndpoint is the one case
	// the ambient fallback must refuse: configured headers are this service's own
	// credential, and an ambient endpoint is a destination it never named.
	//nolint:paralleltest // This test mutates process-global environment or working directory.

	// TestSetupTracingAcceptsAmbientCredentialsForAnAmbientEndpoint records why the
	// conflict rule is scoped to a configured endpoint: when the platform supplies
	// the collector, its credentials belong to that collector, and rejecting them
	// would refuse the ordinary injected-collector deployment.
	//nolint:paralleltest // This test mutates process-global environment or working directory.

	// TestResolveTraceExporterEndpointRejectsInvalidAmbientEndpoint keeps ambient
	// values under the same fail-closed validation as configured ones, and keeps the
	// raw value out of the error because it can carry a credential.
	tests := []struct {
		name      string
		envName   string
		envValue  string
		forbidden []string
	}{
		{
			name:      "header injection",
			envName:   "OTEL_EXPORTER_OTLP_HEADERS",
			envValue:  "authorization=Bearer secret-value",
			forbidden: []string{"Bearer", "secret-value"},
		},
		{
			name:      "trace header injection",
			envName:   "OTEL_EXPORTER_OTLP_TRACES_HEADERS",
			envValue:  "authorization=Bearer secret-value",
			forbidden: []string{"Bearer", "secret-value"},
		},
		{
			name:      "certificate path",
			envName:   "OTEL_EXPORTER_OTLP_CERTIFICATE",
			envValue:  "/tmp/secret-ca.pem",
			forbidden: []string{"/tmp/secret-ca.pem", "secret-ca"},
		},
		{
			name:      "trace certificate path",
			envName:   "OTEL_EXPORTER_OTLP_TRACES_CERTIFICATE",
			envValue:  "/tmp/secret-ca.pem",
			forbidden: []string{"/tmp/secret-ca.pem", "secret-ca"},
		},
		{
			name:      "client certificate path",
			envName:   "OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE",
			envValue:  "/tmp/secret-client.pem",
			forbidden: []string{"/tmp/secret-client.pem", "secret-client"},
		},
		{
			name:      "client key path",
			envName:   "OTEL_EXPORTER_OTLP_CLIENT_KEY",
			envValue:  "/tmp/secret-client.key",
			forbidden: []string{"/tmp/secret-client.key", "secret-client"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			telemetrytest.ClearAmbientExporterEnv(t)
			telemetrytest.RestoreGlobals(t)
			t.Setenv(tt.envName, tt.envValue)

			err := setupTracingForEnvPolicyTest(t, TraceExporterConfig{
				OTLPEndpoint: typedCollector.URL,
			})
			requireAmbientExporterEnvError(t, err, "unsupported ambient otel exporter environment")
			requireErrorDoesNotContain(t, err, tt.envValue)
			for _, forbidden := range tt.forbidden {
				requireErrorDoesNotContain(t, err, forbidden)
			}
		})
	}
}

func TestSetupTracingIgnoresOverriddenAmbientOTLPExporterEnv(t *testing.T) {
	typedCollector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(typedCollector.Close)

	envCollector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(envCollector.Close)

	tests := []struct {
		name     string
		envName  string
		envValue string
	}{
		{
			name:     "generic endpoint is overridden by the configured endpoint",
			envName:  "OTEL_EXPORTER_OTLP_ENDPOINT",
			envValue: envCollector.URL + "/env",
		},
		{
			name:     "trace endpoint is overridden by the configured endpoint",
			envName:  "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
			envValue: envCollector.URL + "/trace-env",
		},
		{
			name:     "insecure is overridden by the configured endpoint scheme",
			envName:  "OTEL_EXPORTER_OTLP_INSECURE",
			envValue: "true",
		},
		{
			name:     "protocol carries no destination or credential",
			envName:  "OTEL_EXPORTER_OTLP_PROTOCOL",
			envValue: "http/protobuf",
		},
		{
			name:     "timeout carries no destination or credential",
			envName:  "OTEL_EXPORTER_OTLP_TIMEOUT",
			envValue: "15000",
		},
		{
			name:     "compression carries no destination or credential",
			envName:  "OTEL_EXPORTER_OTLP_TRACES_COMPRESSION",
			envValue: "gzip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			telemetrytest.ClearAmbientExporterEnv(t)
			telemetrytest.RestoreGlobals(t)
			t.Setenv(tt.envName, tt.envValue)

			if err := setupTracingForEnvPolicyTest(t, TraceExporterConfig{
				OTLPEndpoint: typedCollector.URL,
			}); err != nil {
				t.Fatalf("SetupTracing() error = %v, want tracing to survive an ignorable ambient variable", err)
			}
		})
	}
}

func TestConflictingTraceExporterEnvReportsCredentialAndTrustNamesOnly(t *testing.T) {
	telemetrytest.ClearAmbientExporterEnv(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector.example:4318")
	t.Setenv("OTEL_EXPORTER_OTLP_TIMEOUT", "15000")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "authorization=Bearer secret-value")
	t.Setenv("OTEL_EXPORTER_OTLP_CLIENT_KEY", "/tmp/secret-client.key")

	got := ConflictingTraceExporterEnv()

	want := []string{"OTEL_EXPORTER_OTLP_CLIENT_KEY", "OTEL_EXPORTER_OTLP_HEADERS"}
	if !slices.Equal(got, want) {
		t.Fatalf("ConflictingTraceExporterEnv() = %v, want %v", got, want)
	}
}

func TestSetupTracingExportsToAmbientOTLPEndpointEnv(t *testing.T) {
	tests := []struct {
		name      string
		envName   string
		envSuffix string
		wantPath  string
	}{
		{
			name:     "signal-agnostic root gains the traces path",
			envName:  "OTEL_EXPORTER_OTLP_ENDPOINT",
			wantPath: "/v1/traces",
		},
		{
			name:      "signal-agnostic root keeps its prefix",
			envName:   "OTEL_EXPORTER_OTLP_ENDPOINT",
			envSuffix: "/otlp",
			wantPath:  "/otlp/v1/traces",
		},
		{
			name:      "signal-specific endpoint is used exactly",
			envName:   "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
			envSuffix: "/collect/traces",
			wantPath:  "/collect/traces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			telemetrytest.ClearAmbientExporterEnv(t)
			telemetrytest.RestoreGlobals(t)

			requests := make(chan string, 1)
			collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests <- r.URL.Path
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(collector.Close)
			t.Setenv(tt.envName, collector.URL+tt.envSuffix)

			endpoint, shutdown := setupRecordingTracing(t, TraceExporterConfig{})
			if endpoint.Source != tt.envName {
				t.Fatalf("endpoint source = %q, want %q", endpoint.Source, tt.envName)
			}

			exportOneGlobalSpan(t, "ambient-env-export")
			if err := shutdown(context.Background()); err != nil {
				t.Fatalf("shutdown tracing: %v", err)
			}
			assertCollectorPath(t, requests, tt.wantPath)
		})
	}
}

func TestSetupTracingPrefersConfiguredEndpointOverAmbientEnv(t *testing.T) {
	telemetrytest.ClearAmbientExporterEnv(t)
	telemetrytest.RestoreGlobals(t)

	configured := make(chan string, 1)
	configuredCollector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		configured <- r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(configuredCollector.Close)

	ambient := make(chan string, 1)
	ambientCollector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ambient <- r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(ambientCollector.Close)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", ambientCollector.URL)

	endpoint, shutdown := setupRecordingTracing(t, TraceExporterConfig{OTLPEndpoint: configuredCollector.URL})
	if endpoint.Source != TraceExporterConfigKey {
		t.Fatalf("endpoint source = %q, want %q", endpoint.Source, TraceExporterConfigKey)
	}

	exportOneGlobalSpan(t, "configured-endpoint-wins")
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown tracing: %v", err)
	}
	assertCollectorPath(t, configured, "/v1/traces")
	assertNoCollectorRequest(t, ambient, "ambient endpoint collector")
}

func TestSetupTracingDoesNotSendConfiguredHeadersToAmbientEndpoint(t *testing.T) {
	telemetrytest.ClearAmbientExporterEnv(t)
	telemetrytest.RestoreGlobals(t)

	requests := make(chan string, 1)
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(collector.Close)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", collector.URL)

	endpoint, shutdown := setupRecordingTracing(t, TraceExporterConfig{
		OTLPHeaders: "authorization=Bearer secret-value",
	})
	if endpoint.Configured() {
		t.Fatalf("endpoint = %+v, want no exporter when headers pin an unnamed destination", endpoint)
	}

	exportOneGlobalSpan(t, "headers-pin-destination")
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown tracing: %v", err)
	}
	assertNoCollectorRequest(t, requests, "ambient endpoint collector")
}

func TestSetupTracingAcceptsAmbientCredentialsForAnAmbientEndpoint(t *testing.T) {
	telemetrytest.ClearAmbientExporterEnv(t)
	telemetrytest.RestoreGlobals(t)

	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(collector.Close)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", collector.URL)
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "authorization=Bearer platform-token")

	endpoint, shutdown := setupRecordingTracing(t, TraceExporterConfig{})
	if !endpoint.Configured() {
		t.Fatal("endpoint is not configured, want the platform-supplied collector")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown tracing: %v", err)
	}
}

func TestResolveTraceExporterEndpointRejectsInvalidAmbientEndpoint(t *testing.T) {
	telemetrytest.ClearAmbientExporterEnv(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://user:secret-value@collector.example:4318")

	_, err := resolveTraceExporterEndpoint(TraceExporterConfig{})
	if err == nil {
		t.Fatal("resolveTraceExporterEndpoint() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "OTEL_EXPORTER_OTLP_ENDPOINT") {
		t.Fatalf("error = %v, want the variable name", err)
	}
	if strings.Contains(err.Error(), "secret-value") {
		t.Fatalf("error = %v, leaked the raw value", err)
	}
}

func setupRecordingTracing(t *testing.T, exporter TraceExporterConfig) (TraceExporterEndpoint, func(context.Context) error) {
	t.Helper()

	endpoint, shutdown, err := SetupTracing(context.Background(), TracingConfig{
		Resource: ResourceConfig{
			ServiceName:    "test-service",
			ServiceVersion: "test",
			DeploymentEnv:  "local",
		},
		TracesSampler:    "always_on",
		TracesSamplerArg: 1,
		Exporter:         exporter,
	})
	if err != nil {
		t.Fatalf("SetupTracing() error = %v", err)
	}
	return endpoint, shutdown
}

func exportOneGlobalSpan(t *testing.T, name string) {
	t.Helper()

	provider, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	if !ok {
		t.Fatalf("global tracer provider = %T, want *sdktrace.TracerProvider", otel.GetTracerProvider())
	}
	_, span := provider.Tracer("telemetry-test").Start(context.Background(), name)
	span.End()
	if err := provider.ForceFlush(context.Background()); err != nil {
		t.Fatalf("force flush trace provider: %v", err)
	}
}
