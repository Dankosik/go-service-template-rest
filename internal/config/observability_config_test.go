package config

import (
	"errors"
	"math"
	"strings"
	"testing"
)

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

func TestValidateSamplerAdditionalErrorCoverage(t *testing.T) {
	t.Parallel()

	const supportedSampler = "parentbased_traceidratio"

	if err := validateObservabilitySampler(supportedSampler, 0.5); err != nil {
		t.Fatalf("validateObservabilitySampler(valid) error = %v, want nil", err)
	}

	testCases := []struct {
		name    string
		sampler string
		arg     float64
		wantErr string
	}{
		{
			name:    "unsupported sampler",
			sampler: "not-a-sampler",
			arg:     0.5,
			wantErr: "traces_sampler is unsupported",
		},
		{
			name:    "non finite arg",
			sampler: supportedSampler,
			arg:     math.Inf(1),
			wantErr: "traces_sampler_arg must be finite",
		},
		{
			name:    "out of range arg",
			sampler: supportedSampler,
			arg:     1.1,
			wantErr: "traces_sampler_arg must be in range [0,1]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateObservabilitySampler(tc.sampler, tc.arg)
			if err == nil {
				t.Fatal("validateObservabilitySampler() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("validateObservabilitySampler() error = %q, want to contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}
