package postgresjobs

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/infra/postgresmigrate"
	"github.com/example/go-service-template-rest/internal/jobs"
	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestStoreDatabaseAcceptanceAndClaim(t *testing.T) {
	ctx, store, pool := newDatabaseJobsStore(t)
	if err := store.CheckSchema(ctx); err != nil {
		t.Fatalf("CheckSchema() = %v", err)
	}
	prepared := databasePreparedJob(t, "1", "value", "v1")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		result, err := store.Stage(ctx, tx, prepared)
		if err != nil || result.Outcome != jobs.StageNew {
			t.Fatalf("Stage() = %#v, %v, want new", result, err)
		}
		return err
	}); err != nil {
		t.Fatalf("stage job: %v", err)
	}
	if result, err := store.ResolveAcceptance(ctx, prepared.ReadbackExpectation()); err != nil || result.Outcome != jobs.ReadbackAccepted {
		t.Fatalf("ResolveAcceptance() = %#v, %v, want accepted", result, err)
	}
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		result, err := store.Stage(ctx, tx, prepared)
		if err != nil || result.Outcome != jobs.StageExisting {
			t.Fatalf("Stage(duplicate) = %#v, %v, want existing", result, err)
		}
		return err
	}); err != nil {
		t.Fatalf("stage duplicate job: %v", err)
	}
	conflicting := databasePreparedJob(t, "1", "different", "v1")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		result, err := store.Stage(ctx, tx, conflicting)
		if err != nil || result.Outcome != jobs.StageConflict {
			t.Fatalf("Stage(conflict) = %#v, %v, want conflict", result, err)
		}
		return err
	}); err != nil {
		t.Fatalf("stage conflicting job: %v", err)
	}
	if result, err := store.ResolveAcceptance(ctx, conflicting.ReadbackExpectation()); err != nil || result.Outcome != jobs.ReadbackConflict {
		t.Fatalf("ResolveAcceptance(conflict) = %#v, %v, want conflict", result, err)
	}
	session, err := store.AcquireSession(ctx)
	if err != nil {
		t.Fatalf("AcquireSession() = %v", err)
	}
	defer session.Release(ctx)
	revision := prepared.Revision()
	claim, err := session.Claim(ctx, ClaimOptions{RegistryKeys: []jobs.Revision{revision}, WorkerID: "worker-1", Limit: 1, LeaseDuration: time.Second})
	if err != nil || len(claim.Attempts) != 1 {
		t.Fatalf("Claim() = %#v, %v, want one attempt", claim, err)
	}
	if observation, err := session.Observe(ctx, []jobs.Revision{revision}); err != nil || !observation.Compatible {
		t.Fatalf("Observe() = %#v, %v, want compatible observation", observation, err)
	}
	resolved, err := session.ResolveClaims(ctx, []AttemptIdentity{claim.Attempts[0].Attempt})
	if err != nil || len(resolved) != 1 || !resolved[0].Committed {
		t.Fatalf("ResolveClaims() = %#v, %v, want committed attempt", resolved, err)
	}
	if result, err := session.Finalize(ctx, FinalizeInput{
		Attempt:    claim.Attempts[0].Attempt,
		Transition: jobs.Transition{State: jobs.StateSucceeded, AttemptsUsed: 1, Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectNone},
	}); err != nil || result.Status != TransitionApplied {
		t.Fatalf("Finalize() = %#v, %v, want applied transition", result, err)
	}
	if err := store.CheckProducerPath(ctx); err != nil {
		t.Fatalf("CheckProducerPath() = %v", err)
	}
	unsupported := databasePreparedJob(t, "2", "value", "v2")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		_, err := store.Stage(ctx, tx, unsupported)
		return err
	}); err != nil {
		t.Fatalf("stage unsupported job: %v", err)
	}
	if _, err := session.Claim(ctx, ClaimOptions{RegistryKeys: []jobs.Revision{revision}, WorkerID: "worker-1", Limit: 1, LeaseDuration: time.Second}); !errors.Is(err, jobs.ErrUnsupportedRevision) {
		t.Fatalf("Claim(unsupported revision) = %v, want ErrUnsupportedRevision", err)
	}
}

func newDatabaseJobsStore(t *testing.T) (context.Context, *Store, *postgres.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	t.Cleanup(cancel)
	container, err := tcpostgres.Run(ctx, pgtest.DefaultImage,
		tcpostgres.WithDatabase("app"), tcpostgres.WithUsername("app"), tcpostgres.WithPassword("app"), tcpostgres.BasicWaitStrategies(),
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
	store, err := NewStore(pool, StoreOptions{OperationTimeout: 3 * time.Second, StatementTimeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewStore(): %v", err)
	}
	return ctx, store, pool
}

func databasePreparedJob(t *testing.T, suffix, value, policyVersion string) jobs.Prepared {
	t.Helper()
	definition, err := jobs.NewDefinition(jobs.DefinitionInput[struct{ Value string }]{
		Revision: jobs.Revision{Kind: "acceptance", ArgsVersion: "v1", PolicyVersion: policyVersion}, MaxPayloadBytes: 1024,
		Validate: func(args struct{ Value string }) error {
			if args.Value == "" {
				return errors.New("value is required")
			}
			return nil
		},
		Policy: jobs.Policy{
			Effect:             jobs.EffectPolicy{AmbiguousAction: jobs.AmbiguousEffectOutcomeUnknown},
			Retry:              jobs.RetryPolicy{MaxAttempts: 2, MaxElapsed: time.Hour, InitialBackoff: time.Second, MaxBackoff: time.Minute, HintPolicy: jobs.RetryHintIgnore, Jitter: jobs.JitterNone, MaxRecoveryWave: 1},
			Recovery:           jobs.RecoveryPolicy{Mode: jobs.RecoveryUnavailable, Attempts: jobs.BudgetPreserved, Elapsed: jobs.BudgetPreserved},
			MaxAttemptDuration: time.Minute, TerminationEnvelope: 2 * time.Minute,
		},
	})
	if err != nil {
		t.Fatalf("NewDefinition() = %v", err)
	}
	prepared, err := definition.Prepare(struct{ Value string }{Value: value}, jobs.AcceptanceIdentity{
		LogicalJobID: jobs.LogicalJobID("job-" + suffix), ProducerScope: "acceptance", ProducerKey: jobs.ProducerKey("producer-" + suffix), OccurrenceScope: "acceptance", OccurrenceID: jobs.OccurrenceID("occurrence-" + suffix), EffectScope: "acceptance", EffectKey: jobs.EffectKey("effect-" + suffix),
	}, time.Now().UTC().Add(-time.Minute))
	if err != nil {
		t.Fatalf("Prepare() = %v", err)
	}
	return prepared
}
