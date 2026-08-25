package config

import (
	"errors"
	"strings"
	"testing"
)

//nolint:paralleltest // This test mutates process-global environment or working directory.
func TestMetricsAddressValidation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{name: "disabled", addr: ""},
		{name: "loopback", addr: "127.0.0.1:9090"},
		{name: "ipv6 loopback", addr: "[::1]:9090"},
		{name: "missing port", addr: "127.0.0.1", wantErr: true},
		{name: "zero port", addr: "127.0.0.1:0", wantErr: true},
		{name: "non numeric port", addr: "127.0.0.1:http", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv("APP__OBSERVABILITY__METRICS__ADDR", tc.addr)

			_, _, err := LoadDetailed(LoadOptions{})
			if tc.wantErr && !errors.Is(err, ErrValidate) {
				t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("LoadDetailed() error = %v", err)
			}
		})
	}
}

//nolint:paralleltest // This test mutates process-global environment or working directory.
func TestSamplerValidationUsesConfigError(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP__OBSERVABILITY__OTEL__TRACES_SAMPLER", "not-a-sampler")

	_, _, err := LoadDetailed(LoadOptions{})
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
	}
	const wantDetail = "observability.otel.traces_sampler is unsupported"
	if !strings.Contains(err.Error(), wantDetail) {
		t.Fatalf("LoadDetailed() error = %q, want to contain %q", err.Error(), wantDetail)
	}
}

//nolint:paralleltest // This test mutates process-global environment or working directory.
func TestPprofRequiresDiagnosticsListener(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP__OBSERVABILITY__PPROF__ENABLED", "true")
	t.Setenv("APP__OBSERVABILITY__METRICS__ADDR", "")

	_, _, err := LoadDetailed(LoadOptions{})
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "observability.metrics.addr") {
		t.Fatalf("error = %q, want it to name observability.metrics.addr", err.Error())
	}
}
