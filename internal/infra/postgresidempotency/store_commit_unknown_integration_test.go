//go:build integration

package postgresidempotency

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

func TestExecuteReadsBackCommittedResultAfterLostCommitResponse(t *testing.T) {
	fixture := newCommitUnknownFixture(t)
	store := fixture.store
	commit := store.inTx
	store.inTx = func(ctx context.Context, opts pgx.TxOptions, work func(pgx.Tx) error) error {
		if err := commit(ctx, opts, work); err != nil {
			return err
		}
		return fmt.Errorf("commit response lost: %w", postgres.ErrCommitUnknown)
	}
	request, err := httpidempotency.NewRequest(
		httpidempotency.Scope{Caller: "caller", Operation: "create"},
		"key-a",
		1,
		struct {
			Value string `json:"value"`
		}{Value: "committed"},
	)
	if err != nil {
		t.Fatalf("NewRequest(): %v", err)
	}
	executor, err := NewExecutor(
		store,
		func(tx pgx.Tx) pgx.Tx { return tx },
		httpidempotency.JSONCodec[commitResponse](http.StatusCreated),
	)
	if err != nil {
		t.Fatalf("NewExecutor(): %v", err)
	}
	result, replayed, err := executor(fixture.ctx, request, func(ctx context.Context, tx pgx.Tx) (commitResponse, error) {
		if _, err := tx.Exec(ctx, "INSERT INTO idempotency_commit_effect (value) VALUES ('committed')"); err != nil {
			return commitResponse{}, err
		}
		return commitResponse{Value: "committed"}, nil
	})
	if err != nil || !replayed || result.Value != "committed" {
		t.Fatalf("Execute() = %#v, replayed %v, error %v", result, replayed, err)
	}
	fixture.assertEffects(t, 1)
}

func TestExecuteReturnsUnknownWithoutRetryWhenCommitDidNotPersist(t *testing.T) {
	fixture := newCommitUnknownFixture(t)
	fixture.store.inTx = func(ctx context.Context, opts pgx.TxOptions, work func(pgx.Tx) error) error {
		return postgres.InTx(ctx, fixture.pool, opts, func(tx pgx.Tx) error {
			if err := work(tx); err != nil {
				return err
			}
			return postgres.ErrCommitUnknown
		})
	}
	request, err := httpidempotency.NewRequest(
		httpidempotency.Scope{Caller: "caller", Operation: "create"},
		"key-a",
		1,
		struct{ Value string }{Value: "rolled-back"},
	)
	if err != nil {
		t.Fatalf("NewRequest(): %v", err)
	}
	executor, err := NewExecutor(
		fixture.store,
		func(tx pgx.Tx) pgx.Tx { return tx },
		httpidempotency.JSONCodec[commitResponse](http.StatusCreated),
	)
	if err != nil {
		t.Fatalf("NewExecutor(): %v", err)
	}
	var calls atomic.Int32
	_, replayed, err := executor(fixture.ctx, request, func(ctx context.Context, tx pgx.Tx) (commitResponse, error) {
		calls.Add(1)
		if _, err := tx.Exec(ctx, "INSERT INTO idempotency_commit_effect (value) VALUES ('rolled-back')"); err != nil {
			return commitResponse{}, err
		}
		return commitResponse{Value: "rolled-back"}, nil
	})
	if !errors.Is(err, httpidempotency.ErrOutcomeUnknown) || replayed {
		t.Fatalf("Execute() = replayed %v, error %v; want ErrOutcomeUnknown", replayed, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("work calls = %d, want 1", calls.Load())
	}
	fixture.assertEffects(t, 0)
}

type commitResponse struct {
	Value string `json:"value"`
}

type commitUnknownFixture struct {
	ctx   context.Context
	pool  *pgxpool.Pool
	store *Store
}

func newCommitUnknownFixture(t *testing.T) commitUnknownFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	t.Cleanup(cancel)
	dsn := pgtest.Migrated(t, os.DirFS("../../.."), "migrations")
	pool, err := postgres.Open(ctx, postgres.Options{DSN: dsn, MaxOpenConns: 3})
	if err != nil {
		t.Fatalf("postgres.Open(): %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, "CREATE TABLE idempotency_commit_effect (value text NOT NULL)"); err != nil {
		t.Fatalf("create effect table: %v", err)
	}
	store, err := NewStore(pool, time.Hour)
	if err != nil {
		t.Fatalf("NewStore(): %v", err)
	}
	return commitUnknownFixture{ctx: ctx, pool: pool, store: store}
}

func (f commitUnknownFixture) assertEffects(t *testing.T, want int) {
	t.Helper()
	var effects int
	if err := f.pool.QueryRow(f.ctx, "SELECT count(*) FROM idempotency_commit_effect").Scan(&effects); err != nil {
		t.Fatalf("count effects: %v", err)
	}
	if effects != want {
		t.Fatalf("effects = %d, want %d", effects, want)
	}
}
