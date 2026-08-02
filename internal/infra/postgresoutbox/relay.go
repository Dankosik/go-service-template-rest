package postgresoutbox

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

const publisherJoinTimeout = time.Second

var (
	ErrPublisherStuck  = errors.New("outbox publisher did not stop after cancellation")
	ErrPublisherPanic  = errors.New("outbox publisher panicked")
	ErrProgressUnknown = errors.New("outbox published-state progress is unknown")
)

type RelayConfig struct {
	PollInterval        time.Duration
	PublishTimeout      time.Duration
	LeaseDuration       time.Duration
	MaxAttempts         int
	RetryBase           time.Duration
	RetryMax            time.Duration
	ObservationInterval time.Duration
	CleanupInterval     time.Duration
	PublishedRetention  time.Duration
	CleanupBatchSize    int
}

type RelayResult struct {
	CleanupSafe bool
	Err         error
}

type relayStore interface {
	Claim(ctx context.Context, lease time.Duration) (ClaimedEvent, error)
	MarkPublished(ctx context.Context, id, token string) error
	ScheduleRetry(ctx context.Context, id, token, class string, delay time.Duration) error
	MarkPoisoned(ctx context.Context, id, token, class string) error
	Get(ctx context.Context, id string) (Record, error)
	CleanupPublished(ctx context.Context, retention time.Duration, batch int) (int, error)
	Observe(ctx context.Context) (StateObservation, error)
}

type Relay struct {
	store     relayStore
	publisher Publisher
	telemetry *Telemetry
	config    RelayConfig
	drain     chan struct{}
	drainOnce sync.Once
	ready     atomic.Bool
	observed  atomic.Int64
	inflight  atomic.Int64
	jitter    func(time.Duration) time.Duration
	markEvent func(context.Context, string, string) error
	getEvent  func(context.Context, string) (Record, error)
}

func NewRelay(store *Store, publisher Publisher, telemetry *Telemetry, config RelayConfig) (*Relay, error) {
	if store == nil || store.pool == nil {
		return nil, fmt.Errorf("%w: outbox store is required", ErrConfig)
	}
	return newRelay(store.withTelemetry(telemetry), publisher, telemetry, config)
}

func newRelay(store relayStore, publisher Publisher, telemetry *Telemetry, config RelayConfig) (*Relay, error) {
	if store == nil || nilInterface(store) {
		return nil, fmt.Errorf("%w: outbox store is required", ErrConfig)
	}
	if publisher == nil || nilInterface(publisher) {
		return nil, fmt.Errorf("%w: outbox publisher is required", ErrConfig)
	}
	if err := validateRelayConfig(config); err != nil {
		return nil, err
	}
	return &Relay{
		store:     store,
		publisher: publisher,
		telemetry: telemetry,
		config:    config,
		drain:     make(chan struct{}),
		markEvent: store.MarkPublished,
		getEvent:  store.Get,
		jitter:    fullJitter,
	}, nil
}

func (r *Relay) Ready() bool {
	if r == nil || !r.ready.Load() || done(r.drain) {
		return false
	}
	observedAt := r.observed.Load()
	return observedAt != 0 && time.Since(time.Unix(0, observedAt)) <= r.config.ObservationInterval
}

func (r *Relay) StartDrain() {
	if r == nil {
		return
	}
	r.drainOnce.Do(func() {
		close(r.drain)
		r.recordOperation(context.Background(), "drain", "started", "none", time.Now())
	})
	r.ready.Store(false)
	r.observeProcessState()
}

// Run owns one claim and publication at a time. StartDrain stops new claims but
// lets the current attempt finish; canceling ctx forces that attempt to stop.
func (r *Relay) Run(ctx context.Context) (result RelayResult) {
	result.CleanupSafe = true
	if r == nil {
		result.Err = fmt.Errorf("%w: relay is required", ErrConfig)
		return result
	}
	defer func() {
		r.ready.Store(false)
		r.observeProcessState()
	}()

	if err := r.observe(ctx); err != nil {
		result.Err = err
		return result
	}
	if done(r.drain) || ctx.Err() != nil {
		return result
	}
	r.ready.Store(true)
	r.observeProcessState()
	return r.runLoop(ctx)
}

func (r *Relay) runLoop(ctx context.Context) RelayResult {
	result := RelayResult{CleanupSafe: true}
	nextObservation := time.Now().Add(r.config.ObservationInterval)
	nextCleanup := time.Now().Add(r.config.CleanupInterval)
	for {
		if r.stopped(ctx) {
			return result
		}
		now := time.Now()
		if err := r.maintain(ctx, now, &nextObservation, &nextCleanup); err != nil {
			result.Err = err
			return result
		}
		if r.stopped(ctx) {
			return result
		}
		if attemptResult, keepRunning := r.runAttempt(ctx, nextObservation, nextCleanup); !keepRunning {
			return attemptResult
		}
	}
}

func (r *Relay) maintain(
	ctx context.Context,
	now time.Time,
	nextObservation *time.Time,
	nextCleanup *time.Time,
) error {
	if !now.Before(*nextObservation) {
		if err := r.observe(ctx); err != nil {
			return err
		}
		*nextObservation = now.Add(r.config.ObservationInterval)
	}
	if !now.Before(*nextCleanup) {
		if err := r.cleanup(ctx); err != nil {
			return err
		}
		*nextCleanup = now.Add(r.config.CleanupInterval)
	}
	return nil
}

func (r *Relay) runAttempt(
	ctx context.Context,
	nextObservation time.Time,
	nextCleanup time.Time,
) (RelayResult, bool) {
	result := RelayResult{CleanupSafe: true}
	claim, err := r.store.Claim(ctx, r.config.LeaseDuration)
	if errors.Is(err, ErrNoWork) {
		keepRunning := r.wait(ctx, minDuration(r.config.PollInterval, time.Until(nextObservation), time.Until(nextCleanup)))
		return result, keepRunning
	}
	if err != nil {
		result.Err = fmt.Errorf("claim outbox event: %w", err)
		return result, false
	}
	attempt := r.publish(ctx, claim)
	if !attempt.cleanupSafe {
		return RelayResult{CleanupSafe: false, Err: attempt.err}, false
	}
	if attempt.err != nil && (errors.Is(attempt.err, ErrPublisherPanic) || errors.Is(attempt.err, ErrProgressUnknown)) {
		result.Err = attempt.err
		return result, false
	}
	if ctx.Err() != nil {
		return result, false
	}
	if attempt.err != nil {
		result.Err = attempt.err
		return result, false
	}
	return result, true
}

func (r *Relay) stopped(ctx context.Context) bool {
	return done(r.drain) || ctx.Err() != nil
}

type publishResult struct {
	err         error
	cleanupSafe bool
}

func (r *Relay) publish(ctx context.Context, claim ClaimedEvent) publishResult {
	r.inflight.Store(1)
	r.observeProcessState()
	defer func() {
		r.inflight.Store(0)
		r.observeProcessState()
	}()

	started := time.Now()
	published := r.callPublisher(ctx, claim.Event)
	if !published.cleanupSafe {
		r.recordOperation(ctx, "publish", "error", "stuck", started)
		if r.telemetry != nil {
			r.telemetry.LogPublisherStuck(ctx)
		}
		return published
	}
	if errors.Is(published.err, ErrPublisherPanic) {
		r.recordOperation(ctx, "publish", "error", "panic", started)
		return published
	}
	if published.err == nil {
		r.recordOperation(ctx, "publish", "success", "none", started)
		if err := r.markPublished(ctx, claim); err != nil {
			return publishResult{err: err, cleanupSafe: true}
		}
		return publishResult{cleanupSafe: true}
	}
	r.recordOperation(ctx, "publish", "error", publicationErrorClass(published.err), started)
	if ctx.Err() != nil {
		return publishResult{cleanupSafe: true}
	}

	errorClass := publicationErrorClass(published.err)
	poisonClass := ""
	var err error
	switch {
	case errors.Is(published.err, ErrPermanentPublication):
		poisonClass = "publisher_permanent"
		err = r.store.MarkPoisoned(ctx, claim.Event.ID, claim.Token, poisonClass)
	case claim.CycleAttemptCount >= r.config.MaxAttempts:
		poisonClass = "attempt_exhausted"
		err = r.store.MarkPoisoned(ctx, claim.Event.ID, claim.Token, poisonClass)
	default:
		delay := retryDelay(r.config.RetryBase, r.config.RetryMax, claim.CycleAttemptCount, r.jitter)
		err = r.store.ScheduleRetry(ctx, claim.Event.ID, claim.Token, errorClass, delay)
	}
	if err != nil {
		return publishResult{err: fmt.Errorf("record outbox publication failure: %w", err), cleanupSafe: true}
	}
	if poisonClass != "" && r.telemetry != nil {
		r.telemetry.LogPoison(ctx, claim.Event.ID, poisonClass, claim.CycleAttemptCount)
	}
	return publishResult{cleanupSafe: true}
}

func (r *Relay) callPublisher(ctx context.Context, event Event) publishResult {
	attemptCtx, cancel := context.WithTimeout(ctx, r.config.PublishTimeout)
	defer cancel()
	type completion struct {
		err      error
		panicked bool
	}
	done := make(chan completion, 1)
	go func() {
		result := completion{}
		defer func() {
			if recover() != nil {
				result.err = ErrPublisherPanic
				result.panicked = true
			}
			done <- result
		}()
		result.err = r.publisher.Publish(attemptCtx, event)
	}()

	select {
	case result := <-done:
		if result.panicked {
			return publishResult{err: ErrPublisherPanic, cleanupSafe: true}
		}
		if err := attemptCtx.Err(); err != nil {
			return publishResult{err: err, cleanupSafe: true}
		}
		return publishResult{err: result.err, cleanupSafe: true}
	case <-attemptCtx.Done():
		cancel()
	}

	join := time.NewTimer(publisherJoinTimeout)
	defer join.Stop()
	select {
	case result := <-done:
		if result.panicked {
			return publishResult{err: ErrPublisherPanic, cleanupSafe: true}
		}
		return publishResult{err: attemptCtx.Err(), cleanupSafe: true}
	case <-join.C:
		return publishResult{err: ErrPublisherStuck, cleanupSafe: false}
	}
}

func (r *Relay) markPublished(ctx context.Context, claim ClaimedEvent) error {
	for range 2 {
		err := r.markEvent(ctx, claim.Event.ID, claim.Token)
		if err == nil {
			r.recordProgress(time.Now())
			return nil
		}
		record, getErr := r.getEvent(ctx, claim.Event.ID)
		if getErr == nil && !record.PublishedAt.IsZero() {
			r.recordOperation(ctx, "mark_published", "reconciled", "none", time.Now())
			r.recordProgress(time.Now())
			return nil
		}
		if getErr != nil || record.LeaseToken != claim.Token {
			return fmt.Errorf("%w: reconcile mark: %w", ErrProgressUnknown, errors.Join(err, getErr))
		}
	}
	return ErrProgressUnknown
}

func (r *Relay) observe(ctx context.Context) error {
	observation, err := r.store.Observe(ctx)
	if err != nil {
		return fmt.Errorf("observe outbox: %w", err)
	}
	if r.telemetry != nil {
		observedAt := time.Now()
		r.observed.Store(observedAt.UnixNano())
		r.telemetry.RecordObservation(observation, observedAt)
	} else {
		r.observed.Store(time.Now().UnixNano())
	}
	return nil
}

func (r *Relay) cleanup(ctx context.Context) error {
	_, err := r.store.CleanupPublished(ctx, r.config.PublishedRetention, r.config.CleanupBatchSize)
	if err != nil {
		return fmt.Errorf("cleanup outbox: %w", err)
	}
	return nil
}

func (r *Relay) wait(ctx context.Context, duration time.Duration) bool {
	if duration < 0 {
		duration = 0
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-r.drain:
		return false
	case <-timer.C:
		return true
	}
}

func (r *Relay) observeProcessState() {
	if r.telemetry != nil {
		r.telemetry.SetProcessState(r.Ready(), r.inflight.Load(), r.config.ObservationInterval)
	}
}

func (r *Relay) recordOperation(ctx context.Context, operation, outcome, errorType string, started time.Time) {
	if r.telemetry != nil {
		r.telemetry.RecordOperation(ctx, operation, outcome, errorType, time.Since(started))
	}
}

func (r *Relay) recordProgress(at time.Time) {
	if r.telemetry != nil {
		r.telemetry.RecordProgress(at)
	}
}

func validateRelayConfig(config RelayConfig) error {
	if config.PollInterval <= 0 || config.PublishTimeout <= 0 || config.LeaseDuration <= 0 ||
		config.RetryBase <= 0 || config.RetryMax < config.RetryBase || config.ObservationInterval <= 0 ||
		config.CleanupInterval <= 0 || config.PublishedRetention <= 0 {
		return fmt.Errorf("%w: outbox relay durations are invalid", ErrConfig)
	}
	if config.LeaseDuration <= config.PublishTimeout ||
		config.LeaseDuration-config.PublishTimeout <= publisherJoinTimeout {
		return fmt.Errorf("%w: lease duration must exceed publish and publisher-join timeouts", ErrConfig)
	}
	if config.MaxAttempts < 1 || config.MaxAttempts > 100 {
		return fmt.Errorf("%w: max attempts must be in range [1,100]", ErrConfig)
	}
	if config.CleanupBatchSize < 1 || config.CleanupBatchSize > 10_000 {
		return fmt.Errorf("%w: cleanup batch size must be in range [1,10000]", ErrConfig)
	}
	return nil
}

func fullJitter(limit time.Duration) time.Duration {
	if limit == time.Duration(1<<63-1) {
		// #nosec G404 -- Backoff jitter coordinates retries; it is not a secret or token.
		return time.Duration(rand.Int64())
	}
	// #nosec G404 -- Backoff jitter coordinates retries; it is not a secret or token.
	return time.Duration(rand.Int64N(limit.Nanoseconds() + 1))
}

func retryDelay(base, maximum time.Duration, attempt int, jitter func(time.Duration) time.Duration) time.Duration {
	limit := base
	for current := 1; current < attempt && limit < maximum; current++ {
		if limit > maximum/2 {
			limit = maximum
			break
		}
		limit *= 2
	}
	if limit > maximum {
		limit = maximum
	}
	return jitter(limit)
}

func publicationErrorClass(err error) string {
	switch {
	case errors.Is(err, ErrPermanentPublication):
		return "publisher_permanent"
	case errors.Is(err, context.DeadlineExceeded):
		return "publisher_timeout"
	case errors.Is(err, context.Canceled):
		return "publisher_canceled"
	default:
		return "publisher_temporary"
	}
}

func operationOutcome(err error) string {
	if err != nil {
		return "error"
	}
	return "success"
}

func operationErrorType(err error) string {
	if err != nil {
		return "database"
	}
	return "none"
}

func minDuration(first time.Duration, values ...time.Duration) time.Duration {
	minimum := first
	for _, value := range values {
		if value < minimum {
			minimum = value
		}
	}
	return minimum
}

func done(channel <-chan struct{}) bool {
	select {
	case <-channel:
		return true
	default:
		return false
	}
}

func nilInterface(value any) bool {
	reflected := reflect.ValueOf(value)
	//nolint:exhaustive // Only kinds that can contain a typed nil are relevant.
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
