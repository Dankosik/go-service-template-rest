package postgresoutbox

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
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
	// Constructed enough to pass the entry-point guards, so each assertion below
	// reaches the argument validation it is actually about. A zero-value Store
	// stops earlier, which TestZeroValueStoreRejectsEveryExportedMethod covers.
	store := &Store{pool: &postgres.Pool{}, queries: sqlcgen.New(nil)}
	if err := store.Append(t.Context(), nil, outboxEventForUnit()); !errors.Is(err, ErrConfig) {
		t.Fatalf("Append(nil tx) error = %v", err)
	}
	if _, err := store.Claim(t.Context(), 0, 1); !errors.Is(err, ErrConfig) {
		t.Fatalf("Claim(0 lease) error = %v", err)
	}
	if _, err := store.Claim(t.Context(), time.Second, 0); !errors.Is(err, ErrConfig) {
		t.Fatalf("Claim(0 batch) error = %v", err)
	}
	if err := store.MarkUnorderedPublished(t.Context(), "lease", ""); !errors.Is(err, ErrConfig) {
		t.Fatalf("MarkUnorderedPublished(invalid id) error = %v", err)
	}
	if err := store.MarkOrderedPublished(t.Context(), "lease", OrderedDirective{
		ID: "event", OrderingKey: "key", OrderingSequence: 0,
	}); !errors.Is(err, ErrConfig) {
		t.Fatalf("MarkOrderedPublished(invalid sequence) error = %v", err)
	}
	if _, err := store.MarkPublishedBatch(t.Context(), "lease", nil); !errors.Is(err, ErrConfig) {
		t.Fatalf("MarkPublishedBatch(empty) error = %v", err)
	}
	if _, err := store.MarkOrderedPublishedBatch(t.Context(), "lease", nil); !errors.Is(err, ErrConfig) {
		t.Fatalf("MarkOrderedPublishedBatch(empty) error = %v", err)
	}
	if _, err := store.MarkOrderedPublishedBatch(t.Context(), "lease", []OrderedDirective{
		{ID: "event", OrderingKey: "key", OrderingSequence: 0},
	}); !errors.Is(err, ErrConfig) {
		t.Fatalf("MarkOrderedPublishedBatch(invalid sequence) error = %v", err)
	}
	if err := store.ScheduleRetryBatch(t.Context(), "", []RetryDirective{
		{ID: "event", ErrorClass: "temporary", Delay: time.Second},
	}); !errors.Is(err, ErrConfig) {
		t.Fatalf("ScheduleRetryBatch(invalid token) error = %v", err)
	}
	if err := store.ScheduleRetryBatch(t.Context(), "lease", []RetryDirective{
		{ID: "event", ErrorClass: "temporary", Delay: -1},
	}); !errors.Is(err, ErrConfig) {
		t.Fatalf("ScheduleRetryBatch(negative delay) error = %v", err)
	}
	if err := store.MarkPoisonedBatch(t.Context(), "lease", []PoisonDirective{
		{ID: "event", ErrorClass: ""},
	}); !errors.Is(err, ErrConfig) {
		t.Fatalf("MarkPoisonedBatch(invalid class) error = %v", err)
	}
	if _, err := store.Get(t.Context(), ""); !errors.Is(err, ErrConfig) {
		t.Fatalf("Get(invalid id) error = %v", err)
	}
	if err := store.Redrive(t.Context(), "", "audit"); !errors.Is(err, ErrConfig) {
		t.Fatalf("Redrive(invalid id) error = %v", err)
	}
	if err := store.Redrive(t.Context(), "event", "audit"); !errors.Is(err, postgres.ErrConfig) {
		t.Fatalf("Redrive(unusable pool) error = %v", err)
	}
	if _, err := store.CleanupPublished(t.Context(), 0, 0); !errors.Is(err, ErrConfig) {
		t.Fatalf("CleanupPublished(invalid) error = %v", err)
	}
}

// ErrInvalidEvent means one Event failed Validate and nothing else, so no
// identity check may report it. A store method that borrowed that sentinel for
// a bad id, lease token, or error class would tell a caller to inspect an
// envelope that is not at fault — and would do it through the same
// validateText rules Event.Validate uses, which is exactly how the two drifted
// together before.
func TestStoreIdentityFailuresAreNotEventFailures(t *testing.T) {
	t.Parallel()

	store := &Store{pool: &postgres.Pool{}, queries: sqlcgen.New(nil)}
	for name, call := range map[string]func() error{
		"MarkUnorderedPublished empty id": func() error {
			return store.MarkUnorderedPublished(t.Context(), "lease", "")
		},
		"MarkOrderedPublished empty id": func() error {
			return store.MarkOrderedPublished(t.Context(), "lease", OrderedDirective{
				OrderingKey: "key", OrderingSequence: 1,
			})
		},
		"MarkPublishedBatch empty token": func() error {
			_, err := store.MarkPublishedBatch(t.Context(), "", []string{"event"})
			return err
		},
		"MarkOrderedPublishedBatch empty key": func() error {
			_, err := store.MarkOrderedPublishedBatch(t.Context(), "lease", []OrderedDirective{
				{ID: "event", OrderingKey: "", OrderingSequence: 1},
			})
			return err
		},
		"ScheduleRetryBatch oversized class": func() error {
			return store.ScheduleRetryBatch(t.Context(), "lease", []RetryDirective{
				{ID: "event", ErrorClass: strings.Repeat("c", maxErrorClassBytes+1)},
			})
		},
		"MarkPoisonedBatch empty class": func() error {
			return store.MarkPoisonedBatch(t.Context(), "lease", []PoisonDirective{{ID: "event"}})
		},
		"Get empty id":     func() error { _, err := store.Get(t.Context(), ""); return err },
		"Redrive empty id": func() error { return store.Redrive(t.Context(), "", "audit") },
	} {
		err := call()
		if !errors.Is(err, ErrConfig) {
			t.Errorf("%s error = %v, want ErrConfig", name, err)
		}
		if errors.Is(err, ErrInvalidEvent) {
			t.Errorf("%s reported ErrInvalidEvent; that sentinel belongs to Event.Validate", name)
		}
	}

	// The other direction: a rejected envelope still reports ErrInvalidEvent and
	// never ErrConfig, so the split is readable from both sides.
	invalid := outboxEventForUnit()
	invalid.ID = ""
	err := store.Append(t.Context(), &transactionStub{}, invalid)
	if !errors.Is(err, ErrInvalidEvent) || errors.Is(err, ErrConfig) {
		t.Errorf("Append(invalid event) error = %v, want ErrInvalidEvent alone", err)
	}
}

// stubbedStore builds a Store over a test driver. It sets both fields NewStore
// sets, because Store.valid admits nothing less — half a Store is exactly what
// that guard exists to reject, as the half cases below prove.
func stubbedStore(driver sqlcgen.DBTX) *Store {
	return &Store{pool: &postgres.Pool{}, queries: sqlcgen.New(driver)}
}

// Store is exported, so a zero value can reach a caller that never used
// NewStore. Every exported method opens with Store.valid, so each one rejects
// such a store instead of dereferencing a nil dependency — including the two
// half-built shapes that would otherwise pass whichever single field that
// method happens to read.
//
// The list below is every exported method, not a chosen subset. A method with
// arguments valid enough to reach past its own checks — CleanupPublished is
// the one that used to have none — otherwise panics on a nil *Queries, and the
// count assertion at the end is what fails when a new method is added and left
// out of both the guard and this table.
func TestZeroValueStoreRejectsEveryExportedMethod(t *testing.T) {
	t.Parallel()

	calls := map[string]func(*Store) error{
		"Append": func(s *Store) error {
			return s.Append(t.Context(), &transactionStub{}, outboxEventForUnit())
		},
		"Claim":   func(s *Store) error { _, err := s.Claim(t.Context(), time.Second, 1); return err },
		"Get":     func(s *Store) error { _, err := s.Get(t.Context(), "event"); return err },
		"Observe": func(s *Store) error { _, err := s.Observe(t.Context()); return err },
		"Redrive": func(s *Store) error { return s.Redrive(t.Context(), "event", "audit") },
		"MarkUnorderedPublished": func(s *Store) error {
			return s.MarkUnorderedPublished(t.Context(), "lease", "event")
		},
		"MarkOrderedPublished": func(s *Store) error {
			return s.MarkOrderedPublished(t.Context(), "lease", unitOrdered()[0])
		},
		"MarkPublishedBatch": func(s *Store) error {
			_, err := s.MarkPublishedBatch(t.Context(), "lease", []string{"event"})
			return err
		},
		"MarkOrderedPublishedBatch": func(s *Store) error {
			_, err := s.MarkOrderedPublishedBatch(t.Context(), "lease", unitOrdered())
			return err
		},
		"ScheduleRetryBatch": func(s *Store) error {
			return s.ScheduleRetryBatch(t.Context(), "lease", unitRetries())
		},
		"MarkPoisonedBatch": func(s *Store) error {
			return s.MarkPoisonedBatch(t.Context(), "lease", unitPoisons())
		},
		"CleanupPublished": func(s *Store) error {
			_, err := s.CleanupPublished(t.Context(), time.Hour, 10)
			return err
		},
	}
	if got, want := len(calls), reflect.TypeFor[*Store]().NumMethod(); got != want {
		t.Fatalf("this table drives %d exported methods, Store has %d; add the new one here and give it a valid() guard", got, want)
	}
	for _, store := range map[string]*Store{
		"zero":         {},
		"pool only":    {pool: &postgres.Pool{}},
		"queries only": {queries: sqlcgen.New(databaseStub{})},
		"nil":          nil,
	} {
		for name, call := range calls {
			if err := call(store); !errors.Is(err, ErrConfig) {
				t.Errorf("%s(half-built store) error = %v, want ErrConfig", name, err)
			}
		}
	}
}

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
	if got := unixTime(0); !got.IsZero() {
		t.Errorf("unixTime(0) = %s, want the zero time", got)
	}
	seconds := float64(now.UnixNano()) / float64(time.Second)
	if got := unixTime(seconds); got.Unix() != now.Unix() {
		t.Errorf("unixTime(%f) = %s, want %s", seconds, got, now)
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
	if err := batchProgressResult("mark", 3, 3, nil); err != nil {
		t.Fatalf("batchProgressResult(success) = %v", err)
	}
	if err := batchProgressResult("mark", 2, 3, nil); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("batchProgressResult(short) = %v", err)
	}
	if err := batchProgressResult("mark", 0, 3, databaseErr); !errors.Is(err, databaseErr) {
		t.Fatalf("batchProgressResult(database) = %v", err)
	}

	telemetry := &Telemetry{}
	store := &Store{}
	if got := (*Store)(nil).withTelemetry(telemetry); got != nil {
		t.Errorf("(*Store)(nil).withTelemetry() = %p, want nil", got)
	}
	if got := store.withTelemetry(nil); got != store {
		t.Errorf("withTelemetry(nil) = %p, want the receiver %p unchanged", got, store)
	}
	if clone := store.withTelemetry(telemetry); clone == store || clone.telemetry != telemetry {
		t.Fatal("withTelemetry did not create instrumented view")
	}
	store.telemetry = telemetry
	if store.withTelemetry(telemetry) != store {
		t.Fatal("withTelemetry duplicated the same telemetry")
	}
	// Both recorders run from deferred calls that a zero-value Store can reach
	// before its entry-point guard returns, so the nil receiver is the claim
	// here. What each one records is asserted elsewhere: storeOutcome's
	// classification by TestErrorClassVocabularyIsBounded, and the empty-claim
	// outcome by TestTelemetryBoundedContract.
	(*Store)(nil).recordOperation(context.Background(), "claim", time.Now(), nil)
	(*Store)(nil).recordClaim(context.Background(), ClaimedBatch{}, time.Now(), nil)
}

// A missing row is an absent event rather than a database failure, so only Get
// collapses it into ErrNotFound.
func TestStoreMapsMissingRowToNotFound(t *testing.T) {
	t.Parallel()

	store := stubbedStore(databaseStub{rowErr: pgx.ErrNoRows})
	if _, err := store.Get(t.Context(), "event"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(no rows) error = %v, want ErrNotFound", err)
	}
}

// Every statement reachable through a stubbed driver wraps its failure rather
// than reporting it as lost lease, absent event, or silent success. Append
// takes its transaction from the caller and Redrive owns one, so those two are
// proven against their own doubles and against real PostgreSQL instead.
func TestStorePropagatesDatabaseFailures(t *testing.T) {
	t.Parallel()

	databaseErr := errors.New("database")
	reads := stubbedStore(databaseStub{rowErr: databaseErr, queryErr: databaseErr})
	if _, err := reads.Claim(t.Context(), time.Second, 10); !errors.Is(err, databaseErr) {
		t.Errorf("Claim(database) error = %v, want the driver failure", err)
	}
	if _, err := reads.Get(t.Context(), "event"); !errors.Is(err, databaseErr) {
		t.Errorf("Get(database) error = %v, want the driver failure", err)
	}
	if _, err := reads.Observe(t.Context()); !errors.Is(err, databaseErr) {
		t.Errorf("Observe(database) error = %v, want the driver failure", err)
	}

	writes := stubbedStore(databaseStub{execErr: databaseErr, queryErr: databaseErr})
	if err := writes.MarkPoisonedBatch(t.Context(), "lease", unitPoisons()); !errors.Is(err, databaseErr) {
		t.Errorf("MarkPoisonedBatch(database) error = %v, want the driver failure", err)
	}
	if _, err := writes.MarkPublishedBatch(t.Context(), "lease", []string{"event"}); !errors.Is(err, databaseErr) {
		t.Errorf("MarkPublishedBatch(database) error = %v, want the driver failure", err)
	}
	if _, err := writes.MarkOrderedPublishedBatch(t.Context(), "lease", unitOrdered()); !errors.Is(err, databaseErr) {
		t.Errorf("MarkOrderedPublishedBatch(database) error = %v, want the driver failure", err)
	}
	if _, err := writes.CleanupPublished(t.Context(), time.Hour, 10); !errors.Is(err, databaseErr) {
		t.Errorf("CleanupPublished(database) error = %v, want the driver failure", err)
	}
	if err := writes.ScheduleRetryBatch(t.Context(), "lease", unitRetries()); !errors.Is(err, databaseErr) {
		t.Errorf("ScheduleRetryBatch(database) error = %v, want the driver failure", err)
	}
	// MarkUnorderedPublished takes the single-event path, which reaches a
	// different statement from MarkPublishedBatch and wraps through
	// progressResult.
	if err := writes.MarkUnorderedPublished(t.Context(), "lease", "event"); !errors.Is(err, databaseErr) {
		t.Errorf("MarkUnorderedPublished(database) error = %v, want the driver failure", err)
	}
	if err := writes.MarkOrderedPublished(t.Context(), "lease", unitOrdered()[0]); !errors.Is(err, databaseErr) {
		t.Errorf("MarkOrderedPublished(database) error = %v, want the driver failure", err)
	}
}

// An affected row is the proof the lease still owned the event, so each
// finalization reports success and its own count.
func TestStoreFinalizationSucceedsWhenTheLeaseStillOwnsTheRow(t *testing.T) {
	t.Parallel()

	// One Store per assertion. A shared querySequence is consumed positionally by
	// whichever finalization reaches Query first, so a sixth assertion added above
	// a fifth would silently take the fifth's result set.
	affected := func() databaseStub {
		return databaseStub{
			tag:     pgconn.NewCommandTag("UPDATE 1"),
			queries: &querySequence{sets: [][]pgx.Row{{publishedIDRow("event")}}},
		}
	}
	if err := stubbedStore(affected()).MarkUnorderedPublished(t.Context(), "lease", "event"); err != nil {
		t.Errorf("MarkUnorderedPublished() error = %v", err)
	}
	marked, err := stubbedStore(affected()).MarkPublishedBatch(t.Context(), "lease", []string{"event"})
	if err != nil || len(marked) != 1 || marked[0] != "event" {
		t.Errorf("MarkPublishedBatch() = %v, %v, want the finalized id", marked, err)
	}
	if err := stubbedStore(affected()).ScheduleRetryBatch(t.Context(), "lease", unitRetries()); err != nil {
		t.Errorf("ScheduleRetryBatch() error = %v", err)
	}
	if err := stubbedStore(affected()).MarkPoisonedBatch(t.Context(), "lease", unitPoisons()); err != nil {
		t.Errorf("MarkPoisonedBatch() error = %v", err)
	}
	if deleted, err := stubbedStore(affected()).CleanupPublished(t.Context(), time.Hour, 10); err != nil || deleted != 1 {
		t.Errorf("CleanupPublished() = %d, %v, want 1 deleted", deleted, err)
	}
}

// No affected row means another relay recovered the lease, which every
// single-statement finalization reports as ErrLeaseLost rather than success.
func TestStoreReportsLostLeaseWhenNoRowMatches(t *testing.T) {
	t.Parallel()

	store := stubbedStore(databaseStub{tag: pgconn.NewCommandTag("UPDATE 0")})
	if err := store.MarkUnorderedPublished(t.Context(), "lease", "event"); !errors.Is(err, ErrLeaseLost) {
		t.Errorf("MarkUnorderedPublished(lost) error = %v, want ErrLeaseLost", err)
	}
	if err := store.ScheduleRetryBatch(t.Context(), "lease", unitRetries()); !errors.Is(err, ErrLeaseLost) {
		t.Errorf("ScheduleRetryBatch(lost) error = %v, want ErrLeaseLost", err)
	}
	if err := store.MarkPoisonedBatch(t.Context(), "lease", unitPoisons()); !errors.Is(err, ErrLeaseLost) {
		t.Errorf("MarkPoisonedBatch(lost) error = %v, want ErrLeaseLost", err)
	}
}

// A snapshot conflict resends the conflicted directive and nothing else, and the
// second statement's fresh snapshot resolves it.
//
// The first statement reports its two rows in the opposite order to the input.
// Result rows carry no guaranteed order, which is why conflictedDirectives
// matches on id; resending by position would carry the wrong key here and mark
// an event whose successor is still invisible.
func TestStoreResendsOnlyTheSnapshotConflictedDirective(t *testing.T) {
	t.Parallel()

	sequence := &querySequence{sets: [][]pgx.Row{
		{
			orderedMarkRow{id: "blocked", snapshotConflict: true},
			orderedMarkRow{id: "clear", marked: true},
		},
		{orderedMarkRow{id: "blocked", marked: true}},
	}}
	store := stubbedStore(databaseStub{queries: sequence})

	marked, err := store.MarkOrderedPublishedBatch(t.Context(), "lease", []OrderedDirective{
		{ID: "clear", OrderingKey: "key-a", OrderingSequence: 1},
		{ID: "blocked", OrderingKey: "key-b", OrderingSequence: 1},
	})
	if err != nil {
		t.Fatalf("MarkOrderedPublishedBatch() error = %v", err)
	}
	if len(sequence.sent) != orderedPublishSnapshots {
		t.Fatalf("ordered mark statements = %d, want %d", len(sequence.sent), orderedPublishSnapshots)
	}
	if resent := sentIDs(t, sequence, 1); len(resent) != 1 || resent[0] != "blocked" {
		t.Fatalf("resent ids = %v, want only the conflicted directive", resent)
	}
	if len(marked) != 2 {
		t.Fatalf("marked = %v, want both events finalized across the two statements", marked)
	}
}

// A single ordered claim routes through the same retry, which is what makes
// reconciliation cost orderedPublishSnapshots statements per pass rather than
// one — the second factor in the worst case ErrProgressUnknown derives. The
// product is deliberately not written down here; it moves when either constant
// does.
func TestStoreRetriesOrderedSnapshotConflictForOneClaim(t *testing.T) {
	t.Parallel()

	sequence := &querySequence{sets: [][]pgx.Row{
		{orderedMarkRow{id: "event", snapshotConflict: true}},
		{orderedMarkRow{id: "event", marked: true}},
	}}
	store := stubbedStore(databaseStub{queries: sequence})
	if err := store.MarkOrderedPublished(t.Context(), "lease", unitOrdered()[0]); err != nil {
		t.Fatalf("MarkOrderedPublished(snapshot retry) error = %v", err)
	}
	if len(sequence.sent) != orderedPublishSnapshots {
		t.Fatalf("ordered mark statements = %d, want %d", len(sequence.sent), orderedPublishSnapshots)
	}
}

// An empty result set means the ordered statement finalized nothing. The
// single-event path reports that as a lost lease; the batch path reports an
// empty marked list and leaves the resolution to its caller.
func TestStoreReportsOrderedFinalizationThatMarkedNothing(t *testing.T) {
	t.Parallel()

	store := stubbedStore(databaseStub{queries: &querySequence{}})
	if err := store.MarkOrderedPublished(t.Context(), "lease", unitOrdered()[0]); !errors.Is(err, ErrLeaseLost) {
		t.Errorf("MarkOrderedPublished(lost) error = %v, want ErrLeaseLost", err)
	}
	marked, err := store.MarkOrderedPublishedBatch(t.Context(), "lease", unitOrdered())
	if err != nil || len(marked) != 0 {
		t.Errorf("MarkOrderedPublishedBatch(lost) = %v, %v, want no ids and no error", marked, err)
	}
}

func unitRetries() []RetryDirective {
	return []RetryDirective{{ID: "event", ErrorClass: "temporary", Delay: time.Second}}
}

func unitPoisons() []PoisonDirective {
	return []PoisonDirective{{ID: "event", ErrorClass: "permanent"}}
}

func unitOrdered() []OrderedDirective {
	return []OrderedDirective{{ID: "event", OrderingKey: "key", OrderingSequence: 1}}
}

func TestStoreAppendValidationAndInsert(t *testing.T) {
	t.Parallel()

	store := stubbedStore(databaseStub{})
	tx := &transactionStub{}
	if err := store.Append(t.Context(), tx); err != nil {
		t.Fatalf("Append(no events) error = %v", err)
	}
	event := outboxEventForUnit()
	event.Type, event.Source, event.Destination, event.Schema = "type", "source", "destination", "v1"
	event.OccurredAt = time.Unix(1, 0).UTC()
	// Derived from the valid event, so clearing the id is what makes it invalid.
	// Built the other way round it would be rejected for the fields
	// outboxEventForUnit never sets, and the assertion would hold either way.
	invalid := event
	invalid.ID = ""
	if err := store.Append(t.Context(), tx, invalid); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("Append(invalid) error = %v", err)
	}
	// One invalid event keeps the whole call off the wire, so a caller never
	// commits part of what it asked to append.
	if err := store.Append(t.Context(), tx, event, invalid); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("Append(partly invalid) error = %v", err)
	}
	if tx.statements != 0 {
		t.Fatalf("rejected append sent %d statements, want 0", tx.statements)
	}

	// A call with no ordering key takes the statement that never touches a
	// head, which is the one that carries no ordering columns.
	second := event
	second.ID = "second"
	if err := store.Append(t.Context(), tx, event, second); err != nil {
		t.Fatalf("Append(unordered) error = %v", err)
	}
	if tx.statements != 1 || len(tx.arguments) != 8 {
		t.Fatalf("Append(unordered) sent %d statements with %d column arrays, want 1 and 8",
			tx.statements, len(tx.arguments))
	}

	// Ordered and unordered events travel together, so a mixed call is still
	// one statement however many events it carries.
	tx.statements = 0
	ordered := event
	ordered.ID, ordered.OrderingKey, ordered.OrderingSequence = "ordered", "key", 1
	if err := store.Append(t.Context(), tx, event, second, ordered); err != nil {
		t.Fatalf("Append(mixed) error = %v", err)
	}
	if tx.statements != 1 || len(tx.arguments) != 10 {
		t.Fatalf("Append(mixed) sent %d statements with %d column arrays, want 1 and 10",
			tx.statements, len(tx.arguments))
	}
	if keys, ok := tx.arguments[8].([]string); !ok || len(keys) != 3 ||
		keys[0] != "" || keys[1] != "" || keys[2] != "key" {
		t.Fatalf("Append(mixed) ordering keys = %v", tx.arguments[8])
	}

	// A returned row is a key whose first sequence did not clear its retained
	// high-water mark, and the message names that key rather than the call.
	tx.rejected = []pgx.Row{orderingRejectionRow{key: "key", sequence: 1}}
	err := store.Append(t.Context(), tx, ordered)
	rejection := fmt.Sprintf("%v", err)
	if !errors.Is(err, ErrOrderingSequence) || !strings.Contains(rejection, `key "key" sequence 1`) {
		t.Fatalf("Append(rejected sequence) error = %v", err)
	}
	tx.rejected = nil
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
	queries  *querySequence
}

func (stub databaseStub) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return stub.tag, stub.execErr
}

//nolint:ireturn // The pgx DBTX test double must return pgx's interface.
func (stub databaseStub) Query(_ context.Context, _ string, arguments ...any) (pgx.Rows, error) {
	if stub.queryErr != nil {
		return nil, stub.queryErr
	}
	return stub.queries.next(arguments), nil
}

//nolint:ireturn // The pgx DBTX test double must return pgx's interface.
func (stub databaseStub) QueryRow(context.Context, string, ...any) pgx.Row {
	return rowStub{err: stub.rowErr}
}

type rowStub struct{ err error }

func (row rowStub) Scan(...any) error { return row.err }

// querySequence replays one result set per Query call and records what each call
// was sent, so a test can drive the batch statements that report an outcome per
// event and then assert what a follow-up statement carried. Calls past the end
// see an empty result set, which is what a wholly lost lease looks like.
type querySequence struct {
	sets []([]pgx.Row)
	// sent is one entry per statement, holding that statement's bind arguments.
	// Its length is how many statements ran.
	sent [][]any
}

//nolint:ireturn // The pgx DBTX test double must return pgx's interface.
func (sequence *querySequence) next(arguments []any) pgx.Rows {
	if sequence == nil {
		return &rowsStub{}
	}
	sequence.sent = append(sequence.sent, arguments)
	if len(sequence.sent) > len(sequence.sets) {
		return &rowsStub{}
	}
	return &rowsStub{rows: sequence.sets[len(sequence.sent)-1]}
}

// sentIDs is the id array one recorded statement carried. Every batch statement
// in this package binds its ids first — see the generated
// MarkOrderedOutboxPublishedBatch — so a change to that order fails here rather
// than silently asserting on ordering keys.
func sentIDs(tb testing.TB, sequence *querySequence, statement int) []string {
	tb.Helper()
	if statement >= len(sequence.sent) {
		tb.Fatalf("statement %d never ran; %d did", statement, len(sequence.sent))
	}
	ids, ok := sequence.sent[statement][0].([]string)
	if !ok {
		tb.Fatalf("statement %d bound %T first, want the id array", statement, sequence.sent[statement][0])
	}
	return ids
}

type rowsStub struct {
	pgx.Rows

	rows  []pgx.Row
	index int
}

func (rows *rowsStub) Close() {}

func (rows *rowsStub) Err() error { return nil }

func (rows *rowsStub) Next() bool {
	if rows.index >= len(rows.rows) {
		return false
	}
	rows.index++
	return true
}

func (rows *rowsStub) Scan(destinations ...any) error {
	if err := rows.rows[rows.index-1].Scan(destinations...); err != nil {
		return fmt.Errorf("scan stub row %d: %w", rows.index-1, err)
	}
	return nil
}

// publishedIDRow is one finalized id returned by MarkOutboxPublishedBatch.
type publishedIDRow string

func (row publishedIDRow) Scan(destinations ...any) error {
	id, ok := singleDestination[string](destinations)
	if !ok {
		return errors.New("unexpected published id scan destinations")
	}
	*id = string(row)
	return nil
}

type orderedMarkRow struct {
	id               string
	marked           bool
	snapshotConflict bool
}

func (row orderedMarkRow) Scan(destinations ...any) error {
	if len(destinations) != 3 {
		return errors.New("unexpected ordered mark scan destinations")
	}
	id, idOK := destinations[0].(*string)
	marked, markedOK := destinations[1].(*bool)
	snapshotConflict, snapshotOK := destinations[2].(*bool)
	if !idOK || id == nil || !markedOK || marked == nil || !snapshotOK || snapshotConflict == nil {
		return errors.New("unexpected ordered mark scan destinations")
	}
	*id, *marked, *snapshotConflict = row.id, row.marked, row.snapshotConflict
	return nil
}

func singleDestination[T any](destinations []any) (*T, bool) {
	if len(destinations) != 1 {
		return nil, false
	}
	value, ok := destinations[0].(*T)
	return value, ok && value != nil
}

// transactionStub records what the append statement was sent, so a test can
// assert both the statement count and the column arrays it carried.
type transactionStub struct {
	pgx.Tx

	err        error
	rejected   []pgx.Row
	statements int
	arguments  []any
}

func (tx *transactionStub) Exec(_ context.Context, _ string, arguments ...any) (pgconn.CommandTag, error) {
	tx.record(arguments)
	return pgconn.CommandTag{}, tx.err
}

//nolint:ireturn // The pgx transaction test double must return pgx's interface.
func (tx *transactionStub) Query(_ context.Context, _ string, arguments ...any) (pgx.Rows, error) {
	tx.record(arguments)
	if tx.err != nil {
		return nil, tx.err
	}
	return &rowsStub{rows: tx.rejected}, nil
}

func (tx *transactionStub) record(arguments []any) {
	tx.statements++
	tx.arguments = arguments
}

// orderingRejectionRow is one key the append statement refused, which is the
// only row shape that statement ever returns.
type orderingRejectionRow struct {
	key      string
	sequence int64
}

func (row orderingRejectionRow) Scan(destinations ...any) error {
	if len(destinations) != 2 {
		return errors.New("unexpected ordering rejection scan destinations")
	}
	key, keyOK := destinations[0].(*string)
	sequence, sequenceOK := destinations[1].(*int64)
	if !keyOK || key == nil || !sequenceOK || sequence == nil {
		return errors.New("unexpected ordering rejection scan destinations")
	}
	*key, *sequence = row.key, row.sequence
	return nil
}
