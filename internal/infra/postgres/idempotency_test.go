package postgres

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/idempotency"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// The classification below the SQL is what these tests cover. The statements
// themselves — and the atomicity of the claim, which is the property that matters
// and that no fake can prove — are covered by the testcontainers integration test
// in test/postgres_idempotency_integration_test.go.

type fakeRow struct {
	values []any
	err    error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return fmt.Errorf("fake row: scan arity mismatch: %d destinations, %d values", len(dest), len(r.values))
	}
	for i, target := range dest {
		value := r.values[i]
		switch typed := target.(type) {
		case *string:
			text, ok := value.(string)
			if !ok {
				return fmt.Errorf("fake row: value %d is %T, want string", i, value)
			}
			*typed = text
		case **int32:
			number, ok := value.(*int32)
			if !ok {
				return fmt.Errorf("fake row: value %d is %T, want *int32", i, value)
			}
			*typed = number
		case *[]byte:
			raw, ok := value.([]byte)
			if !ok {
				return fmt.Errorf("fake row: value %d is %T, want []byte", i, value)
			}
			*typed = raw
		case *bool:
			flag, ok := value.(bool)
			if !ok {
				return fmt.Errorf("fake row: value %d is %T, want bool", i, value)
			}
			*typed = flag
		default:
			return fmt.Errorf("fake row: unsupported scan target %T", target)
		}
	}
	return nil
}

type fakeQuerier struct {
	execTag   pgconn.CommandTag
	execErr   error
	row       pgx.Row
	execCalls []string
}

func (q *fakeQuerier) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	q.execCalls = append(q.execCalls, sql)
	return q.execTag, q.execErr
}

func (q *fakeQuerier) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("fake querier: Query is not used by the idempotency store")
}

func (q *fakeQuerier) QueryRow(context.Context, string, ...any) pgx.Row {
	return q.row
}

func newFakeStore(querier *fakeQuerier) *IdempotencyStore {
	return &IdempotencyStore{
		acquire:   func(context.Context) (Querier, func(), error) { return querier, func() {}, nil },
		retention: time.Hour,
	}
}

func TestNewIdempotencyStoreRejectsMissingDependencies(t *testing.T) {
	t.Parallel()

	if _, err := NewIdempotencyStore(nil, time.Hour); !errors.Is(err, ErrConfig) {
		t.Fatalf("NewIdempotencyStore(nil pool) error = %v, want %v", err, ErrConfig)
	}
	if _, err := NewIdempotencyStore(&Pool{}, time.Hour); !errors.Is(err, ErrConfig) {
		t.Fatalf("NewIdempotencyStore(zero pool) error = %v, want %v", err, ErrConfig)
	}
}

func TestReserveOwnsKeyWhenInsertWins(t *testing.T) {
	t.Parallel()

	querier := &fakeQuerier{execTag: pgconn.NewCommandTag("INSERT 0 1")}
	stored, err := newFakeStore(querier).Reserve(context.Background(), "key", "fingerprint")
	if err != nil {
		t.Fatalf("Reserve() error = %v, want nil", err)
	}
	if stored != nil {
		t.Fatalf("Reserve() = %+v, want nil for the attempt that owns the key", stored)
	}
}

func TestReserveReportsInFlightWhileFirstAttemptRuns(t *testing.T) {
	t.Parallel()

	querier := &fakeQuerier{
		execTag: pgconn.NewCommandTag("INSERT 0 0"),
		row:     fakeRow{values: []any{"fingerprint", (*int32)(nil), []byte(nil), []byte(nil), true, false}},
	}
	_, err := newFakeStore(querier).Reserve(context.Background(), "key", "fingerprint")
	if !errors.Is(err, idempotency.ErrInFlight) {
		t.Fatalf("Reserve() error = %v, want %v", err, idempotency.ErrInFlight)
	}
}

// TestReserveReportsKeyReusedOnFingerprintMismatch covers the one case where replay
// would be dangerous: the same key presented for different work.
func TestReserveReportsKeyReusedOnFingerprintMismatch(t *testing.T) {
	t.Parallel()

	querier := &fakeQuerier{
		execTag: pgconn.NewCommandTag("INSERT 0 0"),
		row:     fakeRow{values: []any{"other-fingerprint", (*int32)(nil), []byte(nil), []byte(nil), true, false}},
	}
	_, err := newFakeStore(querier).Reserve(context.Background(), "key", "fingerprint")
	if !errors.Is(err, idempotency.ErrKeyReused) {
		t.Fatalf("Reserve() error = %v, want %v", err, idempotency.ErrKeyReused)
	}
}

func TestReserveReplaysCompletedResponse(t *testing.T) {
	t.Parallel()

	status := int32(http.StatusCreated)
	querier := &fakeQuerier{
		execTag: pgconn.NewCommandTag("INSERT 0 0"),
		row: fakeRow{values: []any{
			"fingerprint",
			&status,
			[]byte(`{"Location":["/api/v1/articles/a"]}`),
			[]byte(`{"slug":"a"}`),
			true,
			true,
		}},
	}

	stored, err := newFakeStore(querier).Reserve(context.Background(), "key", "fingerprint")
	if err != nil {
		t.Fatalf("Reserve() error = %v, want nil", err)
	}
	if stored == nil {
		t.Fatal("Reserve() = nil, want the completed response")
	}
	if stored.Status != http.StatusCreated {
		t.Fatalf("replayed status = %d, want %d", stored.Status, http.StatusCreated)
	}
	if got := stored.Header.Get("Location"); got != "/api/v1/articles/a" {
		t.Fatalf("replayed Location = %q, want %q", got, "/api/v1/articles/a")
	}
	if string(stored.Body) != `{"slug":"a"}` {
		t.Fatalf("replayed body = %q", stored.Body)
	}
	if !stored.Replayable {
		t.Fatal("replayed response is marked unreplayable")
	}
}

// TestReserveReportsSpentKeyForUnreplayableResponse keeps the safety property when
// the exact bytes are gone: the key is still spent, so the work does not run twice.
func TestReserveReportsSpentKeyForUnreplayableResponse(t *testing.T) {
	t.Parallel()

	querier := &fakeQuerier{
		execTag: pgconn.NewCommandTag("INSERT 0 0"),
		row:     fakeRow{values: []any{"fingerprint", (*int32)(nil), []byte(nil), []byte(nil), false, true}},
	}

	stored, err := newFakeStore(querier).Reserve(context.Background(), "key", "fingerprint")
	if err != nil {
		t.Fatalf("Reserve() error = %v, want nil", err)
	}
	if stored == nil || stored.Replayable {
		t.Fatalf("Reserve() = %+v, want a spent, unreplayable outcome", stored)
	}
}

// TestReserveTreatsSweptRowAsInFlight covers the benign race between the failed
// insert and the read that follows it: the row expired in between, so this attempt
// neither owns the key nor has anything to replay.
func TestReserveTreatsSweptRowAsInFlight(t *testing.T) {
	t.Parallel()

	querier := &fakeQuerier{
		execTag: pgconn.NewCommandTag("INSERT 0 0"),
		row:     fakeRow{err: pgx.ErrNoRows},
	}
	_, err := newFakeStore(querier).Reserve(context.Background(), "key", "fingerprint")
	if !errors.Is(err, idempotency.ErrInFlight) {
		t.Fatalf("Reserve() error = %v, want %v", err, idempotency.ErrInFlight)
	}
}

func TestCompleteAndReleaseAndSweepIssueOneStatementEach(t *testing.T) {
	t.Parallel()

	querier := &fakeQuerier{execTag: pgconn.NewCommandTag("UPDATE 1")}
	store := newFakeStore(querier)

	if err := store.Complete(context.Background(), "key", idempotency.StoredResponse{
		Status:     http.StatusCreated,
		Header:     http.Header{"Location": []string{"/a"}},
		Body:       []byte("{}"),
		Replayable: true,
	}); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if err := store.Release(context.Background(), "key"); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	removed, err := store.Sweep(context.Background())
	if err != nil {
		t.Fatalf("Sweep() error = %v", err)
	}
	if removed != 1 {
		t.Fatalf("Sweep() = %d, want 1", removed)
	}

	// One statement per operation is what makes each of them atomic without a
	// transaction; a second statement here would be a race waiting to be found.
	if len(querier.execCalls) != 3 {
		t.Fatalf("exec calls = %d (%v), want one per operation", len(querier.execCalls), querier.execCalls)
	}
}

func TestStoreWrapsDatabaseFailures(t *testing.T) {
	t.Parallel()

	querier := &fakeQuerier{execErr: errors.New("connection reset")}
	store := newFakeStore(querier)

	if _, err := store.Reserve(context.Background(), "key", "fingerprint"); !errors.Is(err, ErrIdempotency) {
		t.Fatalf("Reserve() error = %v, want %v", err, ErrIdempotency)
	}
	if err := store.Release(context.Background(), "key"); !errors.Is(err, ErrIdempotency) {
		t.Fatalf("Release() error = %v, want %v", err, ErrIdempotency)
	}
	if _, err := store.Sweep(context.Background()); !errors.Is(err, ErrIdempotency) {
		t.Fatalf("Sweep() error = %v, want %v", err, ErrIdempotency)
	}
}

// TestStoreSatisfiesTheTransportPort keeps the structural match honest. The port is
// declared by internal/infra/http; this package deliberately does not import it, so
// a signature drift would otherwise only surface at the composition root.
func TestStoreSatisfiesTheTransportPort(t *testing.T) {
	t.Parallel()

	var store any = &IdempotencyStore{}
	if _, ok := store.(interface {
		Reserve(context.Context, string, string) (*idempotency.StoredResponse, error)
		Complete(context.Context, string, idempotency.StoredResponse) error
		Release(context.Context, string) error
	}); !ok {
		t.Fatal("*IdempotencyStore no longer satisfies the store shape internal/infra/http consumes")
	}
}

func TestNewIdempotencyStoreRejectsUnusableRetention(t *testing.T) {
	t.Parallel()

	for _, retention := range []time.Duration{0, -time.Second} {
		if _, err := NewIdempotencyStore(&Pool{pool: nil}, retention); !errors.Is(err, ErrConfig) {
			t.Fatalf("NewIdempotencyStore(retention=%s) error = %v, want %v", retention, err, ErrConfig)
		}
	}
}

// TestStorePropagatesAcquireFailure keeps the pool's own classification intact.
// Every operation takes its connection through Pool.Acquire, so a saturated pool
// reaches the caller as ErrSaturated — the retryable identity the transport turns
// into a 503 with a Retry-After — rather than being flattened into a generic store
// failure.
func TestStorePropagatesAcquireFailure(t *testing.T) {
	t.Parallel()

	store := &IdempotencyStore{acquire: pooledQuerier(&Pool{}), retention: time.Hour}

	if _, err := store.Reserve(context.Background(), "key", "fingerprint"); !errors.Is(err, ErrConfig) {
		t.Fatalf("Reserve() error = %v, want %v", err, ErrConfig)
	}
	if err := store.Complete(context.Background(), "key", idempotency.StoredResponse{}); !errors.Is(err, ErrConfig) {
		t.Fatalf("Complete() error = %v, want %v", err, ErrConfig)
	}
	if err := store.Release(context.Background(), "key"); !errors.Is(err, ErrConfig) {
		t.Fatalf("Release() error = %v, want %v", err, ErrConfig)
	}
	if _, err := store.Sweep(context.Background()); !errors.Is(err, ErrConfig) {
		t.Fatalf("Sweep() error = %v, want %v", err, ErrConfig)
	}
}

func TestIdempotencyHeaderRoundTrip(t *testing.T) {
	t.Parallel()

	encoded, err := encodeIdempotencyHeaders(http.Header{"Location": []string{"/a"}, "X-Trace": []string{"1", "2"}})
	if err != nil {
		t.Fatalf("encodeIdempotencyHeaders() error = %v", err)
	}
	decoded, err := decodeIdempotencyHeaders(encoded)
	if err != nil {
		t.Fatalf("decodeIdempotencyHeaders() error = %v", err)
	}
	if got := decoded.Get("Location"); got != "/a" {
		t.Fatalf("decoded Location = %q, want %q", got, "/a")
	}
	if got := decoded.Values("X-Trace"); len(got) != 2 {
		t.Fatalf("decoded X-Trace = %v, want both values", got)
	}

	// An absent header set must round-trip as empty rather than as SQL NULL, so a
	// replay writes no headers instead of failing to decode.
	empty, err := encodeIdempotencyHeaders(nil)
	if err != nil {
		t.Fatalf("encodeIdempotencyHeaders(nil) error = %v", err)
	}
	if string(empty) != "{}" {
		t.Fatalf("encodeIdempotencyHeaders(nil) = %q, want %q", empty, "{}")
	}
	if header, err := decodeIdempotencyHeaders(nil); err != nil || len(header) != 0 {
		t.Fatalf("decodeIdempotencyHeaders(nil) = (%v, %v), want an empty header", header, err)
	}
	if _, err := decodeIdempotencyHeaders([]byte("not json")); !errors.Is(err, ErrIdempotency) {
		t.Fatalf("decodeIdempotencyHeaders(invalid) error = %v, want %v", err, ErrIdempotency)
	}
}
