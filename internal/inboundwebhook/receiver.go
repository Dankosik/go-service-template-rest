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
	OutcomeUnavailable     Outcome = "unavailable"
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

// Result is one closed acceptance category. Unexpected failures are errors.
type Result struct {
	Outcome Outcome
}

// Receiver is the HTTP-to-durable acceptance port.
type Receiver interface {
	Receive(ctx context.Context, delivery Delivery) (Result, error)
}

var _ Receiver = NoopReceiver{}

// NoopReceiver answers every delivery as an unknown endpoint.
type NoopReceiver struct{}

// Receive reports an unknown endpoint without durable work.
func (NoopReceiver) Receive(context.Context, Delivery) (Result, error) {
	return Result{Outcome: OutcomeUnknownEndpoint}, nil
}

// profile:inbound-webhooks-standard:end
