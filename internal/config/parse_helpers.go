package config

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func parseInt(value any) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int8:
		return int(v), nil
	case int16:
		return int(v), nil
	case int32:
		return int(v), nil
	case int64:
		return intFromInt64(v)
	case uint:
		return intFromUint64(uint64(v))
	case uint8:
		return int(v), nil
	case uint16:
		return int(v), nil
	case uint32:
		return int(v), nil
	case uint64:
		return intFromUint64(v)
	case float32:
		return intFromFloat64(float64(v))
	case float64:
		return intFromFloat64(v)
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, strconv.IntSize)
		if err != nil {
			return 0, fmt.Errorf("invalid integer format")
		}
		return int(n), nil
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
}

func parseInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return int64FromUint64(uint64(v))
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		return int64FromUint64(v)
	case float32:
		return int64FromFloat64(float64(v))
	case float64:
		return int64FromFloat64(v)
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid integer format")
		}
		return n, nil
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
}

func intFromInt64(v int64) (int, error) {
	if v < int64(math.MinInt) || v > int64(math.MaxInt) {
		return 0, fmt.Errorf("integer out of range")
	}
	return int(v), nil
}

func intFromUint64(v uint64) (int, error) {
	if v > uint64(math.MaxInt) {
		return 0, fmt.Errorf("integer out of range")
	}
	return int(v), nil
}

func intFromFloat64(v float64) (int, error) {
	if math.Trunc(v) != v {
		return 0, fmt.Errorf("non-integer numeric value")
	}
	if v < float64(math.MinInt) || v > float64(math.MaxInt) {
		return 0, fmt.Errorf("integer out of range")
	}
	return int(v), nil
}

func int64FromUint64(v uint64) (int64, error) {
	if v > uint64(math.MaxInt64) {
		return 0, fmt.Errorf("integer out of range")
	}
	return int64(v), nil
}

func int64FromFloat64(v float64) (int64, error) {
	if math.Trunc(v) != v {
		return 0, fmt.Errorf("non-integer numeric value")
	}
	if v < float64(math.MinInt64) || v > float64(math.MaxInt64) {
		return 0, fmt.Errorf("integer out of range")
	}
	return int64(v), nil
}

func parseFloat64(value any) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid float format")
		}
		return n, nil
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
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
