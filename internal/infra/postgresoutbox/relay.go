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

const (
	publisherJoinTimeout  = time.Second
	maxBatchSize          = 1000
	maxPublishConcurrency = 256
)

var (
	ErrPublisherStuck  = errors.New("outbox publisher did not stop after cancellation")
	ErrPublisherPanic  = errors.New("outbox publisher panicked")
	ErrProgressUnknown = errors.New("outbox published-state progress is unknown")
)

type RelayConfig struct {
	PollInterval        time.Duration
	BatchSize           int
	PublishConcurrency  int
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
	Claim(ctx context.Context, lease time.Duration, batchSize int) (ClaimedBatch, error)
	MarkPublished(ctx context.Context, token string, claim ClaimedEvent) error
	MarkPublishedBatch(ctx context.Context, token string, ids []string) ([]string, error)
	MarkOrderedPublishedBatch(ctx context.Context, token string, directives []OrderedDirective) ([]string, error)
	ScheduleRetryBatch(ctx context.Context, token string, retries []RetryDirective) error
	MarkPoisonedBatch(ctx context.Context, token string, poisons []PoisonDirective) error
	Get(ctx context.Context, id string) (Record, error)
	CleanupPublished(ctx context.Context, retention time.Duration, batch int) (int, error)
	Observe(ctx context.Context) (StateObservation, error)
}

// listener signals wake whenever PostgreSQL announces a committed append. It
// returns only when ctx is done.
type listener func(ctx context.Context, wake chan<- struct{})

type Relay struct {
	store     relayStore
	publisher Publisher
	telemetry *Telemetry
	config    RelayConfig
	listen    listener
	drain     chan struct{}
	drainOnce sync.Once
	ready     atomic.Bool
	observed  atomic.Int64
	inflight  atomic.Int64
	jitter    func(time.Duration) time.Duration
	markEvent func(context.Context, string, ClaimedEvent) error
	getEvent  func(context.Context, string) (Record, error)
}

func NewRelay(store *Store, publisher Publisher, telemetry *Telemetry, config RelayConfig) (*Relay, error) {
	if store == nil || store.pool == nil {
		return nil, fmt.Errorf("%w: outbox store is required", ErrConfig)
	}
	owned := store.withTelemetry(telemetry)
	relay, err := newRelay(owned, publisher, telemetry, config)
	if err != nil {
		return nil, err
	}
	relay.listen = listenForAppends(owned.listenerConfig(), telemetry)
	return relay, nil
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
	return observedAt != 0 && time.Since(time.Unix(0, observedAt)) <= observationStaleAfter(r.config.ObservationInterval)
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

// Run owns one claimed batch at a time. StartDrain stops new claims but lets the
// current batch finish; canceling ctx forces its publications to stop.
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
	wake := make(chan struct{}, 1)
	stopListener := r.startListener(ctx, wake)
	defer stopListener()

	r.ready.Store(true)
	r.observeProcessState()
	return r.runLoop(ctx, wake)
}

// startListener runs the optional append listener and returns the function that
// stops and joins it.
func (r *Relay) startListener(ctx context.Context, wake chan<- struct{}) func() {
	if r.listen == nil {
		return func() {}
	}
	listenCtx, stop := context.WithCancel(ctx)
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		r.listen(listenCtx, wake)
	}()
	return func() {
		stop()
		<-stopped
	}
}

func (r *Relay) runLoop(ctx context.Context, wake <-chan struct{}) RelayResult {
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
		if cycleResult, keepRunning := r.runCycle(ctx, wake, nextObservation, nextCleanup); !keepRunning {
			return cycleResult
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
		_ = r.observe(ctx)
		*nextObservation = time.Now().Add(r.config.ObservationInterval)
	}
	if !now.Before(*nextCleanup) {
		fullBatch, err := r.cleanup(ctx)
		if err != nil {
			return err
		}
		delay := r.config.CleanupInterval
		if fullBatch {
			delay = minDuration(delay, r.config.PollInterval)
		}
		*nextCleanup = time.Now().Add(delay)
	}
	return nil
}

func (r *Relay) runCycle(
	ctx context.Context,
	wake <-chan struct{},
	nextObservation time.Time,
	nextCleanup time.Time,
) (RelayResult, bool) {
	// The lease is measured from before the claim on this process's own clock.
	// PostgreSQL starts it later and by its own clock, so this deadline is the
	// conservative one under any skew between the two.
	claimedAt := time.Now()
	batch, err := r.store.Claim(ctx, r.config.LeaseDuration, r.config.BatchSize)
	if err != nil {
		return RelayResult{CleanupSafe: true, Err: fmt.Errorf("claim outbox events: %w", err)}, false
	}
	if len(batch.Events) == 0 {
		idle := minDuration(r.config.PollInterval, time.Until(nextObservation), time.Until(nextCleanup))
		return RelayResult{CleanupSafe: true}, r.wait(ctx, wake, idle)
	}
	return r.publishBatch(ctx, batch, claimedAt.Add(r.config.LeaseDuration))
}

func (r *Relay) stopped(ctx context.Context) bool {
	return done(r.drain) || ctx.Err() != nil
}

// poisonedEvent pairs the parked claim with the class that parked it, so the
// operator log keeps the attempt count the store type has no use for.
type poisonedEvent struct {
	claim      ClaimedEvent
	errorClass string
}

// batchOutcomes groups one batch by the durable transition each event needs.
type batchOutcomes struct {
	published []ClaimedEvent
	ordered   []ClaimedEvent
	retries   []RetryDirective
	poisoned  []poisonedEvent
}

func (r *Relay) publishBatch(ctx context.Context, batch ClaimedBatch, leaseExpiry time.Time) (RelayResult, bool) {
	r.inflight.Store(int64(len(batch.Events)))
	r.observeProcessState()
	defer func() {
		r.inflight.Store(0)
		r.observeProcessState()
	}()

	started := time.Now()
	failures, cleanupSafe := r.publishAll(ctx, batch, leaseExpiry)
	if !cleanupSafe {
		r.recordOperation(ctx, "publish", outcomeError, "stuck", started)
		if r.telemetry != nil {
			r.telemetry.LogPublisherStuck(ctx)
		}
		return RelayResult{CleanupSafe: false, Err: ErrPublisherStuck}, false
	}

	// Finalization must reach PostgreSQL even while the process is shutting
	// down: an acknowledged event left unmarked becomes a duplicate after lease
	// recovery, and a failed event left leased blocks its ordering key until the
	// lease expires. The lease this batch already holds bounds the write.
	finalizeCtx, cancel := context.WithDeadline(context.WithoutCancel(ctx), leaseExpiry)
	defer cancel()
	if err := r.finalize(finalizeCtx, batch, failures); err != nil {
		return RelayResult{CleanupSafe: true, Err: err}, false
	}
	if ctx.Err() != nil {
		return RelayResult{CleanupSafe: true}, false
	}
	if err := publisherPanic(failures); err != nil {
		return RelayResult{CleanupSafe: true, Err: err}, false
	}
	return RelayResult{CleanupSafe: true}, true
}

// publishAll publishes the batch through at most PublishConcurrency workers and
// never past the lease it was claimed under. The returned slice holds each
// event's publication error by its index in the batch; a nil entry means the
// broker durably acknowledged that exact event. It reports cleanupSafe = false
// when a publisher ignored cancellation, because its goroutine can still touch
// the dependencies the process is about to close.
//
// The deadline computed here is the whole batch's publication budget. Because
// it starts no later than any single publication does, it is also the budget of
// each one, which is why publishOne needs no timeout of its own.
func (r *Relay) publishAll(ctx context.Context, batch ClaimedBatch, leaseExpiry time.Time) ([]error, bool) {
	deadline := earliest(time.Now().Add(r.config.PublishTimeout), leaseExpiry.Add(-publisherJoinTimeout))
	batchCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	failures := make([]error, len(batch.Events))
	var next atomic.Int64
	var workers sync.WaitGroup
	for range min(r.config.PublishConcurrency, len(batch.Events)) {
		workers.Go(func() {
			for {
				index := int(next.Add(1)) - 1
				if index >= len(batch.Events) {
					return
				}
				failures[index] = r.publishOne(batchCtx, batch.Events[index].Event)
			}
		})
	}
	finished := make(chan struct{})
	go func() {
		workers.Wait()
		close(finished)
	}()

	select {
	case <-finished:
		return failures, true
	case <-batchCtx.Done():
	}
	join := time.NewTimer(publisherJoinTimeout)
	defer join.Stop()
	select {
	case <-finished:
		return failures, true
	case <-join.C:
		return nil, false
	}
}

func (r *Relay) publishOne(ctx context.Context, event Event) (err error) {
	started := time.Now()
	defer func() {
		if recover() != nil {
			err = ErrPublisherPanic
		}
		errorClass := "none"
		if err != nil {
			errorClass = publicationErrorClass(err)
		}
		r.recordOperation(ctx, "publish", operationOutcome(err), errorClass, started)
	}()

	err = r.publisher.Publish(ctx, event)
	if err == nil {
		// A publisher that returns nil after its budget expired cannot prove the
		// broker accepted the event, so the attempt stays retryable.
		if budget := ctx.Err(); budget != nil {
			err = fmt.Errorf("publisher reported success after its budget expired: %w", budget)
		}
	}
	return err
}

func (r *Relay) finalize(ctx context.Context, batch ClaimedBatch, failures []error) error {
	outcomes := r.classify(batch.Events, failures)
	if err := r.markPublished(ctx, batch.Token, outcomes.published); err != nil {
		return err
	}
	if err := r.markOrderedPublished(ctx, batch.Token, outcomes.ordered); err != nil {
		return err
	}
	if len(outcomes.retries) > 0 {
		if err := r.store.ScheduleRetryBatch(ctx, batch.Token, outcomes.retries); err != nil {
			return fmt.Errorf("record outbox publication failure: %w", err)
		}
	}
	return r.markPoisoned(ctx, batch.Token, outcomes.poisoned)
}

// classify groups one batch by the durable transition each event needs.
// failures is indexed by claims, and a nil entry is a durable acknowledgement.
func (r *Relay) classify(claims []ClaimedEvent, failures []error) batchOutcomes {
	var outcomes batchOutcomes
	for index, claim := range claims {
		failure := failures[index]
		switch {
		case failure == nil && claim.Event.OrderingKey == "":
			outcomes.published = append(outcomes.published, claim)
		case failure == nil:
			outcomes.ordered = append(outcomes.ordered, claim)
		case errors.Is(failure, ErrPermanentPublication):
			outcomes.poisoned = append(outcomes.poisoned, poisonedEvent{claim: claim, errorClass: "publisher_permanent"})
		case errors.Is(failure, ErrPublicationNotAccepted) && claim.CycleAttemptCount >= r.config.MaxAttempts:
			outcomes.poisoned = append(outcomes.poisoned, poisonedEvent{claim: claim, errorClass: "attempt_exhausted"})
		default:
			outcomes.retries = append(outcomes.retries, RetryDirective{
				ID:         claim.Event.ID,
				ErrorClass: publicationErrorClass(failure),
				Delay:      retryDelay(r.config.RetryBase, r.config.RetryMax, claim.CycleAttemptCount, r.jitter),
			})
		}
	}
	return outcomes
}

// markPublished finalizes every unordered acknowledgement of one lease in a
// single statement.
func (r *Relay) markPublished(ctx context.Context, token string, claims []ClaimedEvent) error {
	if len(claims) == 0 {
		return nil
	}
	ids := make([]string, len(claims))
	for index, claim := range claims {
		ids[index] = claim.Event.ID
	}
	marked, err := r.store.MarkPublishedBatch(ctx, token, ids)
	return r.reconcileRemainder(ctx, token, claims, marked, err)
}

// markOrderedPublished finalizes every ordered acknowledgement of one lease in
// a single statement, which also advances each event's key head and unblocks
// that key's successor.
func (r *Relay) markOrderedPublished(ctx context.Context, token string, claims []ClaimedEvent) error {
	if len(claims) == 0 {
		return nil
	}
	directives := make([]OrderedDirective, len(claims))
	for index, claim := range claims {
		directives[index] = OrderedDirective{
			ID:               claim.Event.ID,
			OrderingKey:      claim.Event.OrderingKey,
			OrderingSequence: claim.Event.OrderingSequence,
		}
	}
	marked, err := r.store.MarkOrderedPublishedBatch(ctx, token, directives)
	return r.reconcileRemainder(ctx, token, claims, marked, err)
}

// reconcileRemainder closes out a batch finalization. A short or failed write
// means a lost lease, or a crash window between broker acknowledgement and this
// update, so every event the statement did not report as finalized is resolved
// against durable state rather than assumed either way.
func (r *Relay) reconcileRemainder(
	ctx context.Context,
	token string,
	claims []ClaimedEvent,
	marked []string,
	markErr error,
) error {
	if len(marked) > 0 {
		r.recordProgress(time.Now())
	}
	if markErr == nil && len(marked) == len(claims) {
		return nil
	}
	finalized := make(map[string]struct{}, len(marked))
	for _, id := range marked {
		finalized[id] = struct{}{}
	}
	for _, claim := range claims {
		if _, ok := finalized[claim.Event.ID]; ok {
			continue
		}
		if err := r.reconcilePublished(ctx, token, claim); err != nil {
			return err
		}
	}
	return nil
}

func (r *Relay) markPoisoned(ctx context.Context, token string, poisoned []poisonedEvent) error {
	if len(poisoned) == 0 {
		return nil
	}
	directives := make([]PoisonDirective, len(poisoned))
	for index, event := range poisoned {
		directives[index] = PoisonDirective{ID: event.claim.Event.ID, ErrorClass: event.errorClass}
	}
	if err := r.store.MarkPoisonedBatch(ctx, token, directives); err != nil {
		return fmt.Errorf("record outbox publication failure: %w", err)
	}
	if r.telemetry != nil {
		for _, event := range poisoned {
			r.telemetry.LogPoison(ctx, event.claim.Event.ID, event.errorClass, event.claim.CycleAttemptCount)
		}
	}
	return nil
}

func (r *Relay) reconcilePublished(ctx context.Context, token string, claim ClaimedEvent) error {
	for range 2 {
		err := r.markEvent(ctx, token, claim)
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
		if getErr != nil || record.LeaseToken != token {
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
	observedAt := time.Now()
	r.observed.Store(observedAt.UnixNano())
	if r.telemetry != nil {
		r.telemetry.RecordObservation(observation, observedAt)
	}
	return nil
}

func (r *Relay) cleanup(ctx context.Context) (bool, error) {
	deleted, err := r.store.CleanupPublished(ctx, r.config.PublishedRetention, r.config.CleanupBatchSize)
	if err != nil {
		return false, fmt.Errorf("cleanup outbox: %w", err)
	}
	return deleted == r.config.CleanupBatchSize, nil
}

func (r *Relay) wait(ctx context.Context, wake <-chan struct{}, duration time.Duration) bool {
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
	case <-wake:
		return true
	case <-timer.C:
		return true
	}
}

func (r *Relay) observeProcessState() {
	if r.telemetry != nil {
		r.telemetry.SetProcessState(r.Ready(), r.inflight.Load(), observationStaleAfter(r.config.ObservationInterval))
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
	if config.BatchSize < 1 || config.BatchSize > maxBatchSize {
		return fmt.Errorf("%w: batch size must be in range [1,%d]", ErrConfig, maxBatchSize)
	}
	if config.PublishConcurrency < 1 || config.PublishConcurrency > maxPublishConcurrency {
		return fmt.Errorf("%w: publish concurrency must be in range [1,%d]", ErrConfig, maxPublishConcurrency)
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

func publisherPanic(failures []error) error {
	for _, failure := range failures {
		if errors.Is(failure, ErrPublisherPanic) {
			return failure
		}
	}
	return nil
}

func publicationErrorClass(err error) string {
	switch {
	case errors.Is(err, ErrPublisherPanic):
		return "panic"
	case errors.Is(err, ErrPermanentPublication):
		return "publisher_permanent"
	case errors.Is(err, ErrPublicationNotAccepted):
		return "publisher_rejected"
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
		return outcomeError
	}
	return outcomeSuccess
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

func earliest(first, second time.Time) time.Time {
	if second.Before(first) {
		return second
	}
	return first
}

func observationStaleAfter(interval time.Duration) time.Duration {
	const maximum = time.Duration(1<<63 - 1)
	if interval > maximum/2 {
		return maximum
	}
	return 2 * interval
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
