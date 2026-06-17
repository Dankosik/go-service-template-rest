package config

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestParseInt(t *testing.T) {
	t.Parallel()

	t.Run("supports mixed numeric inputs", func(t *testing.T) {
		t.Parallel()

		value, err := parseInt("42")
		if err != nil {
			t.Fatalf("parseInt(string) error = %v", err)
		}
		if value != 42 {
			t.Fatalf("parseInt(string) = %d, want 42", value)
		}

		value, err = parseInt(float64(7))
		if err != nil {
			t.Fatalf("parseInt(float64) error = %v", err)
		}
		if value != 7 {
			t.Fatalf("parseInt(float64) = %d, want 7", value)
		}
	})

	t.Run("rejects non integer floats", func(t *testing.T) {
		t.Parallel()

		if _, err := parseInt(1.25); err == nil {
			t.Fatalf("parseInt() expected non-integer error")
		}
	})

	t.Run("rejects non finite floats", func(t *testing.T) {
		t.Parallel()

		if _, err := parseInt(math.NaN()); err == nil {
			t.Fatalf("parseInt() expected non-finite error for NaN")
		}
		if _, err := parseInt(math.Inf(1)); err == nil {
			t.Fatalf("parseInt() expected non-finite error for +Inf")
		}
	})

	t.Run("rejects conversion unsafe float upper bound", func(t *testing.T) {
		t.Parallel()

		if _, err := parseInt(math.Ldexp(1, strconv.IntSize-1)); err == nil {
			t.Fatalf("parseInt() expected overflow error at first unsafe upper bound")
		}
	})

	t.Run("rejects float above exact integer range on wide int", func(t *testing.T) {
		t.Parallel()

		if strconv.IntSize <= 53 {
			t.Skip("parseInt target range is already narrower than float64 exact integer range")
		}
		if _, err := parseInt(math.Ldexp(1, 53) + 2); err == nil {
			t.Fatalf("parseInt() expected unsafe float integer error")
		}
	})

	t.Run("rejects overflow from unsigned values", func(t *testing.T) {
		t.Parallel()

		overflow := uint(math.MaxInt) + 1
		if _, err := parseInt(overflow); err == nil {
			t.Fatalf("parseInt() expected overflow error for uint value")
		}
		if _, err := parseInt(uint64(math.MaxUint64)); err == nil {
			t.Fatalf("parseInt() expected overflow error for uint64 value")
		}
	})
}

func TestParseInt64(t *testing.T) {
	t.Parallel()

	t.Run("supports mixed numeric inputs", func(t *testing.T) {
		t.Parallel()

		value, err := parseInt64("922")
		if err != nil {
			t.Fatalf("parseInt64(string) error = %v", err)
		}
		if value != 922 {
			t.Fatalf("parseInt64(string) = %d, want 922", value)
		}

		value, err = parseInt64(uint32(11))
		if err != nil {
			t.Fatalf("parseInt64(uint32) error = %v", err)
		}
		if value != 11 {
			t.Fatalf("parseInt64(uint32) = %d, want 11", value)
		}
	})

	t.Run("rejects non integer floats", func(t *testing.T) {
		t.Parallel()

		if _, err := parseInt64(float64(2.5)); err == nil {
			t.Fatalf("parseInt64() expected non-integer error")
		}
	})

	t.Run("rejects non finite floats", func(t *testing.T) {
		t.Parallel()

		if _, err := parseInt64(math.NaN()); err == nil {
			t.Fatalf("parseInt64() expected non-finite error for NaN")
		}
		if _, err := parseInt64(math.Inf(-1)); err == nil {
			t.Fatalf("parseInt64() expected non-finite error for -Inf")
		}
	})

	t.Run("rejects conversion unsafe float upper bound", func(t *testing.T) {
		t.Parallel()

		if _, err := parseInt64(math.Ldexp(1, 63)); err == nil {
			t.Fatalf("parseInt64() expected overflow error at first unsafe upper bound")
		}
	})

	t.Run("rejects float above exact integer range", func(t *testing.T) {
		t.Parallel()

		if _, err := parseInt64(math.Ldexp(1, 53) + 2); err == nil {
			t.Fatalf("parseInt64() expected unsafe float integer error")
		}
	})

	t.Run("rejects overflow from unsigned values", func(t *testing.T) {
		t.Parallel()

		if _, err := parseInt64(uint64(math.MaxUint64)); err == nil {
			t.Fatalf("parseInt64() expected overflow error")
		}
	})
}

func TestParseBool(t *testing.T) {
	t.Parallel()

	value, err := parseBool("true")
	if err != nil {
		t.Fatalf("parseBool(true) error = %v", err)
	}
	if !value {
		t.Fatalf("parseBool(true) = false, want true")
	}

	if _, err := parseBool(1); err == nil {
		t.Fatalf("parseBool() expected unsupported type error")
	}
}

func TestValidateRangeHelpers(t *testing.T) {
	t.Parallel()

	t.Run("int range is inclusive", func(t *testing.T) {
		t.Parallel()

		if err := validateIntRange("postgres.max_open_conns", 1, 1, 100); err != nil {
			t.Fatalf("validateIntRange(min) error = %v", err)
		}
		if err := validateIntRange("postgres.max_open_conns", 100, 1, 100); err != nil {
			t.Fatalf("validateIntRange(max) error = %v", err)
		}
	})

	t.Run("int range out of bounds returns ErrValidate", func(t *testing.T) {
		t.Parallel()

		err := validateIntRange("postgres.max_open_conns", 101, 1, 100)
		if err == nil {
			t.Fatalf("validateIntRange() expected error")
		}
		if !errors.Is(err, ErrValidate) {
			t.Fatalf("error = %v, want ErrValidate", err)
		}
		if !strings.Contains(err.Error(), "postgres.max_open_conns") {
			t.Fatalf("error = %v, want field name in message", err)
		}
	})

	t.Run("duration range is inclusive", func(t *testing.T) {
		t.Parallel()

		if err := validateDurationRange("http.read_timeout", time.Second, time.Second, 10*time.Second); err != nil {
			t.Fatalf("validateDurationRange(min) error = %v", err)
		}
		if err := validateDurationRange("http.read_timeout", 10*time.Second, time.Second, 10*time.Second); err != nil {
			t.Fatalf("validateDurationRange(max) error = %v", err)
		}
	})

	t.Run("duration range out of bounds returns ErrValidate", func(t *testing.T) {
		t.Parallel()

		err := validateDurationRange("http.read_timeout", 11*time.Second, time.Second, 10*time.Second)
		if err == nil {
			t.Fatalf("validateDurationRange() expected error")
		}
		if !errors.Is(err, ErrValidate) {
			t.Fatalf("error = %v, want ErrValidate", err)
		}
		if !strings.Contains(err.Error(), "http.read_timeout") {
			t.Fatalf("error = %v, want field name in message", err)
		}
	})
}
