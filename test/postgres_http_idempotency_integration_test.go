//go:build integration

package integration_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
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
	executor := fixture.executor(t)
	request := httpIDRequest(t, "caller-a", "key-a", 10)

	first, replayed, err := executor.Execute(fixture.ctx, request, fixture.effect("first"))
	if err != nil || replayed {
		t.Fatalf("first Execute() = replayed %v, error %v", replayed, err)
	}
	second, replayed, err := executor.Execute(fixture.ctx, request, func(context.Context, pgx.Tx) (httpIDResponse, error) {
		return httpIDResponse{}, errors.New("replayed work ran")
	})
	if err != nil || !replayed || second != first {
		t.Fatalf("second Execute() = %#v, replayed %v, error %v", second, replayed, err)
	}

	changed := httpIDRequest(t, "caller-a", "key-a", 11)
	if _, _, err := executor.Execute(fixture.ctx, changed, fixture.effect("changed")); !errors.Is(err, httpidempotency.ErrMismatch) {
		t.Fatalf("changed Execute() error = %v, want ErrMismatch", err)
	}
	otherCaller := httpIDRequest(t, "caller-b", "key-a", 10)
	if _, replayed, err := executor.Execute(fixture.ctx, otherCaller, fixture.effect("other")); err != nil || replayed {
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
	executor := fixture.executor(t)
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
		_, replayed, err := executor.Execute(fixture.ctx, request, func(ctx context.Context, tx pgx.Tx) (httpIDResponse, error) {
			workCalls.Add(1)
			if _, err := tx.Exec(ctx, "INSERT INTO test_http_idempotency_effects (value) VALUES ('winner')"); err != nil {
				return httpIDResponse{}, err
			}
			close(started)
			select {
			case <-release:
			case <-ctx.Done():
				return httpIDResponse{}, ctx.Err()
			}
			return httpIDResult("winner"), nil
		})
		firstDone <- outcome{replayed: replayed, err: err}
	}()
	<-started
	go func() {
		_, replayed, err := executor.Execute(fixture.ctx, request, func(ctx context.Context, tx pgx.Tx) (httpIDResponse, error) {
			workCalls.Add(1)
			if _, err := tx.Exec(ctx, "INSERT INTO test_http_idempotency_effects (value) VALUES ('loser')"); err != nil {
				return httpIDResponse{}, err
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
	executor := fixture.executor(t)
	request := httpIDRequest(t, "caller-a", "key-a", 10)
	wantErr := errors.New("business rejected")

	if _, _, err := executor.Execute(fixture.ctx, request, func(ctx context.Context, tx pgx.Tx) (httpIDResponse, error) {
		if _, err := tx.Exec(ctx, "INSERT INTO test_http_idempotency_effects (value) VALUES ('rolled-back')"); err != nil {
			return httpIDResponse{}, err
		}
		return httpIDResponse{}, wantErr
	}); !errors.Is(err, wantErr) {
		t.Fatalf("rollback Execute() error = %v, want business error", err)
	}
	fixture.assertEffects(t, 0)
	if _, replayed, err := executor.Execute(fixture.ctx, request, fixture.effect("committed")); err != nil || replayed {
		t.Fatalf("retry Execute() = replayed %v, error %v", replayed, err)
	}

	if _, err := fixture.pool.Exec(fixture.ctx, "UPDATE postgres_http_idempotency SET expires_at = clock_timestamp() - interval '1 second'"); err != nil {
		t.Fatalf("expire result: %v", err)
	}
	if deleted, err := fixture.store.Cleanup(fixture.ctx); err != nil || deleted != 1 {
		t.Fatalf("Cleanup() = %d, %v, want 1", deleted, err)
	}
	if _, replayed, err := executor.Execute(fixture.ctx, request, fixture.effect("after-expiry")); err != nil || replayed {
		t.Fatalf("expired retry Execute() = replayed %v, error %v", replayed, err)
	}
	fixture.assertEffects(t, 2)
}

func TestPostgresHTTPIdempotencyOversizedResultRollsBack(t *testing.T) {
	fixture := newHTTPIDFixture(t, "idempotency-oversized")
	executor := fixture.executor(t)
	request := httpIDRequest(t, "caller-a", "key-a", 10)

	_, _, err := executor.Execute(fixture.ctx, request, func(ctx context.Context, tx pgx.Tx) (httpIDResponse, error) {
		if _, err := tx.Exec(ctx, "INSERT INTO test_http_idempotency_effects (value) VALUES ('oversized')"); err != nil {
			return httpIDResponse{}, err
		}
		return httpIDResponse{Value: strings.Repeat("x", 1<<20)}, nil
	})
	if !errors.Is(err, httpidempotency.ErrInvalidResult) {
		t.Fatalf("oversized Execute() error = %v, want ErrInvalidResult", err)
	}
	fixture.assertEffects(t, 0)
	if _, replayed, err := executor.Execute(fixture.ctx, request, fixture.effect("retry")); err != nil || replayed {
		t.Fatalf("retry after oversized result = replayed %v, error %v", replayed, err)
	}
	fixture.assertEffects(t, 1)
}

func TestPostgresHTTPIdempotencyCleanupDrainsBacklog(t *testing.T) {
	fixture := newHTTPIDFixture(t, "idempotency-cleanup-backlog")
	if _, err := fixture.pool.Exec(fixture.ctx, `
		INSERT INTO postgres_http_idempotency (
			identity_token, fingerprint_version, fingerprint, result, expires_at
		)
		SELECT
			decode(lpad(to_hex(value), 64, '0'), 'hex'),
			1,
			decode(repeat('01', 32), 'hex'),
			decode('01', 'hex'),
			clock_timestamp() - interval '1 second'
		FROM generate_series(1, 501) AS value`); err != nil {
		t.Fatalf("seed cleanup backlog: %v", err)
	}
	if deleted, err := fixture.store.Cleanup(fixture.ctx); err != nil || deleted != 501 {
		t.Fatalf("Cleanup() = %d, %v, want 501", deleted, err)
	}
	var remaining int
	if err := fixture.pool.QueryRow(fixture.ctx, "SELECT count(*) FROM postgres_http_idempotency").Scan(&remaining); err != nil || remaining != 0 {
		t.Fatalf("remaining rows = %d, error %v", remaining, err)
	}
	var indexDefinition string
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT indexdef FROM pg_indexes
		WHERE schemaname = current_schema()
		  AND indexname = 'postgres_http_idempotency_expires_idx'`).Scan(&indexDefinition); err != nil {
		t.Fatalf("read expiry index: %v", err)
	}
	if !strings.Contains(indexDefinition, "(expires_at, identity_token)") || !strings.Contains(indexDefinition, "WHERE (expires_at IS NOT NULL)") {
		t.Fatalf("expiry index = %q", indexDefinition)
	}
}

func TestPostgresHTTPIdempotencyRejectsReadOnlyAuthority(t *testing.T) {
	fixture := newHTTPIDFixture(t, "idempotency-writer")
	readOnlyPool := newHTTPIDPool(t, fixture.ctx, httpIDDSNParam(t, fixture.dsn, "default_transaction_read_only", "on"), "idempotency-read-only")
	store, err := postgresidempotency.NewStore(readOnlyPool, time.Hour)
	if err != nil {
		t.Fatalf("NewStore(read only): %v", err)
	}
	executor, err := postgresidempotency.NewExecutor(
		store,
		func(tx pgx.Tx) pgx.Tx { return tx },
		httpidempotency.JSONCodec[httpIDResponse](http.StatusCreated),
	)
	if err != nil {
		t.Fatalf("NewExecutor(read only): %v", err)
	}
	called := false
	if _, _, err := executor.Execute(fixture.ctx, httpIDRequest(t, "caller-a", "key-a", 10), func(context.Context, pgx.Tx) (httpIDResponse, error) {
		called = true
		return httpIDResult("unsafe"), nil
	}); !errors.Is(err, httpidempotency.ErrUnavailable) {
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

type httpIDResponse struct {
	Value    string `json:"value"`
	Location string `json:"location"`
}

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
		1,
		struct {
			Amount int `json:"amount"`
		}{Amount: amount},
	)
	if err != nil {
		t.Fatalf("NewRequest(): %v", err)
	}
	return request
}

func httpIDResult(value string) httpIDResponse {
	return httpIDResponse{Value: value, Location: "/effects/" + value}
}

func (f httpIDFixture) executor(t *testing.T) *postgresidempotency.Executor[pgx.Tx, httpIDResponse] {
	t.Helper()
	executor, err := postgresidempotency.NewExecutor(
		f.store,
		func(tx pgx.Tx) pgx.Tx { return tx },
		httpidempotency.JSONCodec[httpIDResponse](http.StatusCreated),
	)
	if err != nil {
		t.Fatalf("NewExecutor(): %v", err)
	}
	return executor
}

func (f httpIDFixture) effect(value string) httpidempotency.Work[pgx.Tx, httpIDResponse] {
	return func(ctx context.Context, tx pgx.Tx) (httpIDResponse, error) {
		if _, err := tx.Exec(ctx, "INSERT INTO test_http_idempotency_effects (value) VALUES ($1)", value); err != nil {
			return httpIDResponse{}, err
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
