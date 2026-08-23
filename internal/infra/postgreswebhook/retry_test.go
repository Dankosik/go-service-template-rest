package postgreswebhook

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

func TestParseRetryAfter(t *testing.T) {
	base := time.Unix(1_700_000_000, 0).UTC()
	if got, ok := parseRetryAfter("120", "", base, time.Minute); !ok || got != time.Minute {
		t.Fatalf("delta Retry-After = %s, %t", got, ok)
	}
	date := base.Add(30 * time.Second).Format(http.TimeFormat)
	if got, ok := parseRetryAfter(date, base.Format(http.TimeFormat), base, time.Minute); !ok || got != 30*time.Second {
		t.Fatalf("date Retry-After = %s, %t", got, ok)
	}
	if _, ok := parseRetryAfter("invalid", "", base, time.Minute); ok {
		t.Fatal("invalid Retry-After accepted")
	}
}

func TestWebhookRetrySchedule(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	job := &river.Job[deliveryArgs]{
		JobRow: &rivertype.JobRow{Attempt: 3, CreatedAt: now.Add(-time.Hour), Metadata: []byte(`{"other":true}`)},
		Args:   deliveryArgs{DeliveryID: "whd_retry"},
	}

	first := webhookNextRetry(job, now)
	if second := webhookNextRetry(job, now); !second.Equal(first) {
		t.Fatalf("retry schedule is not deterministic: %s != %s", first, second)
	}
	if delay := first.Sub(now); delay < 18*time.Second || delay > 22*time.Second {
		t.Fatalf("attempt 3 delay = %s, want 20s with bounded 10%% jitter", delay)
	}
	job.Attempt = 20
	if delay := webhookNextRetry(job, now).Sub(now); delay > webhookMaxBackoff {
		t.Fatalf("attempt 20 delay = %s, want at most %s", delay, webhookMaxBackoff)
	}
	job.Attempt = 3

	hint := now.Add(time.Hour)
	metadata, err := mergeRetryAfterMetadata(job.Metadata, hint)
	if err != nil {
		t.Fatal(err)
	}
	job.Metadata = metadata
	if got := webhookNextRetry(job, now); !got.Equal(hint) {
		t.Fatalf("retry with Retry-After = %s, want %s", got, hint)
	}
	var preserved struct {
		Other bool `json:"other"`
	}
	if err := json.Unmarshal(metadata, &preserved); err != nil || !preserved.Other {
		t.Fatalf("metadata merge lost existing values: %s, %v", metadata, err)
	}

	deadline := job.CreatedAt.Add(webhookMaxElapsed)
	job.Metadata, err = mergeRetryAfterMetadata(job.Metadata, deadline.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if got := webhookNextRetry(job, now); !got.Equal(deadline) {
		t.Fatalf("retry past delivery deadline = %s, want %s", got, deadline)
	}
	if !webhookDeliveryExpired(job, deadline) {
		t.Fatal("delivery remained active at its four-day deadline")
	}
}
