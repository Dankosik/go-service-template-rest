package postgresoutbox

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

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
	marked, err := stubbedStore(affected()).MarkUnorderedPublishedBatch(t.Context(), "lease", []string{"event"})
	if err != nil || len(marked) != 1 || marked[0] != "event" {
		t.Errorf("MarkUnorderedPublishedBatch() = %v, %v, want the finalized id", marked, err)
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
