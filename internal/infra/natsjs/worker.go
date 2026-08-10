package natsjs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/nats-io/nats.go/jetstream"
)

// Worker consumes one durable pull consumer: it fetches up to as many messages
// as it has free handler slots, runs at most MaxConcurrency feature handlers
// over them, and settles each with the broker. [Client.NewWorker] is the only
// constructor, and a client owns at most one Worker.
//
// This file owns the run and drain lifecycle: worker_topology.go proves the
// broker topology before a Worker exists, and worker_delivery.go takes one
// message from admission through to settlement.
//
// Three contexts run at different lengths. fetchCtx stops acquisition the moment
// drain begins; handlerRoot is detached from it so handlers already running may
// finish, and only forceClose cancels it; ctx is the caller's, and observing it
// stops the loop with an error.
type Worker struct {
	client    *Client
	cfg       WorkerConfig
	consumer  pullConsumer
	dlqStream string
	handler   Handler

	draining atomic.Bool
	started  atomic.Bool
	fatal    chan error
	runDone  chan struct{}

	mu            sync.Mutex
	fetchCancel   context.CancelFunc
	handlerCancel context.CancelFunc
}

func (w *Worker) Run(ctx context.Context) error {
	if !w.started.CompareAndSwap(false, true) {
		return fmt.Errorf("%w: worker already started", ErrRejected)
	}
	defer close(w.runDone)
	fetchCtx, fetchCancel := context.WithCancel(ctx)
	handlerRoot, handlerCancel := context.WithCancel(context.WithoutCancel(ctx))
	w.mu.Lock()
	w.fetchCancel = fetchCancel
	w.handlerCancel = handlerCancel
	w.mu.Unlock()
	defer fetchCancel()

	completion := make(chan struct{}, w.cfg.MaxConcurrency)
	active, runErr := w.fetchLoop(ctx, fetchCtx, handlerRoot, completion)
	return w.waitForHandlers(completion, active, runErr)
}

// fetchLoop acquires messages until drain, cancellation, or a fault. It owns
// the concurrency ceiling and the count of handlers still running; pullOnce
// owns everything about a single pull.
//
// Each pull asks for the free slot count, not one message: a pull is one round
// trip whatever it returns, so asking for one would cap sustained throughput at
// 1/RTT however large MaxConcurrency is. The resident bound is unchanged,
// because free never exceeds what the ceiling already allows.
func (w *Worker) fetchLoop(ctx, fetchCtx, handlerRoot context.Context, completion chan struct{}) (int, error) {
	active := 0
	for !w.draining.Load() {
		if err := w.stopReason(ctx); err != nil {
			return active, err
		}
		if active >= w.cfg.MaxConcurrency {
			if err := w.waitForHandler(ctx, completion); err != nil {
				return active, err
			}
			// Reclaim every slot that is already free, not only the one just
			// waited for. Handlers of one batch tend to finish together, and
			// returning to the pull with a single slot would spend a whole
			// round trip on each of the others.
			active -= 1 + drainedHandlers(completion)
			continue
		}
		started, stop, err := w.pullOnce(fetchCtx, handlerRoot, completion, w.cfg.MaxConcurrency-active)
		active += started
		if stop || err != nil {
			return active, err
		}
	}
	return active, nil
}

// drainedHandlers reports how many further handlers had already finished,
// without blocking. Every send on completion is one handler that has returned,
// so consuming the queued ones is only reading a count that was already true.
func drainedHandlers(completion <-chan struct{}) int {
	count := 0
	for {
		select {
		case <-completion:
			count++
		default:
			return count
		}
	}
}

// pullOnce runs one pull for up to free messages and starts a handler for each
// one it delivered.
//
// The pull's context is released by a single defer rather than on each path out:
// the batch's messages and its error are only valid while that context lives, so
// the release has to outlast draining them.
//
// Asking for more than one message costs no extra latency for the first —
// jetstream fills the batch channel as each message arrives — but a partly filled
// batch waits for the pull's own expiry, which is why free slots are reclaimed
// before the next pull rather than during this one.
//
// stop reports that acquisition is over: a cancelled fetch context, or a fault
// this client cannot ride out, and only the fault carries an err. An expired pull
// that delivered nothing is ordinary and stops nothing.
func (w *Worker) pullOnce(
	fetchCtx, handlerRoot context.Context,
	completion chan<- struct{},
	free int,
) (started int, stop bool, err error) {
	pullCtx, cancel := context.WithTimeout(fetchCtx, operationTimeout)
	defer cancel()

	batch, fetchErr := w.consumer.Fetch(free, jetstream.FetchContext(pullCtx))
	if fetchErr != nil {
		if fetchCtx.Err() != nil {
			// Pull cancellation stops acquisition; active handler errors still
			// win during drain.
			return 0, true, nil
		}
		return 0, false, w.resumeAfterFetchFailure(fetchCtx, fetchErr)
	}
	for msg := range batch.Messages() {
		if !w.startHandler(handlerRoot, msg, completion) {
			break
		}
		started++
	}
	if batchErr := batch.Error(); batchErr != nil && !pullExhausted(batchErr) && fetchCtx.Err() == nil {
		return started, false, w.resumeAfterFetchFailure(fetchCtx, batchErr)
	}
	return started, false, nil
}

// resumeAfterFetchFailure decides what a failed pull means. It returns nil to
// pull again, having waited for the connection to come back, and a terminal
// error when the failure is not a reconnect this client can ride out.
func (w *Worker) resumeAfterFetchFailure(fetchCtx context.Context, err error) error {
	if w.client.waitForReconnect(fetchCtx, err) {
		return nil
	}
	runErr := fmt.Errorf("%w: fetch source message", ErrTerminal)
	w.fail(runErr)
	return runErr
}

// pullExhausted is the ordinary end of a pull that delivered nothing: no
// message arrived within the request's own expiry.
func pullExhausted(err error) bool {
	return errors.Is(err, jetstream.ErrNoMessages) || errors.Is(err, context.DeadlineExceeded)
}

// startHandler launches one handler goroutine and reports whether it did.
//
// The check and the launch share one critical section because StartDrain sets
// draining under the same lock. A handler started after drain began would not be
// counted in fetchLoop's active total, so Run would return without joining it.
//
// The fetch counters are recorded against handlerRoot rather than the caller's
// context: it is a context.WithoutCancel of that context, so it carries the same
// values and a counter increment observes no deadline.
func (w *Worker) startHandler(handlerRoot context.Context, msg jetstream.Msg, completion chan<- struct{}) bool {
	w.mu.Lock()
	if w.draining.Load() {
		w.mu.Unlock()
		return false
	}
	// Only past the drain check is msg known to be a real message: a batch that
	// was already in flight when drain began can still yield a nil one.
	messageBytes := wireSize(msg)
	go func() {
		defer func() { completion <- struct{}{} }()
		if err := w.handle(handlerRoot, msg); err != nil {
			w.fail(err)
		}
	}()
	w.mu.Unlock()
	w.client.telemetry.countFetch(handlerRoot, 1, int64(messageBytes))
	return true
}

// stopReason reports why the loop must stop, without blocking, and begins the
// drain when there is one. A nil result means carry on.
func (w *Worker) stopReason(ctx context.Context) error {
	select {
	case err := <-w.fatal:
		w.StartDrain()
		return err
	case <-ctx.Done():
		w.StartDrain()
		return fmt.Errorf("run durable consumer: %w", ctx.Err())
	default:
		return nil
	}
}

func (w *Worker) waitForHandler(ctx context.Context, completion <-chan struct{}) error {
	select {
	case <-completion:
		return nil
	case err := <-w.fatal:
		w.StartDrain()
		return err
	case <-ctx.Done():
		w.StartDrain()
		return fmt.Errorf("run durable consumer: %w", ctx.Err())
	}
}

func (w *Worker) waitForHandlers(completion <-chan struct{}, active int, runErr error) error {
	for active > 0 {
		select {
		case <-completion:
			active--
		case err := <-w.fatal:
			if runErr == nil {
				runErr = err
			}
		}
	}
	// fail publishes the terminal error before the handler publishes its
	// completion. The final completion can still win a select when both channels
	// are ready, so inspect the already-published terminal result once more before
	// reporting a graceful stop.
	if runErr == nil {
		select {
		case runErr = <-w.fatal:
		default:
		}
	}
	return runErr
}

func (w *Worker) StartDrain() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.draining.Load() {
		return
	}
	w.draining.Store(true)
	w.client.StopPublish()
	if w.fetchCancel != nil {
		w.fetchCancel()
	}
}

func (w *Worker) Shutdown(ctx context.Context) error {
	w.StartDrain()
	select {
	case <-w.runDone:
		if err := w.client.Shutdown(ctx); err != nil {
			w.client.telemetry.recordDrain(ctx, outcomeFailed)
			return err
		}
		w.client.telemetry.recordDrain(ctx, outcomeGraceful)
		return nil
	case <-ctx.Done():
		w.forceClose()
		metricCtx := context.WithoutCancel(ctx)
		w.client.telemetry.recordDrain(metricCtx, outcomeForced)
		w.client.telemetry.countForcedShutdown(metricCtx)
		return fmt.Errorf("forced messaging shutdown: %w", ctx.Err())
	}
}

func (w *Worker) forceClose() {
	w.StartDrain()
	w.mu.Lock()
	if w.handlerCancel != nil {
		w.handlerCancel()
	}
	w.mu.Unlock()
	w.client.Close()
}

func (w *Worker) fail(err error) {
	w.StartDrain()
	w.client.signalTerminal(err)
	select {
	case w.fatal <- err:
	default:
	}
}
