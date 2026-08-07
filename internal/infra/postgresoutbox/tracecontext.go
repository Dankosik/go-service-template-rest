package postgresoutbox

import (
	"context"
	"encoding/json"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// This file owns the creation context: the trace context that was active when
// an event was appended, in the carrier shape the trace_context column stores.
// [Store.Append] captures it, [Store.Claim] restores it, and the [Publisher]
// adapter is what finally does something with it.
//
// One rule governs both directions, and it is why neither function below
// returns an error: a trace context can never decide whether an event is
// delivered. An outbox exists so that a broker outage becomes backlog instead
// of a failed request; a telemetry field that could reject an append, or fail a
// publication, would invert exactly that. A context that cannot be captured is
// stored as absent and counted, and one that cannot be decoded is read as
// absent. The event still travels either way.

// maxTraceContextBytes bounds the stored creation context, and
// migrations/000001_postgres_outbox.sql restates it as a CHECK literal.
//
// The bound is small on purpose. W3C Trace Context caps `tracestate` at 512
// bytes and fixes `traceparent` at 55, so the repository's configured
// propagator fits in about 600 bytes of JSON with room for a third key. A
// propagator that emits more — baggage, most likely — degrades to an absent
// context rather than growing the row, which is the trade this file's header
// states.
const maxTraceContextBytes = 1 << 10

// emptyTraceContext is the stored form of "this append carried no propagatable
// context", and matches the column's own default. Every append that has no
// context shares this slice; nothing writes through it, and pgx only reads it
// while encoding.
var emptyTraceContext = []byte("{}")

// captureCreationContext renders the trace context active on ctx into the bytes
// the column stores. degraded reports that a context existed but could not be
// stored, which is a propagator misconfiguration an operator should see — an
// absent context is not degraded, because appending outside a trace is ordinary.
func captureCreationContext(ctx context.Context) (stored []byte, degraded bool) {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	if len(carrier) == 0 {
		return emptyTraceContext, false
	}
	encoded, err := json.Marshal(carrier)
	if err != nil || len(encoded) > maxTraceContextBytes {
		return emptyTraceContext, true
	}
	return encoded, false
}

// creationContextFromStored reads a stored context back into the carrier an
// adapter and the relay's publish span both consume.
//
// It returns nil for an absent context rather than an empty map, which saves an
// allocation per claimed event on the ordinary path and costs nothing to use: a
// nil [propagation.MapCarrier] answers Get with the empty string and Keys with
// nothing, so extracting from it yields the same unset context an empty map
// would.
func creationContextFromStored(stored []byte) propagation.MapCarrier {
	if len(stored) <= len(emptyTraceContext) {
		return nil
	}
	var carrier propagation.MapCarrier
	if err := json.Unmarshal(stored, &carrier); err != nil {
		return nil
	}
	return carrier
}
