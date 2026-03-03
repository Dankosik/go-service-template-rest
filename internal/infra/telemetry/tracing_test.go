package telemetry

import (
	"testing"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
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

func TestExporterOptionTypeCompatibility(t *testing.T) {
	// Guard against accidental option-type drift when upgrading OTLP exporter package.
	var _ []otlptracehttp.Option
}
