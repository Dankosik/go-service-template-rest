package postgresoutbox

import (
	"context"
	"errors"
	"time"
)

// The doubles and builders the four relay test files share. Every other test
// file here is named for the production file it exercises — relay_test.go for
// the cycle and lifecycle, relay_publish_test.go, relay_finalize_test.go,
// relay_config_test.go — so this one holds what belongs to none of them alone.

type publisherFunc func(context.Context, Event) error

func (publish publisherFunc) Publish(ctx context.Context, event Event) error {
	return publish(ctx, event)
}

// unitLeaseToken is the token every unit fixture leases under. relayStoreStub.Get
// and unitBatch must agree on it: reconcilePublished treats a record whose
// LeaseToken differs from the batch's as a lost lease and gives up with
// ErrProgressUnknown, so two different strings here would silently move that
// verdict in every reconciliation test.
const unitLeaseToken = "lease"

// relayStoreStub is the relay's store. A nil field is not "do nothing" — each
// one stands for a specific durable postcondition, named at the method below,
// and a test that leaves it nil is asserting that postcondition.
// The two single-event marks are separate fields, not one, because which of
// them reconciliation reaches is the property that keeps the ordered and
// unordered dispositions apart: a test that leaves one nil is asserting that
// path was never taken.
type relayStoreStub struct {
	claim                       func(context.Context, time.Duration, int) (ClaimedBatch, error)
	markUnorderedPublished      func(context.Context, string, string) error
	markOrderedPublished        func(context.Context, string, OrderedDirective) error
	markUnorderedPublishedBatch func(context.Context, string, []string) ([]string, error)
	markOrderedPublishedBatch   func(context.Context, string, []OrderedDirective) ([]string, error)
	scheduleRetryBatch          func(context.Context, string, []RetryDirective) error
	markPoisonedBatch           func(context.Context, string, []PoisonDirective) error
	get                         func(context.Context, string) (Record, error)
	cleanup                     func(context.Context, time.Duration, int) (int, error)
	observe                     func(context.Context) (StateObservation, error)
}

func (s *relayStoreStub) Claim(ctx context.Context, lease time.Duration, batchSize int) (ClaimedBatch, error) {
	if s.claim == nil {
		return ClaimedBatch{}, nil
	}
	return s.claim(ctx, lease, batchSize)
}

func (s *relayStoreStub) MarkUnorderedPublished(ctx context.Context, token, id string) error {
	if s.markUnorderedPublished == nil {
		return nil
	}
	return s.markUnorderedPublished(ctx, token, id)
}

func (s *relayStoreStub) MarkOrderedPublished(
	ctx context.Context,
	token string,
	directive OrderedDirective,
) error {
	if s.markOrderedPublished == nil {
		return nil
	}
	return s.markOrderedPublished(ctx, token, directive)
}

func (s *relayStoreStub) MarkUnorderedPublishedBatch(ctx context.Context, token string, ids []string) ([]string, error) {
	if s.markUnorderedPublishedBatch == nil {
		// Default: the lease still owned every id, so the statement finalized all
		// of them and nothing reaches reconciliation.
		return ids, nil
	}
	return s.markUnorderedPublishedBatch(ctx, token, ids)
}

func (s *relayStoreStub) MarkOrderedPublishedBatch(
	ctx context.Context,
	token string,
	directives []OrderedDirective,
) ([]string, error) {
	if s.markOrderedPublishedBatch == nil {
		// Default: as above, and every key head advanced with it.
		ids := make([]string, len(directives))
		for index, directive := range directives {
			ids[index] = directive.ID
		}
		return ids, nil
	}
	return s.markOrderedPublishedBatch(ctx, token, directives)
}

func (s *relayStoreStub) ScheduleRetryBatch(ctx context.Context, token string, retries []RetryDirective) error {
	if s.scheduleRetryBatch == nil {
		return nil
	}
	return s.scheduleRetryBatch(ctx, token, retries)
}

func (s *relayStoreStub) MarkPoisonedBatch(ctx context.Context, token string, poisons []PoisonDirective) error {
	if s.markPoisonedBatch == nil {
		return nil
	}
	return s.markPoisonedBatch(ctx, token, poisons)
}

func (s *relayStoreStub) Get(ctx context.Context, id string) (Record, error) {
	if s.get == nil {
		// Default: the row exists, is still unpublished, and this lease still owns
		// it — the one combination reconcilePublished retries rather than resolves.
		return Record{LeaseToken: unitLeaseToken}, nil
	}
	return s.get(ctx, id)
}

func (s *relayStoreStub) CleanupPublished(ctx context.Context, retention time.Duration, batch int) (int, error) {
	if s.cleanup == nil {
		return 0, nil
	}
	return s.cleanup(ctx, retention, batch)
}

func (s *relayStoreStub) Observe(ctx context.Context) (StateObservation, error) {
	if s.observe == nil {
		return StateObservation{}, nil
	}
	return s.observe(ctx)
}

// noopPublisher is the publisher for tests that exercise something other than
// publication.
func noopPublisher() publisherFunc {
	return publisherFunc(func(context.Context, Event) error { return nil })
}

// farSchedule puts both periodic duties out of reach, so a cycle under test is
// not interrupted by observation or cleanup.
func farSchedule() schedule {
	far := time.Now().Add(time.Hour)
	return schedule{observation: far, cleanup: far}
}

type pointerPublisher struct{}

func (*pointerPublisher) Publish(context.Context, Event) error { return nil }

func newUnitRelay(store relayStore, publisher Publisher) *Relay {
	relay, err := newRelay(store, publisher, nil, unitRelayConfig())
	if err != nil {
		panic(err)
	}
	relay.jitter = func(limit time.Duration) time.Duration { return limit }
	return relay
}

func unitRelayConfig() RelayConfig {
	return RelayConfig{
		PollInterval: time.Second, BatchSize: 100, PublishConcurrency: 16,
		PublishTimeout: time.Second, LeaseDuration: 3 * time.Second,
		MaxAttempts: 3, RetryBase: time.Second, RetryMax: time.Minute,
		ObservationInterval: time.Minute, CleanupInterval: time.Hour,
		PublishedRetention: time.Hour, CleanupBatchSize: 100,
	}
}

func unitBatch(events int) ClaimedBatch {
	batch := ClaimedBatch{Token: unitLeaseToken, Events: make([]ClaimedEvent, 0, events)}
	for index := 1; index <= events; index++ {
		batch.Events = append(batch.Events, unitClaim(index))
	}
	return batch
}

func unitClaim(index int) ClaimedEvent {
	event := outboxEventForUnit()
	event.ID = "event-" + string(rune('a'+index-1))
	return ClaimedEvent{Event: event, CycleAttemptCount: 1}
}

func fmtPermanent() error {
	return errors.Join(errors.New("adapter detail"), ErrPermanentPublication)
}

func outboxEventForUnit() Event {
	return Event{ID: "event", Payload: []byte(`{}`), Metadata: []byte(`{}`)}
}
