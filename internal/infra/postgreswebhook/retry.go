package postgreswebhook

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/riverqueue/river"
)

const (
	webhookInitialBackoff = 5 * time.Second
	webhookMaxBackoff     = 24 * time.Hour
	webhookMaxElapsed     = 4 * 24 * time.Hour
	retryAfterMetadataKey = "webhook_retry_after_at"
)

type transportEvidence struct {
	StatusCode        int
	DefinitelyNotSent bool
	MayHaveSent       bool
	LocalDenial       bool
}

func parseRetryAfter(raw, date string, attemptedAt time.Time, maxDelay time.Duration) (time.Duration, bool) {
	if maxDelay <= 0 {
		return 0, false
	}
	raw = strings.TrimSpace(raw)
	if raw != "" && strings.IndexFunc(raw, func(r rune) bool { return r < '0' || r > '9' }) < 0 {
		seconds, err := strconv.ParseUint(raw, 10, 63)
		if err != nil || seconds > uint64(math.MaxInt64/int64(time.Second)) {
			return maxDelay, true
		}
		return min(time.Duration(seconds)*time.Second, maxDelay), true
	}
	when, err := http.ParseTime(raw)
	if err != nil {
		return 0, false
	}
	base := attemptedAt
	if parsedDate, parseErr := http.ParseTime(date); parseErr == nil {
		base = parsedDate
	}
	if !when.After(base) {
		return 0, false
	}
	return min(when.Sub(base), maxDelay), true
}

func rememberRetryAfter(ctx context.Context, job *river.Job[deliveryArgs], retryAt time.Time) error {
	if job == nil || retryAt.IsZero() {
		return fmt.Errorf("%w: webhook retry job and time are required", ErrConfig)
	}
	metadata, err := mergeRetryAfterMetadata(job.Metadata, retryAt.UTC())
	if err != nil {
		return err
	}
	if err := river.MetadataSet(ctx, retryAfterMetadataKey, retryAt.UTC()); err != nil {
		return fmt.Errorf("record webhook Retry-After: %w", err)
	}
	// NextRetry runs before River merges MetadataSet into the durable row, so keep
	// this attempt's in-memory view aligned with the pending durable update.
	job.Metadata = metadata
	return nil
}

func mergeRetryAfterMetadata(raw []byte, retryAt time.Time) ([]byte, error) {
	metadata := make(map[string]json.RawMessage)
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &metadata); err != nil {
			return nil, fmt.Errorf("decode webhook job metadata: %w", err)
		}
	}
	if metadata == nil {
		metadata = make(map[string]json.RawMessage)
	}
	encoded, err := json.Marshal(retryAt.UTC())
	if err != nil {
		return nil, fmt.Errorf("encode webhook Retry-After: %w", err)
	}
	metadata[retryAfterMetadataKey] = encoded
	merged, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("encode webhook job metadata: %w", err)
	}
	return merged, nil
}

func webhookNextRetry(job *river.Job[deliveryArgs], now time.Time) time.Time {
	if job == nil {
		return time.Time{}
	}
	now = now.UTC()
	due := now.Add(webhookBackoff(job.Args.DeliveryID, job.Attempt))
	var metadata struct {
		RetryAfterAt time.Time `json:"webhook_retry_after_at"`
	}
	if json.Unmarshal(job.Metadata, &metadata) == nil && metadata.RetryAfterAt.After(due) {
		due = metadata.RetryAfterAt
	}
	if !job.CreatedAt.IsZero() {
		deadline := job.CreatedAt.Add(webhookMaxElapsed)
		if !deadline.After(now) {
			return now.Add(time.Second)
		}
		if deadline.Before(due) {
			due = deadline
		}
	}
	return due
}

func webhookDeliveryExpired(job *river.Job[deliveryArgs], now time.Time) bool {
	return job != nil && !job.CreatedAt.IsZero() && !now.Before(job.CreatedAt.Add(webhookMaxElapsed))
}

func webhookBackoff(deliveryID string, attempt int) time.Duration {
	attempt = max(attempt, 1)
	delay := webhookInitialBackoff
	for current := 1; current < attempt && delay < webhookMaxBackoff; current++ {
		delay = min(2*delay, webhookMaxBackoff)
	}
	var encodedAttempt [8]byte
	binary.BigEndian.PutUint64(encodedAttempt[:], uint64(attempt))
	hash := sha256.New()
	_, _ = hash.Write([]byte(deliveryID))
	_, _ = hash.Write(encodedAttempt[:])
	jitterPermille := int64(binary.BigEndian.Uint64(hash.Sum(nil)[:8])%201) - 100
	delay += time.Duration(int64(delay) * jitterPermille / 1000)
	return min(delay, webhookMaxBackoff)
}
