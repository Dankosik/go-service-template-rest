package config

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const maxExactIntegerFloat64 = 1 << 53

func parseDuration(raw string) (time.Duration, error) {
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s", sanitizedDurationParseDetail(raw))
	}
	return d, nil
}

func sanitizedDurationParseDetail(raw string) string {
	if !strings.ContainsAny(raw, "hmsuµμn") {
		return "missing duration unit"
	}
	return "invalid duration syntax"
}

func parseSignedInteger(value any, bitSize int) (int64, error) {
	lowerBound, upperBound, err := signedIntegerBounds(bitSize)
	if err != nil {
		return 0, err
	}

	// The arms below are the types a config source can actually produce, and the
	// list is deliberately not wider. koanf hands this hook YAML scalars, the
	// environment provider's strings, and whatever defaultValues put in the
	// confmap — so int, int64, uint64 for a YAML integer past the int64 range,
	// float64, and string. The sized and unsigned variants that used to be here
	// were unreachable through every one of those sources, and an unsupported
	// type is reported by name rather than silently converted.
	switch v := value.(type) {
	case int:
		return signedIntegerFromInt64(int64(v), lowerBound, upperBound)
	case int64:
		return signedIntegerFromInt64(v, lowerBound, upperBound)
	case uint64:
		return signedIntegerFromUint64(v, upperBound)
	case float64:
		return signedIntegerFromFloat64(v, lowerBound, upperBound)
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, bitSize)
		if err != nil {
			return 0, fmt.Errorf("invalid integer format")
		}
		return n, nil
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
}

func signedIntegerBounds(bitSize int) (int64, int64, error) {
	switch {
	case bitSize <= 0 || bitSize > 64:
		return 0, 0, fmt.Errorf("unsupported integer bit size")
	case bitSize == 64:
		return math.MinInt64, math.MaxInt64, nil
	default:
		upperBound := int64(1)<<(bitSize-1) - 1
		lowerBound := -(int64(1) << (bitSize - 1))
		return lowerBound, upperBound, nil
	}
}

func signedIntegerFromInt64(v int64, lowerBound int64, upperBound int64) (int64, error) {
	if v < lowerBound || v > upperBound {
		return 0, fmt.Errorf("integer out of range")
	}
	return v, nil
}

func signedIntegerFromUint64(v uint64, upperBound int64) (int64, error) {
	if upperBound < 0 || v > uint64(math.MaxInt64) {
		return 0, fmt.Errorf("integer out of range")
	}
	n := int64(v)
	if n > upperBound {
		return 0, fmt.Errorf("integer out of range")
	}
	return n, nil
}

func signedIntegerFromFloat64(v float64, lowerBound int64, upperBound int64) (int64, error) {
	if !isFiniteFloat64(v) {
		return 0, fmt.Errorf("non-finite numeric value")
	}
	if math.Trunc(v) != v {
		return 0, fmt.Errorf("non-integer numeric value")
	}
	if math.Abs(v) > maxExactIntegerFloat64 {
		return 0, fmt.Errorf("integer out of range")
	}
	if v < float64(lowerBound) || v > float64(upperBound) {
		return 0, fmt.Errorf("integer out of range")
	}
	return int64(v), nil
}

func parseFloat64(value any) (float64, error) {
	var n float64
	// The same reachable set as parseSignedInteger: a YAML ratio written without
	// a decimal point arrives as an integer, which is why the integer arms are
	// here at all.
	switch v := value.(type) {
	case float64:
		n = v
	case int:
		n = float64(v)
	case int64:
		n = float64(v)
	case uint64:
		n = float64(v)
	case string:
		var err error
		n, err = strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid float format")
		}
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
	if !isFiniteFloat64(n) {
		return 0, fmt.Errorf("non-finite numeric value")
	}
	return n, nil
}

func isFiniteFloat64(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

func parseBool(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		b, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return false, fmt.Errorf("invalid boolean format")
		}
		return b, nil
	default:
		return false, fmt.Errorf("unsupported type %T", value)
	}
}
