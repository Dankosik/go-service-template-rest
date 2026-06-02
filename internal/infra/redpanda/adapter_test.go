package redpanda

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	eventsv1 "github.com/Dankosik/billing-service/internal/api/events/v1"
)

type fakeTerminalStore struct {
	result        TerminalApplyResult
	err           error
	quarantineErr error
	commands      []TerminalEventCommand
	quarantines   []QuarantineRecord
	order         *[]string
}

func (s *fakeTerminalStore) ApplyTerminalEvent(_ context.Context, cmd TerminalEventCommand) (TerminalApplyResult, error) {
	if s.order != nil {
		*s.order = append(*s.order, "apply")
	}
	s.commands = append(s.commands, cmd)
	if s.err != nil {
		return "", s.err
	}
	if s.result == "" {
		return TerminalApplyResultApplied, nil
	}
	return s.result, nil
}

func (s *fakeTerminalStore) QuarantineEvent(_ context.Context, record QuarantineRecord) error {
	if s.order != nil {
		*s.order = append(*s.order, "quarantine")
	}
	s.quarantines = append(s.quarantines, record)
	return s.quarantineErr
}

type fakeCommitter struct {
	commits []Message
	err     error
	order   *[]string
}

func (c *fakeCommitter) CommitOffset(_ context.Context, msg Message) error {
	if c.order != nil {
		*c.order = append(*c.order, "commit")
	}
	c.commits = append(c.commits, msg)
	return c.err
}

type capturedEventObserver struct {
	labels []string
}

func (o *capturedEventObserver) ObserveEvent(topic, eventType, result, reasonClass string) {
	o.labels = append(o.labels, topic+"|"+eventType+"|"+result+"|"+reasonClass)
}

type fakeCheckpointStore struct {
	result        TerminalApplyResult
	err           error
	quarantineErr error
	commands      []CheckpointEventCommand
	quarantines   []QuarantineRecord
	order         *[]string
}

func (s *fakeCheckpointStore) ApplyCheckpointEvent(_ context.Context, cmd CheckpointEventCommand) (TerminalApplyResult, error) {
	if s.order != nil {
		*s.order = append(*s.order, "apply")
	}
	s.commands = append(s.commands, cmd)
	if s.err != nil {
		return "", s.err
	}
	if s.result == "" {
		return TerminalApplyResultApplied, nil
	}
	return s.result, nil
}

func (s *fakeCheckpointStore) QuarantineEvent(_ context.Context, record QuarantineRecord) error {
	if s.order != nil {
		*s.order = append(*s.order, "quarantine")
	}
	s.quarantines = append(s.quarantines, record)
	return s.quarantineErr
}

type fakeCloseStore struct {
	result        TerminalApplyResult
	err           error
	quarantineErr error
	commands      []CloseEventCommand
	quarantines   []QuarantineRecord
	order         *[]string
}

func (s *fakeCloseStore) ApplyCloseEvent(_ context.Context, cmd CloseEventCommand) (TerminalApplyResult, error) {
	if s.order != nil {
		*s.order = append(*s.order, "apply")
	}
	s.commands = append(s.commands, cmd)
	if s.err != nil {
		return "", s.err
	}
	if s.result == "" {
		return TerminalApplyResultApplied, nil
	}
	return s.result, nil
}

func (s *fakeCloseStore) QuarantineEvent(_ context.Context, record QuarantineRecord) error {
	if s.order != nil {
		*s.order = append(*s.order, "quarantine")
	}
	s.quarantines = append(s.quarantines, record)
	return s.quarantineErr
}

func TestCheckpointConsumerAppliesBeforeCommittingOffset(t *testing.T) {
	t.Parallel()

	var order []string
	store := &fakeCheckpointStore{order: &order}
	committer := &fakeCommitter{order: &order}
	observer := &capturedEventObserver{}
	consumer := CheckpointConsumer{
		Store:            store,
		Committer:        committer,
		Observer:         observer,
		AllowedProducers: []string{"gonka-proxy"},
		Now:              fixedAdapterTime,
	}

	if err := consumer.Handle(context.Background(), validCheckpointMessage(t)); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if got, want := strings.Join(order, ","), "apply,commit"; got != want {
		t.Fatalf("order = %s, want %s", got, want)
	}
	if len(store.commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(store.commands))
	}
	cmd := store.commands[0]
	if cmd.MicroleaseID != "11111111-1111-1111-1111-111111111111" || cmd.CheckpointSequence != 3 || cmd.UnresolvedChildCapSumUSDAtoms != 1_000_000 {
		t.Fatalf("checkpoint command = %+v, want durable checkpoint identity and exposure", cmd)
	}
	if len(committer.commits) != 1 {
		t.Fatalf("commits = %d, want 1", len(committer.commits))
	}
	assertObserverLabelsSafe(t, observer.labels)
}

func TestCheckpointConsumerQuarantinesInvalidEventsBeforeCommit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		message    Message
		wantReason string
	}{
		{
			name:       "schema",
			message:    Message{Topic: validCheckpointTopic, Partition: 1, Offset: 44, Value: []byte(`{"bad":`)},
			wantReason: "schema_contract_mismatch",
		},
		{
			name: "fingerprint",
			message: func() Message {
				event := validCheckpointEvent(t)
				event.Envelope.EventFingerprint = "sha256:bad"
				return checkpointMessageFromEvent(t, event)
			}(),
			wantReason: "fingerprint_mismatch",
		},
		{
			name: "negative counter",
			message: func() Message {
				event := validCheckpointEvent(t)
				event.UnresolvedChildCount = -1
				event.Envelope.EventFingerprint = mustCheckpointFingerprint(t, event)
				return checkpointMessageFromEvent(t, event)
			}(),
			wantReason: "schema_contract_mismatch",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := &fakeCheckpointStore{}
			committer := &fakeCommitter{}
			observer := &capturedEventObserver{}
			consumer := CheckpointConsumer{
				Store:            store,
				Committer:        committer,
				Observer:         observer,
				AllowedProducers: []string{"gonka-proxy"},
				Now:              fixedAdapterTime,
			}

			if err := consumer.Handle(context.Background(), tc.message); err != nil {
				t.Fatalf("Handle() error = %v", err)
			}
			if len(store.commands) != 0 {
				t.Fatalf("commands = %d, want 0", len(store.commands))
			}
			if len(store.quarantines) != 1 {
				t.Fatalf("quarantines = %d, want 1", len(store.quarantines))
			}
			if store.quarantines[0].ReasonClass != tc.wantReason {
				t.Fatalf("reason = %q, want %q", store.quarantines[0].ReasonClass, tc.wantReason)
			}
			assertMetadataSafe(t, store.quarantines[0].SafeMetadata)
			if len(committer.commits) != 1 {
				t.Fatalf("commits = %d, want 1", len(committer.commits))
			}
			assertObserverLabelsSafe(t, observer.labels)
		})
	}
}

func TestCheckpointConsumerRetriesWithoutCommitOnStoreFailure(t *testing.T) {
	t.Parallel()

	store := &fakeCheckpointStore{err: errors.New("checkpoint store unavailable")}
	committer := &fakeCommitter{}
	consumer := CheckpointConsumer{
		Store:            store,
		Committer:        committer,
		AllowedProducers: []string{"gonka-proxy"},
		RetryPolicy:      RetryPolicy{BaseDelay: 10 * time.Millisecond, MaxDelay: 50 * time.Millisecond},
		Now:              fixedAdapterTime,
	}

	err := consumer.Handle(context.Background(), Message{
		Topic:     validCheckpointTopic,
		Partition: 1,
		Offset:    45,
		Value:     validCheckpointMessage(t).Value,
		Attempt:   4,
	})
	assertBoundedRetryWithoutCommit(t, err, committer)
}

func TestCloseConsumerAppliesConflictAndRetryPaths(t *testing.T) {
	t.Parallel()

	t.Run("applies before commit", func(t *testing.T) {
		t.Parallel()

		var order []string
		store := &fakeCloseStore{order: &order}
		committer := &fakeCommitter{order: &order}
		observer := &capturedEventObserver{}
		consumer := CloseConsumer{
			Store:            store,
			Committer:        committer,
			Observer:         observer,
			AllowedProducers: []string{"gonka-proxy"},
			Now:              fixedAdapterTime,
		}

		if err := consumer.Handle(context.Background(), validCloseMessage(t)); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		if got, want := strings.Join(order, ","), "apply,commit"; got != want {
			t.Fatalf("order = %s, want %s", got, want)
		}
		if len(store.commands) != 1 || store.commands[0].CloseReason != "normal_close" || store.commands[0].FinalLocalStateFingerprint == "" {
			t.Fatalf("close commands = %+v, want close proof", store.commands)
		}
		assertObserverLabelsSafe(t, observer.labels)
	})

	t.Run("conflict is quarantined then committed", func(t *testing.T) {
		t.Parallel()

		store := &fakeCloseStore{result: TerminalApplyResultConflict}
		committer := &fakeCommitter{}
		observer := &capturedEventObserver{}
		consumer := CloseConsumer{
			Store:            store,
			Committer:        committer,
			Observer:         observer,
			AllowedProducers: []string{"gonka-proxy"},
			Now:              fixedAdapterTime,
		}

		if err := consumer.Handle(context.Background(), validCloseMessage(t)); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		if len(store.quarantines) != 1 || store.quarantines[0].ReasonClass != "payload_conflict" {
			t.Fatalf("quarantines = %+v, want payload conflict", store.quarantines)
		}
		if len(committer.commits) != 1 {
			t.Fatalf("commits = %d, want 1", len(committer.commits))
		}
	})

	t.Run("store failure leaves offset replayable", func(t *testing.T) {
		t.Parallel()

		store := &fakeCloseStore{err: errors.New("close store unavailable")}
		committer := &fakeCommitter{}
		consumer := CloseConsumer{
			Store:            store,
			Committer:        committer,
			AllowedProducers: []string{"gonka-proxy"},
			RetryPolicy:      RetryPolicy{BaseDelay: 10 * time.Millisecond, MaxDelay: 50 * time.Millisecond},
			Now:              fixedAdapterTime,
		}

		err := consumer.Handle(context.Background(), Message{
			Topic:     validCloseTopic,
			Partition: 2,
			Offset:    55,
			Value:     validCloseMessage(t).Value,
			Attempt:   4,
		})
		assertBoundedRetryWithoutCommit(t, err, committer)
	})
}

func TestCloseConsumerQuarantinesMissingCloseProof(t *testing.T) {
	t.Parallel()

	event := validCloseEvent(t)
	event.CloseReason = ""
	event.Checkpoint.Envelope.EventFingerprint = mustCloseFingerprint(t, event)
	store := &fakeCloseStore{}
	committer := &fakeCommitter{}
	consumer := CloseConsumer{
		Store:            store,
		Committer:        committer,
		AllowedProducers: []string{"gonka-proxy"},
		Now:              fixedAdapterTime,
	}

	if err := consumer.Handle(context.Background(), closeMessageFromEvent(t, event)); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(store.commands) != 0 {
		t.Fatalf("commands = %d, want 0", len(store.commands))
	}
	if len(store.quarantines) != 1 || store.quarantines[0].ReasonClass != "schema_contract_mismatch" {
		t.Fatalf("quarantines = %+v, want schema_contract_mismatch", store.quarantines)
	}
	assertMetadataSafe(t, store.quarantines[0].SafeMetadata)
	if len(committer.commits) != 1 {
		t.Fatalf("commits = %d, want 1", len(committer.commits))
	}
}

func TestTerminalConsumerAppliesBeforeCommittingOffset(t *testing.T) {
	t.Parallel()

	var order []string
	store := &fakeTerminalStore{order: &order}
	committer := &fakeCommitter{order: &order}
	observer := &capturedEventObserver{}
	consumer := TerminalConsumer{
		Store:            store,
		Committer:        committer,
		Observer:         observer,
		AllowedProducers: []string{"gonka-proxy"},
		Now:              fixedAdapterTime,
	}

	msg := validTerminalMessage(t)
	if err := consumer.Handle(context.Background(), msg); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if got, want := strings.Join(order, ","), "apply,commit"; got != want {
		t.Fatalf("order = %s, want %s", got, want)
	}
	if len(store.commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(store.commands))
	}
	if store.commands[0].DebitAuthorizationID != "debit-auth-1" {
		t.Fatalf("debit authorization = %q, want event body value", store.commands[0].DebitAuthorizationID)
	}
	if len(committer.commits) != 1 {
		t.Fatalf("commits = %d, want 1", len(committer.commits))
	}
	assertObserverLabelsSafe(t, observer.labels)
}

func TestTerminalConsumerCommitsDuplicatesAndQuarantinesConflicts(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		result           TerminalApplyResult
		wantQuarantines  int
		wantObserverPart string
	}{
		{name: "duplicate replay", result: TerminalApplyResultDuplicate, wantObserverPart: "|duplicate|"},
		{name: "changed fingerprint conflict", result: TerminalApplyResultConflict, wantQuarantines: 1, wantObserverPart: "|conflict|"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := &fakeTerminalStore{result: tc.result}
			committer := &fakeCommitter{}
			observer := &capturedEventObserver{}
			consumer := TerminalConsumer{
				Store:            store,
				Committer:        committer,
				Observer:         observer,
				AllowedProducers: []string{"gonka-proxy"},
				Now:              fixedAdapterTime,
			}

			if err := consumer.Handle(context.Background(), validTerminalMessage(t)); err != nil {
				t.Fatalf("Handle() error = %v", err)
			}
			if len(committer.commits) != 1 {
				t.Fatalf("commits = %d, want 1", len(committer.commits))
			}
			if len(store.quarantines) != tc.wantQuarantines {
				t.Fatalf("quarantines = %d, want %d", len(store.quarantines), tc.wantQuarantines)
			}
			if !containsObserverPart(observer.labels, tc.wantObserverPart) {
				t.Fatalf("observer labels = %v, want part %q", observer.labels, tc.wantObserverPart)
			}
			assertObserverLabelsSafe(t, observer.labels)
		})
	}
}

func TestTerminalConsumerQuarantinesInvalidProducerAndFingerprintWithoutRawPayload(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		mutateEvent func(*eventsv1.MicroleaseChildTerminalSubmitted)
		wantReason  string
	}{
		{
			name: "producer",
			mutateEvent: func(event *eventsv1.MicroleaseChildTerminalSubmitted) {
				event.Envelope.ProducerIdentity = "unexpected-producer"
				event.Envelope.EventFingerprint = mustTerminalFingerprint(t, *event)
			},
			wantReason: "producer_identity",
		},
		{
			name: "fingerprint",
			mutateEvent: func(event *eventsv1.MicroleaseChildTerminalSubmitted) {
				event.Envelope.EventFingerprint = "sha256:bad"
			},
			wantReason: "fingerprint_mismatch",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			event := validTerminalEvent(t)
			tc.mutateEvent(&event)
			msg := terminalMessageFromEvent(t, event)
			store := &fakeTerminalStore{}
			committer := &fakeCommitter{}
			consumer := TerminalConsumer{
				Store:            store,
				Committer:        committer,
				AllowedProducers: []string{"gonka-proxy"},
				Now:              fixedAdapterTime,
			}

			if err := consumer.Handle(context.Background(), msg); err != nil {
				t.Fatalf("Handle() error = %v", err)
			}
			if len(store.commands) != 0 {
				t.Fatalf("commands = %d, want 0", len(store.commands))
			}
			if len(store.quarantines) != 1 {
				t.Fatalf("quarantines = %d, want 1", len(store.quarantines))
			}
			if store.quarantines[0].ReasonClass != tc.wantReason {
				t.Fatalf("reason = %q, want %q", store.quarantines[0].ReasonClass, tc.wantReason)
			}
			assertMetadataSafe(t, store.quarantines[0].SafeMetadata)
			if len(committer.commits) != 1 {
				t.Fatalf("commits = %d, want 1", len(committer.commits))
			}
		})
	}
}

func TestTerminalConsumerReturnsBoundedRetryWithoutOffsetCommit(t *testing.T) {
	t.Parallel()

	store := &fakeTerminalStore{err: errors.New("database temporarily unavailable")}
	committer := &fakeCommitter{}
	consumer := TerminalConsumer{
		Store:            store,
		Committer:        committer,
		AllowedProducers: []string{"gonka-proxy"},
		RetryPolicy:      RetryPolicy{BaseDelay: 10 * time.Millisecond, MaxDelay: 50 * time.Millisecond},
		Now:              fixedAdapterTime,
	}

	err := consumer.Handle(context.Background(), Message{
		Topic:     validTerminalTopic,
		Partition: 1,
		Offset:    22,
		Value:     validTerminalMessage(t).Value,
		Attempt:   4,
	})
	assertBoundedRetryWithoutCommit(t, err, committer)
}

func assertBoundedRetryWithoutCommit(t *testing.T, err error, committer *fakeCommitter) {
	t.Helper()

	if err == nil {
		t.Fatalf("Handle() error = nil, want retry")
	}
	var retryErr RetryAfterError
	if !errors.As(err, &retryErr) {
		t.Fatalf("error = %T, want RetryAfterError", err)
	}
	if retryErr.After != 50*time.Millisecond {
		t.Fatalf("retry after = %s, want capped 50ms", retryErr.After)
	}
	if len(committer.commits) != 0 {
		t.Fatalf("commits = %d, want 0", len(committer.commits))
	}
}

type fakeOutboxStore struct {
	records      []OutboxRecord
	published    []string
	retries      []string
	nextAttempts []time.Time
	claimErr     error
	publishErr   error
	retryErr     error
}

func (s *fakeOutboxStore) ClaimOutbox(_ context.Context, _ time.Time, _ int32) ([]OutboxRecord, error) {
	if s.claimErr != nil {
		return nil, s.claimErr
	}
	return s.records, nil
}

func (s *fakeOutboxStore) MarkOutboxPublished(_ context.Context, outboxID string, _ time.Time) error {
	if s.publishErr != nil {
		return s.publishErr
	}
	s.published = append(s.published, outboxID)
	return nil
}

func (s *fakeOutboxStore) MarkOutboxRetry(_ context.Context, outboxID string, nextAttempt time.Time, reason string) error {
	if s.retryErr != nil {
		return s.retryErr
	}
	s.retries = append(s.retries, outboxID+":"+reason)
	s.nextAttempts = append(s.nextAttempts, nextAttempt)
	return nil
}

type fakeProducer struct {
	messages []ProduceMessage
	err      error
}

func (p *fakeProducer) Produce(_ context.Context, msg ProduceMessage) error {
	p.messages = append(p.messages, msg)
	return p.err
}

func TestOutboxRelayPublishesAfterFingerprintProofAndMarksRetryOnFailure(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"eventType":"MicroleaseIssued","microleaseId":"11111111-1111-1111-1111-111111111111","result":"issued"}`)
	store := &fakeOutboxStore{records: []OutboxRecord{
		{
			OutboxID:         "outbox-1",
			Topic:            "billing.microlease.facts.v1",
			Key:              []byte("microlease-1"),
			EventType:        "MicroleaseIssued",
			EventFingerprint: FingerprintOutboxPayload(payload),
			SafePayload:      payload,
		},
		{
			OutboxID:         "outbox-2",
			Topic:            "billing.microlease.facts.v1",
			Key:              []byte("microlease-2"),
			EventType:        "MicroleaseClosed",
			EventFingerprint: "sha256:bad",
			SafePayload:      payload,
			Attempt:          2,
		},
	}}
	producer := &fakeProducer{}
	observer := &capturedEventObserver{}
	relay := OutboxRelay{
		Store:            store,
		Producer:         producer,
		Observer:         observer,
		ProducerIdentity: "billing-service",
		RetryPolicy:      RetryPolicy{BaseDelay: 10 * time.Millisecond, MaxDelay: 100 * time.Millisecond},
		Now:              fixedAdapterTime,
	}

	published, err := relay.RelayOnce(context.Background(), 10)
	if err != nil {
		t.Fatalf("RelayOnce() error = %v", err)
	}
	if published != 1 {
		t.Fatalf("published = %d, want 1", published)
	}
	if len(producer.messages) != 1 {
		t.Fatalf("produced = %d, want 1", len(producer.messages))
	}
	if got := producer.messages[0].Headers["x-producer-identity"]; got != "billing-service" {
		t.Fatalf("producer identity header = %q, want billing-service", got)
	}
	if len(store.published) != 1 || store.published[0] != "outbox-1" {
		t.Fatalf("published marks = %v, want [outbox-1]", store.published)
	}
	if len(store.retries) != 1 || store.retries[0] != "outbox-2:fingerprint_mismatch" {
		t.Fatalf("retries = %v, want fingerprint retry", store.retries)
	}
	if len(store.nextAttempts) != 1 || !store.nextAttempts[0].Equal(fixedAdapterTime().Add(40*time.Millisecond)) {
		t.Fatalf("next attempts = %v, want capped backoff from attempt", store.nextAttempts)
	}
	assertObserverLabelsSafe(t, observer.labels)
}

func TestOutboxRelayFailureModesAreRetryableAndDoNotPublishAmbiguousEvents(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"result":"ok"}`)
	record := OutboxRecord{
		OutboxID:         "outbox-1",
		Topic:            "billing.microlease.facts.v1",
		Key:              []byte("aggregate-1"),
		EventType:        "MicroleaseIssued",
		EventFingerprint: FingerprintOutboxPayload(payload),
		SafePayload:      payload,
		Attempt:          1,
	}
	if _, err := (OutboxRelay{Producer: &fakeProducer{}}).RelayOnce(context.Background(), 0); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("RelayOnce(missing store) error = %v, want invalid event", err)
	}
	if _, err := (OutboxRelay{Store: &fakeOutboxStore{}}).RelayOnce(context.Background(), 0); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("RelayOnce(missing producer) error = %v, want invalid event", err)
	}
	if _, err := (OutboxRelay{Store: &fakeOutboxStore{claimErr: errors.New("claim failed")}, Producer: &fakeProducer{}}).RelayOnce(context.Background(), 0); !errors.Is(err, ErrRetryable) {
		t.Fatalf("RelayOnce(claim error) error = %v, want retryable", err)
	}

	store := &fakeOutboxStore{records: []OutboxRecord{record}}
	producer := &fakeProducer{err: errors.New("broker down")}
	published, err := (OutboxRelay{Store: store, Producer: producer, RetryPolicy: RetryPolicy{BaseDelay: time.Second}, Now: fixedAdapterTime}).RelayOnce(context.Background(), 0)
	if err != nil || published != 0 || len(store.retries) != 1 || store.retries[0] != "outbox-1:produce_retryable" {
		t.Fatalf("RelayOnce(produce retry) published=%d retries=%v err=%v", published, store.retries, err)
	}

	store = &fakeOutboxStore{records: []OutboxRecord{record}, retryErr: errors.New("retry mark failed")}
	_, err = (OutboxRelay{Store: store, Producer: &fakeProducer{err: errors.New("broker down")}, RetryPolicy: RetryPolicy{BaseDelay: time.Second}, Now: fixedAdapterTime}).RelayOnce(context.Background(), 0)
	if !errors.Is(err, ErrRetryable) {
		t.Fatalf("RelayOnce(retry mark failure) error = %v, want retryable", err)
	}

	store = &fakeOutboxStore{records: []OutboxRecord{record}, publishErr: errors.New("publish mark failed")}
	published, err = (OutboxRelay{Store: store, Producer: &fakeProducer{}, RetryPolicy: RetryPolicy{BaseDelay: time.Second}, Now: fixedAdapterTime}).RelayOnce(context.Background(), 0)
	if published != 0 || !errors.Is(err, ErrRetryable) {
		t.Fatalf("RelayOnce(publish mark failure) published=%d err=%v, want retryable before count", published, err)
	}
}

func validTerminalMessage(t *testing.T) Message {
	t.Helper()
	return terminalMessageFromEvent(t, validTerminalEvent(t))
}

func validCheckpointMessage(t *testing.T) Message {
	t.Helper()
	return checkpointMessageFromEvent(t, validCheckpointEvent(t))
}

func validCloseMessage(t *testing.T) Message {
	t.Helper()
	return closeMessageFromEvent(t, validCloseEvent(t))
}

func terminalMessageFromEvent(t *testing.T, event eventsv1.MicroleaseChildTerminalSubmitted) Message {
	t.Helper()
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal terminal event: %v", err)
	}
	return Message{
		Topic:     validTerminalTopic,
		Partition: 3,
		Offset:    17,
		Key:       []byte(event.Identity.MicroleaseID),
		Value:     data,
	}
}

func checkpointMessageFromEvent(t *testing.T, event eventsv1.MicroleaseCheckpointReported) Message {
	t.Helper()
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal checkpoint event: %v", err)
	}
	return Message{
		Topic:     validCheckpointTopic,
		Partition: 4,
		Offset:    31,
		Key:       []byte(event.Identity.MicroleaseID),
		Value:     data,
	}
}

func closeMessageFromEvent(t *testing.T, event eventsv1.MicroleaseCloseReported) Message {
	t.Helper()
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal close event: %v", err)
	}
	return Message{
		Topic:     validCloseTopic,
		Partition: 5,
		Offset:    41,
		Key:       []byte(event.Checkpoint.Identity.MicroleaseID),
		Value:     data,
	}
}

func validTerminalEvent(t *testing.T) eventsv1.MicroleaseChildTerminalSubmitted {
	t.Helper()
	now := fixedAdapterTime()
	event := eventsv1.MicroleaseChildTerminalSubmitted{
		Envelope: eventsv1.MicroleaseEventEnvelope{
			EventID:            "event-terminal-1",
			EventType:          eventTypeTerminalSubmitted,
			ContractVersion:    "v1",
			SchemaVersion:      "billing.events.v1",
			ProducerIdentity:   "gonka-proxy",
			OccurredAtEpochMS:  now.UnixMilli(),
			ProducedAtEpochMS:  now.UnixMilli(),
			TraceCorrelationID: "trace-terminal-1",
		},
		Identity: eventsv1.MicroleaseIdentity{
			MicroleaseID:          "11111111-1111-1111-1111-111111111111",
			AccountScopeKey:       "acct_test",
			ProxyAllocatorOwnerID: "proxy-owner-1",
			MicroleaseGeneration:  1,
		},
		DebitAuthorizationID:     "debit-auth-1",
		ChildSequence:            7,
		ChildCapUSDAtoms:         10_000_000,
		TerminalKind:             "finalize",
		ChargedUSDAtoms:          4_000_000,
		ReleasedUSDAtoms:         6_000_000,
		RequestBasisFingerprint:  "request-basis-1",
		TerminalBasisFingerprint: "terminal-basis-1",
		Pricing: eventsv1.PricingSnapshotBasis{
			PricingSnapshotID:   "pricing-snapshot-1",
			SnapshotFingerprint: "pricing-fingerprint-1",
			PolicyVersion:       "pricing-policy-v1",
			SelectorKey:         "gnk_usdt:usage_reserve",
			UseClass:            "usage_reserve",
			ContractVersion:     "pricing.v1",
		},
		QualifiedInferenceEvidenceID: "qualified-evidence-1",
		SafeExecutionReference:       "execution-ref-1",
		TerminalDeadlineEpochMS:      now.Add(2 * time.Minute).UnixMilli(),
		ObservedTerminalEpochMS:      now.UnixMilli(),
	}
	event.Envelope.EventFingerprint = mustTerminalFingerprint(t, event)
	return event
}

func validCheckpointEvent(t *testing.T) eventsv1.MicroleaseCheckpointReported {
	t.Helper()
	now := fixedAdapterTime()
	event := eventsv1.MicroleaseCheckpointReported{
		Envelope: eventsv1.MicroleaseEventEnvelope{
			EventID:            "event-checkpoint-1",
			EventType:          eventTypeCheckpointReported,
			ContractVersion:    "v1",
			SchemaVersion:      "billing.events.v1",
			ProducerIdentity:   "gonka-proxy",
			OccurredAtEpochMS:  now.UnixMilli(),
			ProducedAtEpochMS:  now.UnixMilli(),
			TraceCorrelationID: "trace-checkpoint-1",
		},
		Identity: eventsv1.MicroleaseIdentity{
			MicroleaseID:          "11111111-1111-1111-1111-111111111111",
			AccountScopeKey:       "acct_test",
			ProxyAllocatorOwnerID: "proxy-owner-1",
			MicroleaseGeneration:  1,
		},
		CheckpointSequence:            3,
		CheckpointKind:                "periodic",
		AllocatedChildHighWater:       7,
		AllocatedChildCount:           3,
		AllocatedChildCapSumUSDAtoms:  10_000_000,
		TerminalSubmittedCount:        2,
		TerminalPublishedCount:        2,
		TerminalAcceptedCount:         1,
		UnresolvedChildCount:          1,
		UnresolvedChildCapSumUSDAtoms: 1_000_000,
		LocalRemainingUSDAtoms:        40_000_000,
		CheckpointFingerprint:         "checkpoint-fingerprint-1",
	}
	event.Envelope.EventFingerprint = mustCheckpointFingerprint(t, event)
	return event
}

func validCloseEvent(t *testing.T) eventsv1.MicroleaseCloseReported {
	t.Helper()
	now := fixedAdapterTime()
	checkpoint := validCheckpointEvent(t)
	checkpoint.Envelope.EventID = "event-close-1"
	checkpoint.Envelope.EventType = eventTypeCloseReported
	event := eventsv1.MicroleaseCloseReported{
		Checkpoint:                 checkpoint,
		CloseReason:                "normal_close",
		AllocatorClosedEpochMS:     now.UnixMilli(),
		FinalLocalStateFingerprint: "local-state-fingerprint-1",
	}
	event.Checkpoint.Envelope.EventFingerprint = mustCloseFingerprint(t, event)
	return event
}

func mustTerminalFingerprint(t *testing.T, event eventsv1.MicroleaseChildTerminalSubmitted) string {
	t.Helper()
	fingerprint, err := FingerprintTerminalSubmitted(event)
	if err != nil {
		t.Fatalf("fingerprint terminal event: %v", err)
	}
	return fingerprint
}

func mustCheckpointFingerprint(t *testing.T, event eventsv1.MicroleaseCheckpointReported) string {
	t.Helper()
	fingerprint, err := FingerprintCheckpointReported(event)
	if err != nil {
		t.Fatalf("fingerprint checkpoint event: %v", err)
	}
	return fingerprint
}

func mustCloseFingerprint(t *testing.T, event eventsv1.MicroleaseCloseReported) string {
	t.Helper()
	fingerprint, err := FingerprintCloseReported(event)
	if err != nil {
		t.Fatalf("fingerprint close event: %v", err)
	}
	return fingerprint
}

func fixedAdapterTime() time.Time {
	return time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
}

const (
	validTerminalTopic   = "billing.microlease.terminal.v1"
	validCheckpointTopic = "billing.microlease.checkpoint.v1"
	validCloseTopic      = "billing.microlease.close.v1"
)

func assertMetadataSafe(t *testing.T, metadata map[string]string) {
	t.Helper()
	serialized, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	text := string(serialized)
	for _, forbidden := range []string{
		"raw prompt",
		"completion text",
		"sk-",
		"Bearer ",
		"postgres://",
		"payment_secret",
		"acct_test",
		"debit-auth-1",
		"trace-terminal-1",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("metadata leaked forbidden value %q: %s", forbidden, text)
		}
	}
}

func assertObserverLabelsSafe(t *testing.T, labels []string) {
	t.Helper()
	text := strings.Join(labels, "\n")
	for _, forbidden := range []string{"acct_test", "event-terminal-1", "debit-auth-1", "trace-terminal-1", "11111111-1111"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("observer labels leaked forbidden value %q: %s", forbidden, text)
		}
	}
}

func containsObserverPart(labels []string, want string) bool {
	for _, label := range labels {
		if strings.Contains(label, want) {
			return true
		}
	}
	return false
}
