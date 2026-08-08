package postgresoutbox

import (
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5/pgtype"
)

// envelopeColumns is the set of columns every outbox row carries. It is a
// struct rather than a parameter list because five of them are adjacent
// strings: transposing source and destination at a call site would compile, and
// only a test that asserts on those two exact values would notice.
type envelopeColumns struct {
	ID               string
	Type             string
	Source           string
	Destination      string
	Schema           string
	OccurredAt       pgtype.Timestamptz
	Payload          []byte
	Metadata         []byte
	TraceContext     []byte
	OrderingKey      *string
	OrderingSequence *int64
}

// storedEvent maps those columns into an Event. The claim projection and the
// full row are separate generated types with the same columns, so this is the
// one place that knows the mapping: an added Event field costs a field here and
// one line at each call site instead of a second transposition block that can
// silently drift from the first.
//
// It adopts the scanned payload and metadata rather than copying them. pgx
// allocates a fresh slice per row for a bytea scanned into []byte ([pgtype
// bytea scan]), so these bytes are already private to this event and a second
// copy would double the per-event allocation of every claimed batch.
//
// [pgtype bytea scan]: https://pkg.go.dev/github.com/jackc/pgx/v5/pgtype#ByteaCodec
func storedEvent(columns envelopeColumns) Event {
	event := Event{
		ID:          columns.ID,
		Type:        columns.Type,
		Source:      columns.Source,
		Destination: columns.Destination,
		Schema:      columns.Schema,
		OccurredAt:  timeValue(columns.OccurredAt),
		Payload:     columns.Payload,
		Metadata:    columns.Metadata,
		// Decoded rather than adopted like the two byte slices above: the carrier
		// is a map an adapter reads by key, and the ordinary stored value is the
		// empty object, which decodes to no map at all.
		traceContext: creationContextFromStored(columns.TraceContext),
	}
	if columns.OrderingKey != nil {
		event.OrderingKey = *columns.OrderingKey
	}
	if columns.OrderingSequence != nil {
		event.OrderingSequence = *columns.OrderingSequence
	}
	return event
}

func eventFromClaimRow(row sqlcgen.ClaimOutboxEventsRow) Event {
	return storedEvent(envelopeColumns{
		ID: row.ID, Type: row.EventType, Source: row.Source, Destination: row.Destination,
		Schema: row.SchemaName, OccurredAt: row.OccurredAt, Payload: row.Payload,
		Metadata: row.Metadata, TraceContext: row.TraceContext,
		OrderingKey: row.OrderingKey, OrderingSequence: row.OrderingSequence,
	})
}

func recordFromRow(row sqlcgen.OutboxEvent) Record {
	record := Record{
		Event: storedEvent(envelopeColumns{
			ID: row.ID, Type: row.EventType, Source: row.Source, Destination: row.Destination,
			Schema: row.SchemaName, OccurredAt: row.OccurredAt, Payload: row.Payload,
			Metadata: row.Metadata, TraceContext: row.TraceContext,
			OrderingKey: row.OrderingKey, OrderingSequence: row.OrderingSequence,
		}),
		CreatedAt:            timeValue(row.CreatedAt),
		AvailableAt:          timeValue(row.AvailableAt),
		CycleAttemptCount:    int(row.CycleAttemptCount),
		TotalAttemptCount:    row.TotalAttemptCount,
		LastAttemptAt:        timeValue(row.LastAttemptAt),
		LeaseExpiresAt:       timeValue(row.LeaseExpiresAt),
		PublishedAt:          timeValue(row.PublishedAt),
		PoisonedAt:           timeValue(row.PoisonedAt),
		PublicationUncertain: row.PublicationUncertain != nil && *row.PublicationUncertain,
		RedriveCount:         int(row.RedriveCount),
		LastRedrivenAt:       timeValue(row.LastRedrivenAt),
	}
	if row.LeaseToken != nil {
		record.LeaseToken = *row.LeaseToken
	}
	if row.LastErrorClass != nil {
		record.LastErrorClass = *row.LastErrorClass
	}
	if row.LastRedriveID != nil {
		record.LastRedriveID = *row.LastRedriveID
	}
	return record
}

// The identity checks below report [ErrConfig], not [ErrInvalidEvent]: an
// empty id, lease token, or error class is a fault in the call rather than in
// an envelope, and it is the caller — the relay or an operator tool — that has
// to fix it. validateText owns the shared rules; see its comment for the split.
func validateProgressIdentity(id, token string) error {
	if err := validateText(ErrConfig, "id", id, maxTextBytes); err != nil {
		return err
	}
	if err := validateText(ErrConfig, "lease_token", token, maxTextBytes); err != nil {
		return err
	}
	return nil
}

func validateBatchIdentity(token string, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("%w: batch requires at least one event", ErrConfig)
	}
	for _, id := range ids {
		if err := validateProgressIdentity(id, token); err != nil {
			return err
		}
	}
	return nil
}

// maxErrorClassBytes bounds the stored error class. It is far below the general
// text limit because the class is a metric label as well as a column, and its
// values come from this package's own closed set.
const maxErrorClassBytes = 64

func validateErrorClass(value string) error {
	return validateText(ErrConfig, "error_class", value, maxErrorClassBytes)
}

func timestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func timeValue(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time.UTC()
}

// timeFromUnixSeconds reads the fractional epoch seconds the observation
// statement reports. Its inverse, unixSecondsFromTime, is in telemetry.go,
// which puts the same values back on a gauge.
func timeFromUnixSeconds(value float64) time.Time {
	if value == 0 {
		return time.Time{}
	}
	return time.Unix(0, int64(value*float64(time.Second))).UTC()
}

func durationMilliseconds(value time.Duration) float64 {
	return float64(value) / float64(time.Millisecond)
}
