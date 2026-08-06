package postgresoutbox

import (
	"context"
	"errors"
	"fmt"
)

// Publisher returns nil only after the selected broker durably acknowledges
// the same Event.ID. Implementations must stop using Event when Publish returns.
//
// The adapter decides which envelope fields reach the broker, and nothing here
// checks that it forwarded any of them. [Event.Metadata] is the one to decide
// deliberately: the outbox stores, retries, and size-budgets those bytes, and
// they are where correlation and trace context ride, but an adapter whose
// message type has no header carrier for them drops the trace at the outbox
// boundary — silently, because the bytes are still in PostgreSQL. Either
// forward Metadata onto broker headers or state in the adapter that it does
// not, so a feature populating it knows what it is buying.
//
// The relay calls Publish concurrently, up to its configured publish
// concurrency, so implementations must be safe for concurrent use. Concurrent
// calls never carry two events that share an ordering key, because the claim
// query hands out at most one ready event per key. PostgreSQL enforces that,
// not this package: the partial unique index outbox_events_ordering_ready_key
// in migrations/000001_postgres_outbox.sql admits one ready row per key.
//
// ctx carries the deadline of the whole claimed batch rather than a per-call
// timeout: it is the earlier of the configured publish timeout and the batch's
// lease, so a late event of a large batch sees whatever budget the earlier ones
// left. Returning nil once that deadline has passed is treated as unproven and
// retried, because a publisher that stopped waiting cannot have observed the
// acknowledgement. A panic is fatal to the relay process: the event is released
// for retry, the rest of the batch finalizes, and the process exits, because a
// panicking adapter is a deployment fault rather than a transient one.
// profile:messaging-nats-jetstream:start
//
// natsOutboxPublisher in test/postgres_outbox_natsjs_integration_test.go is a
// worked adapter against this contract.
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
	if publisher == nil || nilInterface(publisher) {
		return fmt.Errorf("%w: outbox publisher is not registered", ErrConfig)
	}
	return nil
}

// ErrPermanentPublication lets an adapter reject an occurrence without using
// transport-specific error types in the relay. It poisons the event on the
// first occurrence: the row is never retried, it blocks later work for its
// ordering key, and only an operator redrive releases it. Return it only for a
// rejection that retrying the same bytes cannot fix.
var ErrPermanentPublication = errors.New("permanent outbox publication failure")

// ErrPublicationNotAccepted lets an adapter prove that the broker did not
// durably accept an occurrence. It remains retryable and is the only failure
// the attempt threshold poisons on; unclassified errors stay ambiguous and
// keep retrying rather than risking loss at a strict attempt cap.
var ErrPublicationNotAccepted = errors.New("outbox publication was not accepted")
