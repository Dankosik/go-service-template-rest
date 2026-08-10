package postgresoutbox

import (
	"context"
	"fmt"
)

func (t *Telemetry) LogPoison(ctx context.Context, errorClass string, attempt int) {
	if t != nil {
		t.log.ErrorContext(ctx, "outbox_event_poisoned", "error.type", errorClass, "attempt", attempt)
	}
}

func (t *Telemetry) LogPublisherStuck(ctx context.Context) {
	if t != nil {
		t.log.ErrorContext(ctx, "outbox_publisher_stuck", "error.type", classStuck)
	}
}

// LogPublisherPanic reports an adapter panic with the description the recover
// consumed. It is the one place in this package that logs an unbounded string,
// and it is deliberate: the process is exiting over a deployment fault, and the
// class alone names the category without naming the line of code.
//
// The event is not named. A panicking adapter is reproducible from its stack,
// while an event id on an ERROR line is the identity this package keeps off
// telemetry everywhere else.
func (t *Telemetry) LogPublisherPanic(ctx context.Context, value any, stack []byte) {
	if t != nil {
		t.log.ErrorContext(ctx, "outbox_publisher_panic",
			"error.type", classPanic, "panic", fmt.Sprint(value), "stack", string(stack))
	}
}

func (t *Telemetry) LogRecovery(ctx context.Context, attempt int) {
	if t != nil {
		t.log.WarnContext(ctx, "outbox_lease_recovered", "attempt", attempt)
	}
}

// LogListenerRetry reports that wake-up notifications are unavailable. Pickup
// latency falls back to the poll interval until the listener reconnects.
//
// stage is a bounded class from listenerStage, never the driver's error text.
// A pgx connect failure formats the DSN's user, database, and host into its
// message, and this package promises never to log DSN material.
func (t *Telemetry) LogListenerRetry(ctx context.Context, stage string) {
	if t != nil {
		t.log.WarnContext(ctx, "outbox_listener_retry", "error.type", classDatabase, "stage", stage)
	}
}
