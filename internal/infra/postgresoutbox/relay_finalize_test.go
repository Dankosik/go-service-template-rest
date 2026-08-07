package postgresoutbox

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestRelayPublicationDispositions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		publishErr   error
		attempt      int
		storeErr     error
		wantPoison   string
		wantRetry    string
		wantDelay    time.Duration
		wantStateErr bool
	}{
		{name: "temporary retry", publishErr: errors.New("temporary"), attempt: 1, wantRetry: "publisher_temporary", wantDelay: time.Second},
		{name: "permanent poison", publishErr: ErrPermanentPublication, attempt: 1, wantPoison: "publisher_permanent"},
		{name: "rejected retry", publishErr: ErrPublicationNotAccepted, attempt: 1, wantRetry: "publisher_rejected", wantDelay: time.Second},
		{name: "rejected exhaustion poison", publishErr: ErrPublicationNotAccepted, attempt: 3, wantPoison: "attempt_exhausted"},
		// An ambiguous failure never proves the broker refused the event, so the
		// attempt cap must not trade possible loss for a bounded retry count.
		{name: "ambiguous exhaustion retries", publishErr: errors.New("temporary"), attempt: 3, wantRetry: "publisher_temporary", wantDelay: 4 * time.Second},
		{name: "timeout exhaustion retries", publishErr: context.DeadlineExceeded, attempt: 3, wantRetry: "publisher_timeout", wantDelay: 4 * time.Second},
		{name: "state failure", publishErr: errors.New("temporary"), attempt: 1, storeErr: errors.New("write failed"), wantRetry: "publisher_temporary", wantDelay: time.Second, wantStateErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			poisonClass := ""
			retryClass := ""
			store := &relayStoreStub{
				markPoisonedBatch: func(_ context.Context, _ string, poisons []PoisonDirective) error {
					poisonClass = poisons[0].ErrorClass
					return test.storeErr
				},
				scheduleRetryBatch: func(_ context.Context, _ string, retries []RetryDirective) error {
					retryClass = retries[0].ErrorClass
					if retries[0].Delay != test.wantDelay {
						t.Errorf("retry delay = %s, want %s", retries[0].Delay, test.wantDelay)
					}
					return test.storeErr
				},
			}
			relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return test.publishErr }))
			claim := unitClaim(1)
			claim.CycleAttemptCount = test.attempt
			batch := ClaimedBatch{Token: unitLeaseToken, Events: []ClaimedEvent{claim}}

			result, stop := relay.publishBatch(t.Context(), batch, time.Now().Add(time.Hour))
			if result.CleanupUnsafe || (result.Err != nil) != test.wantStateErr || stop != test.wantStateErr {
				t.Fatalf("publishBatch() = %+v stop=%t", result, stop)
			}
			if poisonClass != test.wantPoison || retryClass != test.wantRetry {
				t.Fatalf("poison=%q retry=%q, want %q/%q", poisonClass, retryClass, test.wantPoison, test.wantRetry)
			}
		})
	}
}

// One batch can end in several dispositions at once, and each one reaches the
// store in a single call.
func TestRelayFinalizesMixedBatchInOneCallPerDisposition(t *testing.T) {
	t.Parallel()

	publishCalls := 0
	retryCalls := 0
	poisonCalls := 0
	var retried []RetryDirective
	var poisoned []PoisonDirective
	store := &relayStoreStub{
		markUnorderedPublishedBatch: func(_ context.Context, _ string, ids []string) ([]string, error) {
			publishCalls++
			return ids, nil
		},
		scheduleRetryBatch: func(_ context.Context, _ string, retries []RetryDirective) error {
			retryCalls++
			retried = retries
			return nil
		},
		markPoisonedBatch: func(_ context.Context, _ string, poisons []PoisonDirective) error {
			poisonCalls++
			poisoned = poisons
			return nil
		},
	}
	relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return nil }))

	batch := unitBatch(5)
	failures := []error{nil, nil, errors.New("temporary"), errors.New("temporary"), ErrPermanentPublication}
	if err := relay.finalize(t.Context(), batch, failures); err != nil {
		t.Fatalf("finalize() error = %v", err)
	}
	if publishCalls != 1 || retryCalls != 1 || poisonCalls != 1 {
		t.Fatalf("store calls published=%d retried=%d poisoned=%d, want 1/1/1", publishCalls, retryCalls, poisonCalls)
	}
	if len(retried) != 2 || len(poisoned) != 1 {
		t.Fatalf("retried=%d poisoned=%d, want 2/1", len(retried), len(poisoned))
	}
}

// Ordered events finalize in their own statement, separate from the unordered
// ones, because each also advances its key head and unblocks that key's
// successor. Both statements cover the whole batch, and neither falls back to
// per-event marking while the lease holds.
func TestRelayFinalizesOrderedEventsInOneStatement(t *testing.T) {
	t.Parallel()

	unorderedCalls := 0
	orderedCalls := 0
	var ordered []OrderedDirective
	var individual []string
	store := &relayStoreStub{
		markUnorderedPublishedBatch: func(_ context.Context, _ string, ids []string) ([]string, error) {
			unorderedCalls++
			return ids, nil
		},
		markOrderedPublishedBatch: func(_ context.Context, _ string, directives []OrderedDirective) ([]string, error) {
			orderedCalls++
			ordered = directives
			ids := make([]string, len(directives))
			for index, directive := range directives {
				ids[index] = directive.ID
			}
			return ids, nil
		},
		markUnorderedPublished: func(_ context.Context, _ string, id string) error {
			individual = append(individual, id)
			return nil
		},
		markOrderedPublished: func(_ context.Context, _ string, directive OrderedDirective) error {
			individual = append(individual, directive.ID)
			return nil
		},
	}
	relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return nil }))

	batch := unitBatch(3)
	for index := 1; index < len(batch.Events); index++ {
		batch.Events[index].Event.OrderingKey = fmt.Sprintf("key-%d", index)
		batch.Events[index].Event.OrderingSequence = int64(index)
	}
	if err := relay.finalize(t.Context(), batch, make([]error, len(batch.Events))); err != nil {
		t.Fatalf("finalize() error = %v", err)
	}
	if unorderedCalls != 1 || orderedCalls != 1 || individual != nil {
		t.Fatalf("calls unordered=%d ordered=%d individual=%v, want 1/1/none",
			unorderedCalls, orderedCalls, individual)
	}
	if len(ordered) != 2 || ordered[0].OrderingKey != "key-1" || ordered[1].OrderingSequence != 2 {
		t.Fatalf("ordered directives = %+v, want both ordered events with their key identity", ordered)
	}
}

// A short batch write is ambiguous only for the events it left out, so exactly
// those are resolved against durable state instead of being assumed published
// or lost. The ones the statement reported are already durable.
func TestRelayReconcilesShortBatchWrite(t *testing.T) {
	t.Parallel()

	var reconciled []string
	store := &relayStoreStub{
		markUnorderedPublishedBatch: func(_ context.Context, _ string, ids []string) ([]string, error) {
			return ids[:len(ids)-1], nil
		},
		markUnorderedPublished: func(_ context.Context, _ string, id string) error {
			reconciled = append(reconciled, id)
			return ErrLeaseLost
		},
		get: func(_ context.Context, id string) (Record, error) {
			return Record{Event: Event{ID: id}, PublishedAt: time.Now()}, nil
		},
	}
	relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return nil }))
	claims := []ClaimedEvent{unitClaim(1), unitClaim(2)}
	if err := relay.finalizeUnordered(t.Context(), unitLeaseToken, claims); err != nil {
		t.Fatalf("finalizeUnordered(short write) error = %v", err)
	}
	if len(reconciled) != 1 || reconciled[0] != claims[1].Event.ID {
		t.Fatalf("reconciled = %v, want only the event the batch left out", reconciled)
	}
	reconciled = nil

	store.get = func(context.Context, string) (Record, error) { return Record{LeaseToken: "other"}, nil }
	if err := relay.finalizeUnordered(t.Context(), unitLeaseToken, []ClaimedEvent{unitClaim(1)}); !errors.Is(err, ErrProgressUnknown) {
		t.Fatalf("finalizeUnordered(lost lease) error = %v, want ErrProgressUnknown", err)
	}
}

// An ordered acknowledgement the batch statement left out reconciles through
// the ordered single-event statement, carrying the key identity that fences the
// head advance. Reconciliation is handed the statement matching the batch it is
// closing out, so a leftover ordered event can never take the unordered path —
// which would mark the row published without advancing its key head, stranding
// every successor of that key until an operator noticed.
func TestRelayReconcilesOrderedRemainderThroughTheOrderedStatement(t *testing.T) {
	t.Parallel()

	var reconciled []OrderedDirective
	store := &relayStoreStub{
		markOrderedPublishedBatch: func(context.Context, string, []OrderedDirective) ([]string, error) {
			return nil, nil
		},
		markOrderedPublished: func(_ context.Context, _ string, directive OrderedDirective) error {
			reconciled = append(reconciled, directive)
			return nil
		},
		// Left nil: the unordered mark must never be reached for an ordered
		// claim, and reaching it would report a success this test cannot see.
		// The assertion below is on what the ordered mark received instead.
	}
	relay := newUnitRelay(store, noopPublisher())
	claim := unitClaim(1)
	claim.Event.OrderingKey, claim.Event.OrderingSequence = "key-a", 7

	if err := relay.finalizeOrdered(t.Context(), unitLeaseToken, []ClaimedEvent{claim}); err != nil {
		t.Fatalf("finalizeOrdered(short write) error = %v", err)
	}
	if len(reconciled) != 1 || reconciled[0] != orderedDirective(claim) {
		t.Fatalf("ordered reconciliation = %+v, want the claim's own key identity", reconciled)
	}
}

// Reconciliation gives up as unknown rather than guessing: the event may
// already be at the broker while PostgreSQL still shows it unpublished.
func TestRelayReconcileReportsUnknownProgress(t *testing.T) {
	t.Parallel()

	store := &relayStoreStub{
		markUnorderedPublished: func(context.Context, string, string) error { return errors.New("mark") },
		get:                    func(context.Context, string) (Record, error) { return Record{LeaseToken: unitLeaseToken}, nil },
	}
	relay := newUnitRelay(store, noopPublisher())
	if err := relay.reconcilePublished(
		t.Context(), unitLeaseToken, unitClaim(1), relay.markOneUnordered,
	); !errors.Is(err, ErrProgressUnknown) {
		t.Fatalf("reconcilePublished(retry exhausted) error = %v", err)
	}
	reconcileErr := errors.New("reconcile")
	store.get = func(context.Context, string) (Record, error) { return Record{}, reconcileErr }
	if err := relay.reconcilePublished(
		t.Context(), unitLeaseToken, unitClaim(1), relay.markOneUnordered,
	); !errors.Is(err, ErrProgressUnknown) || !errors.Is(err, reconcileErr) {
		t.Fatalf("reconcilePublished(reconcile) error = %v", err)
	}
}

// Poisoning reaches the operator log only after the durable write succeeds, and
// a failed write is reported instead.
func TestRelayPoisonLoggingFollowsDurableWrite(t *testing.T) {
	t.Parallel()

	poisonErr := errors.New("poison write failed")
	store := &relayStoreStub{
		markPoisonedBatch: func(context.Context, string, []PoisonDirective) error { return poisonErr },
	}
	// Telemetry stays nil: this asserts markPoisoned's write-then-log ordering
	// through its return value, and a nil *Telemetry is a working no-op. What
	// the log line actually contains is TestTelemetryBoundedContract's claim.
	relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return nil }))
	poisoned := []poisonedEvent{{claim: unitClaim(1), errorClass: "publisher_permanent"}}
	if err := relay.finalizePoisoned(t.Context(), unitLeaseToken, poisoned); !errors.Is(err, poisonErr) {
		t.Fatalf("finalizePoisoned(write failure) error = %v", err)
	}

	store.markPoisonedBatch = nil
	if err := relay.finalizePoisoned(t.Context(), unitLeaseToken, poisoned); err != nil {
		t.Fatalf("finalizePoisoned() error = %v", err)
	}
	if err := relay.finalizePoisoned(t.Context(), unitLeaseToken, nil); err != nil {
		t.Fatalf("finalizePoisoned(empty) error = %v", err)
	}
}

// Finalization failures are reported with the transition that produced them.
func TestRelayFinalizeReportsRetryWriteFailure(t *testing.T) {
	t.Parallel()

	retryErr := errors.New("retry write failed")
	relay := newUnitRelay(&relayStoreStub{
		scheduleRetryBatch: func(context.Context, string, []RetryDirective) error { return retryErr },
	}, publisherFunc(func(context.Context, Event) error { return nil }))
	if err := relay.finalize(t.Context(), unitBatch(1), []error{errors.New("temporary")}); !errors.Is(err, retryErr) {
		t.Fatalf("finalize(retry write failure) error = %v", err)
	}

	publishErr := errors.New("publish write failed")
	relay = newUnitRelay(&relayStoreStub{
		markUnorderedPublishedBatch: func(context.Context, string, []string) ([]string, error) { return nil, publishErr },
		markUnorderedPublished:      func(context.Context, string, string) error { return ErrLeaseLost },
		get:                         func(context.Context, string) (Record, error) { return Record{}, publishErr },
	}, publisherFunc(func(context.Context, Event) error { return nil }))
	if err := relay.finalize(t.Context(), unitBatch(1), []error{nil}); !errors.Is(err, ErrProgressUnknown) {
		t.Fatalf("finalize(publish write failure) error = %v", err)
	}
}
