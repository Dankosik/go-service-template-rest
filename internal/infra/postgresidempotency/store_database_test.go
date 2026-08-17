package postgresidempotency

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/infra/postgresmigrate"
	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestStoreDatabaseLifecycle(t *testing.T) {
	t.Parallel()
	ctx, store, pool := newDatabaseStore(t)
	contract, attempt, resolver := testIdempotencyInputs(t)
	if err := store.Maintain(ctx); err != nil {
		t.Fatalf("Maintain() = %v", err)
	}

	reservation, decision, err := store.Reserve(ctx, contract, attempt, resolver)
	if err != nil || decision.Outcome != httpidempotency.OutcomeExecute {
		t.Fatalf("Reserve() = %#v, %#v, %v, want execute", reservation, decision, err)
	}
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		owned, decision, err := store.Acquire(ctx, tx, contract, reservation, resolver)
		if err != nil || decision.Outcome != httpidempotency.OutcomeExecute {
			t.Fatalf("Acquire() = %#v, %#v, %v, want execute", owned, decision, err)
		}
		return store.Complete(ctx, tx, contract, owned, httpidempotency.Result{
			Status: http.StatusCreated, MediaType: "application/json", Codec: "create/v1", Payload: []byte(`{"id":"widget-1"}`),
		})
	}); err != nil {
		t.Fatalf("complete transaction: %v", err)
	}
	if err := store.Maintain(ctx); err != nil {
		t.Fatalf("Maintain(after completion) = %v", err)
	}
	if _, decision, err = store.Reserve(ctx, contract, attempt, resolver); err != nil || decision.Outcome != httpidempotency.OutcomeReplay {
		t.Fatalf("Reserve(replay) = %#v, %v, want replay", decision, err)
	}
	if _, decision, err = store.Reconcile(ctx, contract, attempt, resolver); err != nil || decision.Outcome != httpidempotency.OutcomeReplay {
		t.Fatalf("Reconcile() = %#v, %v, want replay", decision, err)
	}
	recoveryAttempt, err := httpidempotency.NewAttempt(attempt.Scope, "key-2", attempt.Fingerprint)
	if err != nil {
		t.Fatalf("NewAttempt() = %v", err)
	}
	if _, decision, err = store.Reserve(ctx, contract, recoveryAttempt, resolver); err != nil || decision.Outcome != httpidempotency.OutcomeExecute {
		t.Fatalf("Reserve(recovery) = %#v, %v, want execute", decision, err)
	}
	recovered, decision, err := store.Reconcile(ctx, contract, recoveryAttempt, resolver)
	if err != nil || decision.Outcome != httpidempotency.OutcomeExecute || recovered.Recovery != httpidempotency.ReservationRecoveryReconciled {
		t.Fatalf("Reconcile(reserved) = %#v, %#v, %v, want reconciled execute", recovered, decision, err)
	}
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		owned, decision, err := store.Acquire(ctx, tx, contract, recovered, resolver)
		if err != nil || decision.Outcome != httpidempotency.OutcomeExecute {
			t.Fatalf("Acquire(reconciled) = %#v, %#v, %v, want execute", owned, decision, err)
		}
		return store.Complete(ctx, tx, contract, owned, httpidempotency.Result{
			Status: http.StatusCreated, MediaType: "application/json", Codec: "create/v1", Payload: []byte(`{"id":"widget-2"}`),
		})
	}); err != nil {
		t.Fatalf("complete reconciled transaction: %v", err)
	}
	conflictingFingerprint, err := httpidempotency.NewFingerprint("v1", []byte(`{"name":"different"}`))
	if err != nil {
		t.Fatalf("NewFingerprint() = %v", err)
	}
	conflictAttempt, err := httpidempotency.NewAttempt(attempt.Scope, "key-3", attempt.Fingerprint)
	if err != nil {
		t.Fatalf("NewAttempt() = %v", err)
	}
	if _, decision, err = store.Reserve(ctx, contract, conflictAttempt, resolver); err != nil || decision.Outcome != httpidempotency.OutcomeExecute {
		t.Fatalf("Reserve(conflict seed) = %#v, %v, want execute", decision, err)
	}
	conflictAttempt.Fingerprint = conflictingFingerprint
	conflictResolver := func(string) (httpidempotency.Fingerprint, error) { return conflictingFingerprint, nil }
	if _, decision, err = store.Reserve(ctx, contract, conflictAttempt, conflictResolver); err != nil || decision.Outcome != httpidempotency.OutcomeMismatch {
		t.Fatalf("Reserve(conflict) = %#v, %v, want mismatch", decision, err)
	}
	if _, decision, err = store.Reconcile(ctx, contract, conflictAttempt, conflictResolver); err != nil || decision.Outcome != httpidempotency.OutcomeIntegrityConflict {
		t.Fatalf("Reconcile(conflict) = %#v, %v, want integrity conflict", decision, err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_http_idempotency SET committed_at = clock_timestamp() - interval '3 hours' WHERE identity_token = $1`, identityBytes(attempt)); err != nil {
		t.Fatalf("age completed reservation: %v", err)
	}
	if err := store.Maintain(ctx); err != nil {
		t.Fatalf("Maintain(cleanup) = %v", err)
	}
	var retained int
	if err := pool.PGX().QueryRow(ctx, `SELECT count(*) FROM postgres_http_idempotency WHERE identity_token = $1`, identityBytes(attempt)).Scan(&retained); err != nil {
		t.Fatalf("count cleaned reservation: %v", err)
	}
	if retained != 0 {
		t.Fatalf("cleaned reservation count = %d, want 0", retained)
	}
}

func newDatabaseStore(t *testing.T) (context.Context, *Store, *postgres.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	t.Cleanup(cancel)
	container, err := tcpostgres.Run(ctx, pgtest.DefaultImage,
		tcpostgres.WithDatabase("app"), tcpostgres.WithUsername("app"), tcpostgres.WithPassword("app"),
		tcpostgres.BasicWaitStrategies(), testcontainers.WithCmd("postgres", "-c", "track_commit_timestamp=on"),
	)
	if err != nil {
		t.Skipf("start PostgreSQL: %v", err)
	}
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(container) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("PostgreSQL connection string: %v", err)
	}
	if _, err := postgresmigrate.MigrateUp(ctx, postgresmigrate.MigrationOptions{
		DSN: dsn, SourceFS: os.DirFS("../../.."), SourcePath: "migrations", ConnectTimeout: 3 * time.Second,
		StatementTimeout: time.Minute, LockTimeout: 15 * time.Second, CleanupTimeout: 15 * time.Second,
	}); err != nil {
		t.Fatalf("migrate PostgreSQL: %v", err)
	}
	pool, err := postgres.New(ctx, postgres.Options{
		DSN: dsn, ConnectTimeout: 3 * time.Second, HealthcheckTimeout: 3 * time.Second, MaxOpenConns: 4,
		AcquireTimeout: time.Second, ConnMaxLifetime: time.Hour, StatementTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("postgres.New(): %v", err)
	}
	t.Cleanup(pool.Close)
	store, err := NewStore(pool, StoreOptions{
		OwnerRecoveryDelay: time.Second, CleanupBatchSize: 10, MaxMaintenanceLag: time.Minute,
		MaxRelationBytes: 1 << 30, AdmissionHeadroomBytes: 1 << 20,
	})
	if err != nil {
		t.Fatalf("NewStore(): %v", err)
	}
	return ctx, store, pool
}
