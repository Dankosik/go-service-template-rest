package postgreswebhook

import (
	"testing"
	"time"
)

func TestWebhookWorkerRetryDelayBound(t *testing.T) {
	policy := DeliveryPolicy{BackoffBase: time.Second, BackoffCap: 3 * time.Second}
	if got := retryDelayWithRandom(policy, 1, 0); got != time.Second {
		t.Fatalf("attempt 1 delay = %v", got)
	}
	if got := retryDelayWithRandom(policy, 2, ^uint64(0)); got != 2*time.Second {
		t.Fatalf("attempt 2 delay = %v", got)
	}
	if got := retryDelayWithRandom(policy, 50, ^uint64(0)); got != policy.BackoffCap {
		t.Fatalf("capped delay = %v", got)
	}
}
