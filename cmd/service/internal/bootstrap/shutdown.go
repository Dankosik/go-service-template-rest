package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
)

// shutdownBudget is the one deadline every teardown stage draws from.
// runtimeopts owns what a stage may spend against it; this type owns when the
// clock starts.
//
// The clock starts when teardown begins rather than at startup, because that is
// when the platform's grace period starts. All uses are on the goroutine running
// Run, in sequence; the Once is for the several callers that may each be the
// first to observe that serving has ended.
type shutdownBudget struct {
	grace    time.Duration
	started  sync.Once
	deadline time.Time
}

func newShutdownBudget(grace time.Duration) *shutdownBudget {
	return &shutdownBudget{grace: grace}
}

// start fixes the deadline, and is safe to call from every point that can begin
// a teardown: a signal, a server that stopped on its own, or a failed startup.
func (b *shutdownBudget) start() {
	b.started.Do(func() { b.deadline = time.Now().Add(b.grace) })
}

func (b *shutdownBudget) stage(base context.Context, want time.Duration) context.Context {
	b.start()
	ctx, cancel := runtimeopts.TeardownStage(base, b.deadline, want)
	// Consumed synchronously by the stage. Releasing the timer at the call site
	// would defeat the bound.
	context.AfterFunc(ctx, cancel)
	return ctx
}

// clamp reports how long a stage asking for want may actually take, for the one
// stage that needs the number rather than a context: the drain hands its bound
// to drainAndShutdown.
func (b *shutdownBudget) clamp(want time.Duration) time.Duration {
	b.start()
	return runtimeopts.TeardownBudget(want, b.deadline)
}

// validateShutdownGraceBudget rejects a drain budget that cannot fit inside the
// platform's grace period alongside the teardown that follows it.
//
// It lives here rather than in internal/config because shutdownTailBudget is
// owned by this package: the stage ceilings are process structure, not
// configuration.
func validateShutdownGraceBudget(cfg config.Config) error {
	return runtimeopts.ValidateGracePeriod(
		cfg.HTTP.GracePeriod,
		"http.shutdown_timeout",
		cfg.HTTP.ShutdownTimeout,
		shutdownTailBudget,
	)
}

type startupDrainer interface {
	StartDrain()
}

// profile:grpc:start
type startupDrainSet []startupDrainer

func (set startupDrainSet) StartDrain() {
	for _, drainer := range set {
		drainer.StartDrain()
	}
}

// profile:grpc:end

type shutdownServer interface {
	Shutdown(ctx context.Context) error
	Close() error
}

func drainAndShutdown(ctx context.Context, log *slog.Logger, propagationDelay time.Duration, timeout time.Duration, drainer startupDrainer, servers ...shutdownServer) error {
	log.InfoContext(
		ctx,
		"shutdown_started",
		startupLogArgs(
			"shutdown",
			"shutdown",
			"started",
		)...,
	)
	log.InfoContext(
		ctx,
		"drain_started",
		startupLogArgs(
			"shutdown",
			"drain",
			"started",
		)...,
	)
	drainer.StartDrain()
	log.InfoContext(
		ctx,
		"readiness_disabled",
		startupLogArgs(
			"shutdown",
			"readiness",
			"success",
		)...,
	)

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
	defer cancel()

	if propagationDelay > 0 {
		if deadline, ok := shutdownCtx.Deadline(); ok {
			remaining := time.Until(deadline)
			if remaining < propagationDelay {
				propagationDelay = remaining
			}
		}
		if err := sleepWithContext(shutdownCtx, propagationDelay); err != nil {
			return fmt.Errorf("drain propagation wait failed: %w", err)
		}
	}

	resultCh := make(chan error, len(servers))
	for _, server := range servers {
		go func() {
			resultCh <- server.Shutdown(shutdownCtx)
		}()
	}

	var shutdownErr error
	for completed := 0; completed < len(servers); completed++ {
		select {
		case err := <-resultCh:
			shutdownErr = errors.Join(shutdownErr, err)
		case <-shutdownCtx.Done():
			shutdownErr = errors.Join(shutdownErr, shutdownCtx.Err())
			completed = len(servers)
		}
	}
	if shutdownErr != nil {
		if errors.Is(shutdownErr, context.DeadlineExceeded) {
			log.ErrorContext(
				ctx,
				"shutdown_timeout",
				startupLogArgs(
					"shutdown",
					"drain",
					"error",
					"error.type", "deadline_exceeded",
				)...,
			)
			// Shutdown closed the listeners but waits indefinitely for active
			// connections, so on deadline the handlers it gave up on are still
			// running — and still holding pooled resources that the next
			// shutdown stage has to wait for. Close tears those connections
			// down so the rest of the sequence can proceed. The alternative is
			// not a graceful finish; it is the same abrupt end a moment later
			// when the platform SIGKILLs the process, minus the telemetry.
			shutdownErr = errors.Join(shutdownErr, forceCloseServers(ctx, log, servers...))
		}
		return fmt.Errorf("graceful shutdown failed: %w", shutdownErr)
	}

	log.InfoContext(
		ctx,
		"drain_completed",
		startupLogArgs(
			"shutdown",
			"drain",
			"success",
		)...,
	)
	return nil
}

// forceCloseServers abandons whatever the graceful drain could not finish.
// http.Server.Close is documented as returning any error from closing the
// listeners; already-closed listeners are the expected case here, so a failure is
// recorded rather than treated as fatal.
func forceCloseServers(ctx context.Context, log *slog.Logger, servers ...shutdownServer) error {
	var closeErr error
	for _, server := range servers {
		if err := server.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			closeErr = errors.Join(closeErr, err)
		}
	}

	log.WarnContext(
		ctx,
		"shutdown_forced",
		startupLogArgs(
			"shutdown",
			"drain",
			"degraded",
			"reason", "in_flight_requests_outlived_shutdown_timeout",
		)...,
	)
	return closeErr
}
