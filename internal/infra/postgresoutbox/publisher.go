package postgresoutbox

import (
	"context"
	"fmt"
	"reflect"
)

// Publisher returns nil only after the selected broker durably acknowledges
// the same Event.ID. Implementations must stop using Event when Publish returns.
//
// The adapter decides which envelope fields reach the broker, and nothing here
// checks that it forwarded any of them. Two are worth deciding deliberately,
// because both are carried for a consumer this package cannot see:
// [Event.Metadata] holds whatever correlation the feature owns, and
// [Event.CreationContext] holds the trace context of the operation that
// appended the event:
//
//	ctx = otel.GetTextMapPropagator().Inject(ctx, header)  // onto broker headers
//
// An adapter whose message type has no header carrier drops the consumer's half
// of the trace at the outbox boundary — silently, because the bytes are still in
// PostgreSQL and the relay's own publish span still links to the producer.
// Either forward both onto broker headers or state in the adapter that it does
// not.
//
// The relay calls Publish concurrently, so implementations must be safe for
// concurrent use. Concurrent calls never carry two events that share an ordering
// key, and PostgreSQL enforces that rather than this package: the partial unique
// index outbox_events_ordering_ready_key in
// migrations/000001_postgres_outbox.sql admits one ready row per key.
//
// ctx carries the deadline of the whole claimed batch rather than a per-call
// timeout — the earlier of the configured publish timeout and the batch's lease
// — so a late event of a large batch sees whatever budget the earlier ones
// left. Returning nil once that deadline has passed is treated as unproven and
// retried, because a publisher that stopped waiting cannot have observed the
// acknowledgement. A panic is fatal to the relay process: the event is released
// for retry, the rest of the batch finalizes, and the process exits, because a
// panicking adapter is a deployment fault rather than a transient one.
//
// An adapter that returns a sentinel of its own — one this package does not
// already produce — owes it two edits outside the adapter, because no
// vocabulary here is derived from the error. It needs a case in
// publicationErrorClass in relay_publish.go, the only producer of the class
// that this event's metric sample, its operator log, and its stored
// last_error_class column all carry, and the matching constant in
// boundedErrorType in vocabulary.go. Skipping the first is the expensive one:
// the sentinel falls into that switch's default, so all three surfaces agree on
// publisher_temporary — a class the adapter never meant — and the event opts out
// of the attempt cap, which only [ErrPublicationNotAccepted] trips.
// TestErrorClassVocabularyIsBounded cannot catch this for you: it drives this
// package's own producers, and an adapter is by definition outside them.
//
// None of that reaches failureClass in cmd/outbox-relay/main.go. Every Publish
// error becomes a durable transition — retried or poisoned — and never stops the
// relay, so it is never a process exit class. The exit classes an adapter does
// own are its builder's; see the table in
// docs/postgres-transactional-outbox.md.
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
