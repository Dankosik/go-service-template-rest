package postgresoutbox

import (
	"context"
	"errors"
)

// Publisher returns nil only after the selected broker durably acknowledges
// the same Event.ID. Implementations must stop using Event when Publish returns.
type Publisher interface {
	Publish(ctx context.Context, event Event) error
}

// ErrPermanentPublication lets an adapter reject an occurrence without using
// transport-specific error types in the relay.
var ErrPermanentPublication = errors.New("permanent outbox publication failure")

// ErrPublicationNotAccepted lets an adapter prove that the broker did not
// durably accept an occurrence. It remains retryable and counts toward
// exhaustion; unclassified errors remain ambiguous.
var ErrPublicationNotAccepted = errors.New("outbox publication was not accepted")
