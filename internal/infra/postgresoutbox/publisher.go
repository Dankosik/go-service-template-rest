package postgresoutbox

import (
	"context"
	"errors"
)

// Publisher returns nil only after the selected broker durably acknowledges
// the same Event.ID. Implementations must stop using Event when Publish returns.
//
// The relay calls Publish concurrently, up to its configured publish
// concurrency, so implementations must be safe for concurrent use. Concurrent
// calls never carry two events that share an ordering key, because the claim
// query hands out at most one ready event per key.
type Publisher interface {
	Publish(ctx context.Context, event Event) error
}

// ErrPermanentPublication lets an adapter reject an occurrence without using
// transport-specific error types in the relay.
var ErrPermanentPublication = errors.New("permanent outbox publication failure")

// ErrPublicationNotAccepted lets an adapter prove that the broker did not
// durably accept an occurrence. It remains retryable and is the only failure
// the attempt threshold poisons on; unclassified errors stay ambiguous and
// keep retrying rather than risking loss at a strict attempt cap.
var ErrPublicationNotAccepted = errors.New("outbox publication was not accepted")
