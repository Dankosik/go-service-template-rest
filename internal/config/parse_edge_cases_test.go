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
		// The reachable set, and only it: a YAML scalar, an environment string,
		// or a value defaultValues put in the confmap. A sized or unsigned
		// variant is a type no config source produces, and is now reported by
		// name rather than silently converted.
		{name: "float64", value: float64(1.25), want: 1.25},
		{name: "int", value: int(3), want: 3},
		{name: "int64", value: int64(7), want: 7},
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
		{name: "sized integer no source produces", value: int32(6), wantErr: "unsupported type"},
		{name: "narrow float no source produces", value: float32(2.5), wantErr: "unsupported type"},
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
