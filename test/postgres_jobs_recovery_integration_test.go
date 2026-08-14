//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgresjobs"
	"github.com/example/go-service-template-rest/internal/jobs"
)

func TestPostgresJobsRecovery(t *testing.T) {
	ctx, pool, store := newPostgresJobsFixture(t)

	t.Run("rescue read and write preserve ambiguity and fence the next generation", func(t *testing.T) {
		prepared, claimed := claimPostgresJob(ctx, t, pool, store, "recovery", "worker-lost", time.Minute)
		expirePostgresJobsAttempt(ctx, t, pool, claimed.Attempt)
		session := acquirePostgresJobsSession(ctx, t, store)
		defer session.Release(ctx)
		candidates, err := session.RescueCandidates(ctx, 1)
		if err != nil || len(candidates) != 1 {
			t.Fatalf("RescueCandidates() = %+v, %v", candidates, err)
		}
		candidate := candidates[0]
		if candidate.Attempt != claimed.Attempt || candidate.Revision != prepared.Revision() ||
			candidate.State != jobs.StateRunning || candidate.Elapsed < 0 || !candidate.ObservedAt.After(candidate.LeaseExpiresAt) {
			t.Fatalf("rescue candidate = %+v, want exact expired current attempt", candidate)
		}
		input := postgresjobs.RescueInput{
			Attempt: candidate.Attempt,
			Transition: jobs.Transition{
				State: jobs.StateRetryWait, AttemptsUsed: candidate.AttemptNumber,
				ElapsedUsed: candidate.Elapsed, Outcome: jobs.OutcomeLost, Effect: jobs.EffectUnknown,
			},
			FailureCode: "lease_lost",
		}
		first, err := session.Rescue(ctx, input)
		if err != nil || first.Status != postgresjobs.TransitionApplied || first.Transition.Effect != jobs.EffectUnknown {
			t.Fatalf("Rescue(first) = %+v, %v", first, err)
		}
		repeated, err := session.Rescue(ctx, input)
		if err != nil || repeated.Status != postgresjobs.TransitionRepeated || !repeated.FinalizedAt.Equal(first.FinalizedAt) {
			t.Fatalf("Rescue(repeat) = %+v, %v; want exact replay %+v", repeated, err, first)
		}

		next, err := session.Claim(ctx, postgresjobs.ClaimOptions{
			RegistryKeys: []jobs.Revision{prepared.Revision()}, WorkerID: "worker-recovered",
			Limit: 1, LeaseDuration: time.Minute,
		})
		if err != nil || len(next.Attempts) != 1 || next.Attempts[0].Attempt.AttemptGeneration != claimed.Attempt.AttemptGeneration+1 {
			t.Fatalf("Claim(after rescue) = %+v, %v", next, err)
		}
		stale, err := session.Finalize(ctx, postgresjobs.FinalizeInput{
			Attempt: claimed.Attempt,
			Transition: jobs.Transition{
				State: jobs.StateSucceeded, AttemptsUsed: claimed.AttemptNumber,
				ElapsedUsed: time.Second, Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted,
			},
		})
		if err != nil || stale.Status != postgresjobs.TransitionRepeated || stale.Transition.State != jobs.StateRetryWait {
			t.Fatalf("Finalize(stale rescued generation) = %+v, %v; want retained rescue result", stale, err)
		}
		var state string
		var generation int64
		if err := pool.PGX().QueryRow(ctx, `SELECT state, attempt_generation FROM postgres_jobs WHERE logical_job_id = $1`, string(claimed.Attempt.LogicalJobID)).Scan(&state, &generation); err != nil {
			t.Fatalf("read current recovered generation: %v", err)
		}
		if state != string(jobs.StateRunning) || generation != int64(claimed.Attempt.AttemptGeneration+1) {
			t.Fatalf("current recovered job = state %q generation %d, want running generation %d", state, generation, claimed.Attempt.AttemptGeneration+1)
		}
	})

	t.Run("finalize and rescue equal one job-row lock order", func(t *testing.T) {
		runPostgresJobsFinalizeRescueOrder(ctx, t, pool, store, true)
		runPostgresJobsFinalizeRescueOrder(ctx, t, pool, store, false)
	})
}

func runPostgresJobsFinalizeRescueOrder(ctx context.Context, t *testing.T, pool *postgres.Pool, store *postgresjobs.Store, finalizeFirst bool) {
	t.Helper()
	name := "rescue-first"
	if finalizeFirst {
		name = "finalize-first"
	}
	_, claimed := claimPostgresJob(ctx, t, pool, store, "linearize-"+name, "worker-"+name, time.Minute)
	expirePostgresJobsAttempt(ctx, t, pool, claimed.Attempt)
	locker, err := pool.PGX().Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire %s locker: %v", name, err)
	}
	defer locker.Release()
	lockTx, err := locker.Begin(ctx)
	if err != nil {
		t.Fatalf("begin %s locker: %v", name, err)
	}
	if _, err := lockTx.Exec(ctx, `SELECT logical_job_id FROM postgres_jobs WHERE logical_job_id = $1 FOR UPDATE`, string(claimed.Attempt.LogicalJobID)); err != nil {
		t.Fatalf("lock %s job: %v", name, err)
	}
	finalizeSession := acquirePostgresJobsSession(ctx, t, store)
	defer finalizeSession.Release(ctx)
	rescueSession := acquirePostgresJobsSession(ctx, t, store)
	defer rescueSession.Release(ctx)
	finalizeInput := postgresjobs.FinalizeInput{
		Attempt: claimed.Attempt,
		Transition: jobs.Transition{
			State: jobs.StateSucceeded, AttemptsUsed: claimed.AttemptNumber,
			ElapsedUsed: time.Second, Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted,
		},
	}
	rescueInput := postgresjobs.RescueInput{
		Attempt: claimed.Attempt,
		Transition: jobs.Transition{
			State: jobs.StateOutcomeUnknown, AttemptsUsed: claimed.AttemptNumber,
			ElapsedUsed: 2 * time.Second, Outcome: jobs.OutcomeLost, Effect: jobs.EffectUnknown,
		},
		FailureCode: "lease_lost",
	}
	type transitionCall struct {
		result postgresjobs.PersistedTransition
		err    error
	}
	finalized := make(chan transitionCall, 1)
	rescued := make(chan transitionCall, 1)
	startFinalize := func() {
		result, callErr := finalizeSession.Finalize(ctx, finalizeInput)
		finalized <- transitionCall{result, callErr}
	}
	startRescue := func() {
		result, callErr := rescueSession.Rescue(ctx, rescueInput)
		rescued <- transitionCall{result, callErr}
	}
	firstPID, secondPID := rescueSession.BackendPID(), finalizeSession.BackendPID()
	firstCall, secondCall := startRescue, startFinalize
	if finalizeFirst {
		firstPID, secondPID = finalizeSession.BackendPID(), rescueSession.BackendPID()
		firstCall, secondCall = startFinalize, startRescue
	}
	go firstCall()
	waitForPostgresJobsBlocker(ctx, t, pool, firstPID, locker.Conn().PgConn().PID())
	go secondCall()
	waitForPostgresJobsBlocked(ctx, t, pool, secondPID)
	if err := lockTx.Commit(ctx); err != nil {
		t.Fatalf("release %s job lock: %v", name, err)
	}
	finalizeResult, rescueResult := <-finalized, <-rescued
	if finalizeResult.err != nil || rescueResult.err != nil {
		t.Fatalf("%s Finalize/Rescue errors = %v/%v", name, finalizeResult.err, rescueResult.err)
	}
	firstResult, secondResult := rescueResult.result, finalizeResult.result
	wantState := jobs.StateOutcomeUnknown
	if finalizeFirst {
		firstResult, secondResult = finalizeResult.result, rescueResult.result
		wantState = jobs.StateSucceeded
	}
	if firstResult.Status != postgresjobs.TransitionApplied || secondResult.Status != postgresjobs.TransitionRepeated ||
		firstResult.Transition.State != wantState || secondResult.Transition.State != wantState ||
		!firstResult.FinalizedAt.Equal(secondResult.FinalizedAt) {
		t.Fatalf("%s results = first %+v second %+v, want applied then exact repeated %q", name, firstResult, secondResult, wantState)
	}
}

func expirePostgresJobsAttempt(ctx context.Context, t *testing.T, pool *postgres.Pool, attempt postgresjobs.AttemptIdentity) {
	t.Helper()
	if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_jobs SET lease_expires_at = clock_timestamp() - interval '1 second' WHERE logical_job_id = $1 AND attempt_generation = $2`, string(attempt.LogicalJobID), attempt.AttemptGeneration); err != nil {
		t.Fatalf("expire job lease: %v", err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_job_attempts SET lease_expires_at = clock_timestamp() - interval '1 second' WHERE logical_job_id = $1 AND attempt_generation = $2`, string(attempt.LogicalJobID), attempt.AttemptGeneration); err != nil {
		t.Fatalf("expire attempt lease: %v", err)
	}
}

func waitForPostgresJobsBlocked(ctx context.Context, t *testing.T, pool *postgres.Pool, blockedPID uint32) {
	t.Helper()
	for ctx.Err() == nil {
		var blocked bool
		if err := pool.PGX().QueryRow(ctx, `SELECT cardinality(pg_blocking_pids($1)) > 0`, blockedPID).Scan(&blocked); err != nil {
			t.Fatalf("observe PostgreSQL wait queue: %v", err)
		}
		if blocked {
			return
		}
	}
	t.Fatalf("wait for PostgreSQL wait queue: %v", ctx.Err())
}
