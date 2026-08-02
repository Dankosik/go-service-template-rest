package postgresoutbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestStoreRejectsInvalidUseBeforeDatabaseAccess(t *testing.T) {
	t.Parallel()

	if _, err := NewStore(nil, nil); !errors.Is(err, ErrConfig) {
		t.Fatalf("NewStore(nil) error = %v", err)
	}
	store := &Store{queries: sqlcgen.New(nil)}
	if err := store.Append(t.Context(), nil, outboxEventForUnit()); !errors.Is(err, ErrConfig) {
		t.Fatalf("Append(nil tx) error = %v", err)
	}
	if _, err := store.Claim(t.Context(), 0); !errors.Is(err, ErrConfig) {
		t.Fatalf("Claim(0) error = %v", err)
	}
	if err := store.MarkPublished(t.Context(), "", "lease"); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("MarkPublished(invalid id) error = %v", err)
	}
	if err := store.ScheduleRetry(t.Context(), "event", "", "temporary", time.Second); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("ScheduleRetry(invalid token) error = %v", err)
	}
	if err := store.ScheduleRetry(t.Context(), "event", "lease", "temporary", -1); !errors.Is(err, ErrConfig) {
		t.Fatalf("ScheduleRetry(negative) error = %v", err)
	}
	if err := store.MarkPoisoned(t.Context(), "event", "lease", ""); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("MarkPoisoned(invalid class) error = %v", err)
	}
	if _, err := store.Get(t.Context(), ""); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("Get(invalid id) error = %v", err)
	}
	if err := store.Redrive(t.Context(), "", "audit"); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("Redrive(invalid id) error = %v", err)
	}
	if err := store.Redrive(t.Context(), "event", "audit"); !errors.Is(err, postgres.ErrConfig) {
		t.Fatalf("Redrive(nil pool) error = %v", err)
	}
	if _, err := store.CleanupPublished(t.Context(), 0, 0); !errors.Is(err, ErrConfig) {
		t.Fatalf("CleanupPublished(invalid) error = %v", err)
	}
	if _, err := store.Observe(t.Context()); !errors.Is(err, postgres.ErrConfig) {
		t.Fatalf("Observe(nil pool) error = %v", err)
	}
}

func TestStoreRowConversionsAndHelpers(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 123).UTC()
	stamp := pgtype.Timestamptz{Time: now, Valid: true}
	key, token, class, redriveID := "key", "lease", "temporary", "audit"
	sequence := int64(7)
	row := sqlcgen.ClaimOutboxEventRow{
		ID: "event", EventType: "type", Source: "source", Destination: "destination", SchemaName: "v1",
		OccurredAt: stamp, Payload: []byte(`{"id":1}`), Metadata: []byte(`{"trace":"x"}`),
		OrderingKey: &key, OrderingSequence: &sequence, LeaseToken: &token, LeaseExpiresAt: stamp,
		CycleAttemptCount: 2, TotalAttemptCount: 4,
	}
	event := eventFromClaimRow(row)
	if event.ID != row.ID || event.OrderingKey != key || event.OrderingSequence != sequence || !event.OccurredAt.Equal(now) {
		t.Fatalf("eventFromClaimRow() = %+v", event)
	}
	event.Payload[0] = 'x'
	if row.Payload[0] == 'x' {
		t.Fatal("eventFromClaimRow aliased payload")
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
	record.Event.Metadata[0] = 'x'
	if row.Metadata[0] == 'x' {
		t.Fatal("recordFromRow aliased metadata")
	}

	if got := timestamptz(now); !got.Valid || !got.Time.Equal(now) {
		t.Fatalf("timestamptz() = %+v", got)
	}
	if !timeValue(pgtype.Timestamptz{}).IsZero() || !timeValue(stamp).Equal(now) {
		t.Fatal("timeValue conversion mismatch")
	}
	if !unixTime(0).IsZero() || unixTime(float64(now.UnixNano())/float64(time.Second)).Unix() != now.Unix() {
		t.Fatal("unixTime conversion mismatch")
	}
	if durationMilliseconds(1500*time.Microsecond) != 1.5 {
		t.Fatal("durationMilliseconds conversion mismatch")
	}
}

func TestStoreProgressAndTelemetryHelpers(t *testing.T) {
	t.Parallel()

	if err := validateProgressIdentity("event", "lease"); err != nil {
		t.Fatalf("validateProgressIdentity() error = %v", err)
	}
	if err := validateErrorClass("temporary"); err != nil {
		t.Fatalf("validateErrorClass() error = %v", err)
	}
	databaseErr := errors.New("database")
	if err := progressResult("mark", 1, nil); err != nil {
		t.Fatalf("progressResult(success) = %v", err)
	}
	if err := progressResult("mark", 0, nil); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("progressResult(lost) = %v", err)
	}
	if err := progressResult("mark", 0, databaseErr); !errors.Is(err, databaseErr) {
		t.Fatalf("progressResult(database) = %v", err)
	}

	telemetry := &Telemetry{}
	store := &Store{}
	if (*Store)(nil).withTelemetry(telemetry) != nil || store.withTelemetry(nil) != store {
		t.Fatal("withTelemetry nil behavior mismatch")
	}
	if clone := store.withTelemetry(telemetry); clone == store || clone.telemetry != telemetry {
		t.Fatal("withTelemetry did not create instrumented view")
	}
	store.telemetry = telemetry
	if store.withTelemetry(telemetry) != store {
		t.Fatal("withTelemetry duplicated the same telemetry")
	}
	(*Store)(nil).recordOperation(context.Background(), "claim", time.Now(), nil)
	recorder, err := NewTelemetry(nil, nil)
	if err != nil {
		t.Fatalf("NewTelemetry() error = %v", err)
	}
	t.Cleanup(recorder.Close)
	instrumented := &Store{telemetry: recorder}
	for _, operationErr := range []error{
		nil,
		ErrNoWork,
		ErrLeaseLost,
		ErrInvalidEvent,
		ErrConfig,
		ErrOrderingSequence,
		ErrRedriveRejected,
		ErrRedriveConflict,
	} {
		instrumented.recordOperation(t.Context(), "claim", time.Now(), operationErr)
	}
}

func TestStoreDatabaseResultClassification(t *testing.T) {
	t.Parallel()

	databaseErr := errors.New("database")
	store := &Store{queries: sqlcgen.New(databaseStub{rowErr: pgx.ErrNoRows})}
	if _, err := store.Claim(t.Context(), time.Second); !errors.Is(err, ErrNoWork) {
		t.Fatalf("Claim(no rows) error = %v", err)
	}
	if _, err := store.Get(t.Context(), "event"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(no rows) error = %v", err)
	}
	store.queries = sqlcgen.New(databaseStub{rowErr: databaseErr})
	if _, err := store.Claim(t.Context(), time.Second); !errors.Is(err, databaseErr) {
		t.Fatalf("Claim(database) error = %v", err)
	}
	if _, err := store.Get(t.Context(), "event"); !errors.Is(err, databaseErr) {
		t.Fatalf("Get(database) error = %v", err)
	}

	store.queries = sqlcgen.New(databaseStub{tag: pgconn.NewCommandTag("UPDATE 1")})
	if err := store.MarkPublished(t.Context(), "event", "lease"); err != nil {
		t.Fatalf("MarkPublished() error = %v", err)
	}
	if err := store.ScheduleRetry(t.Context(), "event", "lease", "temporary", time.Second); err != nil {
		t.Fatalf("ScheduleRetry() error = %v", err)
	}
	if err := store.MarkPoisoned(t.Context(), "event", "lease", "permanent"); err != nil {
		t.Fatalf("MarkPoisoned() error = %v", err)
	}
	store.queries = sqlcgen.New(databaseStub{tag: pgconn.NewCommandTag("UPDATE 0")})
	if err := store.MarkPublished(t.Context(), "event", "lease"); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("MarkPublished(lost) error = %v", err)
	}
	store.queries = sqlcgen.New(databaseStub{execErr: databaseErr})
	if err := store.MarkPoisoned(t.Context(), "event", "lease", "permanent"); !errors.Is(err, databaseErr) {
		t.Fatalf("MarkPoisoned(database) error = %v", err)
	}
	store.queries = sqlcgen.New(databaseStub{queryErr: databaseErr})
	if _, err := store.CleanupPublished(t.Context(), time.Hour, 10); !errors.Is(err, databaseErr) {
		t.Fatalf("CleanupPublished(database) error = %v", err)
	}
}

func TestStoreAppendValidationAndInsert(t *testing.T) {
	t.Parallel()

	store := &Store{pool: &postgres.Pool{}}
	tx := &transactionStub{tag: pgconn.NewCommandTag("INSERT 0 1")}
	invalid := outboxEventForUnit()
	invalid.ID = ""
	if err := store.Append(t.Context(), tx, invalid); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("Append(invalid) error = %v", err)
	}
	event := outboxEventForUnit()
	event.Type, event.Source, event.Destination, event.Schema = "type", "source", "destination", "v1"
	event.OccurredAt = time.Unix(1, 0).UTC()
	if err := store.Append(t.Context(), tx, event); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	tx.err = errors.New("insert")
	if err := store.Append(t.Context(), tx, event); !errors.Is(err, tx.err) {
		t.Fatalf("Append(database) error = %v", err)
	}
}

type databaseStub struct {
	tag      pgconn.CommandTag
	execErr  error
	queryErr error
	rowErr   error
}

func (stub databaseStub) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return stub.tag, stub.execErr
}

//nolint:ireturn // The pgx DBTX test double must return pgx's interface.
func (stub databaseStub) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, stub.queryErr
}

//nolint:ireturn // The pgx DBTX test double must return pgx's interface.
func (stub databaseStub) QueryRow(context.Context, string, ...any) pgx.Row {
	return rowStub{err: stub.rowErr}
}

type rowStub struct{ err error }

func (row rowStub) Scan(...any) error { return row.err }

type transactionStub struct {
	pgx.Tx

	tag pgconn.CommandTag
	err error
}

func (tx *transactionStub) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return tx.tag, tx.err
}
