//nolint:dupl // Event consumers intentionally keep event-specific validation, apply, quarantine, and metric labels explicit.
package redpanda

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	eventsv1 "github.com/Dankosik/billing-service/internal/api/events/v1"
)

const (
	eventTypeCheckpointReported = "MicroleaseCheckpointReported"
	eventTypeCloseReported      = "MicroleaseCloseReported"
)

type CheckpointEventStore interface {
	ApplyCheckpointEvent(context.Context, CheckpointEventCommand) (TerminalApplyResult, error)
	QuarantineEvent(context.Context, QuarantineRecord) error
}

type CloseEventStore interface {
	ApplyCloseEvent(context.Context, CloseEventCommand) (TerminalApplyResult, error)
	QuarantineEvent(context.Context, QuarantineRecord) error
}

type CheckpointEventCommand struct {
	Topic                         string
	PartitionID                   int32
	OffsetValue                   int64
	EventID                       string
	ProducerIdentity              string
	EventFingerprint              string
	MicroleaseID                  string
	AccountScopeKey               string
	ProxyAllocatorOwnerID         string
	MicroleaseGeneration          int64
	CheckpointSequence            int64
	CheckpointKind                string
	AllocatedChildHighWater       int64
	AllocatedChildCount           int64
	AllocatedChildCapSumUSDAtoms  int64
	TerminalSubmittedCount        int64
	TerminalPublishedCount        int64
	TerminalAcceptedCount         int64
	UnresolvedChildCount          int64
	UnresolvedChildCapSumUSDAtoms int64
	LocalRemainingUSDAtoms        int64
	CheckpointFingerprint         string
	CreatedAt                     time.Time
	SafeMetadata                  map[string]string
}

type CloseEventCommand struct {
	CheckpointEventCommand
	CloseReason                string
	AllocatorClosedAt          time.Time
	FinalLocalStateFingerprint string
}

type CheckpointConsumer struct {
	Store            CheckpointEventStore
	Committer        OffsetCommitter
	Observer         EventObserver
	AllowedProducers []string
	RetryPolicy      RetryPolicy
	Now              func() time.Time
}

func (c CheckpointConsumer) Handle(ctx context.Context, msg Message) error {
	now := c.now()
	event, decodeErr := decodeCheckpointEvent(msg.Value)
	if decodeErr != nil {
		return c.quarantineAndCommit(ctx, msg, QuarantineRecord{
			Topic:         msg.Topic,
			PartitionID:   msg.Partition,
			OffsetValue:   msg.Offset,
			ReasonClass:   "schema_contract_mismatch",
			SafeMetadata:  safeFailureMetadata(msg, "", "", "schema_contract_mismatch"),
			QuarantinedAt: now,
		})
	}
	if err := c.validateCheckpointEvent(event); err != nil {
		reason := reasonClass(err)
		return c.quarantineAndCommit(ctx, msg, QuarantineRecord{
			Topic:            msg.Topic,
			PartitionID:      msg.Partition,
			OffsetValue:      msg.Offset,
			EventID:          event.Envelope.EventID,
			EventType:        event.Envelope.EventType,
			ProducerIdentity: event.Envelope.ProducerIdentity,
			EventFingerprint: event.Envelope.EventFingerprint,
			ReasonClass:      reason,
			SafeMetadata:     safeFailureMetadata(msg, event.Envelope.EventID, event.Envelope.EventType, reason),
			QuarantinedAt:    now,
		})
	}
	result, err := c.Store.ApplyCheckpointEvent(ctx, checkpointCommandFromEvent(msg, event))
	if err != nil {
		c.observe(msg.Topic, event.Envelope.EventType, applyResultRetry, "store_retryable")
		return retryAfter(c.RetryPolicy, msg.Attempt, fmt.Errorf("%w: apply checkpoint event: %w", ErrRetryable, err))
	}
	if result == TerminalApplyResultConflict {
		if err := c.Store.QuarantineEvent(ctx, QuarantineRecord{
			Topic:            msg.Topic,
			PartitionID:      msg.Partition,
			OffsetValue:      msg.Offset,
			EventID:          event.Envelope.EventID,
			EventType:        event.Envelope.EventType,
			ProducerIdentity: event.Envelope.ProducerIdentity,
			EventFingerprint: event.Envelope.EventFingerprint,
			ReasonClass:      "payload_conflict",
			SafeMetadata:     safeFailureMetadata(msg, event.Envelope.EventID, event.Envelope.EventType, "payload_conflict"),
			QuarantinedAt:    now,
		}); err != nil {
			c.observe(msg.Topic, event.Envelope.EventType, applyResultRetry, "quarantine_retryable")
			return retryAfter(c.RetryPolicy, msg.Attempt, fmt.Errorf("%w: quarantine checkpoint conflict: %w", ErrRetryable, err))
		}
	}
	if err := c.Committer.CommitOffset(ctx, msg); err != nil {
		c.observe(msg.Topic, event.Envelope.EventType, applyResultRetry, "offset_commit_retryable")
		return retryAfter(c.RetryPolicy, msg.Attempt, fmt.Errorf("%w: commit checkpoint offset: %w", ErrRetryable, err))
	}
	c.observe(msg.Topic, event.Envelope.EventType, string(result), "")
	return nil
}

func (c CheckpointConsumer) validateCheckpointEvent(event eventsv1.MicroleaseCheckpointReported) error {
	if c.Store == nil {
		return fmt.Errorf("%w: checkpoint store is required", ErrInvalidEvent)
	}
	if c.Committer == nil {
		return fmt.Errorf("%w: offset committer is required", ErrInvalidEvent)
	}
	if err := validateEventEnvelope(event.Envelope, eventTypeCheckpointReported, c.AllowedProducers); err != nil {
		return err
	}
	fingerprint, err := FingerprintCheckpointReported(event)
	if err != nil {
		return err
	}
	if !fingerprintMatches(event.Envelope.EventFingerprint, fingerprint) {
		return fmt.Errorf("%w: checkpoint event fingerprint mismatch", ErrFingerprint)
	}
	return validateCheckpointBusinessFields(event)
}

func (c CheckpointConsumer) quarantineAndCommit(ctx context.Context, msg Message, record QuarantineRecord) error {
	if c.Store == nil {
		return fmt.Errorf("%w: checkpoint store is required", ErrInvalidEvent)
	}
	if c.Committer == nil {
		return fmt.Errorf("%w: offset committer is required", ErrInvalidEvent)
	}
	if err := c.Store.QuarantineEvent(ctx, record); err != nil {
		c.observe(msg.Topic, record.EventType, applyResultRetry, "quarantine_retryable")
		return retryAfter(c.RetryPolicy, msg.Attempt, fmt.Errorf("%w: quarantine checkpoint event: %w", ErrRetryable, err))
	}
	if err := c.Committer.CommitOffset(ctx, msg); err != nil {
		c.observe(msg.Topic, record.EventType, applyResultRetry, "offset_commit_retryable")
		return retryAfter(c.RetryPolicy, msg.Attempt, fmt.Errorf("%w: commit quarantined checkpoint offset: %w", ErrRetryable, err))
	}
	c.observe(msg.Topic, record.EventType, applyResultQuarantined, record.ReasonClass)
	return nil
}

func (c CheckpointConsumer) now() time.Time {
	if c.Now != nil {
		return c.Now().UTC()
	}
	return time.Now().UTC()
}

func (c CheckpointConsumer) observe(topic, eventType, result, reasonClass string) {
	if c.Observer != nil {
		c.Observer.ObserveEvent(safeTopicLabel(topic), safeEventTypeLabel(eventType), safeEventResultLabel(result), safeReasonClassLabel(reasonClass))
	}
}

type CloseConsumer struct {
	Store            CloseEventStore
	Committer        OffsetCommitter
	Observer         EventObserver
	AllowedProducers []string
	RetryPolicy      RetryPolicy
	Now              func() time.Time
}

func (c CloseConsumer) Handle(ctx context.Context, msg Message) error {
	now := c.now()
	event, decodeErr := decodeCloseEvent(msg.Value)
	if decodeErr != nil {
		return c.quarantineAndCommit(ctx, msg, QuarantineRecord{
			Topic:         msg.Topic,
			PartitionID:   msg.Partition,
			OffsetValue:   msg.Offset,
			ReasonClass:   "schema_contract_mismatch",
			SafeMetadata:  safeFailureMetadata(msg, "", "", "schema_contract_mismatch"),
			QuarantinedAt: now,
		})
	}
	if err := c.validateCloseEvent(event); err != nil {
		reason := reasonClass(err)
		env := event.Checkpoint.Envelope
		return c.quarantineAndCommit(ctx, msg, QuarantineRecord{
			Topic:            msg.Topic,
			PartitionID:      msg.Partition,
			OffsetValue:      msg.Offset,
			EventID:          env.EventID,
			EventType:        env.EventType,
			ProducerIdentity: env.ProducerIdentity,
			EventFingerprint: env.EventFingerprint,
			ReasonClass:      reason,
			SafeMetadata:     safeFailureMetadata(msg, env.EventID, env.EventType, reason),
			QuarantinedAt:    now,
		})
	}
	result, err := c.Store.ApplyCloseEvent(ctx, closeCommandFromEvent(msg, event))
	env := event.Checkpoint.Envelope
	if err != nil {
		c.observe(msg.Topic, env.EventType, applyResultRetry, "store_retryable")
		return retryAfter(c.RetryPolicy, msg.Attempt, fmt.Errorf("%w: apply close event: %w", ErrRetryable, err))
	}
	if result == TerminalApplyResultConflict {
		if err := c.Store.QuarantineEvent(ctx, QuarantineRecord{
			Topic:            msg.Topic,
			PartitionID:      msg.Partition,
			OffsetValue:      msg.Offset,
			EventID:          env.EventID,
			EventType:        env.EventType,
			ProducerIdentity: env.ProducerIdentity,
			EventFingerprint: env.EventFingerprint,
			ReasonClass:      "payload_conflict",
			SafeMetadata:     safeFailureMetadata(msg, env.EventID, env.EventType, "payload_conflict"),
			QuarantinedAt:    now,
		}); err != nil {
			c.observe(msg.Topic, env.EventType, applyResultRetry, "quarantine_retryable")
			return retryAfter(c.RetryPolicy, msg.Attempt, fmt.Errorf("%w: quarantine close conflict: %w", ErrRetryable, err))
		}
	}
	if err := c.Committer.CommitOffset(ctx, msg); err != nil {
		c.observe(msg.Topic, env.EventType, applyResultRetry, "offset_commit_retryable")
		return retryAfter(c.RetryPolicy, msg.Attempt, fmt.Errorf("%w: commit close offset: %w", ErrRetryable, err))
	}
	c.observe(msg.Topic, env.EventType, string(result), "")
	return nil
}

func (c CloseConsumer) validateCloseEvent(event eventsv1.MicroleaseCloseReported) error {
	if c.Store == nil {
		return fmt.Errorf("%w: close store is required", ErrInvalidEvent)
	}
	if c.Committer == nil {
		return fmt.Errorf("%w: offset committer is required", ErrInvalidEvent)
	}
	env := event.Checkpoint.Envelope
	if err := validateEventEnvelope(env, eventTypeCloseReported, c.AllowedProducers); err != nil {
		return err
	}
	fingerprint, err := FingerprintCloseReported(event)
	if err != nil {
		return err
	}
	if !fingerprintMatches(env.EventFingerprint, fingerprint) {
		return fmt.Errorf("%w: close event fingerprint mismatch", ErrFingerprint)
	}
	if event.CloseReason == "" || event.FinalLocalStateFingerprint == "" || event.AllocatorClosedEpochMS <= 0 {
		return fmt.Errorf("%w: close proof incomplete", ErrInvalidEvent)
	}
	return validateCheckpointBusinessFields(event.Checkpoint)
}

func (c CloseConsumer) quarantineAndCommit(ctx context.Context, msg Message, record QuarantineRecord) error {
	if c.Store == nil {
		return fmt.Errorf("%w: close store is required", ErrInvalidEvent)
	}
	if c.Committer == nil {
		return fmt.Errorf("%w: offset committer is required", ErrInvalidEvent)
	}
	if err := c.Store.QuarantineEvent(ctx, record); err != nil {
		c.observe(msg.Topic, record.EventType, applyResultRetry, "quarantine_retryable")
		return retryAfter(c.RetryPolicy, msg.Attempt, fmt.Errorf("%w: quarantine close event: %w", ErrRetryable, err))
	}
	if err := c.Committer.CommitOffset(ctx, msg); err != nil {
		c.observe(msg.Topic, record.EventType, applyResultRetry, "offset_commit_retryable")
		return retryAfter(c.RetryPolicy, msg.Attempt, fmt.Errorf("%w: commit quarantined close offset: %w", ErrRetryable, err))
	}
	c.observe(msg.Topic, record.EventType, applyResultQuarantined, record.ReasonClass)
	return nil
}

func (c CloseConsumer) now() time.Time {
	if c.Now != nil {
		return c.Now().UTC()
	}
	return time.Now().UTC()
}

func (c CloseConsumer) observe(topic, eventType, result, reasonClass string) {
	if c.Observer != nil {
		c.Observer.ObserveEvent(safeTopicLabel(topic), safeEventTypeLabel(eventType), safeEventResultLabel(result), safeReasonClassLabel(reasonClass))
	}
}

func decodeCheckpointEvent(payload []byte) (eventsv1.MicroleaseCheckpointReported, error) {
	var event eventsv1.MicroleaseCheckpointReported
	if err := json.Unmarshal(payload, &event); err != nil {
		return eventsv1.MicroleaseCheckpointReported{}, fmt.Errorf("%w: decode checkpoint event: %w", ErrInvalidEvent, err)
	}
	return event, nil
}

func decodeCloseEvent(payload []byte) (eventsv1.MicroleaseCloseReported, error) {
	var event eventsv1.MicroleaseCloseReported
	if err := json.Unmarshal(payload, &event); err != nil {
		return eventsv1.MicroleaseCloseReported{}, fmt.Errorf("%w: decode close event: %w", ErrInvalidEvent, err)
	}
	return event, nil
}

func validateCheckpointBusinessFields(event eventsv1.MicroleaseCheckpointReported) error {
	if event.Identity.MicroleaseID == "" || event.Identity.AccountScopeKey == "" || event.Identity.ProxyAllocatorOwnerID == "" || event.Identity.MicroleaseGeneration <= 0 {
		return fmt.Errorf("%w: identity incomplete", ErrInvalidEvent)
	}
	if event.CheckpointSequence <= 0 || event.CheckpointKind == "" || event.CheckpointFingerprint == "" {
		return fmt.Errorf("%w: checkpoint incomplete", ErrInvalidEvent)
	}
	if event.AllocatedChildCount < 0 || event.AllocatedChildCapSumUSDAtoms < 0 || event.TerminalSubmittedCount < 0 || event.TerminalPublishedCount < 0 || event.TerminalAcceptedCount < 0 || event.UnresolvedChildCount < 0 || event.UnresolvedChildCapSumUSDAtoms < 0 || event.LocalRemainingUSDAtoms < 0 {
		return fmt.Errorf("%w: checkpoint counters invalid", ErrInvalidEvent)
	}
	return nil
}

func checkpointCommandFromEvent(msg Message, event eventsv1.MicroleaseCheckpointReported) CheckpointEventCommand {
	return CheckpointEventCommand{
		Topic:                         msg.Topic,
		PartitionID:                   msg.Partition,
		OffsetValue:                   msg.Offset,
		EventID:                       event.Envelope.EventID,
		ProducerIdentity:              event.Envelope.ProducerIdentity,
		EventFingerprint:              event.Envelope.EventFingerprint,
		MicroleaseID:                  event.Identity.MicroleaseID,
		AccountScopeKey:               event.Identity.AccountScopeKey,
		ProxyAllocatorOwnerID:         event.Identity.ProxyAllocatorOwnerID,
		MicroleaseGeneration:          event.Identity.MicroleaseGeneration,
		CheckpointSequence:            event.CheckpointSequence,
		CheckpointKind:                event.CheckpointKind,
		AllocatedChildHighWater:       event.AllocatedChildHighWater,
		AllocatedChildCount:           event.AllocatedChildCount,
		AllocatedChildCapSumUSDAtoms:  event.AllocatedChildCapSumUSDAtoms,
		TerminalSubmittedCount:        event.TerminalSubmittedCount,
		TerminalPublishedCount:        event.TerminalPublishedCount,
		TerminalAcceptedCount:         event.TerminalAcceptedCount,
		UnresolvedChildCount:          event.UnresolvedChildCount,
		UnresolvedChildCapSumUSDAtoms: event.UnresolvedChildCapSumUSDAtoms,
		LocalRemainingUSDAtoms:        event.LocalRemainingUSDAtoms,
		CheckpointFingerprint:         event.CheckpointFingerprint,
		CreatedAt:                     time.UnixMilli(event.Envelope.ProducedAtEpochMS).UTC(),
		SafeMetadata: map[string]string{
			"event_type":        event.Envelope.EventType,
			"producer_identity": event.Envelope.ProducerIdentity,
		},
	}
}

func closeCommandFromEvent(msg Message, event eventsv1.MicroleaseCloseReported) CloseEventCommand {
	command := checkpointCommandFromEvent(msg, event.Checkpoint)
	return CloseEventCommand{
		CheckpointEventCommand:     command,
		CloseReason:                event.CloseReason,
		AllocatorClosedAt:          time.UnixMilli(event.AllocatorClosedEpochMS).UTC(),
		FinalLocalStateFingerprint: event.FinalLocalStateFingerprint,
	}
}
