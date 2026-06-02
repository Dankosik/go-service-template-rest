package redpanda

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type OutboxStore interface {
	ClaimOutbox(context.Context, time.Time, int32) ([]OutboxRecord, error)
	MarkOutboxPublished(context.Context, string, time.Time) error
	MarkOutboxRetry(context.Context, string, time.Time, string) error
}

type OutboxRecord struct {
	OutboxID         string
	Topic            string
	Key              []byte
	EventType        string
	EventFingerprint string
	SafePayload      []byte
	Attempt          int
}

type OutboxRelay struct {
	Store            OutboxStore
	Producer         Producer
	Observer         EventObserver
	ProducerIdentity string
	RetryPolicy      RetryPolicy
	Now              func() time.Time
}

func (r OutboxRelay) RelayOnce(ctx context.Context, limit int32) (int, error) {
	if r.Store == nil {
		return 0, fmt.Errorf("%w: outbox store is required", ErrInvalidEvent)
	}
	if r.Producer == nil {
		return 0, fmt.Errorf("%w: producer is required", ErrInvalidEvent)
	}
	if limit <= 0 {
		limit = 100
	}
	now := r.now()
	records, err := r.Store.ClaimOutbox(ctx, now, limit)
	if err != nil {
		return 0, fmt.Errorf("%w: claim outbox: %w", ErrRetryable, err)
	}
	published := 0
	for _, record := range records {
		if err := r.validateOutboxRecord(record); err != nil {
			nextAttempt := now.Add(r.RetryPolicy.NextDelay(record.Attempt))
			if markErr := r.Store.MarkOutboxRetry(ctx, record.OutboxID, nextAttempt, "fingerprint_mismatch"); markErr != nil {
				return published, retryAfter(r.RetryPolicy, record.Attempt, fmt.Errorf("%w: mark outbox retry: %w", ErrRetryable, markErr))
			}
			r.observe(record.Topic, record.EventType, applyResultRetry, "fingerprint_mismatch")
			continue
		}
		message := ProduceMessage{
			Topic: record.Topic,
			Key:   record.Key,
			Value: record.SafePayload,
			Headers: map[string]string{
				"x-producer-identity": r.producerIdentity(),
				"x-event-type":        safeEventTypeLabel(record.EventType),
				"x-event-fingerprint": strings.TrimSpace(record.EventFingerprint),
			},
		}
		if err := r.Producer.Produce(ctx, message); err != nil {
			nextAttempt := now.Add(r.RetryPolicy.NextDelay(record.Attempt))
			if markErr := r.Store.MarkOutboxRetry(ctx, record.OutboxID, nextAttempt, "produce_retryable"); markErr != nil {
				return published, retryAfter(r.RetryPolicy, record.Attempt, fmt.Errorf("%w: mark outbox retry: %w", ErrRetryable, markErr))
			}
			r.observe(record.Topic, record.EventType, applyResultRetry, "produce_retryable")
			continue
		}
		if err := r.Store.MarkOutboxPublished(ctx, record.OutboxID, now); err != nil {
			return published, retryAfter(r.RetryPolicy, record.Attempt, fmt.Errorf("%w: mark outbox published: %w", ErrRetryable, err))
		}
		published++
		r.observe(record.Topic, record.EventType, "published", "")
	}
	return published, nil
}

func (r OutboxRelay) validateOutboxRecord(record OutboxRecord) error {
	if strings.TrimSpace(record.OutboxID) == "" || strings.TrimSpace(record.Topic) == "" || strings.TrimSpace(record.EventType) == "" {
		return fmt.Errorf("%w: outbox record incomplete", ErrInvalidEvent)
	}
	if !fingerprintMatches(record.EventFingerprint, FingerprintOutboxPayload(record.SafePayload)) {
		return fmt.Errorf("%w: outbox payload fingerprint mismatch", ErrFingerprint)
	}
	return nil
}

func (r OutboxRelay) now() time.Time {
	if r.Now != nil {
		return r.Now().UTC()
	}
	return time.Now().UTC()
}

func (r OutboxRelay) producerIdentity() string {
	if strings.TrimSpace(r.ProducerIdentity) == "" {
		return "billing-service"
	}
	return strings.TrimSpace(r.ProducerIdentity)
}

func (r OutboxRelay) observe(topic, eventType, result, reasonClass string) {
	if r.Observer == nil {
		return
	}
	r.Observer.ObserveEvent(safeTopicLabel(topic), safeEventTypeLabel(eventType), safeEventResultLabel(result), safeReasonClassLabel(reasonClass))
}
