package postgreswebhook

import (
	"fmt"
	"math"
	"time"
)

func int32Value(value int) (int32, error) {
	if value < math.MinInt32 || value > math.MaxInt32 {
		return 0, fmt.Errorf("%w: integer value is out of range", ErrConfig)
	}
	return int32(value), nil
}

func remainingBatch(batch int32, used int64) (int32, error) {
	if batch < 0 || used < 0 || used > int64(batch) {
		return 0, fmt.Errorf("%w: database batch result is out of range", ErrConflict)
	}
	// #nosec G115 -- the preceding bounds prove used fits this non-negative int32 batch.
	return batch - int32(used), nil
}

func uint32Value(value int) (uint32, error) {
	if value < 0 || value > math.MaxUint32 {
		return 0, fmt.Errorf("%w: integer value is out of range", ErrConfig)
	}
	return uint32(value), nil
}

func uint64Value(value int64) (uint64, error) {
	if value < 0 {
		return 0, fmt.Errorf("%w: integer value is out of range", ErrConfig)
	}
	return uint64(value), nil
}

func durationValue(value uint64) (time.Duration, error) {
	if value > math.MaxInt64 {
		return 0, fmt.Errorf("%w: duration is out of range", ErrConfig)
	}
	return time.Duration(value), nil
}
