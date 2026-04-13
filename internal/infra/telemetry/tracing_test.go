package telemetry

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
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
		i := i
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

	t.Run("headers without endpoint do not configure sdk default exporter", func(t *testing.T) {
		t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://env-collector.example:4318")

		options, configured, err := buildTraceExporterOptions(TraceExporterConfig{
			OTLPHeaders:  "authorization=Bearer token",
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

	t.Run("generic endpoint base path appends traces path", func(t *testing.T) {
		paths := make(chan string, 1)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			paths <- r.URL.Path
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(server.Close)

		options, configured, err := buildTraceExporterOptions(TraceExporterConfig{
			OTLPEndpoint: server.URL + "/collector",
			OTLPProtocol: "http/protobuf",
		})
		if err != nil {
			t.Fatalf("buildTraceExporterOptions() error = %v", err)
		}
		if !configured {
			t.Fatalf("configured = false, want true")
		}

		exportOneTestSpan(t, options)
		assertCollectorPath(t, paths, "/collector/v1/traces")
	})

	t.Run("traces endpoint overrides generic endpoint and uses path as configured", func(t *testing.T) {
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
			OTLPTracesEndpoint: tracesServer.URL + "/custom/traces",
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
		assertCollectorPath(t, tracesPaths, "/custom/traces")
	})

	t.Run("traces endpoint without path uses root path", func(t *testing.T) {
		paths := make(chan string, 1)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			paths <- r.URL.Path
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(server.Close)

		options, configured, err := buildTraceExporterOptions(TraceExporterConfig{
			OTLPTracesEndpoint: server.URL,
			OTLPProtocol:       "http/protobuf",
		})
		if err != nil {
			t.Fatalf("buildTraceExporterOptions() error = %v", err)
		}
		if !configured {
			t.Fatalf("configured = false, want true")
		}

		exportOneTestSpan(t, options)
		assertCollectorPath(t, paths, "/")
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

func TestDescribeTraceExporterTarget(t *testing.T) {
	testCases := []struct {
		name       string
		cfg        TraceExporterConfig
		wantConfig bool
		wantTarget string
		wantScheme string
		wantErr    string
	}{
		{
			name: "not configured",
		},
		{
			name: "generic endpoint",
			cfg: TraceExporterConfig{
				OTLPEndpoint: "https://otel.example.com:4318",
			},
			wantConfig: true,
			wantTarget: "otel.example.com:4318",
			wantScheme: "https",
		},
		{
			name: "traces endpoint takes precedence",
			cfg: TraceExporterConfig{
				OTLPEndpoint:       "https://generic.example.com:4318",
				OTLPTracesEndpoint: "http://traces.example.com:4318/v1/traces",
			},
			wantConfig: true,
			wantTarget: "traces.example.com:4318",
			wantScheme: "http",
		},
		{
			name: "scheme-less endpoint is http",
			cfg: TraceExporterConfig{
				OTLPEndpoint: "otel.internal:4318",
			},
			wantConfig: true,
			wantTarget: "otel.internal:4318",
			wantScheme: "http",
		},
		{
			name: "headers without endpoint are not a target",
			cfg: TraceExporterConfig{
				OTLPHeaders: "authorization=Bearer token",
			},
		},
		{
			name: "invalid endpoint",
			cfg: TraceExporterConfig{
				OTLPEndpoint: "ftp://otel.example.com:4318",
			},
			wantErr: "unsupported scheme",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			target, err := DescribeTraceExporterTarget(tc.cfg)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("DescribeTraceExporterTarget() error = nil, want non-nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("DescribeTraceExporterTarget() error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("DescribeTraceExporterTarget() error = %v", err)
			}
			if target.Configured != tc.wantConfig {
				t.Fatalf("Configured = %v, want %v", target.Configured, tc.wantConfig)
			}
			if target.Target != tc.wantTarget {
				t.Fatalf("Target = %q, want %q", target.Target, tc.wantTarget)
			}
			if target.Scheme != tc.wantScheme {
				t.Fatalf("Scheme = %q, want %q", target.Scheme, tc.wantScheme)
			}
		})
	}
}

func TestTraceOTLPEndpointRedactsInvalidAndSecretBearingEndpoints(t *testing.T) {
	testCases := []struct {
		name    string
		raw     string
		wantErr string
	}{
		{
			name:    "invalid url",
			raw:     "https://%zz:4318/v1/traces",
			wantErr: "invalid endpoint",
		},
		{
			name:    "unsupported scheme",
			raw:     "ftp://otel.example.com:4318/v1/traces",
			wantErr: "unsupported scheme",
		},
		{
			name:    "userinfo",
			raw:     "https://user:secret-value@otel.example.com:4318/v1/traces",
			wantErr: "userinfo is not supported",
		},
		{
			name:    "query",
			raw:     "https://otel.example.com:4318/v1/traces?authorization=Bearer+secret-value",
			wantErr: "query is not supported",
		},
		{
			name:    "fragment",
			raw:     "https://otel.example.com:4318/v1/traces#secret-value",
			wantErr: "fragment is not supported",
		},
		{
			name:    "empty host",
			raw:     "https:///v1/traces",
			wantErr: "empty host",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DescribeTraceExporterTarget(TraceExporterConfig{
				OTLPEndpoint: tc.raw,
			})
			if err == nil {
				t.Fatal("DescribeTraceExporterTarget() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("DescribeTraceExporterTarget() error = %v, want %q", err, tc.wantErr)
			}
			for _, leaked := range []string{tc.raw, "secret-value", "Bearer"} {
				if strings.Contains(err.Error(), leaked) {
					t.Fatalf("DescribeTraceExporterTarget() error = %v, leaked %q", err, leaked)
				}
			}
		})
	}
}

func TestBuildTraceExporterOptionsUsesRuntimeEndpointParser(t *testing.T) {
	options, configured, err := buildTraceExporterOptions(TraceExporterConfig{
		OTLPEndpoint: "http://localhost:4318/v1/traces",
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

	options, configured, err = buildTraceExporterOptions(TraceExporterConfig{
		OTLPEndpoint: "localhost:4318",
		OTLPProtocol: "http/protobuf",
	})
	if err != nil {
		t.Fatalf("buildTraceExporterOptions() scheme-less error = %v", err)
	}
	if !configured {
		t.Fatalf("scheme-less configured = false, want true")
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
	schemeLessOptions, configured, err := buildTraceExporterOptions(TraceExporterConfig{
		OTLPEndpoint: strings.TrimPrefix(server.URL, "http://"),
		OTLPProtocol: "http/protobuf",
	})
	if err != nil {
		t.Fatalf("buildTraceExporterOptions() scheme-less test server error = %v", err)
	}
	if !configured {
		t.Fatalf("scheme-less test server configured = false, want true")
	}
	exportOneTestSpan(t, schemeLessOptions)
	assertCollectorPath(t, serverPaths, "/v1/traces")

	_, _, err = buildTraceExporterOptions(TraceExporterConfig{
		OTLPEndpoint: "://bad-endpoint",
		OTLPProtocol: "http/protobuf",
	})
	if err == nil {
		t.Fatalf("buildTraceExporterOptions() error = nil, want non-nil")
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

func setupTracingForEnvPolicyTest(t *testing.T, exporter TraceExporterConfig) error {
	t.Helper()

	shutdown, err := SetupTracing(context.Background(), envPolicyTracingConfig(exporter))
	if err == nil {
		if shutdownErr := shutdown(context.Background()); shutdownErr != nil {
			t.Fatalf("shutdown tracing after unexpected setup success: %v", shutdownErr)
		}
	}
	return err
}

func envPolicyTracingConfig(exporter TraceExporterConfig) TracingConfig {
	return TracingConfig{
		ServiceName:      "test-service",
		ServiceVersion:   "test",
		DeploymentEnv:    "local",
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
