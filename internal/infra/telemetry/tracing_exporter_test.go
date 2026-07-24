package telemetry

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildTraceExporterOptions(t *testing.T) {
	t.Run("not configured", func(t *testing.T) {
		t.Parallel()

		options, configured, err := buildTraceExporterOptions(TraceExporterConfig{})
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
			OTLPHeaders: "authorization=Bearer token",
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
		t.Parallel()

		options, configured, err := buildTraceExporterOptions(TraceExporterConfig{
			OTLPEndpoint: "https://otel.example.com:4318",
			OTLPHeaders:  "authorization=Bearer token",
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

	t.Run("endpoint without path defaults to the traces path", func(t *testing.T) {
		t.Parallel()

		paths := make(chan string, 1)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			paths <- r.URL.Path
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(server.Close)

		options, configured, err := buildTraceExporterOptions(TraceExporterConfig{
			OTLPEndpoint: server.URL,
		})
		if err != nil {
			t.Fatalf("buildTraceExporterOptions() error = %v", err)
		}
		if !configured {
			t.Fatalf("configured = false, want true")
		}

		exportOneTestSpan(t, options)
		assertCollectorPath(t, paths, "/v1/traces")
	})

	t.Run("endpoint path is used exactly as configured", func(t *testing.T) {
		t.Parallel()

		paths := make(chan string, 1)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			paths <- r.URL.Path
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(server.Close)

		options, configured, err := buildTraceExporterOptions(TraceExporterConfig{
			OTLPEndpoint: server.URL + "/custom/traces",
		})
		if err != nil {
			t.Fatalf("buildTraceExporterOptions() error = %v", err)
		}
		if !configured {
			t.Fatalf("configured = false, want true")
		}

		exportOneTestSpan(t, options)
		assertCollectorPath(t, paths, "/custom/traces")
	})

	t.Run("scheme-less endpoint is rejected fail-closed", func(t *testing.T) {
		t.Parallel()

		_, _, err := buildTraceExporterOptions(TraceExporterConfig{
			OTLPEndpoint: "otel.internal:4318",
		})
		if err == nil {
			t.Fatal("buildTraceExporterOptions() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "unsupported scheme") {
			t.Fatalf("buildTraceExporterOptions() error = %v, want unsupported scheme", err)
		}
	})
}

func TestTraceOTLPEndpointRedactsInvalidAndSecretBearingEndpoints(t *testing.T) {
	t.Parallel()

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
		{
			name:    "port-only url authority",
			raw:     "http://:4318/v1/traces",
			wantErr: "empty host",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := traceExporterOTLPEndpoint(TraceExporterConfig{
				OTLPEndpoint: tc.raw,
			})
			if err == nil {
				t.Fatal("traceExporterOTLPEndpoint() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("traceExporterOTLPEndpoint() error = %v, want %q", err, tc.wantErr)
			}
			for _, leaked := range []string{tc.raw, "secret-value", "Bearer"} {
				if strings.Contains(err.Error(), leaked) {
					t.Fatalf("traceExporterOTLPEndpoint() error = %v, leaked %q", err, leaked)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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
