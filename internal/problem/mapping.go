package problem

import "time"

// Mapped is the client-visible answer for one classified domain error.
type Mapped struct {
	// Code selects the catalog entry that owns the status, title, and type URI.
	Code Code
	// Detail is shown to the client, and must be the service's own wording.
	// Passing err.Error() through is how an internal message, a SQL fragment, or
	// a dependency's hostname ends up in a response body. Empty falls back to the
	// catalog title.
	Detail string
	// RetryAfter, when positive, is sent as the Retry-After hint. A momentarily
	// saturated connection pool is the case it exists for: with no hint a client
	// library either gives up or retries immediately, and the second one is how a
	// saturated dependency stays saturated.
	RetryAfter time.Duration
}

// Mapper classifies one service-owned error into the problem a client should
// see, and reports false for an error it does not recognize.
//
// This is the seam that keeps error mapping in one place. The generated strict
// server routes any error a handler returns to its response error handler, which
// is the single point where an error becomes a status — but with nothing to
// extend it, the only way to answer 404 for a domain sentinel was a switch inside
// every operation, each arm building a nested generated literal. That is how a
// service ends up with one classification table per handler and no test that any
// of them agree.
//
// It lives here rather than beside the transport that consumes it for the reason
// this package is a leaf at all: the classification is owned by whoever owns the
// error identities, and the repository's depguard contract forbids a feature
// package from importing a concrete infra adapter. A mapper type exported from
// internal/infra/http would be reachable only by another adapter, which is what
// pushed the table back into the handlers it was meant to leave.
type Mapper func(error) (Mapped, bool)

// Classify runs mappers in order and returns the first match.
//
// First match wins, so a service can order specific classifications ahead of
// broad ones without every mapper having to know about the others. Nil entries
// are skipped, which keeps a profile-supplied slice usable without the caller
// filtering it.
func Classify(err error, mappers []Mapper) (Mapped, bool) {
	for _, classify := range mappers {
		if classify == nil {
			continue
		}
		if mapped, ok := classify(err); ok {
			return mapped, true
		}
	}
	return Mapped{}, false
}
