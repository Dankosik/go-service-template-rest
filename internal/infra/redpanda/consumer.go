//nolint:dupl // Event consumers intentionally keep event-specific validation, apply, quarantine, and metric labels explicit.
package redpanda

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	eventsv1 "github.com/Dankosik/billing-service/internal/api/events/v1"
)

const (
	eventTypeTerminalSubmitted = "MicroleaseChildTerminalSubmitted"

	applyResultApplied     = "applied"
	applyResultDuplicate   = "duplicate"
	applyResultConflict    = "conflict"
	applyResultQuarantined = "quarantined"
	applyResultRetry       = "retry"
)

type TerminalApplyResult string

const (
	TerminalApplyResultApplied   TerminalApplyResult = applyResultApplied
	TerminalApplyResultDuplicate TerminalApplyResult = applyResultDuplicate
	TerminalApplyResultConflict  TerminalApplyResult = applyResultConflict
)

type TerminalEventStore interface {
	ApplyTerminalEvent(context.Context, TerminalEventCommand) (TerminalApplyResult, error)
	QuarantineEvent(context.Context, QuarantineRecord) error
}

type TerminalEventCommand struct {
	Topic                      string
	PartitionID                int32
	OffsetValue                int64
	EventID                    string
	ProducerIdentity           string
	EventFingerprint           string
	MicroleaseID               string
	AccountScopeKey            string
	ProxyAllocatorOwnerID      string
	MicroleaseGeneration       int64
	DebitAuthorizationID       string
	ChildSequence              int64
	ChildCapUSDAtoms           int64
	TerminalKind               string
	ChargedUSDAtoms            int64
	ReleasedUSDAtoms           int64
	WriteOffUSDAtoms           int64
	RequestBasisFingerprint    string
	TerminalBasisFingerprint   string
	PricingSnapshotID          string
	PricingSnapshotFingerprint string
	TerminalAt                 time.Time
	SafeMetadata               map[string]string
}

type QuarantineRecord struct {
	Topic            string
	PartitionID      int32
	OffsetValue      int64
	EventID          string
	EventType        string
	ProducerIdentity string
	EventFingerprint string
	ReasonClass      string
	SafeMetadata     map[string]string
	QuarantinedAt    time.Time
}

type TerminalConsumer struct {
	Store            TerminalEventStore
	Committer        OffsetCommitter
	Observer         EventObserver
	AllowedProducers []string
	RetryPolicy      RetryPolicy
	Now              func() time.Time
}

func (c TerminalConsumer) Handle(ctx context.Context, msg Message) error {
	now := c.now()
	event, decodeErr := decodeTerminalEvent(msg.Value)
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
	if err := c.validateTerminalEvent(event); err != nil {
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

	result, err := c.Store.ApplyTerminalEvent(ctx, terminalCommandFromEvent(msg, event))
	if err != nil {
		c.observe(msg.Topic, event.Envelope.EventType, applyResultRetry, "store_retryable")
		return retryAfter(c.RetryPolicy, msg.Attempt, fmt.Errorf("%w: apply terminal event: %w", ErrRetryable, err))
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
			return retryAfter(c.RetryPolicy, msg.Attempt, fmt.Errorf("%w: quarantine conflict: %w", ErrRetryable, err))
		}
	}
	if err := c.Committer.CommitOffset(ctx, msg); err != nil {
		c.observe(msg.Topic, event.Envelope.EventType, applyResultRetry, "offset_commit_retryable")
		return retryAfter(c.RetryPolicy, msg.Attempt, fmt.Errorf("%w: commit offset: %w", ErrRetryable, err))
	}
	c.observe(msg.Topic, event.Envelope.EventType, string(result), "")
	return nil
}

func (c TerminalConsumer) validateTerminalEvent(event eventsv1.MicroleaseChildTerminalSubmitted) error {
	if c.Store == nil {
		return fmt.Errorf("%w: terminal store is required", ErrInvalidEvent)
	}
	if c.Committer == nil {
		return fmt.Errorf("%w: offset committer is required", ErrInvalidEvent)
	}
	env := event.Envelope
	if err := validateTerminalEnvelope(env, c.AllowedProducers); err != nil {
		return err
	}
	if err := validateTerminalFingerprint(event); err != nil {
		return err
	}
	return validateTerminalBusinessFields(event)
}

func validateTerminalEnvelope(env eventsv1.MicroleaseEventEnvelope, allowedProducers []string) error {
	return validateEventEnvelope(env, eventTypeTerminalSubmitted, allowedProducers)
}

func validateEventEnvelope(env eventsv1.MicroleaseEventEnvelope, eventType string, allowedProducers []string) error {
	if env.EventID == "" || env.EventType != eventType || env.ContractVersion == "" || env.SchemaVersion == "" {
		return fmt.Errorf("%w: envelope incomplete", ErrInvalidEvent)
	}
	if !contains(allowedProducers, env.ProducerIdentity) {
		return fmt.Errorf("%w: producer not allowed", ErrProducerIdentity)
	}
	return nil
}

func validateTerminalFingerprint(event eventsv1.MicroleaseChildTerminalSubmitted) error {
	fingerprint, err := FingerprintTerminalSubmitted(event)
	if err != nil {
		return err
	}
	if !fingerprintMatches(event.Envelope.EventFingerprint, fingerprint) {
		return fmt.Errorf("%w: terminal event fingerprint mismatch", ErrFingerprint)
	}
	return nil
}

func validateTerminalBusinessFields(event eventsv1.MicroleaseChildTerminalSubmitted) error {
	if event.Identity.MicroleaseID == "" || event.Identity.AccountScopeKey == "" || event.Identity.ProxyAllocatorOwnerID == "" || event.Identity.MicroleaseGeneration <= 0 {
		return fmt.Errorf("%w: identity incomplete", ErrInvalidEvent)
	}
	if event.DebitAuthorizationID == "" || event.ChildSequence <= 0 || event.ChildCapUSDAtoms <= 0 {
		return fmt.Errorf("%w: child debit incomplete", ErrInvalidEvent)
	}
	total := nonNegative(event.ChargedUSDAtoms) + nonNegative(event.ReleasedUSDAtoms) + nonNegative(event.WriteOffUSDAtoms)
	if total > event.ChildCapUSDAtoms {
		return fmt.Errorf("%w: child cap exceeded", ErrInvalidEvent)
	}
	if event.RequestBasisFingerprint == "" || event.TerminalBasisFingerprint == "" {
		return fmt.Errorf("%w: basis fingerprint missing", ErrInvalidEvent)
	}
	if event.Pricing.PricingSnapshotID == "" || event.Pricing.SnapshotFingerprint == "" || event.Pricing.PolicyVersion == "" {
		return fmt.Errorf("%w: pricing snapshot missing", ErrInvalidEvent)
	}
	return nil
}

func (c TerminalConsumer) quarantineAndCommit(ctx context.Context, msg Message, record QuarantineRecord) error {
	if c.Store == nil {
		return fmt.Errorf("%w: terminal store is required", ErrInvalidEvent)
	}
	if c.Committer == nil {
		return fmt.Errorf("%w: offset committer is required", ErrInvalidEvent)
	}
	if err := c.Store.QuarantineEvent(ctx, record); err != nil {
		c.observe(msg.Topic, record.EventType, applyResultRetry, "quarantine_retryable")
		return retryAfter(c.RetryPolicy, msg.Attempt, fmt.Errorf("%w: quarantine event: %w", ErrRetryable, err))
	}
	if err := c.Committer.CommitOffset(ctx, msg); err != nil {
		c.observe(msg.Topic, record.EventType, applyResultRetry, "offset_commit_retryable")
		return retryAfter(c.RetryPolicy, msg.Attempt, fmt.Errorf("%w: commit quarantined offset: %w", ErrRetryable, err))
	}
	c.observe(msg.Topic, record.EventType, applyResultQuarantined, record.ReasonClass)
	return nil
}

func (c TerminalConsumer) now() time.Time {
	if c.Now != nil {
		return c.Now().UTC()
	}
	return time.Now().UTC()
}

func (c TerminalConsumer) observe(topic, eventType, result, reasonClass string) {
	if c.Observer == nil {
		return
	}
	c.Observer.ObserveEvent(safeTopicLabel(topic), safeEventTypeLabel(eventType), safeEventResultLabel(result), safeReasonClassLabel(reasonClass))
}

func decodeTerminalEvent(payload []byte) (eventsv1.MicroleaseChildTerminalSubmitted, error) {
	var event eventsv1.MicroleaseChildTerminalSubmitted
	if err := json.Unmarshal(payload, &event); err != nil {
		return eventsv1.MicroleaseChildTerminalSubmitted{}, fmt.Errorf("%w: decode terminal event: %w", ErrInvalidEvent, err)
	}
	return event, nil
}

func terminalCommandFromEvent(msg Message, event eventsv1.MicroleaseChildTerminalSubmitted) TerminalEventCommand {
	return TerminalEventCommand{
		Topic:                      msg.Topic,
		PartitionID:                msg.Partition,
		OffsetValue:                msg.Offset,
		EventID:                    event.Envelope.EventID,
		ProducerIdentity:           event.Envelope.ProducerIdentity,
		EventFingerprint:           event.Envelope.EventFingerprint,
		MicroleaseID:               event.Identity.MicroleaseID,
		AccountScopeKey:            event.Identity.AccountScopeKey,
		ProxyAllocatorOwnerID:      event.Identity.ProxyAllocatorOwnerID,
		MicroleaseGeneration:       event.Identity.MicroleaseGeneration,
		DebitAuthorizationID:       event.DebitAuthorizationID,
		ChildSequence:              event.ChildSequence,
		ChildCapUSDAtoms:           event.ChildCapUSDAtoms,
		TerminalKind:               event.TerminalKind,
		ChargedUSDAtoms:            event.ChargedUSDAtoms,
		ReleasedUSDAtoms:           event.ReleasedUSDAtoms,
		WriteOffUSDAtoms:           event.WriteOffUSDAtoms,
		RequestBasisFingerprint:    event.RequestBasisFingerprint,
		TerminalBasisFingerprint:   event.TerminalBasisFingerprint,
		PricingSnapshotID:          event.Pricing.PricingSnapshotID,
		PricingSnapshotFingerprint: event.Pricing.SnapshotFingerprint,
		TerminalAt:                 time.UnixMilli(event.ObservedTerminalEpochMS).UTC(),
		SafeMetadata: map[string]string{
			"event_type":        event.Envelope.EventType,
			"producer_identity": event.Envelope.ProducerIdentity,
		},
	}
}

func safeFailureMetadata(msg Message, eventID, eventType, reasonClass string) map[string]string {
	return map[string]string{
		"topic":        safeTopicLabel(msg.Topic),
		"partition":    fmt.Sprintf("%d", msg.Partition),
		"offset":       fmt.Sprintf("%d", msg.Offset),
		"event_id":     strings.TrimSpace(eventID),
		"event_type":   safeEventTypeLabel(eventType),
		"reason_class": safeReasonClassLabel(reasonClass),
	}
}

func reasonClass(err error) string {
	switch {
	case errors.Is(err, ErrProducerIdentity):
		return "producer_identity"
	case errors.Is(err, ErrFingerprint):
		return "fingerprint_mismatch"
	default:
		return "schema_contract_mismatch"
	}
}

func contains(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func safeTopicLabel(topic string) string {
	switch strings.TrimSpace(topic) {
	case "billing.microlease.terminal.v1":
		return "billing.microlease.terminal.v1"
	case "billing.microlease.checkpoint.v1":
		return "billing.microlease.checkpoint.v1"
	case "billing.microlease.close.v1":
		return "billing.microlease.close.v1"
	case "billing.microlease.facts.v1":
		return "billing.microlease.facts.v1"
	default:
		return "other"
	}
}

func safeEventTypeLabel(eventType string) string {
	switch strings.TrimSpace(eventType) {
	case "MicroleaseChildTerminalSubmitted", "MicroleaseCheckpointReported", "MicroleaseCloseReported", "MicroleaseIssued", "MicroleaseTerminalApplied", "MicroleaseClosed", "MicroleaseAdmissionRejected":
		return strings.TrimSpace(eventType)
	default:
		return "unknown"
	}
}

func safeEventResultLabel(result string) string {
	switch strings.TrimSpace(result) {
	case applyResultApplied, applyResultDuplicate, applyResultConflict, applyResultQuarantined, applyResultRetry, "published":
		return strings.TrimSpace(result)
	default:
		return "other"
	}
}

func safeReasonClassLabel(reason string) string {
	switch strings.TrimSpace(reason) {
	case "", "schema_contract_mismatch", "producer_identity", "fingerprint_mismatch", "payload_conflict", "store_retryable", "quarantine_retryable", "offset_commit_retryable", "produce_retryable":
		return strings.TrimSpace(reason)
	default:
		return "other"
	}
}

func nonNegative(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}
