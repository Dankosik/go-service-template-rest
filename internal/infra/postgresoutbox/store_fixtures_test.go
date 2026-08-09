package postgresoutbox

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// The driver doubles the store test files share. The store prefix on this file
// is load-bearing for the same reason it is on the production statements: the
// pgx and sqlcgen imports above are only allowed under .golangci.yml's
// store*.go exemption, so renaming this to fixtures_test.go would report those
// imports rather than the rename that caused them. See doc.go.

// stubbedStore builds a Store over a test driver. It sets both fields NewStore
// sets, because Store.valid admits nothing less — half a Store is exactly what
// that guard exists to reject, as the half cases below prove.
func stubbedStore(driver sqlcgen.DBTX) *Store {
	return &Store{pool: &postgres.Pool{}, queries: sqlcgen.New(driver)}
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
	sets [][]pgx.Row
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

// singleTextRow is a result row of one text column. It is named for that shape
// rather than for a meaning, because two statements return it and mean
// different things by it: a finalized id from MarkOutboxPublishedBatch, and a
// still-active ordering key from RetireOutboxOrderingKeys. The meaning stays at
// the call site.
type singleTextRow string

func (row singleTextRow) Scan(destinations ...any) error {
	value, ok := singleDestination[string](destinations)
	if !ok {
		return errors.New("unexpected single-text scan destinations")
	}
	*value = string(row)
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
