//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresStatementTimeoutIsPublishedServerSide(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	pool := mustOpenBoundedPool(t, ctx)
	var got string
	if err := pool.QueryRow(ctx, "SHOW statement_timeout").Scan(&got); err != nil {
		t.Fatalf("SHOW statement_timeout error = %v", err)
	}
	if got != "8s" {
		t.Fatalf("statement_timeout = %q, want 8s", got)
	}
}

// TestPostgresInTxRollsBackOnError proves the transaction seam actually isolates
// work, so a service composing two repository calls does not manage commit or rollback.
func TestPostgresInTxRollsBackOnError(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	pool := mustOpenBoundedPool(t, ctx)

	if _, err := pool.Exec(ctx, "CREATE TABLE IF NOT EXISTS intx_probe (id int primary key)"); err != nil {
		t.Fatalf("create probe table: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.WithoutCancel(ctx), "DROP TABLE IF EXISTS intx_probe")
	})

	sentinel := errors.New("business rule rejected the batch")
	err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "INSERT INTO intx_probe (id) VALUES (1)"); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("InTx() error = %v, want wrapped %v", err, sentinel)
	}
	if !errors.Is(err, postgres.ErrTransaction) {
		t.Fatalf("InTx() error = %v, want wrapped ErrTransaction", err)
	}

	var rows int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM intx_probe").Scan(&rows); err != nil {
		t.Fatalf("count probe rows: %v", err)
	}
	if rows != 0 {
		t.Fatalf("probe rows after rollback = %d, want 0", rows)
	}

	if err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		_, execErr := tx.Exec(ctx, "INSERT INTO intx_probe (id) VALUES (2)")
		return execErr
	}); err != nil {
		t.Fatalf("InTx() commit error = %v", err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM intx_probe").Scan(&rows); err != nil {
		t.Fatalf("count probe rows: %v", err)
	}
	if rows != 1 {
		t.Fatalf("probe rows after commit = %d, want 1", rows)
	}
}

// TestPostgresRetryableRecognizesSerializationFailure proves the classifier
// against a real conflict rather than a synthesized error code.
func TestPostgresRetryableRecognizesSerializationFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	pool := mustOpenBoundedPool(t, ctx)

	if _, err := pool.Exec(ctx, "CREATE TABLE IF NOT EXISTS retry_probe (id int primary key, total int)"); err != nil {
		t.Fatalf("create probe table: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.WithoutCancel(ctx), "DROP TABLE IF EXISTS retry_probe")
	})
	if _, err := pool.Exec(ctx, "INSERT INTO retry_probe (id, total) VALUES (1, 0) ON CONFLICT DO NOTHING"); err != nil {
		t.Fatalf("seed probe row: %v", err)
	}

	first, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatalf("begin first transaction: %v", err)
	}
	defer func() { _ = first.Rollback(context.WithoutCancel(ctx)) }()

	second, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatalf("begin second transaction: %v", err)
	}
	defer func() { _ = second.Rollback(context.WithoutCancel(ctx)) }()

	// Both transactions read the same row and then write it, which is the
	// canonical serializable conflict.
	var seen int
	if err := first.QueryRow(ctx, "SELECT total FROM retry_probe WHERE id = 1").Scan(&seen); err != nil {
		t.Fatalf("first read: %v", err)
	}
	if err := second.QueryRow(ctx, "SELECT total FROM retry_probe WHERE id = 1").Scan(&seen); err != nil {
		t.Fatalf("second read: %v", err)
	}
	if _, err := first.Exec(ctx, "UPDATE retry_probe SET total = total + 1 WHERE id = 1"); err != nil {
		t.Fatalf("first update: %v", err)
	}
	if err := first.Commit(ctx); err != nil {
		t.Fatalf("first commit: %v", err)
	}

	_, updateErr := second.Exec(ctx, "UPDATE retry_probe SET total = total + 1 WHERE id = 1")
	commitErr := second.Commit(ctx)
	conflict := errors.Join(updateErr, commitErr)
	if conflict == nil {
		t.Fatal("second transaction succeeded, want a serialization failure")
	}
	if !postgres.Retryable(conflict) {
		t.Fatalf("Retryable(%v) = false, want true for a serialization failure", conflict)
	}
}

func mustOpenBoundedPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	pool, err := postgres.Open(ctx, postgres.Options{
		DSN: pgtest.DSN(t),

		MaxOpenConns: 4,
	})
	if err != nil {
		t.Fatalf("create postgres pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
