package logctx_test

import (
	"errors"
	"runtime"
	"testing"

	"github.com/example/go-service-template-rest/internal/observability/logctx"
)

// TestPanicClassClassifiesRecoveredValues holds the closed set every recovery
// site groups by. runtime_error stays separable because it is always a service
// defect, while a deliberate panic(error) from a library often is not.
func TestPanicClassClassifiesRecoveredValues(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		rec  any
		want string
	}{
		{name: "runtime error", rec: &runtime.TypeAssertionError{}, want: "runtime_error"},
		{name: "error", rec: errors.New("boom"), want: "error"},
		{name: "string", rec: "boom", want: "string"},
		{name: "value", rec: 42, want: "value"},
		{name: "nil", rec: nil, want: "none"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := logctx.PanicClass(tt.rec); got != tt.want {
				t.Fatalf("PanicClass(%v) = %q, want %q", tt.rec, got, tt.want)
			}
		})
	}
}
