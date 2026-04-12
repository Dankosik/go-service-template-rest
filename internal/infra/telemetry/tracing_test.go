package telemetry

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestBuildTraceSampler(t *testing.T) {
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
			_, err := buildTraceSampler(tc.samplerName, tc.samplerArg)
			if (err != nil) != tc.wantErr {
				t.Fatalf("buildTraceSampler() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

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

func TestClampRatio(t *testing.T) {
	testCases := []struct {
		name    string
		input   float64
		wantOut float64
	}{
		{
			name:    "below zero",
			input:   -1,
			wantOut: 0,
		},
		{
			name:    "above one",
			input:   2,
			wantOut: 1,
		},
		{
			name:    "valid range",
			input:   0.42,
			wantOut: 0.42,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := clampRatio(tc.input)
			if got != tc.wantOut {
				t.Fatalf("clampRatio(%v) = %v, want %v", tc.input, got, tc.wantOut)
			}
		})
	}
}

func TestBuildTraceExporterOptions(t *testing.T) {
	t.Run("not configured", func(t *testing.T) {
		options, configured, err := buildTraceExporterOptions(TraceExporterConfig{
			OTLPProtocol: "http/protobuf",
		})
		if err != nil {
			t.Fatalf("buildTraceExporterOptions() error = %v", err)
		}
		if configured {
			t.Fatalf("configured = true, want false")
		}
		if len(options) != 0 {
			t.Fatalf("options len = %d, want 0", len(options))
		}
	})

	t.Run("configured endpoint and headers", func(t *testing.T) {
		options, configured, err := buildTraceExporterOptions(TraceExporterConfig{
			OTLPEndpoint: "https://otel.example.com:4318",
			OTLPHeaders:  "authorization=Bearer token",
			OTLPProtocol: "http/protobuf",
		})
		if err != nil {
			t.Fatalf("buildTraceExporterOptions() error = %v", err)
		}
		if !configured {
			t.Fatalf("configured = false, want true")
		}
		if len(options) == 0 {
			t.Fatalf("options len = 0, want > 0")
		}
	})

	t.Run("traces endpoint overrides generic endpoint path", func(t *testing.T) {
		genericPaths := make(chan string, 1)
		genericServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			genericPaths <- r.URL.Path
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(genericServer.Close)

		tracesPaths := make(chan string, 1)
		tracesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tracesPaths <- r.URL.Path
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(tracesServer.Close)

		options, configured, err := buildTraceExporterOptions(TraceExporterConfig{
			OTLPEndpoint:       genericServer.URL + "/generic",
			OTLPTracesEndpoint: tracesServer.URL,
			OTLPProtocol:       "http/protobuf",
		})
		if err != nil {
			t.Fatalf("buildTraceExporterOptions() error = %v", err)
		}
		if !configured {
			t.Fatalf("configured = false, want true")
		}

		exportOneTestSpan(t, options)

		assertNoCollectorRequest(t, genericPaths, "generic endpoint")
		assertCollectorPath(t, tracesPaths, "/v1/traces")
	})

	t.Run("invalid protocol", func(t *testing.T) {
		_, _, err := buildTraceExporterOptions(TraceExporterConfig{
			OTLPEndpoint: "https://otel.example.com:4318",
			OTLPProtocol: "grpc",
		})
		if err == nil {
			t.Fatalf("buildTraceExporterOptions() error = nil, want non-nil")
		}
	})
}

func TestParseOTLPEndpointOptions(t *testing.T) {
	options, err := parseOTLPEndpointOptions("http://localhost:4318/v1/traces")
	if err != nil {
		t.Fatalf("parseOTLPEndpointOptions() error = %v", err)
	}
	if len(options) == 0 {
		t.Fatalf("options len = 0, want > 0")
	}

	options, err = parseOTLPEndpointOptions("localhost:4318")
	if err != nil {
		t.Fatalf("parseOTLPEndpointOptions() scheme-less error = %v", err)
	}
	if len(options) == 0 {
		t.Fatalf("scheme-less options len = 0, want > 0")
	}
	serverPaths := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverPaths <- r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	schemeLessOptions, err := parseOTLPEndpointOptions(strings.TrimPrefix(server.URL, "http://"))
	if err != nil {
		t.Fatalf("parseOTLPEndpointOptions() scheme-less test server error = %v", err)
	}
	exportOneTestSpan(t, schemeLessOptions)
	assertCollectorPath(t, serverPaths, "/v1/traces")

	_, err = parseOTLPEndpointOptions("://bad-endpoint")
	if err == nil {
		t.Fatalf("parseOTLPEndpointOptions() error = nil, want non-nil")
	}
}

func TestParseOTLPHeaders(t *testing.T) {
	headers, err := parseOTLPHeaders("authorization=Bearer token,x-api-key=abc")
	if err != nil {
		t.Fatalf("parseOTLPHeaders() error = %v", err)
	}
	if headers["authorization"] != "Bearer token" {
		t.Fatalf("headers[authorization] = %q, want %q", headers["authorization"], "Bearer token")
	}
	if headers["x-api-key"] != "abc" {
		t.Fatalf("headers[x-api-key] = %q, want %q", headers["x-api-key"], "abc")
	}

	_, err = parseOTLPHeaders("malformed")
	if err == nil {
		t.Fatalf("parseOTLPHeaders() error = nil, want non-nil")
	}
}

func TestParseOTLPHeadersMalformedEntriesDoNotLeakRawValues(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "authorization token without delimiter",
			raw:  "authorization Bearer secret-value",
		},
		{
			name: "api key without delimiter",
			raw:  "x-api-key secret-value",
		},
		{
			name: "empty authorization value after prior secret",
			raw:  "x-api-key=secret-value,authorization=",
		},
		{
			name: "unsafe empty-value key",
			raw:  "secret-value.=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseOTLPHeaders(tt.raw)
			if err == nil {
				t.Fatalf("parseOTLPHeaders() error = nil, want non-nil")
			}
			for _, leaked := range []string{"secret-value", "Bearer"} {
				if strings.Contains(err.Error(), leaked) {
					t.Fatalf("parseOTLPHeaders() error leaks %q: %v", leaked, err)
				}
			}
			if strings.Contains(err.Error(), tt.raw) {
				t.Fatalf("parseOTLPHeaders() error leaks raw entry: %v", err)
			}
			if !strings.Contains(err.Error(), "position") {
				t.Fatalf("parseOTLPHeaders() error = %v, want position context", err)
			}
		})
	}
}

func TestExporterOptionTypeCompatibility(t *testing.T) {
	// Guard against accidental option-type drift when upgrading OTLP exporter package.
	var _ []otlptracehttp.Option
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
		t.Fatalf("collector path was not requested")
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
