package //nolint:paralleltest // This test mutates process-global environment or working directory.

//nolint:paralleltest // Mutates the process-wide OpenTelemetry MeterProvider.
telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"go.opentelemetry.io/otel/trace"
)

func TestSetupMetricsUsesPrivateRegistryAndConfigResource(t *testing.T) {

	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.name=env-service,env.only=true")
	t.Setenv("OTEL_SERVICE_NAME", "env-service")

	telemetrytest.RestoreGlobals(t)

	metrics := New()
	result, err := SetupMetrics(context.Background(), metrics, MetricsConfig{
		Resource: ResourceConfig{
			ServiceName:       " test-service ",
			ServiceVersion:    " test-version ",
			ServiceInstanceID: " metrics-instance ",
			DeploymentEnv:     " test-env ",
		},
	})
	if err != nil {
		t.Fatalf("SetupMetrics() error = %v", err)
	}
	t.Cleanup(func() {
		if err := result.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown metrics: %v", err)
		}
	})

	meter := metrics.MeterProvider().Meter("metrics-test")
	counter, err := meter.Int64Counter("template.operations")
	if err != nil {
		t.Fatalf("create counter: %v", err)
	}
	counter.Add(context.Background(), 1)
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:     trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		TraceFlags: trace.FlagsSampled,
	})
	counter.Add(trace.ContextWithSpanContext(context.Background(), spanContext), 1)

	metricsText := collectMetricsText(t, metrics)
	for _, pattern := range []string{
		"template_operations_total",
		`service_name="test-service"`,
		`service_version="test-version"`,
		`service_instance_id="metrics-instance"`,
		`deployment_environment_name="test-env"`,
		// Registered on the meter provider rather than on this registry, so the
		// same series also reach a configured OTLP reader. The Prometheus Go
		// collector this replaced reached only the scrape path.
		"go_goroutine_count",
	} {
		if !strings.Contains(metricsText, pattern) {
			t.Fatalf("metrics output does not contain %q\n%s", pattern, metricsText)
		}
	}
	// Configured attributes win the merge with OTEL_RESOURCE_ATTRIBUTES, so the
	// ambient service.name loses — while an attribute this service does not set
	// survives. That is the whole point of no longer unsetting the variable: it is
	// how a platform contributes k8s.pod.name and container.id.
	if strings.Contains(metricsText, "env-service") {
		t.Fatalf("ambient OTEL_SERVICE_NAME overrode the configured service name\n%s", metricsText)
	}
	if !strings.Contains(metricsText, "env_only") {
		t.Fatalf("ambient OTEL_RESOURCE_ATTRIBUTES was discarded instead of merged\n%s", metricsText)
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", nil)
	req.Header.Set("Accept", "application/openmetrics-text; version=1.0.0")
	resp := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(resp, req)
	if got := resp.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/openmetrics-text") {
		t.Fatalf("metrics content type = %q, want OpenMetrics", got)
	}
	for _, label := range []string{
		`trace_id="0102030405060708090a0b0c0d0e0f10"`,
		`span_id="0102030405060708"`,
	} {
		if !strings.Contains(resp.Body.String(), label) {
			t.Fatalf("OpenMetrics output does not contain exemplar %q\n%s", label, resp.Body.String())
		}
	}
}

// A service exporting no traces still answers every request and reports
// healthy, so trace-export state has to leave the boot log and reach a scrape.
//
//nolint:paralleltest // Mutates the process-wide OpenTelemetry MeterProvider.
func TestRecordTraceExporterStateIsScrapable(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name      string
		active    bool
		wantValue string
	}{
		{name: "active", active: true, wantValue: "1"},
		{name: "degraded", active: false, wantValue: "0"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			telemetrytest.RestoreGlobals(t)

			metrics := New()
			result, err := SetupMetrics(context.Background(), metrics, MetricsConfig{
				Resource: ResourceConfig{
					ServiceName:    "test-service",
					ServiceVersion: "test-version",
					DeploymentEnv:  "test-env",
				},
			})
			if err != nil {
				t.Fatalf("SetupMetrics() error = %v", err)
			}
			t.Cleanup(func() {
				if err := result.Shutdown(context.Background()); err != nil {
					t.Fatalf("shutdown metrics: %v", err)
				}
			})

			if err := metrics.RecordTraceExporterState(context.Background(), tt.active); err != nil {
				t.Fatalf("RecordTraceExporterState() error = %v", err)
			}

			gotValue, ok := scrapedSampleValue(
				collectMetricsText(t, metrics),
				"service_startup_trace_exporter_active",
			)
			if !ok {
				t.Fatal("metrics output does not expose service_startup_trace_exporter_active")
			}
			if gotValue != tt.wantValue {
				t.Fatalf("service_startup_trace_exporter_active = %s, want %s", gotValue, tt.wantValue)
			}
		})
	}
}

// scrapedSampleValue returns the value of the first sample line for name,
// skipping the HELP and TYPE metadata lines that share the prefix.
func scrapedSampleValue(metricsText, name string) (string, bool) {
	for line := range strings.Lines(metricsText) {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, name) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		return fields[len(fields)-1], true
	}
	return "", false
}

// A nil or unconfigured Metrics falls back to a no-op provider, so recording
// must not fail startup.
func TestRecordTraceExporterStateToleratesNoopProvider(t *testing.T) {
	t.Parallel()

	for _, metrics := range []*Metrics{nil, {}} {
		if err := metrics.RecordTraceExporterState(context.Background(), true); err != nil {
			t.Fatalf("RecordTraceExporterState() error = %v, want nil for a no-op provider", err)
		}
	}
}

func TestSetupMetricsRequiresRegistry(t *testing.T) {
	t.Parallel()

	for _, metrics := range []*Metrics{nil, {}} {
		if _, err := SetupMetrics(context.Background(), metrics, MetricsConfig{}); err == nil {
			t.Fatal("SetupMetrics() error = nil, want registry error")
		}
	}
}

// TestSetupMetricsDegradesToScrapeOnlyForUnusableEndpoint keeps a bad collector
// address from costing every metric the service has.
//
// Failing as a unit meant no meter provider at all, so the /metrics endpoint
// served only the process collector, otelhttp and otelpgx recorded into a no-op,
// and the gauge that reports degraded telemetry could not be written — the one
// signal an operator needed was removed by the failure it was supposed to report.
//
//nolint:paralleltest // Mutates the process-wide OpenTelemetry MeterProvider.
func TestSetupMetricsDegradesToScrapeOnlyForUnusableEndpoint(t *testing.T) {
	t.Parallel()
	telemetrytest.ClearAmbientExporterEnv(t)
	telemetrytest.RestoreGlobals(t)

	metrics := New()
	result, err := SetupMetrics(context.Background(), metrics, MetricsConfig{
		Resource: ResourceConfig{
			ServiceName:    "degraded-service",
			ServiceVersion: "test-version",
			DeploymentEnv:  "test-env",
		},
		// No scheme, which is what a hand written manifest usually carries and
		// what the endpoint parser refuses fail-closed.
		Exporter: MetricExporterConfig{OTLPEndpoint: "collector:4318"},
	})
	if err != nil {
		t.Fatalf("SetupMetrics() error = %v, want scrape-only degradation rather than no provider", err)
	}
	t.Cleanup(func() {
		if err := result.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown metrics: %v", err)
		}
	})

	if result.ExportErr == nil {
		t.Fatal("ExportErr = nil, want the unusable OTLP destination reported")
	}
	if result.PushConfigured() {
		t.Fatal("PushConfigured() = true, want push reported as unavailable")
	}
	if err := metrics.RecordTraceExporterState(context.Background(), false); err != nil {
		t.Fatalf("RecordTraceExporterState() error = %v, want a usable meter provider", err)
	}
	// The Go runtime instruments only exist once a meter provider is installed,
	// so their presence is what proves scrape survived.
	if scraped := collectMetricsText(t, metrics); !strings.Contains(scraped, "go_goroutine") {
		t.Fatal("scrape carries no Go runtime series; the meter provider was lost with the OTLP destination")
	}
}
