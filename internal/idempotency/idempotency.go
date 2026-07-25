// Package idempotency owns the values shared between the HTTP middleware that
// makes a retried unsafe request safe and the storage that remembers what the
// first attempt answered.
//
// It is a leaf so both sides can depend on it. The store interface itself is
// declared by its consumer — see httpx.IdempotencyStore — which is what lets a
// persistence adapter satisfy it without importing the transport package and
// dragging the router, the OpenAPI validator, and the generated contract into a
// database package's dependency graph.
//
// Nothing here performs I/O or knows the wire format. The three response fields
// are what a replay has to reproduce byte for byte, which is why they are HTTP
// shaped rather than domain shaped.
package idempotency

import (
	"errors"
	"net/http"
)

var (
	// ErrInFlight reports that another attempt with the same key has not finished.
	// The caller retries; it must not be told the work conflicts.
	ErrInFlight = errors.New("idempotency key is in flight")

	// ErrKeyReused reports that the key was already spent on a different request.
	// This is a client defect — the same key must identify the same intent — and it
	// is the one case where replay would be dangerous.
	ErrKeyReused = errors.New("idempotency key was used for a different request")
)

// StoredResponse is a completed response held for replay.
type StoredResponse struct {
	Status int
	Header http.Header
	Body   []byte
	// Replayable is false when the response was too large to hold, or was streamed
	// in a way the recorder could not capture. A repeat then learns the key is
	// spent instead of re-running the work: giving up on the exact bytes must not
	// give up on the work running once.
	Replayable bool
}
