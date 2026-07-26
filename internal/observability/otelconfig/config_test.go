package otelconfig

import (
	"math"
	"strings"
	"testing"
)

func TestTraceSamplerOrDefault(t *testing.T) {
	t.Parallel()

	if got := TraceSamplerOrDefault("  "); got != DefaultTracesSampler {
		t.Fatalf("TraceSamplerOrDefault(empty) = %q, want %q", got, DefaultTracesSampler)
	}
	if got := TraceSamplerOrDefault(" ALWAYS_ON "); got != SamplerAlwaysOn {
		t.Fatalf("TraceSamplerOrDefault() = %q, want %q", got, SamplerAlwaysOn)
	}
}

func TestValidateTraceSamplerAccepts(t *testing.T) {
	t.Parallel()

	samplers := []string{
		"",
		" TRACEIDRATIO ",
		SamplerAlwaysOn,
		SamplerAlwaysOff,
		SamplerTraceIDRatio,
		SamplerParentBasedTraceIDRatio,
	}
	for _, sampler := range samplers {
		for _, arg := range []float64{0, DefaultTracesSamplerArg, 1} {
			if err := ValidateTraceSampler(sampler, arg); err != nil {
				t.Fatalf("ValidateTraceSampler(%q, %v) error = %v, want nil", sampler, arg, err)
			}
		}
	}
}

func TestValidateTraceSamplerRejects(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		sampler string
		arg     float64
		wantErr string
	}{
		{name: "unknown name", sampler: "sometimes", arg: 0.5, wantErr: "traces_sampler is unsupported"},
		{name: "nan arg", sampler: SamplerTraceIDRatio, arg: math.NaN(), wantErr: "traces_sampler_arg must be finite"},
		{name: "positive inf arg", sampler: SamplerTraceIDRatio, arg: math.Inf(1), wantErr: "traces_sampler_arg must be finite"},
		{name: "negative inf arg", sampler: SamplerTraceIDRatio, arg: math.Inf(-1), wantErr: "traces_sampler_arg must be finite"},
		{name: "below range", sampler: SamplerTraceIDRatio, arg: -0.1, wantErr: "traces_sampler_arg must be in range [0,1]"},
		{name: "above range", sampler: SamplerTraceIDRatio, arg: 1.1, wantErr: "traces_sampler_arg must be in range [0,1]"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateTraceSampler(tc.sampler, tc.arg)
			if err == nil {
				t.Fatal("ValidateTraceSampler() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("ValidateTraceSampler() error = %q, want to contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}
