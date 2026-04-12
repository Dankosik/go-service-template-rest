package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type fakePingHistoryQuerier struct {
	create func(ctx context.Context, payload string) (sqlcgen.PingHistory, error)
	list   func(ctx context.Context, limit int32) ([]sqlcgen.PingHistory, error)
}

func newPingHistoryRepositoryWithQuerier(queries pingHistoryQuerier) *PingHistoryRepository {
	return &PingHistoryRepository{queries: queries}
}

func (f fakePingHistoryQuerier) CreatePingHistory(ctx context.Context, payload string) (sqlcgen.PingHistory, error) {
	if f.create == nil {
		return sqlcgen.PingHistory{}, errors.New("unexpected CreatePingHistory call")
	}
	return f.create(ctx, payload)
}

func (f fakePingHistoryQuerier) ListRecentPingHistory(ctx context.Context, limit int32) ([]sqlcgen.PingHistory, error) {
	if f.list == nil {
		return nil, errors.New("unexpected ListRecentPingHistory call")
	}
	return f.list(ctx, limit)
}

type fakePingHistoryDB struct {
	beginTx func(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

func (f fakePingHistoryDB) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (f fakePingHistoryDB) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, nil
}

func (f fakePingHistoryDB) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	return nil
}

func (f fakePingHistoryDB) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	if f.beginTx == nil {
		return nil, errors.New("unexpected BeginTx call")
	}
	return f.beginTx(ctx, txOptions)
}

type fakePingHistoryRow struct {
	row sqlcgen.PingHistory
	err error
}

func (r fakePingHistoryRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != 3 {
		return errors.New("unexpected ping history scan destination count")
	}
	id, ok := dest[0].(*int64)
	if !ok {
		return errors.New("unexpected ping history id destination")
	}
	payload, ok := dest[1].(*string)
	if !ok {
		return errors.New("unexpected ping history payload destination")
	}
	createdAt, ok := dest[2].(*pgtype.Timestamptz)
	if !ok {
		return errors.New("unexpected ping history created_at destination")
	}
	*id = r.row.ID
	*payload = r.row.Payload
	*createdAt = r.row.CreatedAt
	return nil
}

type fakePingHistoryRows struct {
	rows   []sqlcgen.PingHistory
	index  int
	err    error
	closed bool
}

func (r *fakePingHistoryRows) Close() {
	r.closed = true
}

func (r *fakePingHistoryRows) Err() error {
	return r.err
}

func (r *fakePingHistoryRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (r *fakePingHistoryRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (r *fakePingHistoryRows) Next() bool {
	if r.index >= len(r.rows) {
		r.Close()
		return false
	}
	r.index++
	return true
}

func (r *fakePingHistoryRows) Scan(dest ...any) error {
	if r.index == 0 || r.index > len(r.rows) {
		return errors.New("Scan called without a current ping history row")
	}
	return fakePingHistoryRow{row: r.rows[r.index-1]}.Scan(dest...)
}

func (r *fakePingHistoryRows) Values() ([]any, error) {
	if r.index == 0 || r.index > len(r.rows) {
		return nil, errors.New("Values called without a current ping history row")
	}
	row := r.rows[r.index-1]
	return []any{row.ID, row.Payload, row.CreatedAt}, nil
}

func (r *fakePingHistoryRows) RawValues() [][]byte {
	return nil
}

func (r *fakePingHistoryRows) Conn() *pgx.Conn {
	return nil
}

type recordingPingHistoryTx struct {
	queryRow func(context.Context, string, ...any) pgx.Row
	query    func(context.Context, string, ...any) (pgx.Rows, error)
	commit   func(context.Context) error
	rollback func(context.Context) error
}

var _ pgx.Tx = (*recordingPingHistoryTx)(nil)

func (tx *recordingPingHistoryTx) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("nested transactions are not supported by fake tx")
}

func (tx *recordingPingHistoryTx) Commit(ctx context.Context) error {
	if tx.commit != nil {
		return tx.commit(ctx)
	}
	return nil
}

func (tx *recordingPingHistoryTx) Rollback(ctx context.Context) error {
	if tx.rollback != nil {
		return tx.rollback(ctx)
	}
	return nil
}

func (tx *recordingPingHistoryTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("copy is not supported by fake tx")
}

func (tx *recordingPingHistoryTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	return nil
}

func (tx *recordingPingHistoryTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (tx *recordingPingHistoryTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("prepare is not supported by fake tx")
}

func (tx *recordingPingHistoryTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("exec is not supported by fake tx")
}

func (tx *recordingPingHistoryTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if tx.query != nil {
		return tx.query(ctx, sql, args...)
	}
	return nil, errors.New("query is not supported by fake tx")
}

func (tx *recordingPingHistoryTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if tx.queryRow != nil {
		return tx.queryRow(ctx, sql, args...)
	}
	return fakePingHistoryRow{err: errors.New("query row is not supported by fake tx")}
}

func (tx *recordingPingHistoryTx) Conn() *pgx.Conn {
	return nil
}

func TestNewPingHistoryRepositoryRejectsNilPool(t *testing.T) {
	t.Parallel()

	repo, err := NewPingHistoryRepository(nil)
	if err == nil {
		t.Fatal("NewPingHistoryRepository(nil) error = nil, want non-nil")
	}
	if repo != nil {
		t.Fatalf("NewPingHistoryRepository(nil) repo = %#v, want nil", repo)
	}
	if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("NewPingHistoryRepository(nil) error = %v, want ErrPingHistoryRepository", err)
	}
}

func TestPingHistoryRepositoryRejectsNilAndZeroValueUse(t *testing.T) {
	t.Parallel()

	var nilRepo *PingHistoryRepository
	if _, err := nilRepo.Create(context.Background(), "payload"); err == nil {
		t.Fatal("(*PingHistoryRepository)(nil).Create() error = nil, want non-nil")
	} else if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("(*PingHistoryRepository)(nil).Create() error = %v, want ErrPingHistoryRepository", err)
	}
	if _, err := nilRepo.ListRecent(context.Background(), 1); err == nil {
		t.Fatal("(*PingHistoryRepository)(nil).ListRecent() error = nil, want non-nil")
	} else if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("(*PingHistoryRepository)(nil).ListRecent() error = %v, want ErrPingHistoryRepository", err)
	}
	if _, _, err := nilRepo.createAndListRecentInTx(context.Background(), "payload", 1); err == nil {
		t.Fatal("(*PingHistoryRepository)(nil).createAndListRecentInTx() error = nil, want non-nil")
	} else if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("(*PingHistoryRepository)(nil).createAndListRecentInTx() error = %v, want ErrPingHistoryRepository", err)
	}

	zeroRepo := &PingHistoryRepository{}
	if _, err := zeroRepo.Create(context.Background(), "payload"); err == nil {
		t.Fatal("zero-value Create() error = nil, want non-nil")
	} else if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("zero-value Create() error = %v, want ErrPingHistoryRepository", err)
	}
	if _, err := zeroRepo.ListRecent(context.Background(), 1); err == nil {
		t.Fatal("zero-value ListRecent() error = nil, want non-nil")
	} else if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("zero-value ListRecent() error = %v, want ErrPingHistoryRepository", err)
	}
	if _, _, err := zeroRepo.createAndListRecentInTx(context.Background(), "payload", 1); err == nil {
		t.Fatal("zero-value createAndListRecentInTx() error = nil, want non-nil")
	} else if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("zero-value createAndListRecentInTx() error = %v, want ErrPingHistoryRepository", err)
	}
}

func TestPingHistoryRepositoryCreate(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	repo := newPingHistoryRepositoryWithQuerier(fakePingHistoryQuerier{
		create: func(context.Context, string) (sqlcgen.PingHistory, error) {
			return sqlcgen.PingHistory{
				ID:      7,
				Payload: "ok",
				CreatedAt: pgtype.Timestamptz{
					Time:  createdAt,
					Valid: true,
				},
			}, nil
		},
	})

	got, err := repo.Create(context.Background(), "ok")
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}
	if got.ID != 7 || got.Payload != "ok" || !got.CreatedAt.Equal(createdAt) {
		t.Fatalf("Create() = %#v, want ID=7 Payload=ok CreatedAt=%v", got, createdAt)
	}
}

func TestPingHistoryRepositoryCreateWrapsQueryError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("write failed")
	repo := newPingHistoryRepositoryWithQuerier(fakePingHistoryQuerier{
		create: func(context.Context, string) (sqlcgen.PingHistory, error) {
			return sqlcgen.PingHistory{}, sentinel
		},
	})

	_, err := repo.Create(context.Background(), "boom")
	if err == nil {
		t.Fatal("Create() error = nil, want non-nil")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("Create() error = %v, want wrapped %v", err, sentinel)
	}
	if !strings.Contains(err.Error(), "create ping history") {
		t.Fatalf("Create() error = %q, want context prefix", err.Error())
	}
}

func TestPingHistoryRepositoryCreateRejectsNullCreatedAt(t *testing.T) {
	t.Parallel()

	repo := newPingHistoryRepositoryWithQuerier(fakePingHistoryQuerier{
		create: func(context.Context, string) (sqlcgen.PingHistory, error) {
			return sqlcgen.PingHistory{ID: 9, Payload: "x"}, nil
		},
	})

	_, err := repo.Create(context.Background(), "x")
	if err == nil {
		t.Fatal("Create() error = nil, want non-nil")
	}
	if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("Create() error = %v, want ErrPingHistoryRepository", err)
	}
	if !strings.Contains(err.Error(), "created_at is null") {
		t.Fatalf("Create() error = %q, want null created_at detail", err.Error())
	}
}

func TestPingHistoryRepositoryListRecent(t *testing.T) {
	t.Parallel()

	firstAt := time.Date(2026, time.February, 1, 10, 0, 0, 0, time.UTC)
	secondAt := firstAt.Add(-time.Minute)
	repo := newPingHistoryRepositoryWithQuerier(fakePingHistoryQuerier{
		list: func(context.Context, int32) ([]sqlcgen.PingHistory, error) {
			return []sqlcgen.PingHistory{
				{
					ID:      10,
					Payload: "a",
					CreatedAt: pgtype.Timestamptz{
						Time:  firstAt,
						Valid: true,
					},
				},
				{
					ID:      9,
					Payload: "b",
					CreatedAt: pgtype.Timestamptz{
						Time:  secondAt,
						Valid: true,
					},
				},
			}, nil
		},
	})

	got, err := repo.ListRecent(context.Background(), 2)
	if err != nil {
		t.Fatalf("ListRecent() error = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListRecent() len = %d, want 2", len(got))
	}
	if got[0].ID != 10 || got[1].ID != 9 {
		t.Fatalf("ListRecent() IDs = [%d,%d], want [10,9]", got[0].ID, got[1].ID)
	}
	if got[0].Payload != "a" || got[1].Payload != "b" {
		t.Fatalf("ListRecent() payloads = [%q,%q], want [a,b]", got[0].Payload, got[1].Payload)
	}
	if !got[0].CreatedAt.Equal(firstAt) || !got[1].CreatedAt.Equal(secondAt) {
		t.Fatalf("ListRecent() CreatedAt = [%v,%v], want [%v,%v]", got[0].CreatedAt, got[1].CreatedAt, firstAt, secondAt)
	}
}

func TestPingHistoryRepositoryListRecentErrors(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("read failed")
	repo := newPingHistoryRepositoryWithQuerier(fakePingHistoryQuerier{
		list: func(context.Context, int32) ([]sqlcgen.PingHistory, error) {
			return nil, sentinel
		},
	})

	_, err := repo.ListRecent(context.Background(), 1)
	if err == nil {
		t.Fatal("ListRecent() error = nil, want non-nil")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("ListRecent() error = %v, want wrapped %v", err, sentinel)
	}
	if !strings.Contains(err.Error(), "list recent ping history") {
		t.Fatalf("ListRecent() error = %q, want context prefix", err.Error())
	}

	_, err = repo.ListRecent(context.Background(), 0)
	if err == nil {
		t.Fatal("ListRecent(limit=0) error = nil, want non-nil")
	}
	if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("ListRecent(limit=0) error = %v, want ErrPingHistoryRepository", err)
	}
}

func TestPingHistoryRepositoryListRecentRejectsOverSampleLimit(t *testing.T) {
	t.Parallel()

	var listCalled bool
	repo := newPingHistoryRepositoryWithQuerier(fakePingHistoryQuerier{
		list: func(context.Context, int32) ([]sqlcgen.PingHistory, error) {
			listCalled = true
			return nil, nil
		},
	})

	_, err := repo.ListRecent(context.Background(), pingHistorySampleMaxListLimit+1)
	if err == nil {
		t.Fatal("ListRecent(over sample max) error = nil, want non-nil")
	}
	if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("ListRecent(over sample max) error = %v, want ErrPingHistoryRepository", err)
	}
	if !strings.Contains(err.Error(), "limit must be <=") {
		t.Fatalf("ListRecent(over sample max) error = %q, want limit max detail", err.Error())
	}
	if listCalled {
		t.Fatal("ListRecent(over sample max) called query, want rejection before SQL")
	}
}

func TestPingHistoryRepositoryCreateAndListRecentInTxRejectsOverSampleLimit(t *testing.T) {
	t.Parallel()

	var beginCalled bool
	repo := &PingHistoryRepository{
		db: fakePingHistoryDB{
			beginTx: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
				beginCalled = true
				return nil, errors.New("unexpected begin")
			},
		},
	}

	_, _, err := repo.createAndListRecentInTx(context.Background(), "payload", pingHistorySampleMaxListLimit+1)
	if err == nil {
		t.Fatal("createAndListRecentInTx(over sample max) error = nil, want non-nil")
	}
	if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("createAndListRecentInTx(over sample max) error = %v, want ErrPingHistoryRepository", err)
	}
	if beginCalled {
		t.Fatal("createAndListRecentInTx(over sample max) began transaction, want rejection before SQL")
	}
}

func TestPingHistoryRepositoryCreateAndListRecentInTxBeginError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("begin failed")
	repo := &PingHistoryRepository{
		db: fakePingHistoryDB{
			beginTx: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
				return nil, sentinel
			},
		},
	}

	_, _, err := repo.createAndListRecentInTx(context.Background(), "payload", 1)
	if err == nil {
		t.Fatal("createAndListRecentInTx() error = nil, want non-nil")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("createAndListRecentInTx() error = %v, want wrapped %v", err, sentinel)
	}
	if !strings.Contains(err.Error(), "begin ping history transaction") {
		t.Fatalf("createAndListRecentInTx() error = %q, want begin context", err.Error())
	}
}

func TestPingHistoryRepositoryCreateAndListRecentInTxSuccess(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, time.March, 3, 4, 5, 6, 0, time.UTC)
	recentAt := createdAt.Add(-time.Minute)
	var commitCalled bool
	tx := &recordingPingHistoryTx{
		queryRow: func(_ context.Context, sql string, args ...any) pgx.Row {
			if !strings.Contains(sql, "INSERT INTO ping_history") {
				return fakePingHistoryRow{err: errors.New("unexpected create sql")}
			}
			if len(args) != 1 || args[0] != "payload" {
				return fakePingHistoryRow{err: errors.New("unexpected create args")}
			}
			return fakePingHistoryRow{row: sqlcgen.PingHistory{
				ID:      12,
				Payload: "payload",
				CreatedAt: pgtype.Timestamptz{
					Time:  createdAt,
					Valid: true,
				},
			}}
		},
		query: func(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
			if !strings.Contains(sql, "SELECT id, payload, created_at") {
				return nil, errors.New("unexpected list sql")
			}
			if len(args) != 1 || args[0] != int32(2) {
				return nil, errors.New("unexpected list args")
			}
			return &fakePingHistoryRows{rows: []sqlcgen.PingHistory{
				{
					ID:      12,
					Payload: "payload",
					CreatedAt: pgtype.Timestamptz{
						Time:  createdAt,
						Valid: true,
					},
				},
				{
					ID:      11,
					Payload: "previous",
					CreatedAt: pgtype.Timestamptz{
						Time:  recentAt,
						Valid: true,
					},
				},
			}}, nil
		},
		commit: func(context.Context) error {
			commitCalled = true
			return nil
		},
	}
	repo := &PingHistoryRepository{
		db: fakePingHistoryDB{
			beginTx: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
				return tx, nil
			},
		},
	}

	created, recent, err := repo.createAndListRecentInTx(context.Background(), "payload", 2)
	if err != nil {
		t.Fatalf("createAndListRecentInTx() error = %v, want nil", err)
	}
	if !commitCalled {
		t.Fatal("Commit was not called")
	}
	if created.ID != 12 || created.Payload != "payload" || !created.CreatedAt.Equal(createdAt) {
		t.Fatalf("created = %#v, want ID=12 Payload=payload CreatedAt=%v", created, createdAt)
	}
	if len(recent) != 2 {
		t.Fatalf("recent len = %d, want 2", len(recent))
	}
	if recent[0].ID != 12 || recent[1].ID != 11 {
		t.Fatalf("recent IDs = [%d,%d], want [12,11]", recent[0].ID, recent[1].ID)
	}
}

func TestPingHistoryRepositoryCreateAndListRecentInTxCommitError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("commit failed")
	createdAt := time.Date(2026, time.March, 4, 5, 6, 7, 0, time.UTC)
	var rollbackCalled bool
	tx := &recordingPingHistoryTx{
		queryRow: func(context.Context, string, ...any) pgx.Row {
			return fakePingHistoryRow{row: sqlcgen.PingHistory{
				ID:      21,
				Payload: "payload",
				CreatedAt: pgtype.Timestamptz{
					Time:  createdAt,
					Valid: true,
				},
			}}
		},
		query: func(context.Context, string, ...any) (pgx.Rows, error) {
			return &fakePingHistoryRows{rows: []sqlcgen.PingHistory{
				{
					ID:      21,
					Payload: "payload",
					CreatedAt: pgtype.Timestamptz{
						Time:  createdAt,
						Valid: true,
					},
				},
			}}, nil
		},
		commit: func(context.Context) error {
			return sentinel
		},
		rollback: func(context.Context) error {
			rollbackCalled = true
			return nil
		},
	}
	repo := &PingHistoryRepository{
		db: fakePingHistoryDB{
			beginTx: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
				return tx, nil
			},
		},
	}

	_, _, err := repo.createAndListRecentInTx(context.Background(), "payload", 1)
	if err == nil {
		t.Fatal("createAndListRecentInTx() error = nil, want non-nil")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("createAndListRecentInTx() error = %v, want wrapped %v", err, sentinel)
	}
	if !strings.Contains(err.Error(), "commit ping history transaction") {
		t.Fatalf("createAndListRecentInTx() error = %q, want commit context", err.Error())
	}
	if !rollbackCalled {
		t.Fatal("Rollback was not called after commit error")
	}
}

func TestPingHistoryRepositoryRollbackUsesCleanupContext(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("write failed")
	ctx, cancel := context.WithCancel(context.Background())
	var rollbackCalled bool
	var rollbackCtxErr error
	var rollbackHasDeadline bool
	tx := &recordingPingHistoryTx{
		queryRow: func(context.Context, string, ...any) pgx.Row {
			cancel()
			return fakePingHistoryRow{err: sentinel}
		},
		rollback: func(ctx context.Context) error {
			rollbackCalled = true
			rollbackCtxErr = ctx.Err()
			_, rollbackHasDeadline = ctx.Deadline()
			return nil
		},
	}
	repo := &PingHistoryRepository{
		db: fakePingHistoryDB{
			beginTx: func(context.Context, pgx.TxOptions) (pgx.Tx, error) {
				return tx, nil
			},
		},
	}

	_, _, err := repo.createAndListRecentInTx(ctx, "payload", 1)
	if err == nil {
		t.Fatal("createAndListRecentInTx() error = nil, want non-nil")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("createAndListRecentInTx() error = %v, want wrapped %v", err, sentinel)
	}
	if !rollbackCalled {
		t.Fatal("Rollback was not called")
	}
	if rollbackCtxErr != nil {
		t.Fatalf("Rollback() ctx.Err() = %v, want nil", rollbackCtxErr)
	}
	if !rollbackHasDeadline {
		t.Fatal("Rollback() context has no deadline")
	}
}
