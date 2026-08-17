package postgreswebhook

import (
	"errors"
	"math"
	"testing"
)

func TestNumericBoundsRejectOverflow(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		err  error
	}{
		{"int32", func() error { _, err := int32Value(math.MaxInt32 + 1); return err }()},
		{"uint32", func() error { _, err := uint32Value(math.MaxUint32 + 1); return err }()},
		{"duration", func() error { _, err := durationValue(math.MaxInt64 + 1); return err }()},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if !errors.Is(test.err, ErrConfig) {
				t.Fatalf("overflow error = %v", test.err)
			}
		})
	}
}

func TestNumericBoundsAcceptBoundaries(t *testing.T) {
	t.Parallel()
	if value, err := remainingBatch(10, 3); err != nil || value != 7 {
		t.Fatalf("remainingBatch(10, 3) = %d, %v", value, err)
	}
	for _, used := range []int64{-1, 11} {
		if _, err := remainingBatch(10, used); !errors.Is(err, ErrConflict) {
			t.Fatalf("remainingBatch(10, %d) error = %v", used, err)
		}
	}
	if value, err := int32Value(math.MinInt32); err != nil || value != math.MinInt32 {
		t.Fatalf("int32Value(min) = %d, %v", value, err)
	}
	if value, err := int32Value(math.MaxInt32); err != nil || value != math.MaxInt32 {
		t.Fatalf("int32Value(max) = %d, %v", value, err)
	}
	if value, err := uint32Value(math.MaxUint32); err != nil || value != math.MaxUint32 {
		t.Fatalf("uint32Value(max) = %d, %v", value, err)
	}
	if _, err := uint32Value(-1); !errors.Is(err, ErrConfig) {
		t.Fatalf("uint32Value(-1) error = %v", err)
	}
	if value, err := uint64Value(math.MaxInt64); err != nil || value != math.MaxInt64 {
		t.Fatalf("uint64Value(max) = %d, %v", value, err)
	}
	if _, err := uint64Value(-1); !errors.Is(err, ErrConfig) {
		t.Fatalf("uint64Value(-1) error = %v", err)
	}
	if value, err := durationValue(math.MaxInt64); err != nil || value != math.MaxInt64 {
		t.Fatalf("durationValue(max) = %d, %v", value, err)
	}
}
