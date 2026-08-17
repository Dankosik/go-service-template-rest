//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgresjobs"
	"github.com/example/go-service-template-rest/internal/jobs"
	"github.com/jackc/pgx/v5"
)

func TestPostgresJobsAcceptance(t *testing.T) {
	t.Parallel()
	ctx, pool, store := newPostgresJobsFixture(t)
	if _, err := pool.PGX().Exec(ctx, `
		CREATE TABLE postgres_jobs_acceptance_business (
			id text PRIMARY KEY,
			value text NOT NULL
		)`); err != nil {
		t.Fatalf("create business fixture: %v", err)
	}

	t.Run("caller transaction commits and rolls back atomically", func(t *testing.T) {
		t.Parallel()
		committed := postgresJobsPrepared(t, postgresJobsAcceptanceIdentity("commit"), "committed")
		err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, "INSERT INTO postgres_jobs_acceptance_business (id, value) VALUES ('commit', 'committed')"); err != nil {
				return fmt.Errorf("insert committed business row: %w", err)
			}
			result, err := store.Stage(ctx, tx, committed)
			if err != nil {
				return fmt.Errorf("stage committed job: %w", err)
			}
			if result.Outcome != jobs.StageNew || result.LogicalJobID != committed.Identity().LogicalJobID {
				t.Fatalf("Stage(commit) = %+v, want new %q", result, committed.Identity().LogicalJobID)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("commit business plus job: %v", err)
		}
		assertPostgresJobsAcceptanceCounts(ctx, t, pool, "commit", 1, 1)
		fingerprint := committed.Fingerprint()
		var storedFingerprint []byte
		if err := pool.PGX().QueryRow(ctx, `
			SELECT intent_fingerprint FROM postgres_jobs WHERE logical_job_id = $1`,
			string(committed.Identity().LogicalJobID),
		).Scan(&storedFingerprint); err != nil {
			t.Fatalf("read committed fingerprint: %v", err)
		}
		if !bytes.Equal(storedFingerprint, fingerprint[:]) {
			t.Fatalf("stored fingerprint = %x, want %x", storedFingerprint, fingerprint)
		}

		rolledBack := postgresJobsPrepared(t, postgresJobsAcceptanceIdentity("rollback"), "rolled-back")
		rollback := errors.New("rollback fixture")
		err = pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, "INSERT INTO postgres_jobs_acceptance_business (id, value) VALUES ('rollback', 'rolled-back')"); err != nil {
				return fmt.Errorf("insert rolled-back business row: %w", err)
			}
			result, err := store.Stage(ctx, tx, rolledBack)
			if err != nil {
				return fmt.Errorf("stage rolled-back job: %w", err)
			}
			if result.Outcome != jobs.StageNew {
				t.Fatalf("Stage(rollback) outcome = %q, want new", result.Outcome)
			}
			return rollback
		})
		if !errors.Is(err, rollback) {
			t.Fatalf("rollback business plus job error = %v, want fixture cause", err)
		}
		assertPostgresJobsAcceptanceCounts(ctx, t, pool, "rollback", 0, 0)
	})

	t.Run("staging failure rejects the business mutation", func(t *testing.T) {
		t.Parallel()
		err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, "INSERT INTO postgres_jobs_acceptance_business (id, value) VALUES ('rejected', 'rejected')"); err != nil {
				return fmt.Errorf("insert rejected business row: %w", err)
			}
			result, err := store.Stage(ctx, tx, jobs.Prepared{})
			if result.Outcome != jobs.StageRejected || !errors.Is(err, jobs.ErrInvalidAcceptance) {
				t.Fatalf("Stage(invalid) = %+v, %v; want rejected ErrInvalidAcceptance", result, err)
			}
			return fmt.Errorf("reject invalid job: %w", err)
		})
		if !errors.Is(err, jobs.ErrInvalidAcceptance) {
			t.Fatalf("invalid Stage transaction error = %v, want ErrInvalidAcceptance", err)
		}
		assertPostgresJobsAcceptanceCounts(ctx, t, pool, "rejected", 0, 0)

		storageRejected := postgresJobsPrepared(t, postgresJobsAcceptanceIdentity("storage-rejected"), "storage-rejected")
		err = pool.InTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly}, func(tx pgx.Tx) error {
			result, err := store.Stage(ctx, tx, storageRejected)
			if result.Outcome != jobs.StageRejected || err == nil {
				t.Fatalf("Stage(read-only transaction) = %+v, %v; want rejected error", result, err)
			}
			return fmt.Errorf("reject job in read-only transaction: %w", err)
		})
		if err == nil {
			t.Fatal("read-only Stage transaction error = nil")
		}
		assertPostgresJobsAcceptanceCounts(ctx, t, pool, "storage-rejected", 0, 0)
	})

	t.Run("matching intent returns the retained receipt", func(t *testing.T) {
		t.Parallel()
		prepared := postgresJobsPrepared(t, postgresJobsAcceptanceIdentity("duplicate"), "same")
		mustStagePostgresJob(ctx, t, pool, store, prepared)
		rollback := errors.New("duplicate business operation has no matching receipt")
		err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, "INSERT INTO postgres_jobs_acceptance_business (id, value) VALUES ('duplicate', 'must-not-commit')"); err != nil {
				return fmt.Errorf("insert duplicate business row: %w", err)
			}
			result, err := store.Stage(ctx, tx, prepared)
			if err != nil {
				return fmt.Errorf("stage duplicate job: %w", err)
			}
			if result.Outcome != jobs.StageExisting || result.LogicalJobID != prepared.Identity().LogicalJobID {
				t.Fatalf("Stage(duplicate) = %+v, want existing %q", result, prepared.Identity().LogicalJobID)
			}
			return rollback
		})
		if !errors.Is(err, rollback) {
			t.Fatalf("duplicate transaction error = %v, want caller rollback", err)
		}
		assertPostgresJobsAcceptanceCounts(ctx, t, pool, "duplicate", 0, 1)
	})

	t.Run("every conflicting identity leaves the retained receipt unchanged", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name   string
			change func(*jobs.AcceptanceIdentity)
		}{
			{name: "producer", change: func(i *jobs.AcceptanceIdentity) {
				i.LogicalJobID += "-other"
				i.OccurrenceID += "-other"
				i.EffectKey += "-other"
			}},
			{name: "logical", change: func(i *jobs.AcceptanceIdentity) {
				i.ProducerKey += "-other"
				i.OccurrenceID += "-other"
				i.EffectKey += "-other"
			}},
			{name: "occurrence", change: func(i *jobs.AcceptanceIdentity) {
				i.LogicalJobID += "-other"
				i.ProducerKey += "-other"
				i.EffectKey += "-other"
			}},
			{name: "effect", change: func(i *jobs.AcceptanceIdentity) {
				i.LogicalJobID += "-other"
				i.ProducerKey += "-other"
				i.OccurrenceID += "-other"
			}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				seedIdentity := postgresJobsAcceptanceIdentity("conflict-" + tc.name)
				seed := postgresJobsPrepared(t, seedIdentity, "retained")
				mustStagePostgresJob(ctx, t, pool, store, seed)
				before := readPostgresJobsAcceptanceReceipt(ctx, t, pool, seedIdentity.LogicalJobID)
				candidateIdentity := seedIdentity
				tc.change(&candidateIdentity)
				candidate := postgresJobsPrepared(t, candidateIdentity, "different")
				rollback := errors.New("identity conflict")
				err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
					result, err := store.Stage(ctx, tx, candidate)
					if err != nil {
						return fmt.Errorf("stage conflicting job: %w", err)
					}
					if result.Outcome != jobs.StageConflict || result.LogicalJobID != seedIdentity.LogicalJobID {
						t.Fatalf("Stage(conflict) = %+v, want conflict with %q", result, seedIdentity.LogicalJobID)
					}
					return rollback
				})
				if !errors.Is(err, rollback) {
					t.Fatalf("conflict transaction error = %v, want caller rollback", err)
				}
				var count int
				if err := pool.PGX().QueryRow(ctx, `SELECT count(*) FROM postgres_jobs
					WHERE logical_job_id = $1 OR logical_job_id = $2`,
					string(seedIdentity.LogicalJobID), string(candidateIdentity.LogicalJobID),
				).Scan(&count); err != nil {
					t.Fatalf("count conflict receipt: %v", err)
				}
				after := readPostgresJobsAcceptanceReceipt(ctx, t, pool, seedIdentity.LogicalJobID)
				if count != 1 || after != before {
					t.Fatalf("conflict receipt = count %d %+v, want one unchanged %+v", count, after, before)
				}
			})
		}
	})

	t.Run("writer readback resolves every closed outcome", func(t *testing.T) {
		t.Parallel()
		accepted := postgresJobsPrepared(t, postgresJobsAcceptanceIdentity("readback"), "accepted")
		mustStagePostgresJob(ctx, t, pool, store, accepted)
		result, err := store.ResolveAcceptance(ctx, accepted.ReadbackExpectation())
		if err != nil || result.Outcome != jobs.ReadbackAccepted || result.LogicalJobID != accepted.Identity().LogicalJobID {
			t.Fatalf("ResolveAcceptance(accepted) = %+v, %v", result, err)
		}

		absent := postgresJobsPrepared(t, postgresJobsAcceptanceIdentity("absent"), "absent").ReadbackExpectation()
		result, err = store.ResolveAcceptance(ctx, absent)
		if err != nil || result.Outcome != jobs.ReadbackNotAccepted {
			t.Fatalf("ResolveAcceptance(absent) = %+v, %v", result, err)
		}

		conflicting := accepted.Identity()
		conflicting.LogicalJobID += "-other"
		conflicting.OccurrenceID += "-other"
		conflicting.EffectKey += "-other"
		identityConflictBefore := readPostgresJobsAcceptanceReceipt(ctx, t, pool, accepted.Identity().LogicalJobID)
		result, err = store.ResolveAcceptance(ctx, postgresJobsPrepared(t, conflicting, "conflicting identity").ReadbackExpectation())
		if err != nil || result.Outcome != jobs.ReadbackConflict || result.LogicalJobID != accepted.Identity().LogicalJobID {
			t.Fatalf("ResolveAcceptance(conflict) = %+v, %v", result, err)
		}
		var count int
		if err := pool.PGX().QueryRow(ctx, `SELECT count(*) FROM postgres_jobs
			WHERE producer_scope = $1 AND producer_key = $2`,
			string(accepted.Identity().ProducerScope), string(accepted.Identity().ProducerKey),
		).Scan(&count); err != nil {
			t.Fatalf("count retained acceptance after identity conflict: %v", err)
		}
		identityConflictAfter := readPostgresJobsAcceptanceReceipt(ctx, t, pool, accepted.Identity().LogicalJobID)
		if count != 1 || identityConflictAfter != identityConflictBefore {
			t.Fatalf("retained acceptance after identity conflict = count %d %+v, want one unchanged %+v", count, identityConflictAfter, identityConflictBefore)
		}

		differentIntent := postgresJobsPrepared(t, accepted.Identity(), "different intent")
		before := readPostgresJobsAcceptanceReceipt(ctx, t, pool, accepted.Identity().LogicalJobID)
		result, err = store.ResolveAcceptance(ctx, differentIntent.ReadbackExpectation())
		if err != nil || result.Outcome != jobs.ReadbackConflict || result.LogicalJobID != accepted.Identity().LogicalJobID {
			t.Fatalf("ResolveAcceptance(different intent) = %+v, %v", result, err)
		}
		count = 0
		if err := pool.PGX().QueryRow(ctx, `SELECT count(*) FROM postgres_jobs
			WHERE producer_scope = $1 AND producer_key = $2`,
			string(accepted.Identity().ProducerScope), string(accepted.Identity().ProducerKey),
		).Scan(&count); err != nil {
			t.Fatalf("count retained acceptance after intent conflict: %v", err)
		}
		after := readPostgresJobsAcceptanceReceipt(ctx, t, pool, accepted.Identity().LogicalJobID)
		if count != 1 || after != before {
			t.Fatalf("retained acceptance after intent conflict = count %d %+v, want one unchanged %+v", count, after, before)
		}

		dsn := postgresJobsDSN(pool)
		readOnlyPool, readOnlyStore := newPostgresJobsStore(ctx, t, postgresJobsDSNParam(t, dsn, "default_transaction_read_only", "on"), 2, 100*time.Millisecond)
		defer readOnlyPool.Close()
		result, err = readOnlyStore.ResolveAcceptance(ctx, absent)
		if result.Outcome != jobs.ReadbackUnknown || err == nil {
			t.Fatalf("ResolveAcceptance(read-only) = %+v, %v; want unknown error", result, err)
		}

		unavailablePool, unavailableStore := newPostgresJobsStore(ctx, t, dsn, 2, 100*time.Millisecond)
		unavailablePool.Close()
		result, err = unavailableStore.ResolveAcceptance(ctx, absent)
		if result.Outcome != jobs.ReadbackUnknown || err == nil {
			t.Fatalf("ResolveAcceptance(unavailable) = %+v, %v; want unknown error", result, err)
		}

		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		result, err = store.ResolveAcceptance(cancelled, absent)
		if result.Outcome != jobs.ReadbackUnknown || !errors.Is(err, context.Canceled) {
			t.Fatalf("ResolveAcceptance(cancelled) = %+v, %v; want unknown context.Canceled", result, err)
		}

		expired, expire := context.WithDeadline(ctx, time.Now().Add(-time.Second))
		defer expire()
		result, err = store.ResolveAcceptance(expired, absent)
		if result.Outcome != jobs.ReadbackUnknown || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("ResolveAcceptance(expired) = %+v, %v; want unknown context.DeadlineExceeded", result, err)
		}
	})
}

type postgresJobsAcceptanceArgs struct {
	Value string `json:"value"`
}

type postgresJobsAcceptanceReceipt struct {
	LogicalJobID    string
	ProducerScope   string
	ProducerKey     string
	OccurrenceScope string
	OccurrenceID    string
	EffectScope     string
	EffectKey       string
	Fingerprint     string
}

func readPostgresJobsAcceptanceReceipt(
	ctx context.Context,
	t *testing.T,
	pool *postgres.Pool,
	logicalJobID jobs.LogicalJobID,
) postgresJobsAcceptanceReceipt {
	t.Helper()
	var receipt postgresJobsAcceptanceReceipt
	var fingerprint []byte
	if err := pool.PGX().QueryRow(ctx, `SELECT
		logical_job_id, producer_scope, producer_key, occurrence_scope,
		occurrence_id, effect_scope, effect_key, intent_fingerprint
		FROM postgres_jobs WHERE logical_job_id = $1`, string(logicalJobID)).Scan(
		&receipt.LogicalJobID, &receipt.ProducerScope, &receipt.ProducerKey,
		&receipt.OccurrenceScope, &receipt.OccurrenceID, &receipt.EffectScope,
		&receipt.EffectKey, &fingerprint,
	); err != nil {
		t.Fatalf("read postgres jobs acceptance receipt: %v", err)
	}
	receipt.Fingerprint = string(fingerprint)
	return receipt
}

func postgresJobsPrepared(t *testing.T, identity jobs.AcceptanceIdentity, value string) jobs.Prepared {
	t.Helper()
	definition, err := jobs.NewDefinition(jobs.DefinitionInput[postgresJobsAcceptanceArgs]{
		Revision:        jobs.Revision{Kind: "acceptance", ArgsVersion: "v1", PolicyVersion: "v1"},
		MaxPayloadBytes: 1024,
		Validate: func(args postgresJobsAcceptanceArgs) error {
			if args.Value == "" {
				return errors.New("value is required")
			}
			return nil
		},
		Policy: jobs.Policy{
			Effect: jobs.EffectPolicy{AmbiguousAction: jobs.AmbiguousEffectOutcomeUnknown},
			Retry: jobs.RetryPolicy{
				MaxAttempts: 2, MaxElapsed: time.Hour, InitialBackoff: time.Second,
				MaxBackoff: time.Minute, HintPolicy: jobs.RetryHintIgnore, Jitter: jobs.JitterNone,
				MaxRecoveryWave: 1,
			},
			Recovery: jobs.RecoveryPolicy{
				Mode: jobs.RecoveryUnavailable, Attempts: jobs.BudgetPreserved, Elapsed: jobs.BudgetPreserved,
			},
			MaxAttemptDuration: time.Minute, TerminationEnvelope: 2 * time.Minute,
		},
	})
	if err != nil {
		t.Fatalf("jobs.NewDefinition(): %v", err)
	}
	prepared, err := definition.Prepare(
		postgresJobsAcceptanceArgs{Value: value},
		identity,
		time.Date(2030, time.January, 2, 3, 4, 5, 6000, time.UTC),
	)
	if err != nil {
		t.Fatalf("Definition.Prepare(): %v", err)
	}
	return prepared
}

func postgresJobsAcceptanceIdentity(suffix string) jobs.AcceptanceIdentity {
	return jobs.AcceptanceIdentity{
		LogicalJobID:    jobs.LogicalJobID("job-" + suffix),
		ProducerScope:   "acceptance",
		ProducerKey:     jobs.ProducerKey("producer-" + suffix),
		OccurrenceScope: "acceptance",
		OccurrenceID:    jobs.OccurrenceID("occurrence-" + suffix),
		EffectScope:     "acceptance",
		EffectKey:       jobs.EffectKey("effect-" + suffix),
	}
}

func mustStagePostgresJob(
	ctx context.Context,
	t *testing.T,
	pool *postgres.Pool,
	store *postgresjobs.Store,
	prepared jobs.Prepared,
) {
	t.Helper()
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		result, err := store.Stage(ctx, tx, prepared)
		if err == nil && result.Outcome != jobs.StageNew {
			t.Fatalf("Stage() outcome = %q, want new", result.Outcome)
		}
		if err != nil {
			return fmt.Errorf("stage postgres job: %w", err)
		}
		return nil
	}); err != nil {
		t.Fatalf("stage postgres job: %v", err)
	}
}

func assertPostgresJobsAcceptanceCounts(
	ctx context.Context,
	t *testing.T,
	pool *postgres.Pool,
	businessID string,
	wantBusiness int,
	wantJobs int,
) {
	t.Helper()
	var businessRows, jobRows int
	if err := pool.PGX().QueryRow(ctx, "SELECT count(*) FROM postgres_jobs_acceptance_business WHERE id = $1", businessID).Scan(&businessRows); err != nil {
		t.Fatalf("count business rows: %v", err)
	}
	if err := pool.PGX().QueryRow(ctx, "SELECT count(*) FROM postgres_jobs WHERE producer_key = $1", "producer-"+businessID).Scan(&jobRows); err != nil {
		t.Fatalf("count job rows: %v", err)
	}
	if businessRows != wantBusiness || jobRows != wantJobs {
		t.Fatalf("business/jobs rows = %d/%d, want %d/%d", businessRows, jobRows, wantBusiness, wantJobs)
	}
}
