package telemetry

import "testing"

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
