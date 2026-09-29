package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
)

func TestResolveMetricExporterEndpoint(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		cfg                   MetricExporterConfig
		env                   map[string]string
		wantURL               string
		wantSource            string
		wantServiceConfigured bool
	}{
		{
			name: "nothing configured exports nowhere",
		},
		{
			name:                  "own metrics endpoint defaults the signal path",
			cfg:                   MetricExporterConfig{OTLPEndpoint: "https://collector.example"},
			wantURL:               "https://collector.example/v1/metrics",
			wantSource:            MetricExporterConfigKey,
			wantServiceConfigured: true,
		},
		{
			// The point of the shared setting: naming a collector root once is
			// what an operator means, and it must not silently serve one signal.
			name:                  "shared root serves both signals",
			cfg:                   MetricExporterConfig{SharedOTLPEndpoint: "https://collector.example:4318"},
			wantURL:               "https://collector.example:4318/v1/metrics",
			wantSource:            SharedOTLPExporterConfigKey,
			wantServiceConfigured: true,
		},
		{
			// And once it names a path it is a traces endpoint, which says
			// nothing about where metrics go.
			name: "shared traces endpoint does not name a metrics endpoint",
			cfg:  MetricExporterConfig{SharedOTLPEndpoint: "https://collector.example/v1/traces"},
		},
		{
			name:       "platform metrics variable is honored",
			env:        map[string]string{otelExporterMetricsEndpointEnv: "https://platform.example/v1/metrics"},
			wantURL:    "https://platform.example/v1/metrics",
			wantSource: otelExporterMetricsEndpointEnv,
		},
		{
			// The deployment shape the whole change exists for: the platform
			// injects one root, and both signals have to follow it.
			name:       "platform root resolves the metrics path",
			env:        map[string]string{otelExporterEndpointEnv: "https://platform.example:4318"},
			wantURL:    "https://platform.example:4318/v1/metrics",
			wantSource: otelExporterEndpointEnv,
		},
		{
			name:       "platform root keeps a sub-path prefix",
			env:        map[string]string{otelExporterEndpointEnv: "https://platform.example/otlp"},
			wantURL:    "https://platform.example/otlp/v1/metrics",
			wantSource: otelExporterEndpointEnv,
		},
		{
			// Configured headers are a credential, so they pin the destination
			// rather than travelling to an endpoint this service never named.
			name: "configured headers refuse an ambient endpoint",
			cfg:  MetricExporterConfig{OTLPHeaders: "authorization=Bearer token"},
			env:  map[string]string{otelExporterEndpointEnv: "https://platform.example:4318"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			telemetrytest.ClearAmbientExporterEnv(t)
			for name, value := range tc.env {
				t.Setenv(name, value)
			}

			endpoint, err := resolveMetricExporterEndpoint(tc.cfg)
			if err != nil {
				t.Fatalf("resolveMetricExporterEndpoint() error = %v", err)
			}
			if endpoint.URL != tc.wantURL {
				t.Fatalf("URL = %q, want %q", endpoint.URL, tc.wantURL)
			}
			if endpoint.Source != tc.wantSource {
				t.Fatalf("Source = %q, want %q", endpoint.Source, tc.wantSource)
			}
			if endpoint.Configured() != (tc.wantURL != "") {
				t.Fatalf("Configured() = %v for URL %q", endpoint.Configured(), endpoint.URL)
			}
			if endpoint.ConfiguredByService != tc.wantServiceConfigured {
				t.Fatalf("ConfiguredByService = %v, want %v", endpoint.ConfiguredByService, tc.wantServiceConfigured)
			}
		})
	}
}

// TestSetupMetricsPushesToOTLPCollector verifies that the configured collector
// receives the recorded metric over OTLP.
//
//nolint:paralleltest // Mutates the process-wide OpenTelemetry MeterProvider.
func TestSetupMetricsPushesToOTLPCollector(t *testing.T) {
	telemetrytest.ClearAmbientExporterEnv(t)
	telemetrytest.RestoreGlobals(t)

	var exports atomic.Int64
	var sawHeader atomic.Bool
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/metrics" {
			exports.Add(1)
		}
		if r.Header.Get("X-Collector-Token") == "shared-secret" {
			sawHeader.Store(true)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(collector.Close)

	metrics := New()
	result, err := SetupMetrics(context.Background(), metrics, MetricsConfig{
		Resource: ResourceConfig{
			ServiceName:    "push-service",
			ServiceVersion: "test-version",
			DeploymentEnv:  "test-env",
		},
		Exporter: MetricExporterConfig{
			// A bare root, so this also proves the shared setting reaches metrics.
			SharedOTLPEndpoint: collector.URL,
			OTLPHeaders:        "x-collector-token=shared-secret",
		},
	})
	if err != nil {
		t.Fatalf("SetupMetrics() error = %v", err)
	}
	if !result.PushInitialized() {
		t.Fatalf("SetupMetrics() did not initialize push for a configured collector root: %+v", result)
	}
	if result.Endpoint.URL != collector.URL+"/v1/metrics" {
		t.Fatalf("endpoint URL = %q, want %q", result.Endpoint.URL, collector.URL+"/v1/metrics")
	}

	counter, err := metrics.MeterProvider().Meter("metrics-export-test").Int64Counter("template.pushed")
	if err != nil {
		t.Fatalf("create counter: %v", err)
	}
	counter.Add(context.Background(), 1)

	// Shutdown flushes the periodic reader, so the assertion does not wait out an
	// export interval.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := result.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown metrics: %v", err)
	}

	if exports.Load() == 0 {
		t.Fatal("collector received no OTLP metric export")
	}
	if !sawHeader.Load() {
		t.Fatal("OTLP export did not carry the configured collector credential")
	}
}

//nolint:paralleltest // Mutates process-wide exporter environment and meter provider.
func TestSetupMetricsSharedRootRejectsAmbientCredentials(t *testing.T) {
	telemetrytest.ClearAmbientExporterEnv(t)
	telemetrytest.RestoreGlobals(t)
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_HEADERS", "authorization=Bearer injected")

	result, err := SetupMetrics(t.Context(), New(), MetricsConfig{
		Resource: ResourceConfig{ServiceName: "metrics-shared-root-test"},
		Exporter: MetricExporterConfig{SharedOTLPEndpoint: "https://collector.example"},
	})
	if err != nil {
		t.Fatalf("SetupMetrics() error = %v, want scrape-only degradation", err)
	}
	t.Cleanup(func() {
		if shutdownErr := result.Shutdown(context.Background()); shutdownErr != nil {
			t.Errorf("shutdown metrics: %v", shutdownErr)
		}
	})
	if result.Endpoint.Source != SharedOTLPExporterConfigKey {
		t.Fatalf("endpoint source = %q, want shared config key %q", result.Endpoint.Source, SharedOTLPExporterConfigKey)
	}
	if result.ExportErr == nil || result.PushInitialized() {
		t.Fatalf("metrics result = %+v, want ambient credential rejection and scrape-only provider", result)
	}
}
