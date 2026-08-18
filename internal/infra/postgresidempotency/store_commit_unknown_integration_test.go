//go:build integration

package postgresidempotency

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/jackc/pgx/v5"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m, "")) }

func TestExecuteReadsBackCommittedResultAfterLostCommitResponse(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	t.Cleanup(cancel)
	dsn := pgtest.Migrated(t, os.DirFS("../../.."), "migrations")
	pool, err := postgres.New(ctx, postgres.Options{
		DSN: dsn, ConnectTimeout: 3 * time.Second, HealthcheckTimeout: 3 * time.Second,
		MaxOpenConns: 3, AcquireTimeout: time.Second, ConnMaxLifetime: time.Hour,
		StatementTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("postgres.New(): %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.PGX().Exec(ctx, "CREATE TABLE idempotency_commit_effect (value text NOT NULL)"); err != nil {
		t.Fatalf("create effect table: %v", err)
	}
	store, err := NewStore(pool, time.Hour)
	if err != nil {
		t.Fatalf("NewStore(): %v", err)
	}
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
		struct {
			Value string `json:"value"`
		}{Value: "committed"},
	)
	if err != nil {
		t.Fatalf("NewRequest(): %v", err)
	}
	result, replayed, err := store.Execute(ctx, request, func(ctx context.Context, tx pgx.Tx) (httpidempotency.Result, error) {
		if _, err := tx.Exec(ctx, "INSERT INTO idempotency_commit_effect (value) VALUES ('committed')"); err != nil {
			return httpidempotency.Result{}, err
		}
		return httpidempotency.Result{
			Status: http.StatusCreated,
			Header: http.Header{"Content-Type": {"application/json"}},
			Body:   []byte(`{"value":"committed"}`),
		}, nil
	})
	if err != nil || !replayed || !bytes.Equal(result.Body, []byte(`{"value":"committed"}`)) {
		t.Fatalf("Execute() = %#v, replayed %v, error %v", result, replayed, err)
	}
	var effects int
	if err := pool.PGX().QueryRow(ctx, "SELECT count(*) FROM idempotency_commit_effect").Scan(&effects); err != nil {
		t.Fatalf("count effects: %v", err)
	}
	if effects != 1 {
		t.Fatalf("effects = %d, want 1", effects)
	}
}
