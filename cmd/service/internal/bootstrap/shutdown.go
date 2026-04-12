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

func drainAndShutdown(ctx context.Context, log *slog.Logger, propagationDelay time.Duration, timeout time.Duration, drainer startupDrainer, srv shutdownServer) error {
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

	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
		if errors.Is(err, context.DeadlineExceeded) {
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
		return fmt.Errorf("graceful shutdown failed: %w", err)
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
