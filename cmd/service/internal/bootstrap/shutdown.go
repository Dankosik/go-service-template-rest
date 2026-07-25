package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type startupDrainer interface {
	StartDrain()
}

type shutdownServer interface {
	Shutdown(context.Context) error
	Close() error
}

func drainAndShutdown(ctx context.Context, log *slog.Logger, propagationDelay time.Duration, timeout time.Duration, drainer startupDrainer, servers ...shutdownServer) error {
	log.Info(
		"shutdown_started",
		startupLogArgs(
			ctx,
			"shutdown",
			"shutdown",
			"started",
		)...,
	)
	log.Info(
		"drain_started",
		startupLogArgs(
			ctx,
			"shutdown",
			"drain",
			"started",
		)...,
	)
	drainer.StartDrain()
	log.Info(
		"readiness_disabled",
		startupLogArgs(
			ctx,
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
			log.Error(
				"shutdown_timeout",
				startupLogArgs(
					ctx,
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

	log.Info(
		"drain_completed",
		startupLogArgs(
			ctx,
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

	log.Warn(
		"shutdown_forced",
		startupLogArgs(
			ctx,
			"shutdown",
			"drain",
			"degraded",
			"reason", "in_flight_requests_outlived_shutdown_timeout",
		)...,
	)
	return closeErr
}
