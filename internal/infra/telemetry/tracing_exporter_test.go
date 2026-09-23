package telemetry

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
)

// Every subtest clears the ambient exporter environment, because resolution now
// falls back to the standard OpenTelemetry variables and a developer or CI
// machine that exports one would otherwise change what these assert.
// t.Setenv forbids t.Parallel, so these run sequentially.
//
//nolint:paralleltest // ambient env control is process-wide state.
func TestTraceExporterEndpointAndExporter(t *testing.T) {
	t.Run("not configured", func(t *testing.T) {
		telemetrytest.ClearAmbientExporterEnv(t)

		endpoint, err := resolveTraceExporterEndpoint(TraceExporterConfig{})
		if err != nil {
			t.Fatalf("resolveTraceExporterEndpoint() error = %v", err)
		}
		if endpoint.Configured() {
			t.Fatalf("endpoint = %+v, want unconfigured", endpoint)
		}
	})

	t.Run("headers without endpoint do not reach an ambient destination", func(t *testing.T) {
		telemetrytest.ClearAmbientExporterEnv(t)
		t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://env-collector.example:4318")

		endpoint, err := resolveTraceExporterEndpoint(TraceExporterConfig{
			OTLPHeaders: "authorization=Bearer token",
		})
		if err != nil {
			t.Fatalf("resolveTraceExporterEndpoint() error = %v", err)
		}
		if endpoint.Configured() {
			t.Fatalf("endpoint = %+v, want unconfigured", endpoint)
		}
	})

	t.Run("configured endpoint and headers", func(t *testing.T) {
		telemetrytest.ClearAmbientExporterEnv(t)

		cfg := TraceExporterConfig{
			OTLPEndpoint: "https://otel.example.com:4318",
			OTLPHeaders:  "authorization=Bearer token",
		}
		endpoint, err := resolveTraceExporterEndpoint(cfg)
		if err != nil {
			t.Fatalf("resolveTraceExporterEndpoint() error = %v", err)
		}
		if endpoint.Source != SharedOTLPExporterConfigKey {
			t.Fatalf("endpoint source = %q, want %q", endpoint.Source, SharedOTLPExporterConfigKey)
		}
		if !endpoint.ConfiguredByService {
			t.Fatal("ConfiguredByService = false, want true")
		}
		exporter, err := newOTLPTraceExporter(t.Context(), endpoint, cfg)
		if err != nil {
			t.Fatalf("newOTLPTraceExporter() error = %v", err)
		}
		if err := exporter.Shutdown(t.Context()); err != nil {
			t.Fatalf("exporter.Shutdown() error = %v", err)
		}
	})

	t.Run("endpoint without path defaults to the traces path", func(t *testing.T) {
		telemetrytest.ClearAmbientExporterEnv(t)

		paths := make(chan string, 1)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			paths <- r.URL.Path
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(server.Close)

		cfg := TraceExporterConfig{OTLPEndpoint: server.URL}
		endpoint, err := resolveTraceExporterEndpoint(cfg)
		if err != nil {
			t.Fatalf("resolveTraceExporterEndpoint() error = %v", err)
		}
		if !endpoint.Configured() {
			t.Fatalf("endpoint = %+v, want configured", endpoint)
		}

		exportOneTestSpan(t, endpoint, cfg)
		assertCollectorPath(t, paths, "/v1/traces")
	})

	t.Run("endpoint path is used exactly as configured", func(t *testing.T) {
		telemetrytest.ClearAmbientExporterEnv(t)

		paths := make(chan string, 1)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			paths <- r.URL.Path
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(server.Close)

		cfg := TraceExporterConfig{OTLPEndpoint: server.URL + "/custom/traces"}
		endpoint, err := resolveTraceExporterEndpoint(cfg)
		if err != nil {
			t.Fatalf("resolveTraceExporterEndpoint() error = %v", err)
		}
		if !endpoint.Configured() {
			t.Fatalf("endpoint = %+v, want configured", endpoint)
		}

		exportOneTestSpan(t, endpoint, cfg)
		assertCollectorPath(t, paths, "/custom/traces")
	})

	t.Run("malformed headers still report the resolved endpoint", func(t *testing.T) {
		telemetrytest.ClearAmbientExporterEnv(t)
		telemetrytest.RestoreGlobals(t)

		endpoint, shutdown, err := SetupTracing(t.Context(), TracingConfig{
			Exporter: TraceExporterConfig{
				OTLPEndpoint: "https://otel.example.com:4318",
				OTLPHeaders:  "malformed",
			},
		})
		if err == nil {
			t.Fatal("SetupTracing() error = nil, want non-nil")
		}
		if shutdown != nil {
			t.Fatal("SetupTracing() shutdown != nil, want nil on error")
		}
		if endpoint.Source != SharedOTLPExporterConfigKey {
			t.Fatalf("endpoint source = %q, want %q", endpoint.Source, SharedOTLPExporterConfigKey)
		}
	})

	t.Run("scheme-less endpoint is rejected fail-closed", func(t *testing.T) {
		telemetrytest.ClearAmbientExporterEnv(t)

		_, err := resolveTraceExporterEndpoint(TraceExporterConfig{
			OTLPEndpoint: "otel.internal:4318",
		})
		if err == nil {
			t.Fatal("resolveTraceExporterEndpoint() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "unsupported scheme") {
			t.Fatalf("resolveTraceExporterEndpoint() error = %v, want unsupported scheme", err)
		}
	})
}

//nolint:paralleltest // ambient env control is process-wide state.
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
		// #nosec G101 -- Synthetic credentials verify userinfo rejection and error redaction.
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
		{
			name:    "port-only url authority",
			raw:     "http://:4318/v1/traces",
			wantErr: "empty host",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			telemetrytest.ClearAmbientExporterEnv(t)

			_, err := resolveTraceExporterEndpoint(TraceExporterConfig{
				OTLPEndpoint: tc.raw,
			})
			if err == nil {
				t.Fatal("resolveTraceExporterEndpoint() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("resolveTraceExporterEndpoint() error = %v, want %q", err, tc.wantErr)
			}
			for _, leaked := range []string{tc.raw, "secret-value", "Bearer"} {
				if strings.Contains(err.Error(), leaked) {
					t.Fatalf("resolveTraceExporterEndpoint() error = %v, leaked %q", err, leaked)
				}
			}
		})
	}
}

func TestParseOTLPHeaders(t *testing.T) {
	t.Parallel()

	headers, err := parseOTLPHeaders("authorization=Bearer token,x-api-key=abc")
	if err != nil {
		t.Fatalf("parseOTLPHeaders() error = %v", err)
	}
	if headers["Authorization"] != "Bearer token" {
		t.Fatalf("headers[Authorization] = %q, want %q", headers["Authorization"], "Bearer token")
	}
	if headers["X-Api-Key"] != "abc" {
		t.Fatalf("headers[X-Api-Key] = %q, want %q", headers["X-Api-Key"], "abc")
	}

	_, err = parseOTLPHeaders("malformed")
	if err == nil {
		t.Fatal("parseOTLPHeaders() error = nil, want non-nil")
	}
}

func TestParseOTLPHeadersMalformedEntriesDoNotLeakRawValues(t *testing.T) {
	t.Parallel()

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
		{
			name: "invalid header key",
			raw:  "bad key=secret-value",
		},
		{
			name: "invalid header value",
			raw:  "authorization=Bearer secret-value\r\ninjected=true",
		},
		{
			name: "case-insensitive duplicate",
			raw:  "Authorization=first,authorization=secret-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseOTLPHeaders(tt.raw)
			if err == nil {
				t.Fatal("parseOTLPHeaders() error = nil, want non-nil")
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
