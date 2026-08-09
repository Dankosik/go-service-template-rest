//go:build integration

package integration_test

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
)

func TestPostgresOutboxPublishFailure(t *testing.T) {
	tests := []struct {
		name            string
		publisherResult error
		waitForTimeout  bool
		wantClass       string
		wantPoison      bool
	}{
		{name: "temporary before acknowledgement", publisherResult: errors.New("broker unavailable"), wantClass: "publisher_temporary"},
		{name: "timeout before acknowledgement", waitForTimeout: true, wantClass: "publisher_timeout"},
		// A permanent rejection poisons on its first occurrence, which the
		// attempt assertion below is what proves: the row is parked without the
		// relay ever handing the same bytes to the adapter a second time.
		{name: "permanent rejection", publisherResult: postgresoutbox.ErrPermanentPublication, wantClass: "publisher_permanent", wantPoison: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, pool, store := newOutboxFixture(t)
			mustAppendOutbox(t, ctx, pool, store, outboxEvent("publish-failure"))
			entered := make(chan struct{})
			release := make(chan struct{})
			var attempts atomic.Int64
			publisher := testPublisherFunc(func(publishCtx context.Context, _ postgresoutbox.Event) error {
				attempts.Add(1)
				close(entered)
				if test.waitForTimeout {
					<-publishCtx.Done()
					return publishCtx.Err()
				}
				<-release
				return test.publisherResult
			})
			relay := mustNewOutboxRelay(t, store, publisher, nil, testRelayConfig())
			result := runOutboxRelay(ctx, relay)
			<-entered
			relay.StartDrain()
			close(release)
			assertRelayResult(t, result, nil)

			record, err := store.Get(ctx, "publish-failure")
			if err != nil {
				t.Fatalf("Get(): %v", err)
			}
			if record.LastErrorClass != test.wantClass {
				t.Fatalf("error class = %q, want %q", record.LastErrorClass, test.wantClass)
			}
			if gotPoison := !record.PoisonedAt.IsZero(); gotPoison != test.wantPoison {
				t.Fatalf("poisoned = %t, want %t", gotPoison, test.wantPoison)
			}
			if !record.PublishedAt.IsZero() {
				t.Fatal("failed publication was marked published")
			}
			if got := attempts.Load(); got != 1 {
				t.Fatalf("publisher attempts = %d, want 1 before the drain stopped the cycle", got)
			}
		})
	}
}

// A retry that reports its publication was never attempted gives the claim's
// attempt back, and only PostgreSQL can prove it: the two counters are the
// attempt cap's whole authority, and the claim statement is what turns an
// uncertain event that reaches the cap into outcome-unknown quarantine. Without
// the refund, a broker slow enough to leave every batch with a tail would walk
// those events to max_attempts and quarantine them having never attempted one
// publication — an operator action per event for a throughput problem.
func TestPostgresOutboxUnattemptedRetryReturnsTheAttempt(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	mustAppendOutbox(t, ctx, pool, store, outboxEvent("unattempted"))

	// One more cycle than the max-attempts argument claimOutboxEvent passes, so a
	// refund that failed to land would exhaust the cap inside this loop.
	const cycles = 6
	for cycle := range cycles {
		claim := mustClaimOutbox(t, ctx, store)
		if claim.CycleAttemptCount != 1 || claim.TotalAttemptCount != 1 {
			t.Fatalf("cycle %d claimed attempts = %d/%d, want 1/1: the refund did not land",
				cycle, claim.CycleAttemptCount, claim.TotalAttemptCount)
		}
		if err := store.ScheduleRetryBatch(ctx, claim.Token, []postgresoutbox.RetryDirective{{
			ID: claim.Event.ID, ErrorClass: "publisher_not_attempted", NotAttempted: true,
		}}); err != nil {
			t.Fatalf("cycle %d ScheduleRetryBatch(not attempted): %v", cycle, err)
		}

		record, err := store.Get(ctx, claim.Event.ID)
		if err != nil {
			t.Fatalf("cycle %d Get(): %v", cycle, err)
		}
		if record.CycleAttemptCount != 0 || record.TotalAttemptCount != 0 {
			t.Fatalf("cycle %d refunded attempts = %d/%d, want 0/0",
				cycle, record.CycleAttemptCount, record.TotalAttemptCount)
		}
		if record.PublicationUncertain {
			t.Fatalf("cycle %d set publication uncertainty for a publication that never happened", cycle)
		}
		if record.LastErrorClass != "publisher_not_attempted" || !record.PoisonedAt.IsZero() {
			t.Fatalf("cycle %d record = class %q poisoned %v", cycle, record.LastErrorClass, record.PoisonedAt)
		}
	}

	// The refund is not a general reset: a real failure still costs its attempt,
	// which is what keeps the cap meaning attempts actually made.
	claim := mustClaimOutbox(t, ctx, store)
	if err := scheduleOutboxRetry(ctx, store, claim.Event.ID, claim.Token, "publisher_temporary", 0); err != nil {
		t.Fatalf("ScheduleRetryBatch(temporary): %v", err)
	}
	record, err := store.Get(ctx, claim.Event.ID)
	if err != nil {
		t.Fatalf("Get(): %v", err)
	}
	if record.CycleAttemptCount != 1 || record.TotalAttemptCount != 1 {
		t.Fatalf("attempted failure attempts = %d/%d, want 1/1",
			record.CycleAttemptCount, record.TotalAttemptCount)
	}
}

func TestPostgresOutboxAckCrashDuplicate(t *testing.T) {
	ctx, pool, store := newOutboxFixture(t)
	event := outboxEvent("ack-crash")
	event.Payload = []byte(" {\"same\": true} ")
	mustAppendOutbox(t, ctx, pool, store, event)

	attempts := make(chan postgresoutbox.Event, 2)
	releaseSecond := make(chan struct{})
	var callCount atomic.Int64
	publisher := testPublisherFunc(func(_ context.Context, event postgresoutbox.Event) error {
		copy := event
		copy.Payload = append([]byte(nil), event.Payload...)
		copy.Metadata = append([]byte(nil), event.Metadata...)
		attempts <- copy
		if callCount.Add(1) == 2 {
			<-releaseSecond
		}
		return nil
	})

	first, err := claimOutboxEvent(ctx, store, shortOutboxLease)
	if err != nil {
		t.Fatalf("first Claim(): %v", err)
	}
	if err := publisher.Publish(ctx, first.Event); err != nil {
		t.Fatalf("first durable publish: %v", err)
	}
	firstAttempt := <-attempts
	expireOutboxLease(t, ctx, pool)

	relay := mustNewOutboxRelay(t, store, publisher, nil, testRelayConfig())
	result := runOutboxRelay(ctx, relay)
	secondAttempt := <-attempts
	relay.StartDrain()
	close(releaseSecond)
	assertRelayResult(t, result, nil)
	if !reflect.DeepEqual(firstAttempt, secondAttempt) {
		t.Fatalf("duplicate envelopes differ:\nfirst  %+v\nsecond %+v", firstAttempt, secondAttempt)
	}
	record, err := store.Get(ctx, event.ID)
	if err != nil {
		t.Fatalf("Get(): %v", err)
	}
	if record.PublishedAt.IsZero() || record.TotalAttemptCount != 2 {
		t.Fatalf("final record published=%v attempts=%d, want published and 2", record.PublishedAt, record.TotalAttemptCount)
	}
}
