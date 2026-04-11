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

func (f fakePingHistoryQuerier) CreatePingHistory(ctx context.Context, payload string) (sqlcgen.PingHistory, error) {
	return f.create(ctx, payload)
}

func (f fakePingHistoryQuerier) ListRecentPingHistory(ctx context.Context, limit int32) ([]sqlcgen.PingHistory, error) {
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
	return f.beginTx(ctx, txOptions)
}

type fakePingHistoryRow struct {
	err error
}

func (r fakePingHistoryRow) Scan(...any) error {
	return r.err
}

type recordingPingHistoryTx struct {
	queryRow func(context.Context, string, ...any) pgx.Row
	rollback func(context.Context) error
}

var _ pgx.Tx = (*recordingPingHistoryTx)(nil)

func (tx *recordingPingHistoryTx) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("nested transactions are not supported by fake tx")
}

func (tx *recordingPingHistoryTx) Commit(context.Context) error {
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

func (tx *recordingPingHistoryTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
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
		list: func(context.Context, int32) ([]sqlcgen.PingHistory, error) { return nil, nil },
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
		list: func(context.Context, int32) ([]sqlcgen.PingHistory, error) { return nil, nil },
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
		list: func(context.Context, int32) ([]sqlcgen.PingHistory, error) { return nil, nil },
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
		create: func(context.Context, string) (sqlcgen.PingHistory, error) { return sqlcgen.PingHistory{}, nil },
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
}

func TestPingHistoryRepositoryListRecentErrors(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("read failed")
	repo := newPingHistoryRepositoryWithQuerier(fakePingHistoryQuerier{
		create: func(context.Context, string) (sqlcgen.PingHistory, error) { return sqlcgen.PingHistory{}, nil },
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
