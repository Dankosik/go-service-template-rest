package postgresoutbox

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

type publisherFunc func(context.Context, Event) error

func (publish publisherFunc) Publish(ctx context.Context, event Event) error {
	return publish(ctx, event)
}

type relayStoreStub struct {
	claim         func(context.Context, time.Duration) (ClaimedEvent, error)
	markPublished func(context.Context, ClaimedEvent) error
	scheduleRetry func(context.Context, string, string, string, time.Duration) error
	markPoisoned  func(context.Context, string, string, string) error
	get           func(context.Context, string) (Record, error)
	cleanup       func(context.Context, time.Duration, int) (int, error)
	observe       func(context.Context) (StateObservation, error)
}

func (s *relayStoreStub) MarkPublished(ctx context.Context, claim ClaimedEvent) error {
	if s.markPublished == nil {
		return nil
	}
	return s.markPublished(ctx, claim)
}

func (s *relayStoreStub) Claim(ctx context.Context, lease time.Duration) (ClaimedEvent, error) {
	if s.claim == nil {
		return ClaimedEvent{}, ErrNoWork
	}
	return s.claim(ctx, lease)
}

func (s *relayStoreStub) ScheduleRetry(ctx context.Context, id, token, class string, delay time.Duration) error {
	if s.scheduleRetry == nil {
		return nil
	}
	return s.scheduleRetry(ctx, id, token, class, delay)
}

func (s *relayStoreStub) MarkPoisoned(ctx context.Context, id, token, class string) error {
	if s.markPoisoned == nil {
		return nil
	}
	return s.markPoisoned(ctx, id, token, class)
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
		relay := &Relay{
			publisher: publisherFunc(func(ctx context.Context, _ Event) error {
				close(started)
				<-ctx.Done()
				return ctx.Err()
			}),
			config: RelayConfig{PublishTimeout: time.Hour},
		}
		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan publishResult, 1)
		go func() { result <- relay.callPublisher(ctx, outboxEventForUnit()) }()
		<-started
		cancel()
		synctest.Wait()
		got := <-result
		if !got.cleanupSafe || !errors.Is(got.err, context.Canceled) {
			t.Fatalf("callPublisher() = %+v, want joined context cancellation", got)
		}
	})
}

func TestRelayTimeoutRejectsNilCompletion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		relay := &Relay{
			publisher: publisherFunc(func(ctx context.Context, _ Event) error {
				<-ctx.Done()
				return nil
			}),
			config: RelayConfig{PublishTimeout: time.Second},
		}
		got := relay.callPublisher(context.Background(), outboxEventForUnit())
		if !got.cleanupSafe || !errors.Is(got.err, context.DeadlineExceeded) {
			t.Fatalf("callPublisher() = %+v, want deadline failure after nil completion", got)
		}
	})
}

func TestRelayPublisherPanic(t *testing.T) {
	t.Parallel()

	relay := &Relay{
		publisher: publisherFunc(func(context.Context, Event) error { panic("publisher secret") }),
		config:    RelayConfig{PublishTimeout: time.Second},
	}
	result := relay.callPublisher(t.Context(), outboxEventForUnit())
	if !result.cleanupSafe || !errors.Is(result.err, ErrPublisherPanic) {
		t.Fatalf("callPublisher() = %+v, want cleanup-safe ErrPublisherPanic", result)
	}
}

func TestRelayPublisherStuck(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		started := make(chan struct{})
		release := make(chan struct{})
		relay := &Relay{
			publisher: publisherFunc(func(context.Context, Event) error {
				close(started)
				<-release
				return nil
			}),
			config: RelayConfig{PublishTimeout: time.Millisecond},
		}
		result := make(chan publishResult, 1)
		startedAt := time.Now()
		go func() { result <- relay.callPublisher(context.Background(), outboxEventForUnit()) }()
		<-started
		got := <-result
		if elapsed := time.Since(startedAt); elapsed != time.Millisecond+publisherJoinTimeout {
			t.Fatalf("publisher stuck elapsed = %s, want %s", elapsed, time.Millisecond+publisherJoinTimeout)
		}
		close(release)
		synctest.Wait()
		if got.cleanupSafe || !errors.Is(got.err, ErrPublisherStuck) {
			t.Fatalf("callPublisher() = %+v, want cleanup-unsafe ErrPublisherStuck", got)
		}
	})
}

func TestRelayConfigAndErrorClasses(t *testing.T) {
	t.Parallel()

	valid := RelayConfig{
		PollInterval:        time.Millisecond,
		PublishTimeout:      time.Second,
		LeaseDuration:       3 * time.Second,
		MaxAttempts:         10,
		RetryBase:           time.Second,
		RetryMax:            time.Minute,
		ObservationInterval: time.Second,
		CleanupInterval:     time.Minute,
		PublishedRetention:  time.Hour,
		CleanupBatchSize:    1000,
	}
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
	store.claim = func(context.Context, time.Duration) (ClaimedEvent, error) {
		if claimed {
			return ClaimedEvent{}, ErrNoWork
		}
		claimed = true
		return ClaimedEvent{Event: outboxEventForUnit(), Token: "lease", CycleAttemptCount: 1}, nil
	}
	published := false
	relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error {
		published = true
		return nil
	}))
	relay.markEvent = func(_ context.Context, _ ClaimedEvent) error {
		relay.StartDrain() //nolint:contextcheck // Drain has no context-bearing variant.
		return nil
	}

	result := relay.Run(t.Context())
	if result.Err != nil || !result.CleanupSafe || !published || !claimed || relay.Ready() {
		t.Fatalf("Run() = %+v published=%t claimed=%t ready=%t", result, published, claimed, relay.Ready())
	}
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
		store.claim = func(_ context.Context, _ time.Duration) (ClaimedEvent, error) {
			claims++
			if claims == 2 {
				relay.StartDrain() //nolint:contextcheck // Drain has no context-bearing variant.
			}
			return ClaimedEvent{}, ErrNoWork
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
		claim: func(context.Context, time.Duration) (ClaimedEvent, error) { return ClaimedEvent{}, claimErr },
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
				markPoisoned: func(_ context.Context, _, _, class string) error {
					poisonClass = class
					return test.storeErr
				},
				scheduleRetry: func(_ context.Context, _, _, class string, delay time.Duration) error {
					retryClass = class
					if delay != test.wantDelay {
						t.Errorf("retry delay = %s, want %s", delay, test.wantDelay)
					}
					return test.storeErr
				},
			}
			relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return test.publishErr }))
			result := relay.publish(t.Context(), ClaimedEvent{
				Event: outboxEventForUnit(), Token: "lease", CycleAttemptCount: test.attempt,
			})
			if !result.cleanupSafe || (result.err != nil) != test.wantStateErr {
				t.Fatalf("publish() = %+v", result)
			}
			if poisonClass != test.wantPoison || retryClass != test.wantRetry {
				t.Fatalf("poison=%q retry=%q, want %q/%q", poisonClass, retryClass, test.wantPoison, test.wantRetry)
			}
		})
	}
}

func TestRelayHelpersAndValidation(t *testing.T) {
	t.Parallel()

	if got := minDuration(3*time.Second, 2*time.Second, 4*time.Second); got != 2*time.Second {
		t.Fatalf("minDuration() = %s", got)
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

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if relay.wait(ctx, time.Hour) {
		t.Fatal("wait(canceled) = true")
	}
	if !relay.wait(t.Context(), -time.Second) {
		t.Fatal("wait(negative duration) = false")
	}
	relay.StartDrain()
	if relay.wait(t.Context(), time.Hour) {
		t.Fatal("wait(drain) = true")
	}

	markErr := errors.New("mark")
	relay = newUnitRelay(&relayStoreStub{}, publisherFunc(func(context.Context, Event) error { return nil }))
	relay.markEvent = func(context.Context, ClaimedEvent) error { return markErr }
	relay.getEvent = func(context.Context, string) (Record, error) { return Record{LeaseToken: "lease"}, nil }
	if err := relay.markPublished(t.Context(), ClaimedEvent{Event: outboxEventForUnit(), Token: "lease"}); !errors.Is(err, ErrProgressUnknown) {
		t.Fatalf("markPublished(retry exhausted) error = %v", err)
	}
	reconcileErr := errors.New("reconcile")
	relay.getEvent = func(context.Context, string) (Record, error) { return Record{}, reconcileErr }
	if err := relay.markPublished(t.Context(), ClaimedEvent{Event: outboxEventForUnit(), Token: "lease"}); !errors.Is(err, ErrProgressUnknown) || !errors.Is(err, reconcileErr) {
		t.Fatalf("markPublished(reconcile) error = %v", err)
	}
}

func TestRelayAttemptFatalAndStateFailures(t *testing.T) {
	t.Parallel()

	claim := ClaimedEvent{Event: outboxEventForUnit(), Token: "lease", CycleAttemptCount: 1}
	store := &relayStoreStub{claim: func(context.Context, time.Duration) (ClaimedEvent, error) { return claim, nil }}
	relay := newUnitRelay(store, publisherFunc(func(context.Context, Event) error { panic("publisher") }))
	if result, keepRunning := relay.runAttempt(t.Context(), time.Now().Add(time.Hour), time.Now().Add(time.Hour)); keepRunning || !errors.Is(result.Err, ErrPublisherPanic) || !result.CleanupSafe {
		t.Fatalf("runAttempt(panic) = %+v keep=%t", result, keepRunning)
	}

	stateErr := errors.New("state")
	store = &relayStoreStub{
		claim:         func(context.Context, time.Duration) (ClaimedEvent, error) { return claim, nil },
		scheduleRetry: func(context.Context, string, string, string, time.Duration) error { return stateErr },
	}
	relay = newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return errors.New("temporary") }))
	if result, keepRunning := relay.runAttempt(t.Context(), time.Now().Add(time.Hour), time.Now().Add(time.Hour)); keepRunning || !errors.Is(result.Err, stateErr) || !result.CleanupSafe {
		t.Fatalf("runAttempt(state) = %+v keep=%t", result, keepRunning)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	store = &relayStoreStub{claim: func(context.Context, time.Duration) (ClaimedEvent, error) { return claim, nil }}
	relay = newUnitRelay(store, publisherFunc(func(context.Context, Event) error { return context.Canceled }))
	if result, keepRunning := relay.runAttempt(ctx, time.Now().Add(time.Hour), time.Now().Add(time.Hour)); keepRunning || result.Err != nil || !result.CleanupSafe {
		t.Fatalf("runAttempt(canceled) = %+v keep=%t", result, keepRunning)
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
		PollInterval: time.Second, PublishTimeout: time.Second, LeaseDuration: 3 * time.Second,
		MaxAttempts: 3, RetryBase: time.Second, RetryMax: time.Minute,
		ObservationInterval: time.Minute, CleanupInterval: time.Hour,
		PublishedRetention: time.Hour, CleanupBatchSize: 100,
	}
}

func fmtPermanent() error {
	return errors.Join(errors.New("adapter detail"), ErrPermanentPublication)
}

func outboxEventForUnit() Event {
	return Event{ID: "event", Payload: []byte(`{}`), Metadata: []byte(`{}`)}
}
