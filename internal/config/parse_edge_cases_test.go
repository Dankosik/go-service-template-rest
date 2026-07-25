package config

import (
	"math"
	"strings"
	"testing"
)

func TestParseFloat64SupportedTypes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		value any
		want  float64
	}{
		{name: "float64", value: float64(1.25), want: 1.25},
		{name: "float32", value: float32(2.5), want: 2.5},
		{name: "int", value: int(3), want: 3},
		{name: "int8", value: int8(4), want: 4},
		{name: "int16", value: int16(5), want: 5},
		{name: "int32", value: int32(6), want: 6},
		{name: "int64", value: int64(7), want: 7},
		{name: "uint", value: uint(8), want: 8},
		{name: "uint8", value: uint8(9), want: 9},
		{name: "uint16", value: uint16(10), want: 10},
		{name: "uint32", value: uint32(11), want: 11},
		{name: "uint64", value: uint64(12), want: 12},
		{name: "string", value: " 13.5 ", want: 13.5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseFloat64(tc.value)
			if err != nil {
				t.Fatalf("parseFloat64(%T) error = %v, want nil", tc.value, err)
			}
			if got != tc.want {
				t.Fatalf("parseFloat64(%T) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}

func TestParseFloat64RejectsInvalidValues(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		value   any
		wantErr string
	}{
		{name: "invalid string", value: "not-a-float", wantErr: "invalid float format"},
		{name: "unsupported type", value: struct{}{}, wantErr: "unsupported type"},
		{name: "infinity", value: math.Inf(1), wantErr: "non-finite numeric value"},
		{name: "nan", value: math.NaN(), wantErr: "non-finite numeric value"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseFloat64(tc.value)
			if err == nil {
				t.Fatal("parseFloat64() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("parseFloat64() error = %q, want to contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestSignedIntegerBounds(t *testing.T) {
	t.Parallel()

	lower, upper, err := signedIntegerBounds(8)
	if err != nil {
		t.Fatalf("signedIntegerBounds(8) error = %v, want nil", err)
	}
	if lower != math.MinInt8 || upper != math.MaxInt8 {
		t.Fatalf("signedIntegerBounds(8) = [%d,%d], want [%d,%d]", lower, upper, math.MinInt8, math.MaxInt8)
	}

	lower, upper, err = signedIntegerBounds(64)
	if err != nil {
		t.Fatalf("signedIntegerBounds(64) error = %v, want nil", err)
	}
	if lower != math.MinInt64 || upper != math.MaxInt64 {
		t.Fatalf("signedIntegerBounds(64) = [%d,%d], want [%d,%d]", lower, upper, int64(math.MinInt64), int64(math.MaxInt64))
	}

	for _, bitSize := range []int{0, 65} {
		if _, _, err := signedIntegerBounds(bitSize); err == nil {
			t.Fatalf("signedIntegerBounds(%d) error = nil, want non-nil", bitSize)
		}
	}
}

func TestParseBoolAdditionalErrorCoverage(t *testing.T) {
	t.Parallel()

	got, err := parseBool(true)
	if err != nil {
		t.Fatalf("parseBool(true) error = %v, want nil", err)
	}
	if !got {
		t.Fatal("parseBool(true) = false, want true")
	}

	got, err = parseBool(" false ")
	if err != nil {
		t.Fatalf("parseBool(string) error = %v, want nil", err)
	}
	if got {
		t.Fatal("parseBool(\" false \") = true, want false")
	}

	for _, tc := range []struct {
		name    string
		value   any
		wantErr string
	}{
		{name: "invalid string", value: "definitely", wantErr: "invalid boolean format"},
		{name: "unsupported type", value: 1, wantErr: "unsupported type"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseBool(tc.value)
			if err == nil {
				t.Fatal("parseBool() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("parseBool() error = %q, want to contain %q", err.Error(), tc.wantErr)
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
