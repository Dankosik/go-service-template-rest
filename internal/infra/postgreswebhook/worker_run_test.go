package postgreswebhook

import (
	"errors"
	"testing"
	"time"
)

func TestWebhookWorkerTerminalResult(t *testing.T) {
	var worker *Worker
	result := worker.Run(t.Context())
	if !errors.Is(result.Err, ErrConfig) || result.CleanupUnsafe {
		t.Fatalf("Run(nil) = %+v", result)
	}
}

func TestWebhookWorkerLocalLifecycleGuards(t *testing.T) {
	worker := &Worker{config: WorkerConfig{AttemptTimeout: time.Second, StoreOperationTimeout: time.Second, DrainTimeout: time.Second}}
	result := worker.Run(t.Context())
	if !errors.Is(result.Err, ErrConfig) || result.CleanupUnsafe {
		t.Fatalf("Run(invalid store) = %+v", result)
	}
	if got := worker.leaseDuration(); got != 4*time.Second {
		t.Fatalf("leaseDuration() = %v", got)
	}
	if err := worker.drain(); err != nil {
		t.Fatalf("drain() error = %v", err)
	}
	if err := worker.runAttempt(t.Context(), ClaimedAttempt{}); !errors.Is(err, ErrConfig) {
		t.Fatalf("runAttempt(invalid) error = %v", err)
	}
}
