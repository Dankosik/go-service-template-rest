package postgresoutbox

import (
	"context"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

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

	columns, err := newAppendColumns(events)
	if err != nil {
		return err
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
	ids               []string
	types             []string
	sources           []string
	destinations      []string
	schemas           []string
	occurredAt        []pgtype.Timestamptz
	payloads          [][]byte
	metadatas         [][]byte
	orderingKeys      []string
	orderingSequences []int64
	ordered           bool
}

func (c appendColumns) withoutOrdering() sqlcgen.InsertOutboxEventsParams {
	return sqlcgen.InsertOutboxEventsParams{
		Ids: c.ids, EventTypes: c.types, Sources: c.sources, Destinations: c.destinations,
		SchemaNames: c.schemas, OccurredAts: c.occurredAt, Payloads: c.payloads, Metadatas: c.metadatas,
	}
}

func (c appendColumns) withOrdering() sqlcgen.InsertOutboxEventsWithOrderingParams {
	return sqlcgen.InsertOutboxEventsWithOrderingParams{
		Ids: c.ids, EventTypes: c.types, Sources: c.sources, Destinations: c.destinations,
		SchemaNames: c.schemas, OccurredAts: c.occurredAt, Payloads: c.payloads, Metadatas: c.metadatas,
		OrderingKeys: c.orderingKeys, OrderingSequences: c.orderingSequences,
	}
}

// newAppendColumns validates every event and transposes the call into those
// arrays. An event that owns no ordering head keeps the zero key and sequence
// it already has in Go, which is what the ordered statement reads as an absent
// head; a call where every event is like that skips that statement entirely.
func newAppendColumns(events []Event) (appendColumns, error) {
	columns := appendColumns{
		ids:               make([]string, len(events)),
		types:             make([]string, len(events)),
		sources:           make([]string, len(events)),
		destinations:      make([]string, len(events)),
		schemas:           make([]string, len(events)),
		occurredAt:        make([]pgtype.Timestamptz, len(events)),
		payloads:          make([][]byte, len(events)),
		metadatas:         make([][]byte, len(events)),
		orderingKeys:      make([]string, len(events)),
		orderingSequences: make([]int64, len(events)),
	}
	for index, event := range events {
		event = event.withDefaults()
		if err := event.Validate(); err != nil {
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
		columns.orderingKeys[index] = event.OrderingKey
		columns.orderingSequences[index] = event.OrderingSequence
		columns.ordered = columns.ordered || event.OrderingKey != ""
	}
	return columns, nil
}
