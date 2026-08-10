package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/observability/logctx"
)

// errRelayPanic and errPublisherPanic report a run loop that ended in a
// recovered panic.
//
// They gate cleanup rather than only explaining the exit. A loop that panicked
// cannot account for the goroutines it started, so the publisher adapter and the
// pool stay owned by process exit — which is what relayTeardown's unsafe path
// already exists to do for a publisher goroutine that outlived cancellation.
var (
	errRelayPanic     = errors.New("outbox relay panicked")
	errPublisherPanic = errors.New("outbox publisher supervisor panicked")
)

// The post-drain budgets, kept together because validateRuntimeConfig charges
// http.grace_period for their sum and every one of them is spent on the same
// shutdown path.
const (
	diagnosticsClose = 2 * time.Second
	publisherClose   = 5 * time.Second
	telemetryClose   = 5 * time.Second
	forcedJoin       = 2 * time.Second
)

func validateRuntimeConfig(cfg config.Config) error {
	if !cfg.Outbox.Enabled {
		return fmt.Errorf("%w: outbox must be enabled for relay", postgresoutbox.ErrConfig)
	}
	if !cfg.Postgres.Enabled {
		return fmt.Errorf("%w: postgres must be enabled for relay", postgresoutbox.ErrConfig)
	}
	if strings.TrimSpace(cfg.Observability.Metrics.Addr) == "" {
		return fmt.Errorf("%w: outbox diagnostics address is required", postgresoutbox.ErrConfig)
	}
	// Relay and publisher supervisors have distinct joins and can both consume
	// their bound before dependency shutdown begins.
	return runtimeopts.ValidateGracePeriod(
		cfg.HTTP.GracePeriod,
		"outbox.drain_timeout",
		cfg.Outbox.DrainTimeout,
		2*forcedJoin+diagnosticsClose+publisherClose+telemetryClose,
	)
}

// relayTeardown releases the publisher and then the pool, and only while
// releasing them is safe. Its fields are filled in as each dependency appears,
// so a startup failure releases exactly what had been built by then.
//
// Publisher before pool is deliberate and is not reverse construction order:
// adapter cleanup can still be draining sends, and the pool is what a last-moment
// durable write would need.
//
// unsafe is the relay's report that a publisher goroutine outlived cancellation.
// That goroutine can still reach both the adapter and the pool, so nothing is
// released and process exit owns them instead.
//
// release carries no idempotence latch because run defers it exactly once. A
// second call site would need one: PublisherBuilder does not promise its cleanup
// tolerates being called twice.
type relayTeardown struct {
	publisher *PublisherRuntime
	pool      func()
	unsafe    bool
}

func (t *relayTeardown) release(signalCtx context.Context) error {
	if t.unsafe {
		return nil
	}
	var err error
	if t.publisher != nil {
		var unsafe bool
		err, unsafe = closePublisher(signalCtx, t.publisher.shutdown)
		if unsafe {
			return err
		}
		t.publisher.shutdown = nil
	}
	if t.pool != nil {
		t.pool()
	}
	return err
}

func closePublisher(parent context.Context, cleanup func(context.Context) error) (error, bool) {
	if cleanup == nil {
		return nil, false
	}
	// The zero deadline a parent without one yields is what runtimeopts reads as
	// an unarmed process budget, which is the startup-failure path: this cleanup
	// runs from run's defer against the signal context and gets its whole bound.
	deadline, _ := parent.Deadline()
	ctx, cancel := runtimeopts.TeardownStage(parent, deadline, publisherClose)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		var runErr error
		defer func() {
			if recover() != nil {
				runErr = errors.New("outbox publisher cleanup panicked")
			}
			result <- runErr
		}()
		runErr = cleanup(ctx)
	}()
	select {
	case err := <-result:
		return err, false
	case <-ctx.Done():
		return fmt.Errorf("close outbox publisher: %w", ctx.Err()), true
	}
}

// runRelayLifecycle also reports the process-wide teardown deadline it armed, so
// the telemetry flush run defers takes its bound from the same grace period the
// stages here spend rather than from a fresh ceiling of its own. The zero time it
// returns before that deadline exists means no bound was armed.
func runRelayLifecycle(
	signalCtx context.Context,
	startupCtx context.Context,
	cfg config.Config,
	log *slog.Logger,
	metrics *telemetry.Metrics,
	relay relayRunner,
	publisher *PublisherRuntime,
) (postgresoutbox.RelayResult, time.Time) {
	ready := func() bool { return relay.Ready() && publisher.ready() }
	diag, err := runtimeopts.ListenDiagnostics(startupCtx, cfg.Observability.Metrics.Addr, "outbox", ready, metrics)
	if err != nil {
		return postgresoutbox.RelayResult{Err: err}, time.Time{}
	}
	relayCtx, relayCancel := context.WithCancel(context.WithoutCancel(signalCtx))
	defer relayCancel()
	publisherCtx, publisherCancel := context.WithCancel(context.WithoutCancel(signalCtx))
	defer publisherCancel()
	// Both loops run under superviseRunLoop rather than bare. Neither can be a
	// background.Supervisor task: the select below reads each exit on its own and
	// drains it on its own budget, which the supervisor's single aggregated
	// failure channel cannot express. Left bare, an unrecovered panic in either
	// would take the process down without the ordered drain below or the
	// telemetry flush that records it.
	relayResult := make(chan postgresoutbox.RelayResult, 1)
	go superviseRunLoop(
		relayCtx, log, "relay", relayResult,
		postgresoutbox.RelayResult{Err: errRelayPanic, CleanupUnsafe: true}, relay.Run,
	)
	publisherResult := make(chan error, 1)
	go superviseRunLoop(publisherCtx, log, "publisher", publisherResult, errPublisherPanic, publisher.run)

	// Three ways out, and only the signal is an ordinary one. A relay or a
	// diagnostics server that returned by itself is a fault, so each supplies
	// the reason the other two would otherwise leave unexplained.
	var result postgresoutbox.RelayResult
	var triggerErr error
	relayStopped := false
	publisherStopped := false
	select {
	case <-signalCtx.Done():
	case result = <-relayResult:
		relayStopped = true
		if result.Err == nil {
			result.Err = errors.New("outbox relay stopped unexpectedly")
		}
	case <-diag.Stopped():
		// diag.Stop below carries whatever Serve reported.
		triggerErr = errors.New("outbox diagnostics stopped unexpectedly")
	case triggerErr = <-publisherResult:
		publisherStopped = true
		if triggerErr == nil {
			triggerErr = errors.New("outbox publisher supervisor stopped unexpectedly")
		}
	}

	processCtx, processCancel, shutdownDeadline := runtimeopts.ArmTeardown(signalCtx, cfg.HTTP.GracePeriod)
	defer processCancel()
	relay.StartDrain()
	if !relayStopped {
		result = drainRelay(processCtx, cfg.Outbox.DrainTimeout, relayCancel, relayResult)
	}

	publisherErr, publisherUnsafe := stopPublisherSupervisor(
		processCtx, publisherCancel, publisherResult, publisherStopped,
	)
	result.Err = errors.Join(result.Err, publisherErr)
	// A publisher that panicked is unsafe for the same reason one that outlived
	// cancellation is: neither can account for what it left running, and the
	// adapter cleanup below would run underneath it.
	//
	// Both arrival paths are checked because the select above takes whichever
	// exit came first: a panic can reach this either as the trigger or as the
	// joined result. It is also read after drainRelay, which replaces result
	// wholesale and would otherwise discard a flag set before it.
	result.CleanupUnsafe = result.CleanupUnsafe || publisherUnsafe ||
		errors.Is(triggerErr, errPublisherPanic) || errors.Is(publisherErr, errPublisherPanic)

	diagnosticsErr := diag.Stop(processCtx, diagnosticsClose)
	var shutdownErr error
	if !result.CleanupUnsafe {
		var unsafe bool
		shutdownErr, unsafe = closePublisher(processCtx, publisher.shutdown)
		if !unsafe {
			publisher.shutdown = nil
		} else {
			result.CleanupUnsafe = true
		}
	}
	result.Err = errors.Join(triggerErr, result.Err, diagnosticsErr, shutdownErr)
	return result, shutdownDeadline
}

// superviseRunLoop runs one relay run loop and reports its outcome exactly once,
// substituting onPanic when the loop panicked.
//
// The result is assigned and then sent from the deferred call, the shape
// closePublisher above already uses: sending on the normal path and again from
// the recovery would deliver twice, and the second delivery would be read as a
// second exit by a select that has already acted on the first.
//
// It is generic because the two loops answer with different types — the relay
// with a [postgresoutbox.RelayResult] the drain reads fields from, the publisher
// with a plain error — and only the type differs. Two copies is how the recovery
// ends up on one of them and not the other.
func superviseRunLoop[T any](
	ctx context.Context,
	log *slog.Logger,
	loop string,
	results chan<- T,
	onPanic T,
	run func(context.Context) T,
) {
	var result T
	defer func() {
		if recovered := recover(); recovered != nil {
			logRunLoopPanic(ctx, log, loop, recovered)
			result = onPanic
		}
		results <- result
	}()
	result = run(ctx)
}

// logRunLoopPanic records where a recovered run-loop panic came from.
//
// The sentinel the loop reports names only that a panic happened. This is the
// only record of the defect behind it, and it is written from inside the
// deferred recovery because that is the one point the panicking frames still
// exist.
func logRunLoopPanic(ctx context.Context, log *slog.Logger, loop string, recovered any) {
	log.ErrorContext(
		ctx,
		"outbox_run_loop_panic",
		append(
			[]any{"component", "outbox_relay", "loop", loop},
			logctx.PanicAttrs(recovered, debug.Stack())...,
		)...,
	)
}

func stopPublisherSupervisor(
	processCtx context.Context,
	cancel context.CancelFunc,
	result <-chan error,
	alreadyStopped bool,
) (error, bool) {
	cancel()
	if alreadyStopped {
		return nil, false
	}
	joinCtx, joinCancel := context.WithTimeout(processCtx, forcedJoin)
	defer joinCancel()
	select {
	case runErr := <-result:
		if errors.Is(runErr, context.Canceled) {
			return nil, false
		}
		return runErr, false
	case <-joinCtx.Done():
		return fmt.Errorf("join outbox publisher supervisor: %w", joinCtx.Err()), true
	}
}

// drainRelay waits out a relay that has not reported yet and returns its result.
// It escalates in two steps: the drain budget lets the current batch finish, then
// cancellation forces its publications to stop. Only the final step can fail to
// produce a result, and the one it synthesizes reports cleanup as unsafe because
// the relay goroutine is still running.
func drainRelay(
	processCtx context.Context,
	drainTimeout time.Duration,
	runtimeCancel context.CancelFunc,
	relayResult <-chan postgresoutbox.RelayResult,
) postgresoutbox.RelayResult {
	drainCtx, drainCancel := context.WithTimeout(processCtx, drainTimeout)
	defer drainCancel()
	select {
	case result := <-relayResult:
		return result
	case <-drainCtx.Done():
	}

	runtimeCancel()
	joinCtx, joinCancel := context.WithTimeout(processCtx, forcedJoin)
	defer joinCancel()
	select {
	case result := <-relayResult:
		return result
	case <-joinCtx.Done():
		return postgresoutbox.RelayResult{
			CleanupUnsafe: true,
			Err:           fmt.Errorf("join outbox relay: %w", joinCtx.Err()),
		}
	}
}
