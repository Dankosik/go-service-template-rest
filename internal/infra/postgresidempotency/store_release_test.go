package postgresidempotency

import (
	"testing"
	"time"
)

func TestRecoveryDelayRoundsUpToWriterMicros(t *testing.T) {
	if got := durationMicros(time.Microsecond + time.Nanosecond); got != 2 {
		t.Fatalf("recovery delay micros = %d, want 2", got)
	}
}
