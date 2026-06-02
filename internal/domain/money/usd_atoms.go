package money

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const USDAtomScale int64 = 100_000_000

var (
	ErrInvalidAmount = errors.New("invalid USD amount")
	ErrAmountRange   = errors.New("USD amount out of range")
)

func ParseUSDAtoms(raw string, allowSigned bool) (int64, error) {
	if raw == "" {
		return 0, fmt.Errorf("%w: empty", ErrInvalidAmount)
	}
	if strings.TrimSpace(raw) != raw {
		return 0, fmt.Errorf("%w: surrounding whitespace", ErrInvalidAmount)
	}

	amount, sign, err := parseUSDAtomSign(raw, allowSigned)
	if err != nil {
		return 0, err
	}
	raw = amount

	whole, frac, hasFrac := strings.Cut(raw, ".")
	if err := validateUSDDecimalParts(whole, frac, hasFrac); err != nil {
		return 0, err
	}

	wholeAtoms, err := parseWholeAtoms(whole)
	if err != nil {
		return 0, err
	}
	fracAtoms, err := parseFractionAtoms(frac)
	if err != nil {
		return 0, err
	}
	if wholeAtoms > math.MaxInt64-fracAtoms {
		return 0, fmt.Errorf("%w: %q", ErrAmountRange, raw)
	}

	atoms := wholeAtoms + fracAtoms
	if atoms == 0 {
		return 0, nil
	}
	if sign < 0 {
		return -atoms, nil
	}
	return atoms, nil
}

func parseUSDAtomSign(raw string, allowSigned bool) (string, int64, error) {
	sign := int64(1)
	if raw[0] != '+' && raw[0] != '-' {
		return raw, sign, nil
	}
	if raw[0] == '-' {
		if !allowSigned {
			return "", 0, fmt.Errorf("%w: negative amount not allowed", ErrInvalidAmount)
		}
		sign = -1
	}
	raw = raw[1:]
	if raw == "" {
		return "", 0, fmt.Errorf("%w: sign without digits", ErrInvalidAmount)
	}
	return raw, sign, nil
}

func validateUSDDecimalParts(whole, frac string, hasFrac bool) error {
	switch {
	case whole == "":
		return fmt.Errorf("%w: missing integer digits", ErrInvalidAmount)
	case hasFrac && frac == "":
		return fmt.Errorf("%w: missing fractional digits", ErrInvalidAmount)
	case strings.Contains(frac, "."):
		return fmt.Errorf("%w: multiple decimal points", ErrInvalidAmount)
	case len(frac) > 8:
		return fmt.Errorf("%w: excess precision", ErrInvalidAmount)
	case !allDecimalDigits(whole) || (hasFrac && !allDecimalDigits(frac)):
		return fmt.Errorf("%w: non-decimal digit", ErrInvalidAmount)
	default:
		return nil
	}
}

func FormatUSDAtoms(atoms int64) string {
	negative := atoms < 0
	abs := absUint64(atoms)
	whole := abs / uint64(USDAtomScale)
	frac := abs % uint64(USDAtomScale)

	var b strings.Builder
	if negative {
		b.WriteByte('-')
	}
	b.WriteString(strconv.FormatUint(whole, 10))
	if frac == 0 {
		return b.String()
	}

	fracText := fmt.Sprintf("%08d", frac)
	fracText = strings.TrimRight(fracText, "0")
	b.WriteByte('.')
	b.WriteString(fracText)
	return b.String()
}

func RoundUpUSDAtoms(numerator, denominator int64) (int64, error) {
	if numerator < 0 {
		return 0, fmt.Errorf("%w: negative numerator", ErrInvalidAmount)
	}
	if denominator <= 0 {
		return 0, fmt.Errorf("%w: non-positive denominator", ErrInvalidAmount)
	}
	q := numerator / denominator
	if numerator%denominator != 0 {
		if q == math.MaxInt64 {
			return 0, ErrAmountRange
		}
		q++
	}
	return q, nil
}

func RoundHalfUpUSDAtoms(numerator, denominator int64) (int64, error) {
	if numerator < 0 {
		return 0, fmt.Errorf("%w: negative numerator", ErrInvalidAmount)
	}
	if denominator <= 0 {
		return 0, fmt.Errorf("%w: non-positive denominator", ErrInvalidAmount)
	}

	q := numerator / denominator
	remainder := numerator % denominator
	if remainder == 0 {
		return q, nil
	}
	if remainder > (denominator-1)/2 {
		if q == math.MaxInt64 {
			return 0, ErrAmountRange
		}
		q++
	}
	return q, nil
}

func parseWholeAtoms(whole string) (int64, error) {
	var value uint64
	for _, r := range whole {
		digit := decimalDigitValue(r)
		if value > uint64(math.MaxInt64)/10 {
			return 0, fmt.Errorf("%w: %q", ErrAmountRange, whole)
		}
		value *= 10
		if value > uint64(math.MaxInt64)-digit {
			return 0, fmt.Errorf("%w: %q", ErrAmountRange, whole)
		}
		value += digit
	}
	if value > uint64(math.MaxInt64)/uint64(USDAtomScale) {
		return 0, fmt.Errorf("%w: %q", ErrAmountRange, whole)
	}
	atoms := value * uint64(USDAtomScale)
	if atoms > uint64(math.MaxInt64) {
		return 0, fmt.Errorf("%w: %q", ErrAmountRange, whole)
	}
	return int64(atoms), nil
}

func parseFractionAtoms(frac string) (int64, error) {
	if frac == "" {
		return 0, nil
	}
	padded := frac + strings.Repeat("0", 8-len(frac))
	value, err := strconv.ParseInt(padded, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %q", ErrInvalidAmount, frac)
	}
	return value, nil
}

func allDecimalDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func decimalDigitValue(r rune) uint64 {
	switch r {
	case '0':
		return 0
	case '1':
		return 1
	case '2':
		return 2
	case '3':
		return 3
	case '4':
		return 4
	case '5':
		return 5
	case '6':
		return 6
	case '7':
		return 7
	case '8':
		return 8
	case '9':
		return 9
	default:
		return 0
	}
}

func absUint64(v int64) uint64 {
	if v >= 0 {
		return uint64(v) // #nosec G115 -- non-negative int64 values always fit into uint64.
	}
	if v == math.MinInt64 {
		return uint64(math.MaxInt64) + 1
	}
	return uint64(-v) // #nosec G115 -- v is negative but not MinInt64, so -v is in int64 range and fits uint64.
}
