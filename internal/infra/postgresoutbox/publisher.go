package postgresoutbox

import (
	"context"
	"fmt"
	"reflect"
)

// Publisher returns nil only after the selected broker acknowledges the same
// Event.ID under the adapter's configured durability contract. Publish may be
// called concurrently.
//
// Implementations must treat Event and all data it references as read-only and
// stop using them before Publish returns. In particular, they must neither
// mutate nor retain Payload, Metadata, or the map returned by CreationContext.
//
// The result vocabulary is closed:
//
//   - nil means the broker acknowledged this Event.ID under that contract;
//   - [ErrPermanentPublication] means retrying the same bytes cannot succeed;
//   - [ErrPublicationNotAccepted] means the broker definitely did not accept
//     the event;
//   - every other error is an ambiguous, retryable publication failure.
//
// Relay-owned sentinels, including [ErrPublisherPanic], are reserved for the
// relay and must not be returned by an adapter.
//
// The adapter decides which envelope fields reach the broker. Nothing here
// verifies that it forwards [Event.Metadata] or [Event.CreationContext], so the
// adapter must either forward each one deliberately or document that it drops
// it. Concurrent calls never carry two events with the same ordering key;
// PostgreSQL enforces that invariant.
//
// ctx carries the deadline of the whole claimed batch rather than a per-call
// timeout — the earlier of the configured publish timeout and the batch's lease
// — so a late event of a large batch sees whatever budget the earlier ones
// left. Returning nil once that deadline has passed is treated as unproven and
// retried, because a publisher that stopped waiting cannot have observed the
// acknowledgement. A panic is fatal to the relay process: the event is released
// for retry, the rest of the batch finalizes, and the process exits, because a
// panicking adapter is a deployment fault rather than a transient one.
// profile:messaging-nats-jetstream:start
//
// natsjs.NewOutboxPublisher is the selected adapter for this template.
// profile:messaging-nats-jetstream:end
type Publisher interface {
	Publish(ctx context.Context, event Event) error
}

// ValidatePublisher rejects a publisher that cannot publish: an untyped nil,
// and a typed nil that satisfies the interface without being usable. [NewRelay]
// applies it, and cmd/outbox-relay applies it at startup so a service that
// never registered an adapter fails before the process builds telemetry and a
// PostgreSQL pool for a relay that cannot run.
func ValidatePublisher(publisher Publisher) error {
	if publisher == nil || holdsTypedNil(publisher) {
		return fmt.Errorf("%w: outbox publisher is not registered", ErrConfig)
	}
	return nil
}

// holdsTypedNil reports an interface value holding a typed nil, which `== nil`
// does not. [ValidatePublisher] and newRelay both need it, for the same reason:
// each takes an interface a composition root may have left unset.
func holdsTypedNil(value any) bool {
	reflected := reflect.ValueOf(value)
	//nolint:exhaustive // Only kinds that can contain a typed nil are relevant.
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
