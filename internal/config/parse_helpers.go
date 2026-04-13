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

func parseInt(value any) (int, error) {
	n, err := parseSignedInteger(value, strconv.IntSize)
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

func parseInt64(value any) (int64, error) {
	return parseSignedInteger(value, 64)
}

func parseSignedInteger(value any, bitSize int) (int64, error) {
	lowerBound, upperBound, err := signedIntegerBounds(bitSize)
	if err != nil {
		return 0, err
	}

	switch v := value.(type) {
	case int:
		return signedIntegerFromInt64(int64(v), lowerBound, upperBound)
	case int8:
		return signedIntegerFromInt64(int64(v), lowerBound, upperBound)
	case int16:
		return signedIntegerFromInt64(int64(v), lowerBound, upperBound)
	case int32:
		return signedIntegerFromInt64(int64(v), lowerBound, upperBound)
	case int64:
		return signedIntegerFromInt64(v, lowerBound, upperBound)
	case uint:
		return signedIntegerFromUint64(uint64(v), upperBound)
	case uint8:
		return signedIntegerFromUint64(uint64(v), upperBound)
	case uint16:
		return signedIntegerFromUint64(uint64(v), upperBound)
	case uint32:
		return signedIntegerFromUint64(uint64(v), upperBound)
	case uint64:
		return signedIntegerFromUint64(v, upperBound)
	case float32:
		return signedIntegerFromFloat64(float64(v), lowerBound, upperBound)
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
	switch v := value.(type) {
	case float64:
		n = v
	case float32:
		n = float64(v)
	case int:
		n = float64(v)
	case int8:
		n = float64(v)
	case int16:
		n = float64(v)
	case int32:
		n = float64(v)
	case int64:
		n = float64(v)
	case uint:
		n = float64(v)
	case uint8:
		n = float64(v)
	case uint16:
		n = float64(v)
	case uint32:
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
