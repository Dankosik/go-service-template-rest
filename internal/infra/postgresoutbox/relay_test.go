package postgresoutbox

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

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
	claim                     func(context.Context, time.Duration, int) (ClaimedBatch, error)
	markUnorderedPublished    func(context.Context, string, string) error
	markOrderedPublished      func(context.Context, string, OrderedDirective) error
	markPublishedBatch        func(context.Context, string, []string) ([]string, error)
	markOrderedPublishedBatch func(context.Context, string, []OrderedDirective) ([]string, error)
	scheduleRetryBatch        func(context.Context, string, []RetryDirective) error
	markPoisonedBatch         func(context.Context, string, []PoisonDirective) error
	get                       func(context.Context, string) (Record, error)
	cleanup                   func(context.Context, time.Duration, int) (int, error)
	observe                   func(context.Context) (StateObservation, error)
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

func (s *relayStoreStub) MarkPublishedBatch(ctx context.Context, token string, ids []string) ([]string, error) {
	if s.markPublishedBatch == nil {
		// Default: the lease still owned every id, so the statement finalized all
		// of them and nothing reaches reconciliation.
		return ids, nil
	}
	return s.markPublishedBatch(ctx, token, ids)
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

func TestBackoff(t *testing.T) {
	t.Parallel()

	const (
		base    = time.Second
		maximum = 5 * time.Second
	)
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: 1, want: time.Second},
		{attempt: 2, want: 2 * time.Second},
		{attempt: 3, want: 4 * time.Second},
		{attempt: 4, want: maximum},
		{attempt: 100, want: maximum},
	}
	for _, test := range tests {
		got := retryDelay(base, maximum, test.attempt, func(limit time.Duration) time.Duration { return limit })
		if got != test.want {
			t.Errorf("attempt %d delay cap = %s, want %s", test.attempt, got, test.want)
		}
	}
	if got := retryDelay(base, maximum, 3, func(time.Duration) time.Duration { return 0 }); got != 0 {
		t.Fatalf("full-jitter lower bound = %s, want 0", got)
	}
	maxDuration := time.Duration(1<<63 - 1)
	if got := fullJitter(maxDuration); got < 0 || got > maxDuration {
		t.Fatalf("maximum full jitter = %s, want [0,%s]", got, maxDuration)
	}
}

func TestRelayCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		started := make(chan struct{})
		relay := newUnitRelay(&relayStoreStub{}, publisherFunc(func(ctx context.Context, _ Event) error {
			close(started)
			<-ctx.Done()
			return ctx.Err()
		}))
		// No PublishTimeout here on purpose: publishOne carries no budget of its
		// own, so the only deadline it can observe is the one its caller passes.
		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan error, 1)
		go func() { result <- relay.publishOne(ctx, unitClaim(1).Event) }()
		<-started
		cancel()
		synctest.Wait()
		if got := <-result; !errors.Is(got, context.Canceled) {
			t.Fatalf("publishOne() = %v, want context cancellation", got)
		}
	})
}

// A publisher that reports success once the batch's publication budget is gone
// has not proven the broker accepted the event, so the attempt stays retryable.
func TestRelayTimeoutRejectsNilCompletion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		relay := newUnitRelay(&relayStoreStub{}, publisherFunc(func(ctx context.Context, _ Event) error {
			<-ctx.Done()
			return nil
		}))
		failures, cleanupSafe := relay.publishAll(
			context.Background(), unitBatch(1), time.Now().Add(time.Hour))
		if !cleanupSafe || len(failures) != 1 || !errors.Is(failures[0], context.DeadlineExceeded) {
			t.Fatalf("publishAll() = %v cleanupSafe=%t, want deadline failure after nil completion",
				failures, cleanupSafe)
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
			failures    []error
			cleanupSafe bool
		}
		result := make(chan outcome, 1)
		startedAt := time.Now()
		go func() {
			failures, cleanupSafe := relay.publishAll(
				context.Background(), unitBatch(1), time.Now().Add(time.Hour))
			result <- outcome{failures: failures, cleanupSafe: cleanupSafe}
		}()
		<-started
		got := <-result
		if elapsed := time.Since(startedAt); elapsed != time.Millisecond+PublisherJoinTimeout {
			t.Fatalf("publisher stuck elapsed = %s, want %s", elapsed, time.Millisecond+PublisherJoinTimeout)
		}
		close(release)
		synctest.Wait()
		if got.cleanupSafe || got.failures != nil {
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
		failures, cleanupSafe := relay.publishAll(context.Background(), unitBatch(1), leaseExpiry)
		if !cleanupSafe || len(failures) != 1 || !errors.Is(failures[0], context.DeadlineExceeded) {
			t.Fatalf("publishAll() = %v cleanupSafe=%t, want lease-bounded deadline", failures, cleanupSafe)
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
	failures, cleanupSafe := relay.publishAll(t.Context(), batch, time.Now().Add(time.Hour))
	if !cleanupSafe || len(failures) != events {
		t.Fatalf("publishAll() outcomes = %d cleanupSafe=%t, want %d/true", len(failures), cleanupSafe, events)
	}
	if got := total.Load(); got != events {
		t.Fatalf("published %d events, want %d", got, events)
	}
	if peak > concurrency || peak == 0 {
		t.Fatalf("peak concurrency = %d, want (0,%d]", peak, concurrency)
	}
	for index, failure := range failures {
		if failure != nil || batch.Events[index].Event.ID != unitClaim(index+1).Event.ID {
			t.Fatalf("outcome %d = %v, want the matching claim published", index, failure)
		}
	}
}

// The lease has to outlast the publication budget plus the publisher-join
// bound, and saying so must survive both a too-short lease and durations near
// the overflow edge.
func TestRelayConfigRejectsLeaseThatCannotHoldPublication(t *testing.T) {
	t.Parallel()

	valid := unitRelayConfig()
	if err := ValidateRelayConfig(valid); err != nil {
		t.Fatalf("valid config: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(RelayConfig) RelayConfig
	}{
		{name: "lease equals publish plus join", mutate: func(c RelayConfig) RelayConfig {
			c.LeaseDuration = c.PublishTimeout + PublisherJoinTimeout
			return c
		}},
		{name: "lease and publish both saturate", mutate: func(c RelayConfig) RelayConfig {
			c.PublishTimeout = math.MaxInt64
			c.LeaseDuration = math.MaxInt64
			return c
		}},
	}
	for _, test := range tests {
		if err := ValidateRelayConfig(test.mutate(valid)); !errors.Is(err, ErrConfig) {
			t.Errorf("ValidateRelayConfig(%s) error = %v, want ErrConfig", test.name, err)
		}
	}
}

// Backoff at the maximum representable duration must stay inside its own bound
// rather than overflowing into a negative delay.
func TestRelayRetryDelayHoldsAtMaximumDuration(t *testing.T) {
	t.Parallel()

	config := unitRelayConfig()
	config.RetryBase = math.MaxInt64
	config.RetryMax = math.MaxInt64
	if err := ValidateRelayConfig(config); err != nil {
		t.Fatalf("maximum retry config: %v", err)
	}
	if got := retryDelay(config.RetryBase, config.RetryMax, 1, fullJitter); got < 0 || got > config.RetryMax {
		t.Fatalf("maximum retry delay = %s, want [0,%s]", got, config.RetryMax)
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

func TestRelayReadinessRequiresFreshObservation(t *testing.T) {
	t.Parallel()

	// Built through the constructor like every other relay here, so this test
	// keeps working if Ready starts reading a field only newRelay populates.
	// unitRelayConfig already uses a one-minute observation interval.
	relay := newUnitRelay(&relayStoreStub{}, noopPublisher())
	relay.ready.Store(true)
	relay.observedAtUnixNano.Store(time.Now().UnixNano())
	if !relay.Ready() {
		t.Fatal("Ready() = false after a fresh successful observation")
	}
	relay.observedAtUnixNano.Store(time.Now().Add(-90 * time.Second).UnixNano())
	if !relay.Ready() {
		t.Fatal("Ready() = false within the two-interval freshness window")
	}
	relay.observedAtUnixNano.Store(time.Now().Add(-3 * time.Minute).UnixNano())
	if relay.Ready() {
		t.Fatal("Ready() = true after the observation became stale")
	}
	relay.observedAtUnixNano.Store(time.Now().UnixNano())
	relay.StartDrain()
	if relay.Ready() {
		t.Fatal("Ready() = true while draining")
	}
	if got := observationStaleAfter(time.Duration(1<<63 - 1)); got != time.Duration(1<<63-1) {
		t.Fatalf("observationStaleAfter(max) = %s", got)
	}
}

func TestRelayRunPublishesAndDrains(t *testing.T) {
	t.Parallel()

	store := &relayStoreStub{}
	claimed := false
	store.claim = func(context.Context, time.Duration, int) (ClaimedBatch, error) {
		if claimed {
			return ClaimedBatch{}, nil
		}
		claimed = true
		return unitBatch(3), nil
	}
	var published atomic.Int64
	relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error {
		published.Add(1)
		return nil
	}))
	store.markPublishedBatch = func(_ context.Context, _ string, ids []string) ([]string, error) {
		relay.StartDrain() //nolint:contextcheck // Drain has no context-bearing variant.
		return ids, nil
	}

	result := relay.Run(t.Context())
	if result.Err != nil {
		t.Errorf("Run() stopped with an error after an ordinary drain: %v", result.Err)
	}
	if result.CleanupUnsafe {
		t.Error("Run() reported cleanup unsafe though no publisher outlived cancellation")
	}
	if !claimed {
		t.Error("Run() drained without ever claiming a batch")
	}
	if got := published.Load(); got != 3 {
		t.Errorf("published %d events of the claimed batch, want 3", got)
	}
	if relay.Ready() {
		t.Error("Ready() = true after Run() returned; a stopped relay is never ready")
	}
}

// A wake-up signal replaces the poll wait, so an append committed during an
// idle cycle is claimed without waiting out the poll interval.
func TestRelayRunWakesOnNotification(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		claims := 0
		store := &relayStoreStub{}
		relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return nil }))
		relay.config.PollInterval = time.Hour
		relay.listen = func(ctx context.Context, wake chan<- struct{}) {
			signal(wake)
			<-ctx.Done()
		}
		store.claim = func(context.Context, time.Duration, int) (ClaimedBatch, error) {
			claims++
			if claims == 2 {
				relay.StartDrain() //nolint:contextcheck // Drain has no context-bearing variant.
			}
			return ClaimedBatch{}, nil
		}

		startedAt := time.Now()
		result := relay.Run(context.Background())
		if result.Err != nil || result.CleanupUnsafe {
			t.Fatalf("Run() = %+v", result)
		}
		if claims != 2 || time.Since(startedAt) != 0 {
			t.Fatalf("claims=%d elapsed=%s, want 2 claims without a poll wait", claims, time.Since(startedAt))
		}
	})
}

func TestRelayRunMaintainsWhileEmpty(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		claims := 0
		observations := 0
		cleanups := 0
		store := &relayStoreStub{
			observe: func(context.Context) (StateObservation, error) {
				observations++
				return StateObservation{}, nil
			},
			cleanup: func(context.Context, time.Duration, int) (int, error) {
				cleanups++
				return 0, nil
			},
		}
		relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return nil }))
		relay.config.PollInterval = 2 * time.Second
		relay.config.ObservationInterval = time.Second
		relay.config.CleanupInterval = time.Second
		store.claim = func(context.Context, time.Duration, int) (ClaimedBatch, error) {
			claims++
			if claims == 2 {
				relay.StartDrain() //nolint:contextcheck // Drain has no context-bearing variant.
			}
			return ClaimedBatch{}, nil
		}

		started := time.Now()
		result := relay.Run(context.Background())
		if result.Err != nil || result.CleanupUnsafe || time.Since(started) != time.Second {
			t.Fatalf("Run() = %+v elapsed=%s", result, time.Since(started))
		}
		if claims != 2 || observations != 2 || cleanups != 1 {
			t.Fatalf("claims=%d observations=%d cleanups=%d, want 2/2/1", claims, observations, cleanups)
		}
	})
}

func TestRelayRunFailures(t *testing.T) {
	t.Parallel()

	if result := (*Relay)(nil).Run(t.Context()); !errors.Is(result.Err, ErrConfig) || result.CleanupUnsafe {
		t.Fatalf("nil Run() = %+v", result)
	}
	observeErr := errors.New("observe failed")
	relay := newUnitRelay(&relayStoreStub{
		observe: func(context.Context) (StateObservation, error) { return StateObservation{}, observeErr },
	}, publisherFunc(func(context.Context, Event) error { return nil }))
	if result := relay.Run(t.Context()); !errors.Is(result.Err, observeErr) {
		t.Fatalf("observe failure Run() = %+v", result)
	}
	claimErr := errors.New("claim failed")
	relay = newUnitRelay(&relayStoreStub{
		claim: func(context.Context, time.Duration, int) (ClaimedBatch, error) { return ClaimedBatch{}, claimErr },
	}, publisherFunc(func(context.Context, Event) error { return nil }))
	if result := relay.Run(t.Context()); !errors.Is(result.Err, claimErr) {
		t.Fatalf("claim failure Run() = %+v", result)
	}
}

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
		markPublishedBatch: func(_ context.Context, _ string, ids []string) ([]string, error) {
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
		markPublishedBatch: func(_ context.Context, _ string, ids []string) ([]string, error) {
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
		markPublishedBatch: func(_ context.Context, _ string, ids []string) ([]string, error) {
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

// A typed nil satisfies the interface but cannot publish, so construction has
// to reject it as firmly as an untyped nil.
func TestRelayRejectsMissingDependencies(t *testing.T) {
	t.Parallel()

	var typedNil *pointerPublisher
	if !nilInterface(typedNil) {
		t.Error("nilInterface(typed nil pointer) = false, want true")
	}
	if nilInterface(1) {
		t.Error("nilInterface(int) = true, want false")
	}
	if _, err := NewRelay(nil, noopPublisher(), nil, RelayConfig{}); !errors.Is(err, ErrConfig) {
		t.Fatalf("NewRelay(nil) error = %v", err)
	}
	var nilStore *relayStoreStub
	if _, err := newRelay(nilStore, noopPublisher(), nil, unitRelayConfig()); !errors.Is(err, ErrConfig) {
		t.Fatalf("newRelay(typed nil store) error = %v", err)
	}
	if _, err := newRelay(&relayStoreStub{}, nil, nil, unitRelayConfig()); !errors.Is(err, ErrConfig) {
		t.Fatalf("newRelay(nil publisher) error = %v", err)
	}
}

func TestRelayConfigRejectsOutOfRangeFields(t *testing.T) {
	t.Parallel()

	valid := unitRelayConfig()
	invalid := []struct {
		name   string
		mutate func(RelayConfig) RelayConfig
	}{
		{name: "zero value", mutate: func(RelayConfig) RelayConfig { return RelayConfig{} }},
		{name: "zero publish timeout", mutate: func(c RelayConfig) RelayConfig { c.PublishTimeout = 0; return c }},
		{name: "zero max attempts", mutate: func(c RelayConfig) RelayConfig { c.MaxAttempts = 0; return c }},
		{name: "zero retry base", mutate: func(c RelayConfig) RelayConfig { c.RetryBase = 0; return c }},
		{name: "retry max below base", mutate: func(c RelayConfig) RelayConfig { c.RetryMax = c.RetryBase - 1; return c }},
		{name: "zero observation interval", mutate: func(c RelayConfig) RelayConfig { c.ObservationInterval = 0; return c }},
		{name: "zero cleanup interval", mutate: func(c RelayConfig) RelayConfig { c.CleanupInterval = 0; return c }},
		{name: "zero published retention", mutate: func(c RelayConfig) RelayConfig { c.PublishedRetention = 0; return c }},
		{name: "zero cleanup batch size", mutate: func(c RelayConfig) RelayConfig { c.CleanupBatchSize = 0; return c }},
		{name: "zero batch size", mutate: func(c RelayConfig) RelayConfig { c.BatchSize = 0; return c }},
		{name: "batch size over maximum", mutate: func(c RelayConfig) RelayConfig { c.BatchSize = MaxBatchSize + 1; return c }},
		{name: "zero publish concurrency", mutate: func(c RelayConfig) RelayConfig { c.PublishConcurrency = 0; return c }},
		{name: "publish concurrency over maximum", mutate: func(c RelayConfig) RelayConfig {
			c.PublishConcurrency = MaxPublishConcurrency + 1
			return c
		}},
	}
	for _, test := range invalid {
		config := test.mutate(valid)
		if err := ValidateRelayConfig(config); !errors.Is(err, ErrConfig) {
			t.Errorf("ValidateRelayConfig(%s) error = %v, want ErrConfig", test.name, err)
		}
		if _, err := newRelay(&relayStoreStub{}, publisherFunc(func(context.Context, Event) error { return nil }), nil, config); !errors.Is(err, ErrConfig) {
			t.Errorf("newRelay(%s) error = %v, want ErrConfig", test.name, err)
		}
	}
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

// The next observation is measured from when the slow one finished, not from
// when it came due, so a store slower than the interval cannot queue up.
func TestRelayMaintainReschedulesFromEndOfSlowObservation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		relay := newUnitRelay(&relayStoreStub{
			observe: func(context.Context) (StateObservation, error) {
				time.Sleep(2 * time.Second)
				return StateObservation{}, errors.New("observe")
			},
		}, noopPublisher())
		relay.config.ObservationInterval = time.Second
		now := time.Now()
		due, err := relay.maintain(t.Context(), schedule{observation: now, cleanup: now.Add(time.Hour)})
		if err != nil {
			t.Fatalf("maintain(periodic observation failure) error = %v", err)
		}
		if elapsed := time.Since(now); elapsed != 2*time.Second {
			t.Fatalf("observation elapsed = %s, want 2s", elapsed)
		}
		if delay := due.observation.Sub(now); delay != 3*time.Second {
			t.Fatalf("next observation = %s after start, want 3s", delay)
		}
	})
}

// Retention is how the table stays bounded, so a failed cleanup stops the loop.
func TestRelayMaintainStopsOnCleanupFailure(t *testing.T) {
	t.Parallel()

	cleanupErr := errors.New("cleanup")
	relay := newUnitRelay(&relayStoreStub{
		cleanup: func(context.Context, time.Duration, int) (int, error) { return 0, cleanupErr },
	}, noopPublisher())
	now := time.Now()
	if _, err := relay.maintain(
		t.Context(), schedule{observation: now.Add(time.Hour), cleanup: now},
	); !errors.Is(err, cleanupErr) {
		t.Fatalf("maintain(cleanup) error = %v", err)
	}
}

// A full delete batch means more is waiting, so the next cleanup comes at poll
// speed rather than after the whole interval.
func TestRelayMaintainShortensDelayAfterFullCleanupBatch(t *testing.T) {
	t.Parallel()

	relay := newUnitRelay(&relayStoreStub{
		cleanup: func(_ context.Context, _ time.Duration, batch int) (int, error) { return batch, nil },
	}, noopPublisher())
	now := time.Now()
	due, err := relay.maintain(t.Context(), schedule{observation: now.Add(time.Hour), cleanup: now})
	if err != nil {
		t.Fatalf("maintain(full cleanup batch) error = %v", err)
	}
	if delay := time.Until(due.cleanup); delay <= 0 || delay > relay.config.PollInterval {
		t.Fatalf("full cleanup batch delay = %s, want (0,%s]", delay, relay.config.PollInterval)
	}
}

func TestRelayWaitStopsOnCancellationAndDrain(t *testing.T) {
	t.Parallel()

	relay := newUnitRelay(&relayStoreStub{}, noopPublisher())
	wake := make(chan struct{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if !relay.wait(ctx, wake, time.Hour) {
		t.Fatal("wait(canceled) did not report stop")
	}
	if relay.wait(t.Context(), wake, -time.Second) {
		t.Fatal("wait(negative duration) reported stop")
	}
	relay.StartDrain()
	if !relay.wait(t.Context(), wake, time.Hour) {
		t.Fatal("wait(drain) did not report stop")
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
			markPublishedBatch: func(_ context.Context, _ string, ids []string) ([]string, error) {
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

// Maintenance owns the cycle: a failed cleanup stops the relay before it claims,
// and a drain raised during maintenance starts no claim at all.
func TestRelayLoopStopsOnMaintenanceFailureAndDrain(t *testing.T) {
	t.Parallel()

	cleanupErr := errors.New("cleanup failed")
	claims := 0
	store := &relayStoreStub{
		cleanup: func(context.Context, time.Duration, int) (int, error) { return 0, cleanupErr },
		claim: func(context.Context, time.Duration, int) (ClaimedBatch, error) {
			claims++
			return ClaimedBatch{}, nil
		},
	}
	relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return nil }))
	relay.config.CleanupInterval = 0
	if result := relay.runLoop(t.Context(), nil); !errors.Is(result.Err, cleanupErr) || result.CleanupUnsafe {
		t.Fatalf("runLoop(cleanup failure) = %+v", result)
	}
	if claims != 0 {
		t.Fatalf("claims after maintenance failure = %d, want 0", claims)
	}

	store.cleanup = func(context.Context, time.Duration, int) (int, error) { return 0, nil }
	relay = newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return nil }))
	relay.config.CleanupInterval = 0
	store.cleanup = func(context.Context, time.Duration, int) (int, error) {
		relay.StartDrain() //nolint:contextcheck // Drain has no context-bearing variant.
		return 0, nil
	}
	if result := relay.runLoop(t.Context(), nil); result.Err != nil || result.CleanupUnsafe {
		t.Fatalf("runLoop(drain during maintenance) = %+v", result)
	}
	if claims != 0 {
		t.Fatalf("claims after a drain raised during maintenance = %d, want 0", claims)
	}
}

// Run and StartDrain stay safe on a nil relay and on a relay already draining,
// because the process lifecycle calls both without owning that state.
func TestRelayDrainBeforeRunStartsNoLoop(t *testing.T) {
	t.Parallel()

	(*Relay)(nil).StartDrain()

	claims := 0
	relay := newUnitRelay(&relayStoreStub{
		claim: func(context.Context, time.Duration, int) (ClaimedBatch, error) {
			claims++
			return ClaimedBatch{}, nil
		},
	}, publisherFunc(func(context.Context, Event) error { return nil }))
	relay.StartDrain()
	if result := relay.Run(t.Context()); result.Err != nil || result.CleanupUnsafe {
		t.Fatalf("Run(already draining) = %+v", result)
	}
	if claims != 0 || relay.Ready() {
		t.Fatalf("claims=%d ready=%t after running an already-drained relay", claims, relay.Ready())
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
		markPublishedBatch:     func(context.Context, string, []string) ([]string, error) { return nil, publishErr },
		markUnorderedPublished: func(context.Context, string, string) error { return ErrLeaseLost },
		get:                    func(context.Context, string) (Record, error) { return Record{}, publishErr },
	}, publisherFunc(func(context.Context, Event) error { return nil }))
	if err := relay.finalize(t.Context(), unitBatch(1), []error{nil}); !errors.Is(err, ErrProgressUnknown) {
		t.Fatalf("finalize(publish write failure) error = %v", err)
	}
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
