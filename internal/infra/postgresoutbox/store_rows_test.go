package postgresoutbox

import (
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestStoreRowConversionsAndHelpers(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 123).UTC()
	stamp := pgtype.Timestamptz{Time: now, Valid: true}
	key, token, class, redriveID := "key", "lease", "temporary", "audit"
	sequence := int64(7)
	row := sqlcgen.ClaimOutboxEventsRow{
		ID: "event", EventType: "type", Source: "source", Destination: "destination", SchemaName: "v1",
		OccurredAt: stamp, Payload: []byte(`{"id":1}`), Metadata: []byte(`{"trace":"x"}`),
		OrderingKey: &key, OrderingSequence: &sequence,
		CycleAttemptCount: 2, TotalAttemptCount: 4,
	}
	event := eventFromClaimRow(row)
	if event.ID != row.ID || event.OrderingKey != key || event.OrderingSequence != sequence || !event.OccurredAt.Equal(now) {
		t.Fatalf("eventFromClaimRow() = %+v", event)
	}
	// The scanned bytes are already private to this row, so the event adopts
	// them instead of paying a second copy per claimed event.
	if &event.Payload[0] != &row.Payload[0] || &event.Metadata[0] != &row.Metadata[0] {
		t.Fatal("eventFromClaimRow copied payload or metadata it already owns")
	}

	record := recordFromRow(sqlcgen.OutboxEvent{
		ID: row.ID, EventType: row.EventType, Source: row.Source, Destination: row.Destination, SchemaName: row.SchemaName,
		OccurredAt: stamp, Payload: row.Payload, Metadata: row.Metadata, OrderingKey: &key, OrderingSequence: &sequence,
		CreatedAt: stamp, AvailableAt: stamp, CycleAttemptCount: 2, TotalAttemptCount: 4, LastAttemptAt: stamp,
		LeaseToken: &token, LeaseExpiresAt: stamp, PublishedAt: stamp, PoisonedAt: stamp,
		LastErrorClass: &class, RedriveCount: 3, LastRedriveID: &redriveID, LastRedrivenAt: stamp,
	})
	if record.LeaseToken != token || record.LastErrorClass != class || record.LastRedriveID != redriveID ||
		record.RedriveCount != 3 || !record.PublishedAt.Equal(now) {
		t.Fatalf("recordFromRow() = %+v", record)
	}
	if &record.Event.Metadata[0] != &row.Metadata[0] {
		t.Fatal("recordFromRow copied metadata it already owns")
	}

	if got := timestamptz(now); !got.Valid || !got.Time.Equal(now) {
		t.Fatalf("timestamptz() = %+v", got)
	}
	if got := timeValue(pgtype.Timestamptz{}); !got.IsZero() {
		t.Errorf("timeValue(invalid) = %s, want the zero time", got)
	}
	if got := timeValue(stamp); !got.Equal(now) {
		t.Errorf("timeValue(%s) = %s, want %s", stamp.Time, got, now)
	}
	if got := timeFromUnixSeconds(0); !got.IsZero() {
		t.Errorf("timeFromUnixSeconds(0) = %s, want the zero time", got)
	}
	seconds := float64(now.UnixNano()) / float64(time.Second)
	if got := timeFromUnixSeconds(seconds); got.Unix() != now.Unix() {
		t.Errorf("timeFromUnixSeconds(%f) = %s, want %s", seconds, got, now)
	}
	if durationMilliseconds(1500*time.Microsecond) != 1.5 {
		t.Fatal("durationMilliseconds conversion mismatch")
	}
}
