package httpx

import (
	"errors"
	"runtime"
	"testing"
)

func TestPanicClassClassifiesRecoveredValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		rec  any
		want string
	}{
		{name: "runtime error", rec: &runtime.TypeAssertionError{}, want: "runtime_error"},
		{name: "error", rec: errors.New("boom"), want: "error"},
		{name: "string", rec: "boom", want: "string"},
		{name: "value", rec: 42, want: "value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := panicClass(tt.rec); got != tt.want {
				t.Fatalf("panicClass(%v) = %q, want %q", tt.rec, got, tt.want)
			}
		})
	}
}
