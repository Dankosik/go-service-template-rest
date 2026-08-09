package postgresoutbox

import (
	"context"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Appender is the write path's whole view of this package: one call, inside the
// transaction that already owns a domain mutation. *[Store] is its only
// implementation here.
//
// It exists because the consumer that would otherwise declare it lives in an
// adopting service rather than in this repository — the PostgreSQL repository
// adapter doc.go names. Taking this type instead of *Store keeps the operator
// tooling and the whole relay half of Store out of reach of the request path,
// and the compiler rather than a comment is then what says the write path only
// appends.
type Appender interface {
	Append(ctx context.Context, tx pgx.Tx, events ...Event) error
}

// The one in-package use of the type above, and the reason it needs no other: a
// signature change to Append that leaves the interface behind fails to compile
// here rather than at the composition root of an adopting service.
var _ Appender = (*Store)(nil)

// Append stores every event in the transaction owned by the feature caller. It
// never begins or commits a transaction itself.
//
// One call is one statement and one round trip whatever mix of ordered and
// unordered events it carries, because the events travel as one array per
// column rather than one statement per event. A feature that emits several
// events per business transaction therefore pays the cost of a single append
// and holds its own row locks for that much less time. Nothing is sent unless
// every event is valid, and one rejected ordering sequence stores nothing.
//
// One call is one append operation in telemetry, because that is what the
// recorded duration measures; backlog gauges report events.
func (s *Store) Append(ctx context.Context, tx pgx.Tx, events ...Event) (err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "append", started, err) }()
	if !s.valid() {
		return errStoreRequired()
	}
	if tx == nil {
		return fmt.Errorf("%w: transaction is required", ErrConfig)
	}
	if len(events) == 0 {
		return nil
	}

	columns, err := newAppendColumns(ctx, events)
	if err != nil {
		return err
	}
	if columns.traceContextDegraded {
		// Counted rather than logged: this is a property of the propagator the
		// process is configured with, so it holds for every append until an
		// operator changes it, and a log line per append would say the same thing
		// at the rate of the write path.
		s.telemetry.CountOperation(ctx, "trace_capture", outcomeRejected, classValidation)
	}
	queries := sqlcgen.New(tx)
	if !columns.ordered {
		if err := queries.InsertOutboxEvents(ctx, columns.withoutOrdering()); err != nil {
			return fmt.Errorf("insert outbox events: %w", err)
		}
		return nil
	}

	rejected, err := queries.InsertOutboxEventsWithOrdering(ctx, columns.withOrdering())
	if err != nil {
		return fmt.Errorf("insert outbox events: %w", err)
	}
	if len(rejected) > 0 {
		return fmt.Errorf(
			"%w: key %q sequence %d is not above the retained high-water mark",
			ErrOrderingSequence, rejected[0].OrderingKey, rejected[0].FirstSequence,
		)
	}
	return nil
}

// appendColumns is one append laid out the way its statement reads it: one
// array per column instead of one struct per event.
type appendColumns struct {
	ids                  []string
	types                []string
	sources              []string
	destinations         []string
	schemas              []string
	occurredAt           []pgtype.Timestamptz
	payloads             [][]byte
	metadatas            [][]byte
	traceContexts        [][]byte
	envelopeFingerprints [][]byte
	orderingKeys         []string
	orderingSequences    []int64
	ordered              bool
	// traceContextDegraded reports that this call had a trace context that could
	// not be stored, so every event in it carries an absent one.
	traceContextDegraded bool
}

func (c appendColumns) withoutOrdering() sqlcgen.InsertOutboxEventsParams {
	return sqlcgen.InsertOutboxEventsParams{
		Ids: c.ids, EventTypes: c.types, Sources: c.sources, Destinations: c.destinations,
		SchemaNames: c.schemas, OccurredAts: c.occurredAt, Payloads: c.payloads, Metadatas: c.metadatas,
		TraceContexts: c.traceContexts, EnvelopeFingerprints: c.envelopeFingerprints,
	}
}

func (c appendColumns) withOrdering() sqlcgen.InsertOutboxEventsWithOrderingParams {
	return sqlcgen.InsertOutboxEventsWithOrderingParams{
		Ids: c.ids, EventTypes: c.types, Sources: c.sources, Destinations: c.destinations,
		SchemaNames: c.schemas, OccurredAts: c.occurredAt, Payloads: c.payloads, Metadatas: c.metadatas,
		TraceContexts: c.traceContexts, EnvelopeFingerprints: c.envelopeFingerprints,
		OrderingKeys: c.orderingKeys, OrderingSequences: c.orderingSequences,
	}
}

// newAppendColumns validates every event and transposes the call into those
// arrays. An event that owns no ordering head keeps the zero key and sequence
// it already has in Go, which is what the ordered statement reads as an absent
// head; a call where every event is like that skips that statement entirely.
func newAppendColumns(ctx context.Context, events []Event) (appendColumns, error) {
	// One capture for the whole call. Every event of one Append shares the
	// caller's context, so they share its creation context too — and the column
	// array then holds one slice repeated rather than one encoding per event.
	traceContext, degraded := captureCreationContext(ctx)
	columns := appendColumns{
		ids:                  make([]string, len(events)),
		types:                make([]string, len(events)),
		sources:              make([]string, len(events)),
		destinations:         make([]string, len(events)),
		schemas:              make([]string, len(events)),
		occurredAt:           make([]pgtype.Timestamptz, len(events)),
		payloads:             make([][]byte, len(events)),
		metadatas:            make([][]byte, len(events)),
		traceContexts:        make([][]byte, len(events)),
		envelopeFingerprints: make([][]byte, len(events)),
		orderingKeys:         make([]string, len(events)),
		orderingSequences:    make([]int64, len(events)),
		traceContextDegraded: degraded,
	}
	for index, event := range events {
		columns.traceContexts[index] = traceContext
		event = event.withDefaults()
		fingerprint, err := commitReceiptFingerprint(event)
		if err != nil {
			return appendColumns{}, err
		}
		columns.ids[index] = event.ID
		columns.types[index] = event.Type
		columns.sources[index] = event.Source
		columns.destinations[index] = event.Destination
		columns.schemas[index] = event.Schema
		columns.occurredAt[index] = timestamptz(event.OccurredAt)
		columns.payloads[index] = event.Payload
		columns.metadatas[index] = event.Metadata
		columns.envelopeFingerprints[index] = fingerprint[:]
		columns.orderingKeys[index] = event.OrderingKey
		columns.orderingSequences[index] = event.OrderingSequence
		columns.ordered = columns.ordered || event.OrderingKey != ""
	}
	return columns, nil
}
