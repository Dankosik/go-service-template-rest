package postgreswebhook

import (
	"errors"
	"testing"
)

func TestWebhookWorkerMaintenanceRequiresStore(t *testing.T) {
	t.Parallel()
	worker := &Worker{config: WorkerConfig{MaintenanceBatch: 1}}
	_, err := worker.maintain(t.Context())
	if !errors.Is(err, ErrConfig) {
		t.Fatalf("maintain() error = %v", err)
	}
}
