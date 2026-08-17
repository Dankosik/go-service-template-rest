//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgresjobs"
	"github.com/example/go-service-template-rest/internal/jobs"
)

func TestPostgresJobsClaim(t *testing.T) {
	ctx, pool, store := newPostgresJobsFixture(t)
	keys := []jobs.Revision{{Kind: "acceptance", ArgsVersion: "v1", PolicyVersion: "v1"}}

	t.Run("required revision coverage fails closed in the claim snapshot", func(t *testing.T) {
		unknown := stageDuePostgresJob(ctx, t, pool, store, "claim-unknown")
		if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_jobs SET kind = 'unknown' WHERE logical_job_id = $1`, string(unknown.Identity().LogicalJobID)); err != nil {
			t.Fatalf("make required revision unknown: %v", err)
		}
		session := acquirePostgresJobsSession(ctx, t, store)
		defer session.Release(ctx)
		result, err := session.Claim(ctx, postgresjobs.ClaimOptions{RegistryKeys: keys, WorkerID: "worker-coverage", Limit: 1, LeaseDuration: time.Minute})
		if !errors.Is(err, jobs.ErrUnsupportedRevision) || len(result.Attempts) != 0 {
			t.Fatalf("Claim(unknown revision) = %+v, %v; want no claims and ErrUnsupportedRevision", result, err)
		}
		assertPostgresJobsAttemptCount(ctx, t, pool, unknown.Identity().LogicalJobID, 0)
		if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_jobs SET state = 'permanent', terminal_at = clock_timestamp() WHERE logical_job_id = $1`, string(unknown.Identity().LogicalJobID)); err != nil {
			t.Fatalf("make unknown revision terminal history: %v", err)
		}
		result, err = session.Claim(ctx, postgresjobs.ClaimOptions{RegistryKeys: keys, WorkerID: "worker-history", Limit: 1, LeaseDuration: time.Minute})
		if err != nil || len(result.Attempts) != 0 {
			t.Fatalf("Claim(terminal unknown revision) = %+v, %v; want compatible empty claim", result, err)
		}
		if _, err := pool.PGX().Exec(ctx, `DELETE FROM postgres_jobs WHERE logical_job_id = $1`, string(unknown.Identity().LogicalJobID)); err != nil {
			t.Fatalf("remove unknown revision fixture: %v", err)
		}
	})

	t.Run("a revision committed after one snapshot closes the next claim", func(t *testing.T) {
		due := stageDuePostgresJob(ctx, t, pool, store, "claim-before-late-revision")
		late := postgresJobsPrepared(t, postgresJobsAcceptanceIdentity("claim-late-revision"), "late")
		tx, err := pool.PGX().Begin(ctx)
		if err != nil {
			t.Fatalf("begin late revision transaction: %v", err)
		}
		t.Cleanup(func() { _ = tx.Rollback(context.WithoutCancel(ctx)) })
		if result, stageErr := store.Stage(ctx, tx, late); stageErr != nil || result.Outcome != jobs.StageNew {
			t.Fatalf("stage late revision = %+v, %v", result, stageErr)
		}
		if _, err := tx.Exec(ctx, `UPDATE postgres_jobs SET kind = 'late' WHERE logical_job_id = $1`, string(late.Identity().LogicalJobID)); err != nil {
			t.Fatalf("make late revision unknown: %v", err)
		}
		session := acquirePostgresJobsSession(ctx, t, store)
		defer session.Release(ctx)
		first, err := session.Claim(ctx, postgresjobs.ClaimOptions{RegistryKeys: keys, WorkerID: "worker-before-late", Limit: 1, LeaseDuration: time.Minute})
		if err != nil || len(first.Attempts) != 1 || first.Attempts[0].Identity.LogicalJobID != due.Identity().LogicalJobID {
			t.Fatalf("Claim(before late commit) = %+v, %v", first, err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit late revision: %v", err)
		}
		second, err := session.Claim(ctx, postgresjobs.ClaimOptions{RegistryKeys: keys, WorkerID: "worker-after-late", Limit: 1, LeaseDuration: time.Minute})
		if !errors.Is(err, jobs.ErrUnsupportedRevision) || len(second.Attempts) != 0 {
			t.Fatalf("Claim(after late commit) = %+v, %v; want closed admission", second, err)
		}
		if _, err := pool.PGX().Exec(ctx, `DELETE FROM postgres_jobs WHERE logical_job_id = $1`, string(late.Identity().LogicalJobID)); err != nil {
			t.Fatalf("remove late revision fixture: %v", err)
		}
	})

	t.Run("pause and claim serialize on the neutral scope", func(t *testing.T) {
		prepared := stageDuePostgresJob(ctx, t, pool, store, "claim-paused")
		pauseConn, err := pool.PGX().Acquire(ctx)
		if err != nil {
			t.Fatalf("acquire pause connection: %v", err)
		}
		defer pauseConn.Release()
		pauseTx, err := pauseConn.Begin(ctx)
		if err != nil {
			t.Fatalf("begin pause transaction: %v", err)
		}
		defer func() { _ = pauseTx.Rollback(context.WithoutCancel(ctx)) }()
		if _, err := pauseTx.Exec(ctx, `UPDATE postgres_job_claim_scopes SET paused = true, scope_generation = scope_generation + 1 WHERE work_class = 'neutral'`); err != nil {
			t.Fatalf("pause scope: %v", err)
		}
		session := acquirePostgresJobsSession(ctx, t, store)
		defer session.Release(ctx)
		claimDone := make(chan struct{})
		var claimResult postgresjobs.ClaimResult
		var claimErr error
		go func() {
			defer close(claimDone)
			claimResult, claimErr = session.Claim(ctx, postgresjobs.ClaimOptions{RegistryKeys: keys, WorkerID: "worker-paused", Limit: 1, LeaseDuration: time.Minute})
		}()
		waitForPostgresJobsBlocker(ctx, t, pool, session.BackendPID(), pauseConn.Conn().PgConn().PID())
		if err := pauseTx.Commit(ctx); err != nil {
			t.Fatalf("commit pause: %v", err)
		}
		select {
		case <-claimDone:
		case <-ctx.Done():
			t.Fatalf("wait for paused claim: %v", ctx.Err())
		}
		if claimErr != nil || !claimResult.Paused || len(claimResult.Attempts) != 0 {
			t.Fatalf("Claim(after pause commit) = %+v, %v", claimResult, claimErr)
		}
		assertPostgresJobsAttemptCount(ctx, t, pool, prepared.Identity().LogicalJobID, 0)
		if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_job_claim_scopes SET paused = false, scope_generation = scope_generation + 1 WHERE work_class = 'neutral'`); err != nil {
			t.Fatalf("resume scope fixture: %v", err)
		}
	})

	t.Run("claim and pause serialize in the opposite lock order", func(t *testing.T) {
		if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_jobs SET state = 'scheduled', available_at = '2100-01-01' WHERE state IN ('ready', 'scheduled', 'retry_wait')`); err != nil {
			t.Fatalf("isolate claim-first candidate: %v", err)
		}
		prepared := stageDuePostgresJob(ctx, t, pool, store, "claim-before-pause")
		if _, err := pool.PGX().Exec(ctx, `
			CREATE FUNCTION test_block_postgres_jobs_attempt_insert() RETURNS trigger
			LANGUAGE plpgsql AS $$
			BEGIN
				PERFORM pg_advisory_xact_lock(74004);
				RETURN NEW;
			END
			$$
		`); err != nil {
			t.Fatalf("create attempt-insert blocker function: %v", err)
		}
		if _, err := pool.PGX().Exec(ctx, `
			CREATE TRIGGER test_block_postgres_jobs_attempt_insert
			BEFORE INSERT ON postgres_job_attempts
			FOR EACH ROW EXECUTE FUNCTION test_block_postgres_jobs_attempt_insert()
		`); err != nil {
			t.Fatalf("create attempt-insert blocker trigger: %v", err)
		}
		t.Cleanup(func() {
			cleanupCtx := context.WithoutCancel(ctx)
			_, _ = pool.PGX().Exec(cleanupCtx, `DROP TRIGGER IF EXISTS test_block_postgres_jobs_attempt_insert ON postgres_job_attempts`)
			_, _ = pool.PGX().Exec(cleanupCtx, `DROP FUNCTION IF EXISTS test_block_postgres_jobs_attempt_insert()`)
		})
		conflictConn, err := pool.PGX().Acquire(ctx)
		if err != nil {
			t.Fatalf("acquire attempt-conflict connection: %v", err)
		}
		defer conflictConn.Release()
		conflictTx, err := conflictConn.Begin(ctx)
		if err != nil {
			t.Fatalf("begin attempt-conflict transaction: %v", err)
		}
		defer func() { _ = conflictTx.Rollback(context.WithoutCancel(ctx)) }()
		if _, err := conflictTx.Exec(ctx, `SELECT pg_advisory_xact_lock(74004)`); err != nil {
			t.Fatalf("hold attempt-insert advisory lock: %v", err)
		}

		session := acquirePostgresJobsSession(ctx, t, store)
		defer session.Release(ctx)
		claimDone := make(chan struct{})
		var claimResult postgresjobs.ClaimResult
		var claimErr error
		go func() {
			defer close(claimDone)
			claimResult, claimErr = session.Claim(ctx, postgresjobs.ClaimOptions{RegistryKeys: keys, WorkerID: "worker-before-pause", Limit: 1, LeaseDuration: time.Minute})
		}()
		waitForPostgresJobsBlocker(ctx, t, pool, session.BackendPID(), conflictConn.Conn().PgConn().PID())

		pauseConn, err := pool.PGX().Acquire(ctx)
		if err != nil {
			t.Fatalf("acquire claim-first pause connection: %v", err)
		}
		defer pauseConn.Release()
		pauseTx, err := pauseConn.Begin(ctx)
		if err != nil {
			t.Fatalf("begin claim-first pause transaction: %v", err)
		}
		defer func() { _ = pauseTx.Rollback(context.WithoutCancel(ctx)) }()
		pauseDone := make(chan error, 1)
		go func() {
			_, updateErr := pauseTx.Exec(ctx, `UPDATE postgres_job_claim_scopes SET paused = true, scope_generation = scope_generation + 1 WHERE work_class = 'neutral'`)
			pauseDone <- updateErr
		}()
		waitForPostgresJobsBlocker(ctx, t, pool, pauseConn.Conn().PgConn().PID(), session.BackendPID())
		if err := conflictTx.Rollback(ctx); err != nil {
			t.Fatalf("release attempt conflict: %v", err)
		}
		select {
		case <-claimDone:
		case <-ctx.Done():
			t.Fatalf("wait for claim-first claim: %v", ctx.Err())
		}
		if claimErr != nil || len(claimResult.Attempts) != 1 || claimResult.Attempts[0].Identity.LogicalJobID != prepared.Identity().LogicalJobID {
			t.Fatalf("Claim(before pause) = %+v, %v", claimResult, claimErr)
		}
		if err := <-pauseDone; err != nil {
			t.Fatalf("pause after claim: %v", err)
		}
		if err := pauseTx.Commit(ctx); err != nil {
			t.Fatalf("commit pause after claim: %v", err)
		}

		claim := claimResult.Attempts[0]
		stale := claim.Attempt
		stale.WorkerID = "stale-worker"
		resolved, err := session.ResolveClaims(ctx, []postgresjobs.AttemptIdentity{claim.Attempt, stale})
		if err != nil || len(resolved) != 2 {
			t.Fatalf("ResolveClaims() = %+v, %v", resolved, err)
		}
		var committed, rejected bool
		for _, resolution := range resolved {
			switch resolution.Attempt.WorkerID {
			case claim.Attempt.WorkerID:
				committed = resolution.Committed
			case stale.WorkerID:
				rejected = !resolution.Committed
			}
		}
		if !committed || !rejected {
			t.Fatalf("ResolveClaims() = %+v, want exact attempt committed and stale attempt rejected", resolved)
		}
		afterPause, err := session.Claim(ctx, postgresjobs.ClaimOptions{RegistryKeys: keys, WorkerID: "worker-after-claim-first-pause", Limit: 1, LeaseDuration: time.Minute})
		if err != nil || !afterPause.Paused || len(afterPause.Attempts) != 0 {
			t.Fatalf("Claim(after claim-first pause) = %+v, %v", afterPause, err)
		}
		if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_job_claim_scopes SET paused = false, scope_generation = scope_generation + 1 WHERE work_class = 'neutral'`); err != nil {
			t.Fatalf("resume claim-first scope fixture: %v", err)
		}
	})

	t.Run("skip locked creates one atomic generation and attempt", func(t *testing.T) {
		if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_jobs SET state = 'scheduled', available_at = '2100-01-01' WHERE state IN ('ready', 'scheduled', 'retry_wait')`); err != nil {
			t.Fatalf("isolate SKIP LOCKED candidates: %v", err)
		}
		locked := stageDuePostgresJob(ctx, t, pool, store, "claim-locked")
		available := stageDuePostgresJob(ctx, t, pool, store, "claim-skip")
		locker, err := pool.PGX().Begin(ctx)
		if err != nil {
			t.Fatalf("begin row locker: %v", err)
		}
		defer func() { _ = locker.Rollback(context.WithoutCancel(ctx)) }()
		if _, err := locker.Exec(ctx, `SELECT logical_job_id FROM postgres_jobs WHERE logical_job_id = $1 FOR UPDATE`, string(locked.Identity().LogicalJobID)); err != nil {
			t.Fatalf("lock first job: %v", err)
		}
		session := acquirePostgresJobsSession(ctx, t, store)
		defer session.Release(ctx)
		result, err := session.Claim(ctx, postgresjobs.ClaimOptions{RegistryKeys: keys, WorkerID: "worker-skip", Limit: 1, LeaseDuration: time.Minute})
		if err != nil || len(result.Attempts) != 1 || result.Attempts[0].Identity.LogicalJobID != available.Identity().LogicalJobID {
			t.Fatalf("Claim(SKIP LOCKED) = %+v, %v; want %q", result, err, available.Identity().LogicalJobID)
		}
		if err := locker.Commit(ctx); err != nil {
			t.Fatalf("release first job: %v", err)
		}
		claim := result.Attempts[0]
		if claim.Attempt.AttemptGeneration != 1 || claim.AttemptNumber != 1 || !claim.LeaseExpiresAt.After(claim.StartedAt) {
			t.Fatalf("claimed attempt = %+v, want generation/attempt 1 and future lease", claim)
		}
		assertPostgresJobsAttemptCount(ctx, t, pool, available.Identity().LogicalJobID, 1)
	})

	t.Run("future work follows the writer clock and survives downtime", func(t *testing.T) {
		if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_jobs SET state = 'scheduled', available_at = '2100-01-01' WHERE state IN ('ready', 'scheduled', 'retry_wait')`); err != nil {
			t.Fatalf("isolate future candidate: %v", err)
		}
		future := stageDuePostgresJob(ctx, t, pool, store, "claim-future")
		if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_jobs SET state = 'scheduled', available_at = clock_timestamp() + interval '150 milliseconds' WHERE logical_job_id = $1`, string(future.Identity().LogicalJobID)); err != nil {
			t.Fatalf("schedule future job: %v", err)
		}
		session := acquirePostgresJobsSession(ctx, t, store)
		defer session.Release(ctx)
		before, err := session.Claim(ctx, postgresjobs.ClaimOptions{RegistryKeys: keys, WorkerID: "worker-future", Limit: 8, LeaseDuration: time.Minute})
		if err != nil || containsPostgresJobsClaim(before, future.Identity().LogicalJobID) {
			t.Fatalf("Claim(before writer instant) = %+v, %v", before, err)
		}
		waitForPostgresJobsWriterInstant(ctx, t, pool, future.Identity().LogicalJobID)
		after, err := session.Claim(ctx, postgresjobs.ClaimOptions{RegistryKeys: keys, WorkerID: "worker-future", Limit: 8, LeaseDuration: time.Minute})
		if err != nil || !containsPostgresJobsClaim(after, future.Identity().LogicalJobID) {
			t.Fatalf("Claim(after writer instant) = %+v, %v", after, err)
		}
	})
}

func stageDuePostgresJob(ctx context.Context, t *testing.T, pool *postgres.Pool, store *postgresjobs.Store, suffix string) jobs.Prepared {
	t.Helper()
	prepared := postgresJobsPrepared(t, postgresJobsAcceptanceIdentity(suffix), suffix)
	mustStagePostgresJob(ctx, t, pool, store, prepared)
	if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_jobs SET state = 'ready', available_at = clock_timestamp() - interval '1 second' WHERE logical_job_id = $1`, string(prepared.Identity().LogicalJobID)); err != nil {
		t.Fatalf("make job due: %v", err)
	}
	return prepared
}

func acquirePostgresJobsSession(ctx context.Context, t *testing.T, store *postgresjobs.Store) *postgresjobs.Session {
	t.Helper()
	session, err := store.AcquireSession(ctx)
	if err != nil {
		t.Fatalf("AcquireSession(): %v", err)
	}
	return session
}

func assertPostgresJobsAttemptCount(ctx context.Context, t *testing.T, pool *postgres.Pool, logicalJobID jobs.LogicalJobID, want int) {
	t.Helper()
	var count int
	if err := pool.PGX().QueryRow(ctx, `SELECT count(*) FROM postgres_job_attempts WHERE logical_job_id = $1`, string(logicalJobID)).Scan(&count); err != nil {
		t.Fatalf("count attempts: %v", err)
	}
	if count != want {
		t.Fatalf("attempt count for %q = %d, want %d", logicalJobID, count, want)
	}
}

func waitForPostgresJobsBlocker(ctx context.Context, t *testing.T, pool *postgres.Pool, blockedPID, blockerPID uint32) {
	t.Helper()
	for ctx.Err() == nil {
		var blocked bool
		if err := pool.PGX().QueryRow(ctx, `SELECT $2::integer = ANY(pg_blocking_pids($1))`, blockedPID, blockerPID).Scan(&blocked); err != nil {
			t.Fatalf("observe PostgreSQL blocker: %v", err)
		}
		if blocked {
			return
		}
	}
	t.Fatalf("wait for PostgreSQL blocker: %v", ctx.Err())
}

func waitForPostgresJobsWriterInstant(ctx context.Context, t *testing.T, pool *postgres.Pool, logicalJobID jobs.LogicalJobID) {
	t.Helper()
	for ctx.Err() == nil {
		var crossed bool
		if err := pool.PGX().QueryRow(ctx, `SELECT clock_timestamp() >= available_at FROM postgres_jobs WHERE logical_job_id = $1`, string(logicalJobID)).Scan(&crossed); err != nil {
			t.Fatalf("observe writer instant: %v", err)
		}
		if crossed {
			return
		}
	}
	t.Fatalf("wait for writer instant: %v", ctx.Err())
}

func containsPostgresJobsClaim(result postgresjobs.ClaimResult, logicalJobID jobs.LogicalJobID) bool {
	for _, claim := range result.Attempts {
		if claim.Identity.LogicalJobID == logicalJobID {
			return true
		}
	}
	return false
}

func claimPostgresJob(ctx context.Context, t *testing.T, pool *postgres.Pool, store *postgresjobs.Store, suffix, worker string, lease time.Duration) (jobs.Prepared, postgresjobs.ClaimedAttempt) {
	t.Helper()
	prepared := stageDuePostgresJob(ctx, t, pool, store, suffix)
	session := acquirePostgresJobsSession(ctx, t, store)
	defer session.Release(ctx)
	result, err := session.Claim(ctx, postgresjobs.ClaimOptions{RegistryKeys: []jobs.Revision{prepared.Revision()}, WorkerID: worker, Limit: 1, LeaseDuration: lease})
	if err != nil || len(result.Attempts) != 1 {
		t.Fatalf("Claim(%s) = %+v, %v", suffix, result, err)
	}
	return prepared, result.Attempts[0]
}
