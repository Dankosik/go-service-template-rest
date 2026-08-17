//go:build integration

package referenceservice_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/jackc/pgx/v5"
)

// reconciliationAdapter is the concrete repository pattern for a write whose
// commit response can be lost. The stable event exists before InTx, and only a
// writer-primary CommitNotApplied result can send the mutation again.
type reconciliationAdapter struct {
	pool  *postgres.Pool
	store *postgresoutbox.Store
	inTx  func(context.Context, pgx.TxOptions, func(pgx.Tx) error) error
}

func (adapter reconciliationAdapter) create(ctx context.Context, mutationID, eventID string) error {
	event := postgresoutbox.Event{
		ID:          eventID,
		Type:        "article.created",
		Source:      "reference-service",
		Destination: "articles.events",
		Schema:      "v1",
		OccurredAt:  time.Now().UTC(),
		Payload:     []byte(fmt.Sprintf(`{"id":%q}`, mutationID)),
	}

mutation:
	for {
		err := adapter.inTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx,
				"INSERT INTO reconciliation_writes (id, event_id) VALUES ($1, $2)",
				mutationID, event.ID,
			); err != nil {
				return err
			}
			return adapter.store.Append(ctx, tx, event)
		})
		if err == nil {
			return nil
		}
		if !errors.Is(err, postgres.ErrCommitUnknown) {
			return err
		}

		for {
			outcome, reconcileErr := adapter.store.ReconcileCommit(ctx, event)
			switch outcome {
			case postgresoutbox.CommitApplied:
				return nil
			case postgresoutbox.CommitNotApplied:
				continue mutation
			case postgresoutbox.CommitStillUnknown:
				if errors.Is(reconcileErr, postgresoutbox.ErrReceiptConflict) {
					return reconcileErr
				}
				if ctx.Err() != nil {
					return errors.Join(ctx.Err(), reconcileErr)
				}
			}
		}
	}
}

func TestPostgresOutboxWriterPrimaryReconciliation(t *testing.T) {
	t.Run("committed result lost", func(t *testing.T) {
		adapter := newReconciliationAdapter(t)
		realInTx := adapter.inTx
		calls := 0
		adapter.inTx = func(ctx context.Context, opts pgx.TxOptions, fn func(pgx.Tx) error) error {
			calls++
			if err := realInTx(ctx, opts, fn); err != nil {
				return err
			}
			return fmt.Errorf("%w: commit result lost", postgres.ErrCommitUnknown)
		}

		if err := adapter.create(t.Context(), "committed", "committed-event"); err != nil {
			t.Fatalf("create() error = %v", err)
		}
		assertReconciliationCounts(t, adapter.pool, "committed", "committed-event", 1, 1)
		if calls != 1 {
			t.Fatalf("mutation transaction calls = %d, want 1", calls)
		}
	})

	t.Run("authoritative absence retries same event", func(t *testing.T) {
		adapter := newReconciliationAdapter(t)
		realInTx := adapter.inTx
		calls := 0
		adapter.inTx = func(ctx context.Context, opts pgx.TxOptions, fn func(pgx.Tx) error) error {
			calls++
			if calls == 1 {
				return fmt.Errorf("%w: connection lost before commit", postgres.ErrCommitUnknown)
			}
			return realInTx(ctx, opts, fn)
		}

		if err := adapter.create(t.Context(), "retried", "retried-event"); err != nil {
			t.Fatalf("create() error = %v", err)
		}
		assertReconciliationCounts(t, adapter.pool, "retried", "retried-event", 1, 1)
		if calls != 2 {
			t.Fatalf("mutation transaction calls = %d, want 2", calls)
		}
	})

	t.Run("conflicting receipt fails without mutation", func(t *testing.T) {
		adapter := newReconciliationAdapter(t)
		if _, err := adapter.pool.PGX().Exec(t.Context(), `
			INSERT INTO outbox_commit_receipts (event_id, fingerprint_version, envelope_fingerprint)
			VALUES ('conflict-event', 1, decode(repeat('01', 32), 'hex'))`); err != nil {
			t.Fatalf("seed conflicting receipt: %v", err)
		}
		calls := 0
		adapter.inTx = func(context.Context, pgx.TxOptions, func(pgx.Tx) error) error {
			calls++
			return fmt.Errorf("%w: connection lost", postgres.ErrCommitUnknown)
		}

		err := adapter.create(t.Context(), "conflict", "conflict-event")
		if !errors.Is(err, postgresoutbox.ErrReceiptConflict) {
			t.Fatalf("create() error = %v, want ErrReceiptConflict", err)
		}
		assertReconciliationCounts(t, adapter.pool, "conflict", "conflict-event", 0, 1)
		if calls != 1 {
			t.Fatalf("mutation transaction calls = %d, want 1", calls)
		}
	})

	t.Run("inconclusive read never retries mutation", func(t *testing.T) {
		adapter := newReconciliationAdapter(t)
		calls := 0
		adapter.inTx = func(context.Context, pgx.TxOptions, func(pgx.Tx) error) error {
			calls++
			return fmt.Errorf("%w: connection lost", postgres.ErrCommitUnknown)
		}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err := adapter.create(ctx, "unknown", "unknown-event")
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("create() error = %v, want context.Canceled", err)
		}
		assertReconciliationCounts(t, adapter.pool, "unknown", "unknown-event", 0, 0)
		if calls != 1 {
			t.Fatalf("mutation transaction calls = %d, want 1", calls)
		}
	})
}

func newReconciliationAdapter(t *testing.T) reconciliationAdapter {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	dsn := pgtest.Migrated(t, os.DirFS("../.."), "migrations")
	pool, err := postgres.New(ctx, postgres.Options{
		DSN:                dsn,
		ConnectTimeout:     3 * time.Second,
		HealthcheckTimeout: 3 * time.Second,
		MaxOpenConns:       4,
		AcquireTimeout:     time.Second,
		ConnMaxLifetime:    time.Hour,
		StatementTimeout:   5 * time.Second,
	})
	if err != nil {
		t.Fatalf("postgres.New() error = %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.PGX().Exec(ctx, `
		CREATE TABLE reconciliation_writes (
			id text PRIMARY KEY,
			event_id text NOT NULL
		)`); err != nil {
		t.Fatalf("create reconciliation fixture: %v", err)
	}
	store, err := postgresoutbox.NewStore(pool, nil)
	if err != nil {
		t.Fatalf("postgresoutbox.NewStore() error = %v", err)
	}
	return reconciliationAdapter{pool: pool, store: store, inTx: pool.InTx}
}

func assertReconciliationCounts(
	t *testing.T,
	pool *postgres.Pool,
	mutationID, eventID string,
	wantMutations, wantReceipts int,
) {
	t.Helper()
	var mutations, receipts int
	if err := pool.PGX().QueryRow(t.Context(), `
		SELECT
			(SELECT count(*) FROM reconciliation_writes WHERE id = $1 AND event_id = $2),
			(SELECT count(*) FROM outbox_commit_receipts WHERE event_id = $2)`,
		mutationID, eventID,
	).Scan(&mutations, &receipts); err != nil {
		t.Fatalf("count reconciliation state: %v", err)
	}
	if mutations != wantMutations || receipts != wantReceipts {
		t.Fatalf("mutation/receipt counts = %d/%d, want %d/%d", mutations, receipts, wantMutations, wantReceipts)
	}
}
