package postgresoutbox

import (
	"context"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/jackc/pgx/v5"
)

const (
	// appendChannel is the PostgreSQL NOTIFY channel the append trigger signals.
	appendChannel = "outbox_appended"
	// listenerRetryDelay bounds how fast a failed listener reconnects. A missing
	// listener only costs pickup latency, so the retry is deliberately unhurried.
	listenerRetryDelay = time.Second
	// listenerCloseTimeout bounds returning the connection after cancellation.
	listenerCloseTimeout = time.Second
)

// listenForAppends turns committed appends into relay wake-ups, so pickup
// latency is a round trip rather than a poll interval.
//
// It owns one connection outside the pool: the relay's own budget stays sized
// for its claim and finalization statements, and a blocked listener can never
// starve them. Every failure here is a latency concern rather than a
// correctness one, because the relay still claims on its poll timer.
func listenForAppends(pool *postgres.Pool, telemetry *Telemetry) listener {
	return func(ctx context.Context, wake chan<- struct{}) {
		for ctx.Err() == nil {
			err := consumeAppends(ctx, pool, wake)
			if err != nil && ctx.Err() == nil {
				telemetry.LogListenerRetry(ctx, err)
				sleep(ctx, listenerRetryDelay)
			}
		}
	}
}

func consumeAppends(ctx context.Context, pool *postgres.Pool, wake chan<- struct{}) error {
	conn, err := pgx.ConnectConfig(ctx, pool.PGX().Config().ConnConfig.Copy())
	if err != nil {
		return fmt.Errorf("connect outbox listener: %w", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), listenerCloseTimeout)
		defer cancel()
		_ = conn.Close(closeCtx)
	}()

	if _, err := conn.Exec(ctx, "LISTEN "+pgx.Identifier{appendChannel}.Sanitize()); err != nil {
		return fmt.Errorf("listen for outbox appends: %w", err)
	}
	// Appends committed while no listener was subscribed produced no signal, so
	// re-check the backlog once the subscription exists.
	signal(wake)

	for {
		if _, err := conn.WaitForNotification(ctx); err != nil {
			return fmt.Errorf("wait for outbox append: %w", err)
		}
		signal(wake)
	}
}

// signal never blocks: one pending wake-up already tells the relay to claim
// again, and the claim it triggers sees every event committed before it ran.
func signal(wake chan<- struct{}) {
	select {
	case wake <- struct{}{}:
	default:
	}
}

func sleep(ctx context.Context, duration time.Duration) {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
