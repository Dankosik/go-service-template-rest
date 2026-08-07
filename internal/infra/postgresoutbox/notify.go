package postgresoutbox

import (
	"context"
	"errors"
	"fmt"
	"time"

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

// The three stages a listener cycle can fail at. They are sentinels rather than
// plain wrap text because listenerStage classifies on them, and a stage is the
// only thing the operator log is allowed to carry — see LogListenerRetry.
var (
	errListenerConnect   = errors.New("connect outbox listener")
	errListenerSubscribe = errors.New("listen for outbox appends")
	errListenerWait      = errors.New("wait for outbox append")
)

// listenForAppends turns committed appends into relay wake-ups, so pickup
// latency is a round trip rather than a poll interval.
//
// It owns one connection outside the pool: the relay's own budget stays sized
// for its claim and finalization statements, and a blocked listener can never
// starve them. Every failure here is a latency concern rather than a
// correctness one, because the relay still claims on its poll timer.
func listenForAppends(connConfig *pgx.ConnConfig, telemetry *Telemetry) listener {
	return func(ctx context.Context, wake chan<- struct{}) {
		for ctx.Err() == nil {
			err := consumeAppends(ctx, connConfig, wake)
			if err != nil && ctx.Err() == nil {
				telemetry.LogListenerRetry(ctx, listenerStage(err))
				sleep(ctx, listenerRetryDelay)
			}
		}
	}
}

// listenerStage names which step of a listener cycle failed. It is a closed
// vocabulary for the same reason the error classes in vocabulary.go are. It
// stays here rather than joining them because its values label one log field of
// one function and never reach a metric attribute.
func listenerStage(err error) string {
	switch {
	case errors.Is(err, errListenerConnect):
		return "connect"
	case errors.Is(err, errListenerSubscribe):
		return "subscribe"
	case errors.Is(err, errListenerWait):
		return "wait"
	default:
		return boundedOther
	}
}

func consumeAppends(ctx context.Context, connConfig *pgx.ConnConfig, wake chan<- struct{}) error {
	conn, err := pgx.ConnectConfig(ctx, connConfig.Copy())
	if err != nil {
		return fmt.Errorf("%w: %w", errListenerConnect, err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), listenerCloseTimeout)
		defer cancel()
		_ = conn.Close(closeCtx)
	}()

	if _, err := conn.Exec(ctx, "LISTEN "+pgx.Identifier{appendChannel}.Sanitize()); err != nil {
		return fmt.Errorf("%w: %w", errListenerSubscribe, err)
	}
	// Appends committed while no listener was subscribed produced no signal, so
	// re-check the backlog once the subscription exists.
	signal(wake)

	for {
		if _, err := conn.WaitForNotification(ctx); err != nil {
			return fmt.Errorf("%w: %w", errListenerWait, err)
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
