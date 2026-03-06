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

func drainAndShutdown(ctx context.Context, propagationDelay time.Duration, timeout time.Duration, drainer startupDrainer, srv shutdownServer) error {
	slog.Info(
		"shutdown_started",
		startupLogArgs(
			ctx,
			"shutdown",
			"shutdown",
			"started",
		)...,
	)
	slog.Info(
		"drain_started",
		startupLogArgs(
			ctx,
			"shutdown",
			"drain",
			"started",
		)...,
	)
	drainer.StartDrain()
	slog.Info(
		"readiness_disabled",
		startupLogArgs(
			ctx,
			"shutdown",
			"readiness",
			"success",
		)...,
	)
	if propagationDelay > 0 {
		if err := sleepWithContext(context.WithoutCancel(ctx), propagationDelay); err != nil {
			return fmt.Errorf("drain propagation wait failed: %w", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
		if errors.Is(err, context.DeadlineExceeded) {
			slog.Error(
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

	slog.Info(
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
