package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
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
	cleanupReserve := 2*forcedJoin + diagnosticsClose + publisherClose + telemetryClose
	if cfg.HTTP.GracePeriod < cfg.Outbox.DrainTimeout ||
		cfg.HTTP.GracePeriod-cfg.Outbox.DrainTimeout < cleanupReserve {
		return fmt.Errorf(
			"%w: http.grace_period must fit outbox drain and post-drain cleanup (%s + %s)",
			config.ErrValidate,
			cfg.Outbox.DrainTimeout,
			cleanupReserve,
		)
	}
	return nil
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
	base := context.WithoutCancel(parent)
	if deadline, ok := parent.Deadline(); ok {
		var cancelDeadline context.CancelFunc
		base, cancelDeadline = context.WithDeadline(base, deadline)
		defer cancelDeadline()
	}
	ctx, cancel := context.WithTimeout(base, publisherClose)
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

func runRelayLifecycle(
	signalCtx context.Context,
	startupCtx context.Context,
	cfg config.Config,
	metrics *telemetry.Metrics,
	relay relayRunner,
	publisher *PublisherRuntime,
) postgresoutbox.RelayResult {
	ready := func() bool { return relay.Ready() && publisher.ready() }
	diag, err := startDiagnostics(startupCtx, cfg.Observability.Metrics.Addr, ready, metrics)
	if err != nil {
		return postgresoutbox.RelayResult{Err: err}
	}
	relayCtx, relayCancel := context.WithCancel(context.WithoutCancel(signalCtx))
	defer relayCancel()
	publisherCtx, publisherCancel := context.WithCancel(context.WithoutCancel(signalCtx))
	defer publisherCancel()
	relayResult := make(chan postgresoutbox.RelayResult, 1)
	go func() { relayResult <- relay.Run(relayCtx) }()
	publisherResult := make(chan error, 1)
	go func() { publisherResult <- publisher.run(publisherCtx) }()

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
	case <-diag.watch():
		// diag.stop below carries whatever Serve reported.
		triggerErr = errors.New("outbox diagnostics stopped unexpectedly")
	case triggerErr = <-publisherResult:
		publisherStopped = true
		if triggerErr == nil {
			triggerErr = errors.New("outbox publisher supervisor stopped unexpectedly")
		}
	}

	processCtx, processCancel := context.WithTimeout(context.WithoutCancel(signalCtx), cfg.HTTP.GracePeriod)
	defer processCancel()
	relay.StartDrain()
	if !relayStopped {
		result = drainRelay(processCtx, cfg.Outbox.DrainTimeout, relayCancel, relayResult)
	}

	publisherErr, publisherUnsafe := stopPublisherSupervisor(
		processCtx, publisherCancel, publisherResult, publisherStopped,
	)
	result.Err = errors.Join(result.Err, publisherErr)
	result.CleanupUnsafe = result.CleanupUnsafe || publisherUnsafe

	diagnosticsErr := diag.stop(processCtx)
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
	return result
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
