package postgresoutbox

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

// This block owns the relay's fixed ceilings, and the reasoning about them that
// the two sites restating these values point back to.
//
// [ValidateRelayConfig] checks a RelayConfig against them at construction, and
// internal/config restates them so an operator's settings are rejected at load
// time instead of at startup. That copy is not a shortcut and cannot be removed
// by importing this package: depguard's config_no_runtime_owners rule forbids
// internal/config from importing any repository runtime package, so that
// loading configuration does not link the PostgreSQL adapter into every binary
// that merely reads it. The copy survives only because the composition root —
// cmd/outbox-relay/internal/bootstrap, the one package that wires both — pins
// it from outside; its tests say which value each of them holds.
//
// Those pins deliberately stop short of the lease relation, because the two
// sides check different ones: internal/config also charges the lease for the
// PostgreSQL acquire and statement budgets a RelayConfig cannot see, so it is
// strictly the stricter of the two.
const (
	// PublisherJoinTimeout is the grace a publisher goroutine gets to stop after
	// cancellation before the relay reports it stuck and declares cleanup unsafe.
	PublisherJoinTimeout = time.Second
	// MaxBatchSize bounds RelayConfig.BatchSize, and so the relay's peak memory
	// in stored envelopes.
	MaxBatchSize = 1000
	// MaxPublishConcurrency bounds RelayConfig.PublishConcurrency.
	MaxPublishConcurrency = 256
	// MaxAttempts bounds RelayConfig.MaxAttempts, the cycle attempt count at
	// which an adapter-proven rejection poisons.
	MaxAttempts = 100
	// MaxCleanupBatchSize bounds RelayConfig.CleanupBatchSize.
	MaxCleanupBatchSize = 10_000
)

var (
	// ErrPublisherStuck means a publisher goroutine ignored cancellation past
	// the join bound. Its durable work is unresolved and it can still reach the
	// process's dependencies, which is why it also makes cleanup unsafe.
	ErrPublisherStuck = errors.New("outbox publisher did not stop after cancellation")
	// ErrPublisherPanic means an adapter panicked. That event is released for
	// retry and the rest of its batch still finalizes, but the relay stops so a
	// broken adapter surfaces as a process failure instead of a silent backlog.
	ErrPublisherPanic = errors.New("outbox publisher panicked")
	// ErrProgressUnknown means the relay could neither record nor disprove a
	// publication: the event may already be at the broker while PostgreSQL still
	// shows it unpublished, so lease recovery will republish it. It stops the
	// relay, and two paths reach it. The ordinary one is a lost lease that
	// reconcilePublished could not resolve against durable state. The rare one is
	// an ordered acknowledgement whose key was still snapshot-conflicted on every
	// statement it was sent on: orderedPublishSnapshots of them inside
	// Store.MarkOrderedPublishedBatch, then that many again on each of
	// reconcilePasses passes, because Store.MarkOrderedPublished routes a single
	// ordered claim back through the same retry. An appended successor stayed invisible to all
	// of them. This product is the whole worst case, and this is the only place
	// that states it; both constants point here rather than restating it, and no
	// comment or test writes the number down.
	//
	// Those statements share one deadline, the lease the batch was claimed under
	// — see publishBatch. Nothing sizes the lease for that worst case, and
	// deliberately so: a finalization that runs out of lease ends here, which is
	// the outcome this error exists to report, rather than claiming a publication
	// it cannot prove.
	ErrProgressUnknown = errors.New("outbox published-state progress is unknown")
)

// RelayConfig is the relay's runtime budget. [ValidateRelayConfig] rejects the
// combinations that cannot hold: notably LeaseDuration must exceed
// PublishTimeout by more than the publisher join bound, because the batch's
// publication deadline is derived from the lease.
type RelayConfig struct {
	// PollInterval bounds how long an idle relay waits before claiming again.
	// An append notification normally arrives first; this is the fallback.
	PollInterval time.Duration
	// BatchSize is how many events one lease covers, and so the relay's peak
	// memory in stored envelopes.
	BatchSize int
	// PublishConcurrency is how many events of a batch may be in the publisher
	// at once. It never reorders an ordering key, because a batch holds at most
	// one claimable event per key — the partial unique index
	// outbox_events_ordering_ready_key is what enforces that, not this package.
	PublishConcurrency int
	// PublishTimeout budgets one whole batch's publication, not one event's.
	PublishTimeout time.Duration
	// LeaseDuration is how long a claim fences its events against another relay.
	// It also caps the publication budget, so it must exceed PublishTimeout plus
	// the publisher join bound.
	LeaseDuration time.Duration
	// MaxAttempts is the cycle attempt count at which an adapter-proven
	// rejection poisons. Ambiguous failures ignore it and keep retrying.
	MaxAttempts int
	// RetryBase and RetryMax bound full-jitter exponential backoff.
	RetryBase time.Duration
	RetryMax  time.Duration
	// ObservationInterval is how often relay state is sampled. Readiness treats
	// a sample as fresh for two of these intervals.
	ObservationInterval time.Duration
	// CleanupInterval is the normal retention cadence. A full delete batch
	// shortens the next one to PollInterval so a backlog drains.
	CleanupInterval time.Duration
	// PublishedRetention is how long a published row is kept before deletion.
	PublishedRetention time.Duration
	// CleanupBatchSize bounds one retention delete transaction.
	CleanupBatchSize int
}

// RelayResult is why Run stopped. CleanupUnsafe is the process's cleanup
// contract rather than a failure flag: true means a publisher goroutine
// outlived cancellation and can still reach the pool and the adapter, so the
// process must exit without closing them. Err is nil for an ordinary drain or
// cancellation.
//
// The field is stated negatively so the zero value is the safe one. Only the
// two paths that actually observed a stuck goroutine say so; every other stop
// — and any stop added later — leaves it alone and stays closeable.
type RelayResult struct {
	CleanupUnsafe bool
	Err           error
}

// relayStore is the relay's view of [Store], which is its only production
// implementation. The interface exists so the whole cycle — claim, finalize,
// reconcile, cleanup, observe — is unit-testable without PostgreSQL.
type relayStore interface {
	Claim(ctx context.Context, lease time.Duration, batchSize int) (ClaimedBatch, error)
	MarkUnorderedPublished(ctx context.Context, token, id string) error
	MarkOrderedPublished(ctx context.Context, token string, directive OrderedDirective) error
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
	// observedAtUnixNano is when the last state observation succeeded. Zero
	// means none has, which is not the same as a stale one: readiness requires
	// a sample, so zero never passes.
	observedAtUnixNano atomic.Int64
	inflight           atomic.Int64
	jitter             func(time.Duration) time.Duration
}

// NewRelay applies Store.valid rather than checking the one field it goes on to
// use: that predicate is what every exported Store method opens with, and a
// field added to Store later must reach this entry point too.
// The nil arm is what Store.valid already answers; it is spelled out because the
// listener below dereferences the store, and that is the one flow a nil checker
// cannot derive from the predicate.
func NewRelay(store *Store, publisher Publisher, telemetry *Telemetry, config RelayConfig) (*Relay, error) {
	if store == nil || !store.valid() {
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
	if err := ValidatePublisher(publisher); err != nil {
		return nil, err
	}
	if err := ValidateRelayConfig(config); err != nil {
		return nil, err
	}
	return &Relay{
		store:     store,
		publisher: publisher,
		telemetry: telemetry,
		config:    config,
		drain:     make(chan struct{}),
		jitter:    fullJitter,
	}, nil
}

func (r *Relay) Ready() bool {
	if r == nil {
		return false
	}
	return readyWithFreshObservation(
		r.running(),
		r.lastObservation(),
		observationStaleAfter(r.config.ObservationInterval),
	)
}

// running reports the lifecycle half of readiness: the loop started and drain
// has not begun. The freshness half is observation state, which
// readyWithFreshObservation owns.
func (r *Relay) running() bool {
	return r.ready.Load() && !done(r.drain)
}

// lastObservation is when the last state observation succeeded, or the zero
// time when none has. Those are different conditions — readiness requires a
// sample, so "never observed" must not read as an old one.
func (r *Relay) lastObservation() time.Time {
	nanos := r.observedAtUnixNano.Load()
	if nanos == 0 {
		return time.Time{}
	}
	return time.Unix(0, nanos)
}

func (r *Relay) StartDrain() {
	if r == nil {
		return
	}
	r.drainOnce.Do(func() {
		close(r.drain)
		r.telemetry.CountOperation(context.Background(), "drain", "started", classNone)
	})
	r.ready.Store(false)
	r.observeProcessState()
}

// Run owns one claimed batch at a time. StartDrain stops new claims but lets the
// current batch finish; canceling ctx forces its publications to stop.
func (r *Relay) Run(ctx context.Context) (result RelayResult) {
	if r == nil {
		result.Err = fmt.Errorf("%w: relay is required", ErrConfig)
		return result
	}
	defer func() {
		r.ready.Store(false)
		r.observeProcessState()
	}()

	// This first observation is also the schema gate: one statement reads
	// outbox_events, counts outbox_ordering_heads, and sizes outbox_redrives,
	// so a missing relation fails startup here rather than at the first claim
	// or the first operator redrive. Dropping a column from ObserveOutbox drops
	// that relation's share of the gate with it — see
	// TestPostgresOutboxStartupRequiresRedriveLedger.
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
	var result RelayResult
	due := newSchedule(r.config)
	for {
		if r.stopped(ctx) {
			return result
		}
		updated, err := r.maintain(ctx, due)
		if err != nil {
			result.Err = err
			return result
		}
		due = updated
		if r.stopped(ctx) {
			return result
		}
		if cycleResult, stop := r.runCycle(ctx, wake, due); stop {
			return cycleResult
		}
	}
}

// schedule is when the relay's two periodic duties come due. Each is measured
// from the moment its work finished rather than from when it was due, so a slow
// observation or cleanup cannot queue up behind itself.
type schedule struct {
	observation time.Time
	cleanup     time.Time
}

func newSchedule(config RelayConfig) schedule {
	now := time.Now()
	return schedule{
		observation: now.Add(config.ObservationInterval),
		cleanup:     now.Add(config.CleanupInterval),
	}
}

// maintain runs whichever duties are due and returns the schedule they earned.
// A failed observation is not fatal — readiness already goes stale on its own —
// but a failed cleanup is, because retention is how the table stays bounded.
//
// It reads the clock three times on purpose: once to decide what is due, and
// once after each duty finishes, because the next due time is measured from the
// end of the work rather than from when it came due.
func (r *Relay) maintain(ctx context.Context, due schedule) (schedule, error) {
	now := time.Now()
	if !now.Before(due.observation) {
		_ = r.observe(ctx)
		due.observation = time.Now().Add(r.config.ObservationInterval)
	}
	if !now.Before(due.cleanup) {
		fullBatch, err := r.cleanup(ctx)
		if err != nil {
			return due, err
		}
		delay := r.config.CleanupInterval
		if fullBatch {
			// A full batch means more is waiting; come back at poll speed.
			delay = min(delay, r.config.PollInterval)
		}
		due.cleanup = time.Now().Add(delay)
	}
	return due, nil
}

// runCycle claims once and publishes what it claimed. stop reports whether the
// loop must end, whether or not result carries an error: an ordinary drain and a
// cancellation both stop with a nil Err. Every stop signal in this file reads the
// same way — see Relay.stopped and Relay.wait.
func (r *Relay) runCycle(
	ctx context.Context,
	wake <-chan struct{},
	due schedule,
) (result RelayResult, stop bool) {
	// The lease is measured from before the claim on this process's own clock.
	// PostgreSQL starts it later and by its own clock, so this deadline is the
	// conservative one under any skew between the two.
	claimedAt := time.Now()
	batch, err := r.store.Claim(ctx, r.config.LeaseDuration, r.config.BatchSize)
	if err != nil {
		return RelayResult{Err: fmt.Errorf("claim outbox events: %w", err)}, true
	}
	if len(batch.Events) == 0 {
		idle := min(r.config.PollInterval, time.Until(due.observation), time.Until(due.cleanup))
		return RelayResult{}, r.wait(ctx, wake, idle)
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

// publishBatch publishes one claimed batch and finalizes every event in it.
// stop is true for a stuck publisher, a finalization failure, an observed
// cancellation, and a publisher panic. Only the cancellation stop carries a nil
// Err, because shutting down is not a failure.
func (r *Relay) publishBatch(
	ctx context.Context,
	batch ClaimedBatch,
	leaseExpiry time.Time,
) (result RelayResult, stop bool) {
	// The whole claimed batch, not the events inside Publish right now — that is
	// bounded by PublishConcurrency and this gauge is deliberately not. What an
	// operator needs from outbox.relay.inflight is how much durable work one
	// lease is holding, which is what a crash would redeliver.
	r.inflight.Store(int64(len(batch.Events)))
	r.observeProcessState()
	defer func() {
		r.inflight.Store(0)
		r.observeProcessState()
	}()

	failures, cleanupSafe := r.publishAll(ctx, batch, leaseExpiry)
	if !cleanupSafe {
		// A batch-scoped verdict, not an event's latency: publishOne already
		// timed whatever each event managed before the join bound expired.
		r.telemetry.CountOperation(ctx, "publish", outcomeError, classStuck)
		r.telemetry.LogPublisherStuck(ctx)
		return RelayResult{CleanupUnsafe: true, Err: ErrPublisherStuck}, true
	}

	// Finalization must reach PostgreSQL even while the process is shutting
	// down: an acknowledged event left unmarked becomes a duplicate after lease
	// recovery, and a failed event left leased blocks its ordering key until the
	// lease expires. The lease this batch already holds bounds the write.
	finalizeCtx, cancel := context.WithDeadline(context.WithoutCancel(ctx), leaseExpiry)
	defer cancel()
	if err := r.finalize(finalizeCtx, batch, failures); err != nil {
		return RelayResult{Err: err}, true
	}
	// Order matters between these two, and shutdown wins. A batch that was being
	// cancelled reports an ordinary drain even when an adapter also panicked in
	// it: the process is already stopping for a reason the operator asked for,
	// and naming a second one would only obscure that. The panic still reached
	// the publish metric, and the event was still released for retry. A stop
	// reason added below this point inherits the same precedence, so put it
	// above if it must outrank a shutdown in progress.
	if ctx.Err() != nil {
		return RelayResult{}, true
	}
	if err := publisherPanic(failures); err != nil {
		return RelayResult{Err: err}, true
	}
	return RelayResult{}, false
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
//
// cleanupSafe is stated positively even though [RelayResult.CleanupUnsafe] is
// not: this reports what it observed, and the result field is stated so its
// zero value is the safe one. Both results are named so the polarity is
// readable at the signature rather than a line into the caller.
func (r *Relay) publishAll(
	ctx context.Context,
	batch ClaimedBatch,
	leaseExpiry time.Time,
) (failures []error, cleanupSafe bool) {
	deadline := earliest(time.Now().Add(r.config.PublishTimeout), leaseExpiry.Add(-PublisherJoinTimeout))
	batchCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	// The workers write to a local rather than to the named result. A stuck
	// batch returns nil while they are still running, and assigning that to the
	// named result would hand a live goroutine a nil slice to index into.
	outcomes := make([]error, len(batch.Events))
	var next atomic.Int64
	var workers sync.WaitGroup
	for range min(r.config.PublishConcurrency, len(batch.Events)) {
		workers.Go(func() {
			for {
				index := int(next.Add(1)) - 1
				if index >= len(batch.Events) {
					return
				}
				outcomes[index] = r.publishOne(batchCtx, batch.Events[index].Event)
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
		return outcomes, true
	case <-batchCtx.Done():
	}
	join := time.NewTimer(PublisherJoinTimeout)
	defer join.Stop()
	select {
	case <-finished:
		return outcomes, true
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
		errorClass := classNone
		if err != nil {
			errorClass = publicationErrorClass(err)
		}
		r.telemetry.RecordOperation(ctx, "publish", operationOutcome(err), errorClass, time.Since(started))
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

// finalize records the durable transition every event of one batch earned. Each
// of the four statements can come back short — reporting fewer events than it
// was given — and the two halves answer that differently on purpose:
//
//   - An acknowledgement the statement did not report may already be at the
//     broker. Leaving it unmarked creates a duplicate, so finalizeUnordered and
//     finalizeOrdered resolve each missing event against durable state instead
//     of assuming either outcome. That resolution is not free: it costs up to
//     reconcilePasses statements per leftover event, and for an ordered one
//     each of those passes is itself worth orderedPublishSnapshots statements
//     inside the store. [ErrProgressUnknown] owns what that product means for
//     the lease.
//   - A retry or poison the statement did not report means this lease no longer
//     owns that event, so another relay does and will deliver it. There is
//     nothing to reconcile and nothing at risk, but the lease was overrun, which
//     is a lease or replica misconfiguration rather than a transient fault:
//     Store reports ErrLeaseLost and the relay stops so an operator sees it.
//
// Every error returned here stops the relay. failureClass in
// cmd/outbox-relay/main.go turns each one into its operator-facing exit class,
// so a new stop reason belongs in that switch too.
func (r *Relay) finalize(ctx context.Context, batch ClaimedBatch, failures []error) error {
	outcomes := r.classify(batch.Events, failures)
	if err := r.finalizeUnordered(ctx, batch.Token, outcomes.published); err != nil {
		return err
	}
	if err := r.finalizeOrdered(ctx, batch.Token, outcomes.ordered); err != nil {
		return err
	}
	if len(outcomes.retries) > 0 {
		if err := r.store.ScheduleRetryBatch(ctx, batch.Token, outcomes.retries); err != nil {
			// Named per transition: both this and the poison write below stop the
			// relay under the same exit class, so the message is all an operator
			// has to tell which statement failed.
			return fmt.Errorf("schedule outbox retry: %w", err)
		}
	}
	return r.finalizePoisoned(ctx, batch.Token, outcomes.poisoned)
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
			// The adapter's own class, so a poisoned row and the publish metric
			// that recorded the same failure agree without a second literal.
			outcomes.poisoned = append(outcomes.poisoned, poisonedEvent{claim: claim, errorClass: publicationErrorClass(failure)})
		case errors.Is(failure, ErrPublicationNotAccepted) && claim.CycleAttemptCount >= r.config.MaxAttempts:
			// Exhaustion is this package's policy rather than the adapter's
			// verdict, so it is the one class publicationErrorClass cannot name.
			outcomes.poisoned = append(outcomes.poisoned, poisonedEvent{claim: claim, errorClass: classAttemptExhausted})
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

// finalizeUnordered finalizes every unordered acknowledgement of one lease in a
// single statement.
func (r *Relay) finalizeUnordered(ctx context.Context, token string, claims []ClaimedEvent) error {
	if len(claims) == 0 {
		return nil
	}
	ids := make([]string, len(claims))
	for index, claim := range claims {
		ids[index] = claim.Event.ID
	}
	marked, err := r.store.MarkPublishedBatch(ctx, token, ids)
	return r.reconcileRemainder(ctx, token, claims, marked, err, r.markOneUnordered)
}

// finalizeOrdered finalizes every ordered acknowledgement of one lease in a
// single statement, which also advances each event's key head and unblocks that
// key's successor.
func (r *Relay) finalizeOrdered(ctx context.Context, token string, claims []ClaimedEvent) error {
	if len(claims) == 0 {
		return nil
	}
	directives := make([]OrderedDirective, len(claims))
	for index, claim := range claims {
		directives[index] = orderedDirective(claim)
	}
	marked, err := r.store.MarkOrderedPublishedBatch(ctx, token, directives)
	return r.reconcileRemainder(ctx, token, claims, marked, err, r.markOneOrdered)
}

// The two single-event marks reconciliation falls back to. Each caller above
// supplies the one matching the batch it just sent, so the ordered/unordered
// split stays where classify made it: nothing below re-derives it from the
// event, and a batch routed on some other rule cannot reach the wrong
// statement.
//
// Both pass the store's error through unchanged: reconcilePublished decides
// what it means, either discarding it once durable state proves the event
// published or joining it into [ErrProgressUnknown]. A wrap here would only
// prefix a message that ends up inside that join.
func (r *Relay) markOneUnordered(ctx context.Context, token string, claim ClaimedEvent) error {
	//nolint:wrapcheck // reconcilePublished classifies this error; see above.
	return r.store.MarkUnorderedPublished(ctx, token, claim.Event.ID)
}

func (r *Relay) markOneOrdered(ctx context.Context, token string, claim ClaimedEvent) error {
	//nolint:wrapcheck // As above.
	return r.store.MarkOrderedPublished(ctx, token, orderedDirective(claim))
}

func orderedDirective(claim ClaimedEvent) OrderedDirective {
	return OrderedDirective{
		ID:               claim.Event.ID,
		OrderingKey:      claim.Event.OrderingKey,
		OrderingSequence: claim.Event.OrderingSequence,
	}
}

// reconcileRemainder closes out a batch finalization. A short or failed write
// means a lost lease, or a crash window between broker acknowledgement and this
// update, so every event the statement did not report as finalized is resolved
// against durable state rather than assumed either way. markOne is the
// single-event statement matching the batch that came back short.
func (r *Relay) reconcileRemainder(
	ctx context.Context,
	token string,
	claims []ClaimedEvent,
	marked []string,
	markErr error,
	markOne markOneFunc,
) error {
	if len(marked) > 0 {
		r.telemetry.RecordProgress(time.Now())
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
		if err := r.reconcilePublished(ctx, token, claim, markOne); err != nil {
			return err
		}
	}
	return nil
}

func (r *Relay) finalizePoisoned(ctx context.Context, token string, poisoned []poisonedEvent) error {
	if len(poisoned) == 0 {
		return nil
	}
	directives := make([]PoisonDirective, len(poisoned))
	for index, event := range poisoned {
		directives[index] = PoisonDirective{ID: event.claim.Event.ID, ErrorClass: event.errorClass}
	}
	if err := r.store.MarkPoisonedBatch(ctx, token, directives); err != nil {
		return fmt.Errorf("poison outbox events: %w", err)
	}
	for _, event := range poisoned {
		r.telemetry.LogPoison(ctx, event.claim.Event.ID, event.errorClass, event.claim.CycleAttemptCount)
	}
	return nil
}

// reconcilePasses is how many times reconcilePublished re-resolves one event
// against durable state. A second pass is worth exactly one case: the mark
// failed, the event is still unpublished, and this lease still owns it, which is
// a lost race rather than a lost lease. Every other combination returns inside
// the first pass, so a third would only repeat a settled verdict.
//
// [ErrProgressUnknown] derives its worst case from this count and
// orderedPublishSnapshots, and owns the derivation; state it there, not here.
const reconcilePasses = 2

// markOneFunc is one lease's single-event finalization for one disposition.
// reconcileRemainder takes it rather than picking a statement itself, so the
// ordered/unordered split is written once, in Relay.classify.
type markOneFunc func(ctx context.Context, token string, claim ClaimedEvent) error

// reconcilePublished resolves one event whose batch finalization did not report
// it.
func (r *Relay) reconcilePublished(
	ctx context.Context,
	token string,
	claim ClaimedEvent,
	markOne markOneFunc,
) error {
	for range reconcilePasses {
		err := markOne(ctx, token, claim)
		if err == nil {
			r.telemetry.RecordProgress(time.Now())
			return nil
		}
		record, getErr := r.store.Get(ctx, claim.Event.ID)
		if getErr == nil && !record.PublishedAt.IsZero() {
			r.telemetry.CountOperation(ctx, "mark_published", "reconciled", classNone)
			r.telemetry.RecordProgress(time.Now())
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
	r.observedAtUnixNano.Store(observedAt.UnixNano())
	r.telemetry.RecordObservation(observation, observedAt)
	return nil
}

func (r *Relay) cleanup(ctx context.Context) (bool, error) {
	deleted, err := r.store.CleanupPublished(ctx, r.config.PublishedRetention, r.config.CleanupBatchSize)
	if err != nil {
		return false, fmt.Errorf("cleanup outbox: %w", err)
	}
	return deleted == r.config.CleanupBatchSize, nil
}

// wait blocks until there is reason to claim again, and reports whether the loop
// must stop instead. An append notification and the poll timer both mean carry
// on; cancellation and drain both mean stop.
func (r *Relay) wait(ctx context.Context, wake <-chan struct{}, duration time.Duration) (stop bool) {
	if duration < 0 {
		duration = 0
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return true
	case <-r.drain:
		return true
	case <-wake:
		return false
	case <-timer.C:
		return false
	}
}

// observeProcessState republishes the readiness inputs the metric callback
// reads. Only the relay knows when they change, so every lifecycle edge calls
// it. It publishes the inputs rather than the verdict: the callback runs on
// scrape and re-derives readiness itself, so an observation that goes stale
// between two lifecycle edges still turns the gauge off.
func (r *Relay) observeProcessState() {
	r.telemetry.SetProcessState(r.running(), r.inflight.Load(), observationStaleAfter(r.config.ObservationInterval))
}

// ValidateRelayConfig rejects a configuration the relay cannot run under. Every
// message names the field and the value it was given, because the operator who
// reads it is holding a config file rather than this source.
//
// [NewRelay] applies it, and it is exported for the same reason
// [ValidatePublisher] is: the composition root is the only place that can hold
// this package's rules beside internal/config's copies of them, so it is the
// only place that can prove the two still describe the same budget. See the
// ceiling block above for which tests do that.
func ValidateRelayConfig(config RelayConfig) error {
	for _, field := range []struct {
		name  string
		value time.Duration
	}{
		{name: "poll_interval", value: config.PollInterval},
		{name: "publish_timeout", value: config.PublishTimeout},
		{name: "lease_duration", value: config.LeaseDuration},
		{name: "retry_base", value: config.RetryBase},
		{name: "observation_interval", value: config.ObservationInterval},
		{name: "cleanup_interval", value: config.CleanupInterval},
		{name: "published_retention", value: config.PublishedRetention},
	} {
		if field.value <= 0 {
			return fmt.Errorf("%w: %s must be positive, got %s", ErrConfig, field.name, field.value)
		}
	}
	for _, field := range []struct {
		name      string
		value     int
		low, high int
	}{
		{name: "batch_size", value: config.BatchSize, low: 1, high: MaxBatchSize},
		{name: "publish_concurrency", value: config.PublishConcurrency, low: 1, high: MaxPublishConcurrency},
		{name: "max_attempts", value: config.MaxAttempts, low: 1, high: MaxAttempts},
		{name: "cleanup_batch_size", value: config.CleanupBatchSize, low: 1, high: MaxCleanupBatchSize},
	} {
		if field.value < field.low || field.value > field.high {
			return fmt.Errorf(
				"%w: %s must be in range [%d,%d], got %d",
				ErrConfig, field.name, field.low, field.high, field.value,
			)
		}
	}
	if config.RetryMax < config.RetryBase {
		return fmt.Errorf(
			"%w: retry_max (%s) must be at least retry_base (%s)",
			ErrConfig, config.RetryMax, config.RetryBase,
		)
	}
	// The batch's publication deadline is derived from the lease, and a stuck
	// publisher is given PublisherJoinTimeout to stop before the relay gives up
	// on it, so the lease has to outlast both.
	if config.LeaseDuration <= config.PublishTimeout ||
		config.LeaseDuration-config.PublishTimeout <= PublisherJoinTimeout {
		return fmt.Errorf(
			"%w: lease_duration (%s) must exceed publish_timeout (%s) by more than the %s publisher-join timeout",
			ErrConfig, config.LeaseDuration, config.PublishTimeout, PublisherJoinTimeout,
		)
	}
	return nil
}

// fullJitter draws uniformly from [0,limit]. Spreading the whole interval, not
// a band around it, is what keeps a batch of events that failed together from
// retrying together.
func fullJitter(limit time.Duration) time.Duration {
	if limit == math.MaxInt64 {
		// limit+1 would overflow, and Int64 already covers the whole range.
		// #nosec G404 -- Backoff jitter coordinates retries; it is not a secret or token.
		return time.Duration(rand.Int64())
	}
	// #nosec G404 -- Backoff jitter coordinates retries; it is not a secret or token.
	return time.Duration(rand.Int64N(limit.Nanoseconds() + 1))
}

// retryDelay is full-jitter exponential backoff: the delay is drawn from
// [0, base*2^(attempt-1)] with that limit clamped to maximum. attempt is the
// claim's 1-based cycle attempt count, so the first retry waits within base.
func retryDelay(base, maximum time.Duration, attempt int, jitter func(time.Duration) time.Duration) time.Duration {
	limit := base
	for range max(attempt-1, 0) {
		// Doubling only below half the ceiling keeps the limit from overflowing.
		if limit > maximum/2 {
			return jitter(maximum)
		}
		limit *= 2
	}
	return jitter(min(limit, maximum))
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
		return classPanic
	case errors.Is(err, ErrPermanentPublication):
		return classPublisherPermanent
	case errors.Is(err, ErrPublicationNotAccepted):
		return classPublisherRejected
	case errors.Is(err, context.DeadlineExceeded):
		return classPublisherTimeout
	case errors.Is(err, context.Canceled):
		return classPublisherCanceled
	default:
		return classPublisherTemporary
	}
}

func operationOutcome(err error) string {
	if err != nil {
		return outcomeError
	}
	return outcomeSuccess
}

func earliest(first, second time.Time) time.Time {
	if second.Before(first) {
		return second
	}
	return first
}

// readyWithFreshObservation is the whole readiness policy, and the only copy of
// it. Relay.Ready answers the diagnostics probe from live relay state and the
// metric callback answers a scrape from the published snapshot, so both ask
// this rather than each spelling the rule out. A readiness input added here
// therefore reaches the probe and the gauge together; one added to only one
// caller would silently disagree between them.
func readyWithFreshObservation(running bool, observedAt time.Time, staleAfter time.Duration) bool {
	return running && !observedAt.IsZero() && time.Since(observedAt) <= staleAfter
}

// observationStaleAfter is how long a state observation still counts as fresh
// for readiness: two intervals, so one missed sample is tolerated and two are
// not. The guard keeps the doubling from overflowing.
func observationStaleAfter(interval time.Duration) time.Duration {
	if interval > math.MaxInt64/2 {
		return math.MaxInt64
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
