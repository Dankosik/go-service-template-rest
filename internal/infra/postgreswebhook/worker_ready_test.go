package postgreswebhook

import (
	"testing"
	"time"
)

func TestWebhookWorkerReadinessFreshness(t *testing.T) {
	state := readinessState{interval: time.Second}
	state.observed()
	if state.ready() {
		t.Fatal("observation without successful maintenance opened readiness")
	}
	state.maintained()
	if !state.ready() {
		t.Fatal("fresh admitted worker is not ready")
	}
	state.mu.Lock()
	state.lastObservation = time.Now().Add(-2*time.Second - time.Nanosecond)
	state.mu.Unlock()
	if state.ready() {
		t.Fatal("observation older than exactly two intervals remained ready")
	}
	state.observed()
	state.close()
	if state.ready() {
		t.Fatal("closed admission remained ready")
	}
}
