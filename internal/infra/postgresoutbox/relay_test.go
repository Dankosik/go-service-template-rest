package postgresoutbox

import (
	"context"
	"errors"
	"fmt"
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

type relayStoreStub struct {
	claim                     func(context.Context, time.Duration, int) (ClaimedBatch, error)
	markPublished             func(context.Context, string, ClaimedEvent) error
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

func (s *relayStoreStub) MarkPublished(ctx context.Context, token string, claim ClaimedEvent) error {
	if s.markPublished == nil {
		return nil
	}
	return s.markPublished(ctx, token, claim)
}

func (s *relayStoreStub) MarkPublishedBatch(ctx context.Context, token string, ids []string) ([]string, error) {
	if s.markPublishedBatch == nil {
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
		return Record{LeaseToken: "lease"}, nil
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
		relay.config.PublishTimeout = time.Hour
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

func TestRelayTimeoutRejectsNilCompletion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		relay := newUnitRelay(&relayStoreStub{}, publisherFunc(func(ctx context.Context, _ Event) error {
			<-ctx.Done()
			return nil
		}))
		got := relay.publishOne(context.Background(), unitClaim(1).Event)
		if !errors.Is(got, context.DeadlineExceeded) {
			t.Fatalf("publishOne() = %v, want deadline failure after nil completion", got)
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
		if elapsed := time.Since(startedAt); elapsed != time.Millisecond+publisherJoinTimeout {
			t.Fatalf("publisher stuck elapsed = %s, want %s", elapsed, time.Millisecond+publisherJoinTimeout)
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
		leaseExpiry := startedAt.Add(3 * publisherJoinTimeout)
		failures, cleanupSafe := relay.publishAll(context.Background(), unitBatch(1), leaseExpiry)
		if !cleanupSafe || len(failures) != 1 || !errors.Is(failures[0], context.DeadlineExceeded) {
			t.Fatalf("publishAll() = %v cleanupSafe=%t, want lease-bounded deadline", failures, cleanupSafe)
		}
		if elapsed := time.Since(startedAt); elapsed != 2*publisherJoinTimeout {
			t.Fatalf("lease-bounded elapsed = %s, want %s", elapsed, 2*publisherJoinTimeout)
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

func TestRelayConfigAndErrorClasses(t *testing.T) {
	t.Parallel()

	valid := unitRelayConfig()
	if err := validateRelayConfig(valid); err != nil {
		t.Fatalf("valid config: %v", err)
	}
	invalid := valid
	invalid.LeaseDuration = invalid.PublishTimeout + publisherJoinTimeout
	if err := validateRelayConfig(invalid); !errors.Is(err, ErrConfig) {
		t.Fatalf("invalid lease error = %v, want ErrConfig", err)
	}
	invalid = valid
	invalid.PublishTimeout = time.Duration(1<<63 - 1)
	invalid.LeaseDuration = time.Duration(1<<63 - 1)
	if err := validateRelayConfig(invalid); !errors.Is(err, ErrConfig) {
		t.Fatalf("overflowing lease error = %v, want ErrConfig", err)
	}
	extremeRetry := valid
	extremeRetry.RetryBase = time.Duration(1<<63 - 1)
	extremeRetry.RetryMax = time.Duration(1<<63 - 1)
	if err := validateRelayConfig(extremeRetry); err != nil {
		t.Fatalf("maximum retry config: %v", err)
	}
	if got := retryDelay(extremeRetry.RetryBase, extremeRetry.RetryMax, 1, fullJitter); got < 0 || got > extremeRetry.RetryMax {
		t.Fatalf("maximum retry delay = %s, want [0,%s]", got, extremeRetry.RetryMax)
	}

	if got := publicationErrorClass(fmtPermanent()); got != "publisher_permanent" {
		t.Fatalf("permanent class = %q", got)
	}
	if got := publicationErrorClass(context.DeadlineExceeded); got != "publisher_timeout" {
		t.Fatalf("timeout class = %q", got)
	}
}

func TestRelayReadinessRequiresFreshObservation(t *testing.T) {
	t.Parallel()

	relay := &Relay{config: RelayConfig{ObservationInterval: time.Minute}, drain: make(chan struct{})}
	relay.ready.Store(true)
	relay.observed.Store(time.Now().UnixNano())
	if !relay.Ready() {
		t.Fatal("Ready() = false after a fresh successful observation")
	}
	relay.observed.Store(time.Now().Add(-90 * time.Second).UnixNano())
	if !relay.Ready() {
		t.Fatal("Ready() = false within the two-interval freshness window")
	}
	relay.observed.Store(time.Now().Add(-3 * time.Minute).UnixNano())
	if relay.Ready() {
		t.Fatal("Ready() = true after the observation became stale")
	}
	relay.observed.Store(time.Now().UnixNano())
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
	if result.Err != nil || !result.CleanupSafe || published.Load() != 3 || !claimed || relay.Ready() {
		t.Fatalf("Run() = %+v published=%d claimed=%t ready=%t", result, published.Load(), claimed, relay.Ready())
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
		if result.Err != nil || !result.CleanupSafe {
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
		if result.Err != nil || !result.CleanupSafe || time.Since(started) != time.Second {
			t.Fatalf("Run() = %+v elapsed=%s", result, time.Since(started))
		}
		if claims != 2 || observations != 2 || cleanups != 1 {
			t.Fatalf("claims=%d observations=%d cleanups=%d, want 2/2/1", claims, observations, cleanups)
		}
	})
}

func TestRelayRunFailures(t *testing.T) {
	t.Parallel()

	if result := (*Relay)(nil).Run(t.Context()); !errors.Is(result.Err, ErrConfig) || !result.CleanupSafe {
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
			batch := ClaimedBatch{Token: "lease", Events: []ClaimedEvent{claim}}

			result, keepRunning := relay.publishBatch(t.Context(), batch, time.Now().Add(time.Hour))
			if !result.CleanupSafe || (result.Err != nil) != test.wantStateErr || keepRunning == test.wantStateErr {
				t.Fatalf("publishBatch() = %+v keep=%t", result, keepRunning)
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
		markPublished: func(_ context.Context, _ string, claim ClaimedEvent) error {
			individual = append(individual, claim.Event.ID)
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
		markPublished: func(_ context.Context, _ string, claim ClaimedEvent) error {
			reconciled = append(reconciled, claim.Event.ID)
			return ErrLeaseLost
		},
		get: func(_ context.Context, id string) (Record, error) {
			return Record{Event: Event{ID: id}, PublishedAt: time.Now()}, nil
		},
	}
	relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return nil }))
	claims := []ClaimedEvent{unitClaim(1), unitClaim(2)}
	if err := relay.markPublished(t.Context(), "lease", claims); err != nil {
		t.Fatalf("markPublished(short write) error = %v", err)
	}
	if len(reconciled) != 1 || reconciled[0] != claims[1].Event.ID {
		t.Fatalf("reconciled = %v, want only the event the batch left out", reconciled)
	}
	reconciled = nil

	store.get = func(context.Context, string) (Record, error) { return Record{LeaseToken: "other"}, nil }
	if err := relay.markPublished(t.Context(), "lease", []ClaimedEvent{unitClaim(1)}); !errors.Is(err, ErrProgressUnknown) {
		t.Fatalf("markPublished(lost lease) error = %v, want ErrProgressUnknown", err)
	}
}

func TestRelayHelpersAndValidation(t *testing.T) {
	t.Parallel()

	if got := minDuration(3*time.Second, 2*time.Second, 4*time.Second); got != 2*time.Second {
		t.Fatalf("minDuration() = %s", got)
	}
	now := time.Now()
	if got := earliest(now, now.Add(time.Second)); !got.Equal(now) {
		t.Fatalf("earliest() = %s, want %s", got, now)
	}
	if got := earliest(now.Add(time.Second), now); !got.Equal(now) {
		t.Fatalf("earliest(reversed) = %s, want %s", got, now)
	}
	if operationOutcome(nil) != "success" || operationOutcome(errors.New("x")) != "error" ||
		operationErrorType(nil) != "none" || operationErrorType(errors.New("x")) != "database" {
		t.Fatal("operation outcome/error type classification mismatch")
	}
	var typedNil *pointerPublisher
	if !nilInterface(typedNil) || nilInterface(1) {
		t.Fatal("nilInterface classification mismatch")
	}
	if _, err := NewRelay(nil, publisherFunc(func(context.Context, Event) error { return nil }), nil, RelayConfig{}); !errors.Is(err, ErrConfig) {
		t.Fatalf("NewRelay(nil) error = %v", err)
	}
	var nilStore *relayStoreStub
	if _, err := newRelay(nilStore, publisherFunc(func(context.Context, Event) error { return nil }), nil, unitRelayConfig()); !errors.Is(err, ErrConfig) {
		t.Fatalf("newRelay(typed nil store) error = %v", err)
	}
	if _, err := newRelay(&relayStoreStub{}, nil, nil, unitRelayConfig()); !errors.Is(err, ErrConfig) {
		t.Fatalf("newRelay(nil publisher) error = %v", err)
	}
	valid := unitRelayConfig()
	invalid := []RelayConfig{
		{},
		func() RelayConfig { value := valid; value.PublishTimeout = 0; return value }(),
		func() RelayConfig { value := valid; value.MaxAttempts = 0; return value }(),
		func() RelayConfig { value := valid; value.RetryBase = 0; return value }(),
		func() RelayConfig { value := valid; value.RetryMax = value.RetryBase - 1; return value }(),
		func() RelayConfig { value := valid; value.ObservationInterval = 0; return value }(),
		func() RelayConfig { value := valid; value.CleanupInterval = 0; return value }(),
		func() RelayConfig { value := valid; value.PublishedRetention = 0; return value }(),
		func() RelayConfig { value := valid; value.CleanupBatchSize = 0; return value }(),
		func() RelayConfig { value := valid; value.BatchSize = 0; return value }(),
		func() RelayConfig { value := valid; value.BatchSize = maxBatchSize + 1; return value }(),
		func() RelayConfig { value := valid; value.PublishConcurrency = 0; return value }(),
		func() RelayConfig { value := valid; value.PublishConcurrency = maxPublishConcurrency + 1; return value }(),
	}
	for index, config := range invalid {
		if err := validateRelayConfig(config); !errors.Is(err, ErrConfig) {
			t.Errorf("invalid config %d error = %v", index, err)
		}
		if _, err := newRelay(&relayStoreStub{}, publisherFunc(func(context.Context, Event) error { return nil }), nil, config); !errors.Is(err, ErrConfig) {
			t.Errorf("newRelay(invalid config %d) error = %v", index, err)
		}
	}
}

func TestRelayMaintenanceWaitAndProgressFailures(t *testing.T) {
	observeErr := errors.New("observe")
	synctest.Test(t, func(t *testing.T) {
		relay := newUnitRelay(&relayStoreStub{
			observe: func(context.Context) (StateObservation, error) {
				time.Sleep(2 * time.Second)
				return StateObservation{}, observeErr
			},
		}, publisherFunc(func(context.Context, Event) error { return nil }))
		relay.config.ObservationInterval = time.Second
		now := time.Now()
		nextObservation, nextCleanup := now, now.Add(time.Hour)
		if err := relay.maintain(t.Context(), now, &nextObservation, &nextCleanup); err != nil {
			t.Fatalf("maintain(periodic observation failure) error = %v", err)
		}
		if elapsed := time.Since(now); elapsed != 2*time.Second {
			t.Fatalf("observation elapsed = %s, want 2s", elapsed)
		}
		if delay := nextObservation.Sub(now); delay != 3*time.Second {
			t.Fatalf("next observation = %s after start, want 3s", delay)
		}
	})

	now := time.Now()
	cleanupErr := errors.New("cleanup")
	relay := newUnitRelay(&relayStoreStub{
		cleanup: func(context.Context, time.Duration, int) (int, error) { return 0, cleanupErr },
	}, publisherFunc(func(context.Context, Event) error { return nil }))
	nextObservation, nextCleanup := now.Add(time.Hour), now
	if err := relay.maintain(t.Context(), now, &nextObservation, &nextCleanup); !errors.Is(err, cleanupErr) {
		t.Fatalf("maintain(cleanup) error = %v", err)
	}
	relay = newUnitRelay(&relayStoreStub{
		cleanup: func(context.Context, time.Duration, int) (int, error) { return 100, nil },
	}, publisherFunc(func(context.Context, Event) error { return nil }))
	nextObservation, nextCleanup = now.Add(time.Hour), now
	if err := relay.maintain(t.Context(), now, &nextObservation, &nextCleanup); err != nil {
		t.Fatalf("maintain(full cleanup batch) error = %v", err)
	}
	if delay := time.Until(nextCleanup); delay <= 0 || delay > relay.config.PollInterval {
		t.Fatalf("full cleanup batch delay = %s, want (0,%s]", delay, relay.config.PollInterval)
	}

	wake := make(chan struct{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if relay.wait(ctx, wake, time.Hour) {
		t.Fatal("wait(canceled) = true")
	}
	if !relay.wait(t.Context(), wake, -time.Second) {
		t.Fatal("wait(negative duration) = false")
	}
	relay.StartDrain()
	if relay.wait(t.Context(), wake, time.Hour) {
		t.Fatal("wait(drain) = true")
	}

	markErr := errors.New("mark")
	relay = newUnitRelay(&relayStoreStub{}, publisherFunc(func(context.Context, Event) error { return nil }))
	relay.markEvent = func(context.Context, string, ClaimedEvent) error { return markErr }
	relay.getEvent = func(context.Context, string) (Record, error) { return Record{LeaseToken: "lease"}, nil }
	if err := relay.reconcilePublished(t.Context(), "lease", unitClaim(1)); !errors.Is(err, ErrProgressUnknown) {
		t.Fatalf("reconcilePublished(retry exhausted) error = %v", err)
	}
	reconcileErr := errors.New("reconcile")
	relay.getEvent = func(context.Context, string) (Record, error) { return Record{}, reconcileErr }
	if err := relay.reconcilePublished(t.Context(), "lease", unitClaim(1)); !errors.Is(err, ErrProgressUnknown) || !errors.Is(err, reconcileErr) {
		t.Fatalf("reconcilePublished(reconcile) error = %v", err)
	}
}

// A panicking publisher still lets the rest of the batch finalize, then stops
// the relay so the process restarts on a broken adapter.
func TestRelayCycleFatalAndStateFailures(t *testing.T) {
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
	result, keepRunning := relay.runCycle(t.Context(), nil, time.Now().Add(time.Hour), time.Now().Add(time.Hour))
	if keepRunning || !errors.Is(result.Err, ErrPublisherPanic) || !result.CleanupSafe || retried != 2 {
		t.Fatalf("runCycle(panic) = %+v keep=%t retried=%d", result, keepRunning, retried)
	}

	stateErr := errors.New("state")
	store = &relayStoreStub{
		claim: func(context.Context, time.Duration, int) (ClaimedBatch, error) { return unitBatch(1), nil },
		scheduleRetryBatch: func(context.Context, string, []RetryDirective) error {
			return stateErr
		},
	}
	relay = newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return errors.New("temporary") }))
	result, keepRunning = relay.runCycle(t.Context(), nil, time.Now().Add(time.Hour), time.Now().Add(time.Hour))
	if keepRunning || !errors.Is(result.Err, stateErr) || !result.CleanupSafe {
		t.Fatalf("runCycle(state) = %+v keep=%t", result, keepRunning)
	}

	// Cancellation still finalizes: releasing the lease now beats waiting for
	// recovery, and the cycle then stops without reporting a failure.
	released := 0
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	store = &relayStoreStub{
		claim: func(context.Context, time.Duration, int) (ClaimedBatch, error) { return unitBatch(1), nil },
		scheduleRetryBatch: func(_ context.Context, _ string, retries []RetryDirective) error {
			released = len(retries)
			return nil
		},
	}
	relay = newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return context.Canceled }))
	result, keepRunning = relay.runCycle(ctx, nil, time.Now().Add(time.Hour), time.Now().Add(time.Hour))
	if keepRunning || result.Err != nil || !result.CleanupSafe || released != 1 {
		t.Fatalf("runCycle(canceled) = %+v keep=%t released=%d", result, keepRunning, released)
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

		result, keepRunning := relay.runCycle(context.Background(), nil, time.Now().Add(time.Hour), time.Now().Add(time.Hour))
		if keepRunning || result.CleanupSafe || !errors.Is(result.Err, ErrPublisherStuck) {
			t.Fatalf("runCycle(stuck) = %+v keep=%t", result, keepRunning)
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

	telemetry, err := NewTelemetry(nil, nil)
	if err != nil {
		t.Fatalf("NewTelemetry(): %v", err)
	}
	t.Cleanup(telemetry.Close)

	poisonErr := errors.New("poison write failed")
	store := &relayStoreStub{
		markPoisonedBatch: func(context.Context, string, []PoisonDirective) error { return poisonErr },
	}
	relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return nil }))
	relay.telemetry = telemetry
	poisoned := []poisonedEvent{{claim: unitClaim(1), errorClass: "publisher_permanent"}}
	if err := relay.markPoisoned(t.Context(), "lease", poisoned); !errors.Is(err, poisonErr) {
		t.Fatalf("markPoisoned(write failure) error = %v", err)
	}

	store.markPoisonedBatch = nil
	if err := relay.markPoisoned(t.Context(), "lease", poisoned); err != nil {
		t.Fatalf("markPoisoned() error = %v", err)
	}
	if err := relay.markPoisoned(t.Context(), "lease", nil); err != nil {
		t.Fatalf("markPoisoned(empty) error = %v", err)
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
	if result := relay.runLoop(t.Context(), nil); !errors.Is(result.Err, cleanupErr) || !result.CleanupSafe {
		t.Fatalf("runLoop(cleanup failure) = %+v", result)
	}
	if claims != 0 {
		t.Fatalf("claims after maintenance failure = %d, want 0", claims)
	}

	store.cleanup = func(context.Context, time.Duration, int) (int, error) { return 0, nil }
	relay = newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return nil }))
	relay.config.CleanupInterval = 0
	relay.markEvent = func(context.Context, string, ClaimedEvent) error { return nil }
	store.cleanup = func(context.Context, time.Duration, int) (int, error) {
		relay.StartDrain() //nolint:contextcheck // Drain has no context-bearing variant.
		return 0, nil
	}
	if result := relay.runLoop(t.Context(), nil); result.Err != nil || !result.CleanupSafe {
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
	if result := relay.Run(t.Context()); result.Err != nil || !result.CleanupSafe {
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
		markPublishedBatch: func(context.Context, string, []string) ([]string, error) { return nil, publishErr },
		markPublished:      func(context.Context, string, ClaimedEvent) error { return ErrLeaseLost },
		get:                func(context.Context, string) (Record, error) { return Record{}, publishErr },
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
	batch := ClaimedBatch{Token: "lease", Events: make([]ClaimedEvent, 0, events)}
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
