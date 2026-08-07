package postgresoutbox

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

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
	MarkUnorderedPublishedBatch(ctx context.Context, token string, ids []string) ([]string, error)
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

// Relay is one outbox relay process: it repeats claim, publish, finalize until
// drain or cancellation, and runs the two periodic duties beside that cycle.
// One Relay owns one claimed batch at a time, so a deployment scales by adding
// replicas rather than by widening a single relay — the lease token is what
// fences them apart.
//
// This file owns the cycle and the lifecycle state, and nothing else. Each of
// the parts around it has its own file: relay_publish.go publishes a claimed
// batch, relay_finalize.go turns each publication outcome into its durable
// transition, relay_maintain.go runs the two periodic duties beside the cycle,
// relay_ready.go owns the readiness policy the probe and the gauge share, and
// relay_config.go holds the budget every step reads.
//
// Only [NewRelay] builds a usable Relay. Every exported method tolerates a nil
// receiver, because the composition root can reach one on a startup path that
// failed earlier.
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
	if store == nil || holdsTypedNil(store) {
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

func (r *Relay) StartDrain() {
	if r == nil {
		return
	}
	r.drainOnce.Do(func() {
		close(r.drain)
		r.telemetry.CountOperation(context.Background(), "drain", "started", classNone)
	})
	r.ready.Store(false)
	r.reportProcessState()
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
		r.reportProcessState()
	}()

	// This first observation is also the schema gate: one statement reads
	// outbox_events, counts outbox_ordering_heads, and sizes outbox_redrives,
	// so a missing relation fails startup here rather than at the first claim
	// or the first operator redrive. Dropping a column from ObserveOutbox drops
	// that relation's share of the gate with it — see
	// TestPostgresOutboxStartupRequiresRedriveLedger.
	if err := r.sampleState(ctx); err != nil {
		result.Err = err
		return result
	}
	if isClosed(r.drain) || ctx.Err() != nil {
		return result
	}
	wake := make(chan struct{}, 1)
	stopListener := r.startListener(ctx, wake)
	defer stopListener()

	r.ready.Store(true)
	r.reportProcessState()
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
		if r.loopMustStop(ctx) {
			return result
		}
		updated, err := r.runDueMaintenance(ctx, due)
		if err != nil {
			result.Err = err
			return result
		}
		due = updated
		if r.loopMustStop(ctx) {
			return result
		}
		if cycleResult, stop := r.runCycle(ctx, wake, due); stop {
			return cycleResult
		}
	}
}

// runCycle claims once and publishes what it claimed. stop reports whether the
// loop must end, whether or not result carries an error: an ordinary drain and a
// cancellation both stop with a nil Err. Every stop signal in this file reads the
// same way — see Relay.loopMustStop and Relay.wait.
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

// loopMustStop reports that the cycle must end: drain began, or ctx was
// cancelled. It is deliberately not the negation of readyToServe — see there.
func (r *Relay) loopMustStop(ctx context.Context) bool {
	return isClosed(r.drain) || ctx.Err() != nil
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

func isClosed(channel <-chan struct{}) bool {
	select {
	case <-channel:
		return true
	default:
		return false
	}
}
