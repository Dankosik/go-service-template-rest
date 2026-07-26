package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
)

// TestMetricsHandlerExposesProcessCollectorOnly pins the split between the two
// runtime signals. Process-level series come from the operating system and are
// registered here, so they exist on the scrape path only. Go runtime series come
// from the OpenTelemetry instruments SetupMetrics registers on the meter provider,
// so they reach the OTLP reader too — which is why the Prometheus Go collector
// that used to be registered here is gone, and why go_gc_duration_seconds must
// not come back: two naming schemes for the same facts is what this replaced.
func TestMetricsHandlerExposesProcessCollectorOnly(t *testing.T) {
	t.Parallel()

	metricsText := collectMetricsText(t, New())
	if !strings.Contains(metricsText, "process_cpu_seconds_total") {
		t.Fatal("metrics output does not contain process_cpu_seconds_total")
	}
	if strings.Contains(metricsText, "go_gc_duration_seconds") {
		t.Fatal("metrics output contains the Prometheus Go collector, which the OTel runtime instruments replaced")
	}
	for _, removed := range []string{
		"config_load_duration_seconds",
		"config_failures_total",
		"startup_rejections_total",
		"telemetry_init_failure_total",
		"config_unknown_key_warnings_total",
		"config_startup_outcome_total",
		"startup_dependency_status",
	} {
		if strings.Contains(metricsText, removed) {
			t.Fatalf("metrics output unexpectedly contains removed startup-only series %q", removed)
		}
	}
}

func TestMetricsNilAndZeroValueHandlersReturnNotFound(t *testing.T) {
	t.Parallel()

	for _, metrics := range []*Metrics{nil, {}} {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		resp := httptest.NewRecorder()
		metrics.Handler().ServeHTTP(resp, req)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("zero-value metrics handler status = %d, want %d", resp.Code, http.StatusNotFound)
		}
	}
}

func collectMetricsText(t *testing.T, metrics *Metrics) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	resp := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("metrics handler status = %d, want %d", resp.Code, http.StatusOK)
	}

	return resp.Body.String()
}

// TestServerLoadIsScrapable is what makes http.max_in_flight tunable. otelhttp
// derives its instruments from the request and supplies neither of these, so
// without them a shed request is one more 503 on a route that also answers 503
// when the connection pool saturates, and nothing reports how close the service
// runs to its limit.
func TestServerLoadIsScrapable(t *testing.T) {
	telemetrytest.RestoreGlobals(t)
	telemetrytest.ClearAmbientExporterEnv(t)

	metrics := New()
	result, err := SetupMetrics(context.Background(), metrics, MetricsConfig{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("SetupMetrics() error = %v", err)
	}
	t.Cleanup(func() {
		if shutdownErr := result.Shutdown(context.Background()); shutdownErr != nil {
			t.Errorf("shutdown meter provider: %v", shutdownErr)
		}
	})

	load := metrics.ServerLoad()
	release := load.Admitted(context.Background())
	load.Shed(context.Background())

	if got := scrapedSeriesValue(t, collectMetricsText(t, metrics), "http_server_active_requests"); got != "1" {
		t.Fatalf("http_server_active_requests = %s, want 1 while a request is admitted", got)
	}
	if got := scrapedSeriesValue(t, collectMetricsText(t, metrics), "http_server_shed_requests_total"); got != "1" {
		t.Fatalf("http_server_shed_requests_total = %s, want 1", got)
	}

	release()
	if got := scrapedSeriesValue(t, collectMetricsText(t, metrics), "http_server_active_requests"); got != "0" {
		t.Fatalf("http_server_active_requests = %s, want 0 after release", got)
	}
}

// scrapedSeriesValue returns the value of the first sample whose metric name
// matches, so an assertion does not have to restate the scope labels the
// exporter attaches.
func scrapedSeriesValue(t *testing.T, scraped, name string) string {
	t.Helper()

	for line := range strings.Lines(scraped) {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, name+"{") && !strings.HasPrefix(line, name+" ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		return fields[len(fields)-1]
	}
	t.Fatalf("scrape has no sample named %q:\n%s", name, scraped)
	return ""
}

// TestServerLoadToleratesNoopProvider keeps a chain built before telemetry setup
// — the reference example is one — from panicking on a nil instrument.
func TestServerLoadToleratesNoopProvider(t *testing.T) {
	t.Parallel()

	load := New().ServerLoad()
	load.Shed(context.Background())
	load.Admitted(context.Background())()

	var zero ServerLoad
	zero.Shed(context.Background())
	zero.Admitted(context.Background())()
}
