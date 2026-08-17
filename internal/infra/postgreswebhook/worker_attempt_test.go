package postgreswebhook

import (
	"testing"
	"time"
)

func TestWebhookWorkerRetryDelayBound(t *testing.T) {
	t.Parallel()
	policy := DeliveryPolicy{BackoffBase: time.Second, BackoffCap: 3 * time.Second}
	for range 100 {
		delay := retryDelay(policy, 2*time.Second)
		if delay < policy.BackoffBase || delay > 3*policy.BackoffBase {
			t.Fatalf("retry delay = %v", delay)
		}
	}
}
