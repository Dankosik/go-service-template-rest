//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/infra/postgresidempotency"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresHTTPIdempotencyReplayMismatchAndScope(t *testing.T) {
	fixture := newHTTPIDFixture(t, "idempotency-replay")
	request := httpIDRequest(t, "caller-a", "key-a", 10)

	first, replayed, err := fixture.store.Execute(fixture.ctx, request, fixture.effect("first"))
	if err != nil || replayed {
		t.Fatalf("first Execute() = replayed %v, error %v", replayed, err)
	}
	second, replayed, err := fixture.store.Execute(fixture.ctx, request, func(context.Context, pgx.Tx) (httpidempotency.Result, error) {
		return httpidempotency.Result{}, errors.New("replayed work ran")
	})
	if err != nil || !replayed || string(second.Body) != string(first.Body) {
		t.Fatalf("second Execute() = %#v, replayed %v, error %v", second, replayed, err)
	}

	changed := httpIDRequest(t, "caller-a", "key-a", 11)
	if _, _, err := fixture.store.Execute(fixture.ctx, changed, fixture.effect("changed")); !errors.Is(err, postgresidempotency.ErrMismatch) {
		t.Fatalf("changed Execute() error = %v, want ErrMismatch", err)
	}
	otherCaller := httpIDRequest(t, "caller-b", "key-a", 10)
	if _, replayed, err := fixture.store.Execute(fixture.ctx, otherCaller, fixture.effect("other")); err != nil || replayed {
		t.Fatalf("other caller Execute() = replayed %v, error %v", replayed, err)
	}
	fixture.assertEffects(t, 2)
}

func TestPostgresHTTPIdempotencyFeatureExecutorBindsTransactionRepository(t *testing.T) {
	fixture := newHTTPIDFixture(t, "idempotency-feature-executor")
	type response struct {
		Value string `json:"value"`
	}
	executor, err := postgresidempotency.NewExecutor(
		fixture.store,
		func(tx pgx.Tx) effectRepository { return postgresEffectRepository{tx: tx} },
		httpidempotency.JSONCodec[response](http.StatusCreated),
	)
	if err != nil {
		t.Fatalf("NewExecutor(): %v", err)
	}
	request := httpIDRequest(t, "caller-a", "key-a", 10)
	work := func(ctx context.Context, effects effectRepository) (response, error) {
		return response{Value: "created"}, effects.Insert(ctx, "created")
	}
	if got, replayed, err := executor.Execute(fixture.ctx, request, work); err != nil || replayed || got.Value != "created" {
		t.Fatalf("first feature Execute() = %#v, replayed %v, error %v", got, replayed, err)
	}
	if got, replayed, err := executor.Execute(fixture.ctx, request, work); err != nil || !replayed || got.Value != "created" {
		t.Fatalf("replayed feature Execute() = %#v, replayed %v, error %v", got, replayed, err)
	}
	fixture.assertEffects(t, 1)
}

func TestPostgresHTTPIdempotencySerializesConcurrentReplicas(t *testing.T) {
	fixture := newHTTPIDFixture(t, "idempotency-concurrent")
	request := httpIDRequest(t, "caller-a", "key-a", 10)
	started := make(chan struct{})
	release := make(chan struct{})
	var workCalls atomic.Int32
	type outcome struct {
		replayed bool
		err      error
	}
	firstDone := make(chan outcome, 1)
	secondDone := make(chan outcome, 1)

	go func() {
		_, replayed, err := fixture.store.Execute(fixture.ctx, request, func(ctx context.Context, tx pgx.Tx) (httpidempotency.Result, error) {
			workCalls.Add(1)
			if _, err := tx.Exec(ctx, "INSERT INTO test_http_idempotency_effects (value) VALUES ('winner')"); err != nil {
				return httpidempotency.Result{}, err
			}
			close(started)
			<-release
			return httpIDResult("winner"), nil
		})
		firstDone <- outcome{replayed: replayed, err: err}
	}()
	<-started
	go func() {
		_, replayed, err := fixture.store.Execute(fixture.ctx, request, func(ctx context.Context, tx pgx.Tx) (httpidempotency.Result, error) {
			workCalls.Add(1)
			if _, err := tx.Exec(ctx, "INSERT INTO test_http_idempotency_effects (value) VALUES ('loser')"); err != nil {
				return httpidempotency.Result{}, err
			}
			return httpIDResult("loser"), nil
		})
		secondDone <- outcome{replayed: replayed, err: err}
	}()

	waittest.UntilFunc(t, 5*time.Second, func() bool {
		var waiters int
		err := fixture.pool.QueryRow(fixture.ctx, `
			SELECT count(*) FROM pg_stat_activity
			WHERE application_name = 'idempotency-concurrent'
			  AND query LIKE '%INSERT INTO postgres_http_idempotency%'
			  AND wait_event_type = 'Lock'`).Scan(&waiters)
		return err == nil && waiters == 1
	}, func() string { return "second replica to wait on PostgreSQL uniqueness" })
	close(release)

	first := <-firstDone
	second := <-secondDone
	if first.err != nil || first.replayed {
		t.Fatalf("first outcome = %+v", first)
	}
	if second.err != nil || !second.replayed {
		t.Fatalf("second outcome = %+v", second)
	}
	if workCalls.Load() != 1 {
		t.Fatalf("work calls = %d, want 1", workCalls.Load())
	}
	fixture.assertEffects(t, 1)
}

func TestPostgresHTTPIdempotencyRollbackAndExpiryPermitRetry(t *testing.T) {
	fixture := newHTTPIDFixture(t, "idempotency-retry")
	request := httpIDRequest(t, "caller-a", "key-a", 10)
	wantErr := errors.New("business rejected")

	if _, _, err := fixture.store.Execute(fixture.ctx, request, func(ctx context.Context, tx pgx.Tx) (httpidempotency.Result, error) {
		if _, err := tx.Exec(ctx, "INSERT INTO test_http_idempotency_effects (value) VALUES ('rolled-back')"); err != nil {
			return httpidempotency.Result{}, err
		}
		return httpidempotency.Result{}, wantErr
	}); !errors.Is(err, wantErr) {
		t.Fatalf("rollback Execute() error = %v, want business error", err)
	}
	fixture.assertEffects(t, 0)
	if _, replayed, err := fixture.store.Execute(fixture.ctx, request, fixture.effect("committed")); err != nil || replayed {
		t.Fatalf("retry Execute() = replayed %v, error %v", replayed, err)
	}

	if _, err := fixture.pool.Exec(fixture.ctx, "UPDATE postgres_http_idempotency SET expires_at = clock_timestamp() - interval '1 second'"); err != nil {
		t.Fatalf("expire result: %v", err)
	}
	if deleted, err := fixture.store.Cleanup(fixture.ctx); err != nil || deleted != 1 {
		t.Fatalf("Cleanup() = %d, %v, want 1", deleted, err)
	}
	if _, replayed, err := fixture.store.Execute(fixture.ctx, request, fixture.effect("after-expiry")); err != nil || replayed {
		t.Fatalf("expired retry Execute() = replayed %v, error %v", replayed, err)
	}
	fixture.assertEffects(t, 2)
}

func TestPostgresHTTPIdempotencyRejectsReadOnlyAuthority(t *testing.T) {
	fixture := newHTTPIDFixture(t, "idempotency-writer")
	readOnlyPool := newHTTPIDPool(t, fixture.ctx, httpIDDSNParam(t, fixture.dsn, "default_transaction_read_only", "on"), "idempotency-read-only")
	store, err := postgresidempotency.NewStore(readOnlyPool, time.Hour)
	if err != nil {
		t.Fatalf("NewStore(read only): %v", err)
	}
	called := false
	if _, _, err := store.Execute(fixture.ctx, httpIDRequest(t, "caller-a", "key-a", 10), func(context.Context, pgx.Tx) (httpidempotency.Result, error) {
		called = true
		return httpIDResult("unsafe"), nil
	}); !errors.Is(err, postgresidempotency.ErrUnavailable) {
		t.Fatalf("read-only Execute() error = %v, want ErrUnavailable", err)
	}
	if called {
		t.Fatal("read-only authority executed business work")
	}
}

type httpIDFixture struct {
	ctx   context.Context
	pool  *pgxpool.Pool
	store *postgresidempotency.Store
	dsn   string
}

type effectRepository interface {
	Insert(context.Context, string) error
}

type postgresEffectRepository struct{ tx pgx.Tx }

func (r postgresEffectRepository) Insert(ctx context.Context, value string) error {
	_, err := r.tx.Exec(ctx, "INSERT INTO test_http_idempotency_effects (value) VALUES ($1)", value)
	return err
}

func newHTTPIDFixture(t *testing.T, applicationName string) httpIDFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	t.Cleanup(cancel)
	dsn := pgtest.Migrated(t, os.DirFS(".."), "migrations")
	pool := newHTTPIDPool(t, ctx, dsn, applicationName)
	store, err := postgresidempotency.NewStore(pool, time.Hour)
	if err != nil {
		t.Fatalf("NewStore(): %v", err)
	}
	if _, err := pool.Exec(ctx, "CREATE TABLE test_http_idempotency_effects (value text NOT NULL)"); err != nil {
		t.Fatalf("create effects table: %v", err)
	}
	return httpIDFixture{ctx: ctx, pool: pool, store: store, dsn: dsn}
}

func newHTTPIDPool(t *testing.T, ctx context.Context, dsn, applicationName string) *pgxpool.Pool {
	t.Helper()
	dsn = httpIDDSNParam(t, dsn, "application_name", applicationName)
	pool, err := postgres.Open(ctx, postgres.Options{DSN: dsn, MaxOpenConns: 4})
	if err != nil {
		t.Fatalf("postgres.Open(): %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func httpIDDSNParam(t *testing.T, dsn, name, value string) string {
	t.Helper()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse DSN: %v", err)
	}
	query := parsed.Query()
	query.Set(name, value)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func httpIDRequest(t *testing.T, caller, key string, amount int) httpidempotency.Request {
	t.Helper()
	request, err := httpidempotency.NewRequest(
		httpidempotency.Scope{Caller: caller, Operation: "test.create"},
		key,
		struct {
			Amount int `json:"amount"`
		}{Amount: amount},
	)
	if err != nil {
		t.Fatalf("NewRequest(): %v", err)
	}
	return request
}

func httpIDResult(value string) httpidempotency.Result {
	return httpidempotency.Result{
		Status: http.StatusCreated,
		Header: http.Header{"Content-Type": {"application/json"}, "Location": {"/effects/" + value}},
		Body:   []byte(fmt.Sprintf(`{"value":%q}`, value)),
	}
}

func (f httpIDFixture) effect(value string) func(context.Context, pgx.Tx) (httpidempotency.Result, error) {
	return func(ctx context.Context, tx pgx.Tx) (httpidempotency.Result, error) {
		if _, err := tx.Exec(ctx, "INSERT INTO test_http_idempotency_effects (value) VALUES ($1)", value); err != nil {
			return httpidempotency.Result{}, err
		}
		return httpIDResult(value), nil
	}
}

func (f httpIDFixture) assertEffects(t *testing.T, want int) {
	t.Helper()
	var got int
	if err := f.pool.QueryRow(f.ctx, "SELECT count(*) FROM test_http_idempotency_effects").Scan(&got); err != nil {
		t.Fatalf("count effects: %v", err)
	}
	if got != want {
		t.Fatalf("effects = %d, want %d", got, want)
	}
}
