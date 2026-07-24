package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

type startupDrainer interface {
	StartDrain()
}

type shutdownServer interface {
	Shutdown(context.Context) error
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
