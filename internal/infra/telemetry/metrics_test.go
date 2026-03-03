package telemetry

import "testing"

func TestNormalizeFieldGroupLabel(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "http field", input: "http.addr", want: "http"},
		{name: "observability field", input: "observability.otel.service_name", want: "observability"},
		{name: "redis group", input: "redis", want: "redis"},
		{name: "unknown group", input: "custom.group.field", want: "other"},
		{name: "empty", input: "", want: "other"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeFieldGroupLabel(tc.input)
			if got != tc.want {
				t.Fatalf("normalizeFieldGroupLabel(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeTelemetryFailureReason(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "setup error", input: "setup_error", want: "setup_error"},
		{name: "deadline exceeded", input: "deadline_exceeded", want: "deadline_exceeded"},
		{name: "canceled upper", input: "CANCELED", want: "canceled"},
		{name: "unknown", input: "dns_failure", want: "other"},
		{name: "empty", input: "", want: "other"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeTelemetryFailureReason(tc.input)
			if got != tc.want {
				t.Fatalf("normalizeTelemetryFailureReason(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
