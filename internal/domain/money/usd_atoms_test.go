package money

import (
	"errors"
	"testing"
)

func TestParseUSDAtoms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		allowSigned bool
		want        int64
		wantErr     error
	}{
		{name: "whole", input: "12", want: 1_200_000_000},
		{name: "fraction padded", input: "12.34", want: 1_234_000_000},
		{name: "max precision", input: "0.00000001", want: 1},
		{name: "positive sign", input: "+1.00000001", allowSigned: true, want: 100_000_001},
		{name: "negative signed", input: "-1.25", allowSigned: true, want: -125_000_000},
		{name: "negative zero canonicalized", input: "-0.00000000", allowSigned: true, want: 0},
		{name: "negative rejected", input: "-1", wantErr: ErrInvalidAmount},
		{name: "exponent rejected", input: "1e2", wantErr: ErrInvalidAmount},
		{name: "grouping rejected", input: "1,000", wantErr: ErrInvalidAmount},
		{name: "currency rejected", input: "$1.00", wantErr: ErrInvalidAmount},
		{name: "whitespace rejected", input: " 1.00", wantErr: ErrInvalidAmount},
		{name: "nan rejected", input: "NaN", wantErr: ErrInvalidAmount},
		{name: "infinity rejected", input: "Inf", wantErr: ErrInvalidAmount},
		{name: "excess precision rejected", input: "0.123456789", wantErr: ErrInvalidAmount},
		{name: "range rejected", input: "92233720368.54775808", wantErr: ErrAmountRange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseUSDAtoms(tt.input, tt.allowSigned)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ParseUSDAtoms(%q) error = %v, want %v", tt.input, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseUSDAtoms(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("ParseUSDAtoms(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatUSDAtoms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		atoms int64
		want  string
	}{
		{name: "zero", atoms: 0, want: "0"},
		{name: "whole", atoms: 1_200_000_000, want: "12"},
		{name: "trims fractional zeroes", atoms: 1_234_000_000, want: "12.34"},
		{name: "retains atom precision", atoms: 1, want: "0.00000001"},
		{name: "negative", atoms: -125_000_000, want: "-1.25"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := FormatUSDAtoms(tt.atoms); got != tt.want {
				t.Fatalf("FormatUSDAtoms(%d) = %q, want %q", tt.atoms, got, tt.want)
			}
		})
	}
}

func TestRoundingRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		fn    func(int64, int64) (int64, error)
		num   int64
		den   int64
		want  int64
		error bool
	}{
		{name: "reserve ceiling exact", fn: RoundUpUSDAtoms, num: 100, den: 10, want: 10},
		{name: "reserve ceiling rounds up", fn: RoundUpUSDAtoms, num: 101, den: 10, want: 11},
		{name: "final charge below half", fn: RoundHalfUpUSDAtoms, num: 104, den: 10, want: 10},
		{name: "final charge half up", fn: RoundHalfUpUSDAtoms, num: 105, den: 10, want: 11},
		{name: "reject negative rational", fn: RoundUpUSDAtoms, num: -1, den: 10, error: true},
		{name: "reject zero denominator", fn: RoundHalfUpUSDAtoms, num: 1, den: 0, error: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.fn(tt.num, tt.den)
			if tt.error {
				if err == nil {
					t.Fatal("rounding error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("rounding unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("rounding = %d, want %d", got, tt.want)
			}
		})
	}
}
