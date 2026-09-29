// Package inboundwebhook owns the provider-neutral inbound webhook contracts.
//
// profile:inbound-webhooks-standard:start
package inboundwebhook

import (
	"bytes"
	"context"
	"errors"
)

var ErrUnavailable = errors.New("inbound webhook unavailable")

// Outcome is the closed synchronous acceptance category.
type Outcome string

const (
	OutcomeAccepted        Outcome = "accepted"
	OutcomeDuplicate       Outcome = "duplicate"
	OutcomeUnknownEndpoint Outcome = "unknown_endpoint"
	OutcomeRejected        Outcome = "rejected"
	OutcomeConflict        Outcome = "conflict"
)

// Delivery is the raw signed request the HTTP adapter hands the receiver.
type Delivery struct {
	EndpointID string
	DeliveryID string
	Timestamp  string
	Signature  string
	Body       []byte
}

// Clone copies the body so the adapter's buffer and the receiver can coexist.
func (d Delivery) Clone() Delivery {
	d.Body = bytes.Clone(d.Body)
	return d
}

// Receiver is the HTTP-to-durable acceptance port. A non-nil error means the
// receiver could not confirm acceptance, and the outcome is meaningless; callers
// must treat acceptance as unavailable and retry with the same delivery
// identity. [ErrUnavailable] is the expected error for that case. With a nil
// error the outcome is one of:
//   - Accepted: a new receipt and processing job were durably accepted.
//   - Duplicate: the same delivery identity and body were already accepted.
//   - Conflict: that identity was previously used with a different body.
//   - Rejected: the delivery failed verification.
//   - UnknownEndpoint: no endpoint is configured for the delivery.
type Receiver interface {
	Receive(ctx context.Context, delivery Delivery) (Outcome, error)
}

var _ Receiver = UnknownEndpointReceiver{}

// UnknownEndpointReceiver answers every delivery as an unknown endpoint.
type UnknownEndpointReceiver struct{}

// Receive reports an unknown endpoint without durable work.
func (UnknownEndpointReceiver) Receive(context.Context, Delivery) (Outcome, error) {
	return OutcomeUnknownEndpoint, nil
}

// profile:inbound-webhooks-standard:end
