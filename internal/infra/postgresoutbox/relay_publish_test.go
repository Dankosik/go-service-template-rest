package postgresoutbox

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

// A publisher that reports success once the batch's publication budget is gone
// has not proven the broker accepted the event, so the attempt stays retryable.
func TestRelayTimeoutRejectsNilCompletion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		relay := newUnitRelay(&relayStoreStub{}, publisherFunc(func(ctx context.Context, _ Event) error {
			<-ctx.Done()
			return nil
		}))
		publications, cleanupSafe := relay.publishAll(
			context.Background(), unitBatch(1), time.Now().Add(time.Hour))
		if !cleanupSafe || len(publications) != 1 || !errors.Is(publications[0], context.DeadlineExceeded) {
			t.Fatalf("publishAll() = %v cleanupSafe=%t, want deadline failure after nil completion",
				publications, cleanupSafe)
		}
	})
}

func TestRelayPublisherPanic(t *testing.T) {
	t.Parallel()

	relay := newUnitRelay(&relayStoreStub{},
		publisherFunc(func(context.Context, Event) error { panic("publisher secret") }))
	got := relay.publishOne(t.Context(), unitClaim(1).Event)
	if !errors.Is(got, ErrPublisherPanic) {
		t.Fatalf("publishOne() = %v, want ErrPublisherPanic", got)
	}
	if class := publicationErrorClass(got); class != "panic" {
		t.Fatalf("panic class = %q, want panic", class)
	}
}

func TestRelayPublisherStuck(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		started := make(chan struct{})
		release := make(chan struct{})
		relay := newUnitRelay(&relayStoreStub{}, publisherFunc(func(context.Context, Event) error {
			close(started)
			<-release
			return nil
		}))
		relay.config.PublishTimeout = time.Millisecond

		type outcome struct {
			publications []error
			cleanupSafe  bool
		}
		result := make(chan outcome, 1)
		startedAt := time.Now()
		go func() {
			publications, cleanupSafe := relay.publishAll(
				context.Background(), unitBatch(1), time.Now().Add(time.Hour))
			result <- outcome{publications: publications, cleanupSafe: cleanupSafe}
		}()
		<-started
		got := <-result
		if elapsed := time.Since(startedAt); elapsed != time.Millisecond+PublisherJoinTimeout {
			t.Fatalf("publisher stuck elapsed = %s, want %s", elapsed, time.Millisecond+PublisherJoinTimeout)
		}
		close(release)
		synctest.Wait()
		if got.cleanupSafe || got.publications != nil {
			t.Fatalf("publishAll() = %+v, want cleanup-unsafe with no outcomes", got)
		}
	})
}

// A lease that is about to expire bounds the batch even when the publish
// timeout would allow more, so every claimed event is finalized while the
// relay still owns it.
func TestRelayPublishStopsBeforeLeaseExpiry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		relay := newUnitRelay(&relayStoreStub{}, publisherFunc(func(ctx context.Context, _ Event) error {
			<-ctx.Done()
			return ctx.Err()
		}))
		relay.config.PublishTimeout = time.Hour
		startedAt := time.Now()
		leaseExpiry := startedAt.Add(3 * PublisherJoinTimeout)
		publications, cleanupSafe := relay.publishAll(context.Background(), unitBatch(1), leaseExpiry)
		if !cleanupSafe || len(publications) != 1 || !errors.Is(publications[0], context.DeadlineExceeded) {
			t.Fatalf("publishAll() = %v cleanupSafe=%t, want lease-bounded deadline", publications, cleanupSafe)
		}
		if elapsed := time.Since(startedAt); elapsed != 2*PublisherJoinTimeout {
			t.Fatalf("lease-bounded elapsed = %s, want %s", elapsed, 2*PublisherJoinTimeout)
		}
	})
}

// The batch runs at most PublishConcurrency publications at a time and still
// publishes every claimed event exactly once.
func TestRelayPublishConcurrencyBound(t *testing.T) {
	t.Parallel()

	const (
		events      = 12
		concurrency = 4
	)
	var mutex sync.Mutex
	var live, peak int
	var total atomic.Int64
	var opened sync.Once
	release := make(chan struct{})
	relay := newUnitRelay(&relayStoreStub{}, publisherFunc(func(context.Context, Event) error {
		mutex.Lock()
		live++
		peak = max(peak, live)
		saturated := live >= concurrency
		mutex.Unlock()
		// The first full set of workers must be in the publisher at once before
		// any of them returns, which is what proves the bound is reached.
		if saturated {
			opened.Do(func() { close(release) })
		}
		<-release
		total.Add(1)
		mutex.Lock()
		live--
		mutex.Unlock()
		return nil
	}))
	relay.config.PublishConcurrency = concurrency

	batch := unitBatch(events)
	publications, cleanupSafe := relay.publishAll(t.Context(), batch, time.Now().Add(time.Hour))
	if !cleanupSafe || len(publications) != events {
		t.Fatalf("publishAll() outcomes = %d cleanupSafe=%t, want %d/true", len(publications), cleanupSafe, events)
	}
	if got := total.Load(); got != events {
		t.Fatalf("published %d events, want %d", got, events)
	}
	if peak > concurrency || peak == 0 {
		t.Fatalf("peak concurrency = %d, want (0,%d]", peak, concurrency)
	}
	for index, publication := range publications {
		if publication != nil || batch.Events[index].Event.ID != unitClaim(index+1).Event.ID {
			t.Fatalf("outcome %d = %v, want the matching claim published", index, publication)
		}
	}
}

func TestRelayPublicationErrorClasses(t *testing.T) {
	t.Parallel()

	if got := publicationErrorClass(fmtPermanent()); got != "publisher_permanent" {
		t.Fatalf("permanent class = %q", got)
	}
	if got := publicationErrorClass(context.DeadlineExceeded); got != "publisher_timeout" {
		t.Fatalf("timeout class = %q", got)
	}
}

func TestRelayDurationHelpers(t *testing.T) {
	t.Parallel()

	now := time.Now()
	if got := earliest(now, now.Add(time.Second)); !got.Equal(now) {
		t.Fatalf("earliest() = %s, want %s", got, now)
	}
	if got := earliest(now.Add(time.Second), now); !got.Equal(now) {
		t.Fatalf("earliest(reversed) = %s, want %s", got, now)
	}
}

func TestRelayOperationClassification(t *testing.T) {
	t.Parallel()

	failure := errors.New("x")
	if got := operationOutcome(nil); got != outcomeSuccess {
		t.Errorf("operationOutcome(nil) = %q, want %q", got, outcomeSuccess)
	}
	if got := operationOutcome(failure); got != outcomeError {
		t.Errorf("operationOutcome(error) = %q, want %q", got, outcomeError)
	}
}

// A panicking publisher still lets the rest of the batch finalize, then stops
// the relay so the process restarts on a broken adapter.
func TestRelayCycleFinalizesBatchAfterPublisherPanic(t *testing.T) {
	t.Parallel()

	retried := 0
	store := &relayStoreStub{
		claim: func(context.Context, time.Duration, int) (ClaimedBatch, error) { return unitBatch(2), nil },
		scheduleRetryBatch: func(_ context.Context, _ string, retries []RetryDirective) error {
			retried = len(retries)
			return nil
		},
	}
	relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error { panic("publisher") }))
	result, stop := relay.runCycle(t.Context(), nil, farSchedule())
	if !stop || !errors.Is(result.Err, ErrPublisherPanic) || result.CleanupUnsafe || retried != 2 {
		t.Fatalf("runCycle(panic) = %+v stop=%t retried=%d", result, stop, retried)
	}
}

// A failed finalization stops the cycle: the lease state is now unknown, so
// claiming more work would risk publishing it twice.
func TestRelayCycleStopsOnFinalizationFailure(t *testing.T) {
	t.Parallel()

	stateErr := errors.New("state")
	store := &relayStoreStub{
		claim: func(context.Context, time.Duration, int) (ClaimedBatch, error) { return unitBatch(1), nil },
		scheduleRetryBatch: func(context.Context, string, []RetryDirective) error {
			return stateErr
		},
	}
	relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return errors.New("temporary") }))
	result, stop := relay.runCycle(t.Context(), nil, farSchedule())
	if !stop || !errors.Is(result.Err, stateErr) || result.CleanupUnsafe {
		t.Fatalf("runCycle(state) = %+v stop=%t", result, stop)
	}
}

// Cancellation still finalizes: releasing the lease now beats waiting for
// recovery, and the cycle then stops without reporting a failure.
func TestRelayCycleFinalizesUnderCancellation(t *testing.T) {
	t.Parallel()

	released := 0
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	store := &relayStoreStub{
		claim: func(context.Context, time.Duration, int) (ClaimedBatch, error) { return unitBatch(1), nil },
		scheduleRetryBatch: func(_ context.Context, _ string, retries []RetryDirective) error {
			released = len(retries)
			return nil
		},
	}
	relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return context.Canceled }))
	result, stop := relay.runCycle(ctx, nil, farSchedule())
	if !stop || result.Err != nil || result.CleanupUnsafe || released != 1 {
		t.Fatalf("runCycle(canceled) = %+v stop=%t released=%d", result, stop, released)
	}
}

// A publisher that ignores cancellation makes cleanup unsafe for the whole
// cycle: its goroutine can still touch dependencies the process is closing.
func TestRelayCycleStuckPublisherIsCleanupUnsafe(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		finalized := false
		store := &relayStoreStub{
			claim: func(context.Context, time.Duration, int) (ClaimedBatch, error) { return unitBatch(1), nil },
			markUnorderedPublishedBatch: func(_ context.Context, _ string, ids []string) ([]string, error) {
				finalized = true
				return ids, nil
			},
		}
		relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error {
			<-release
			return nil
		}))
		relay.config.PublishTimeout = time.Millisecond

		result, stop := relay.runCycle(context.Background(), nil, farSchedule())
		if !stop || !result.CleanupUnsafe || !errors.Is(result.Err, ErrPublisherStuck) {
			t.Fatalf("runCycle(stuck) = %+v stop=%t", result, stop)
		}
		if finalized {
			t.Fatal("a stuck cycle finalized a claim whose outcome is unknown")
		}
		close(release)
		synctest.Wait()
	})
}
