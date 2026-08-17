//go:build integration

package integration_test

import (
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgresjobs"
	"github.com/example/go-service-template-rest/internal/jobs"
)

func TestPostgresJobsFinalize(t *testing.T) {
	ctx, pool, store := newPostgresJobsFixture(t)

	t.Run("finalization stores one replayable transition", func(t *testing.T) {
		_, claimed := claimPostgresJob(ctx, t, pool, store, "finalize-retry", "worker-finalize", time.Minute)
		session := acquirePostgresJobsSession(ctx, t, store)
		defer session.Release(ctx)
		input := postgresjobs.FinalizeInput{
			Attempt: claimed.Attempt,
			Transition: jobs.Transition{
				State: jobs.StateRetryWait, Delay: 7 * time.Second, AttemptsUsed: claimed.AttemptNumber,
				ElapsedUsed: 1250 * time.Millisecond, Outcome: jobs.OutcomeRetryable, Effect: jobs.EffectNone,
			},
			FailureCode: "temporary",
		}
		first, err := session.Finalize(ctx, input)
		if err != nil || first.Status != postgresjobs.TransitionApplied || first.RetryAt.IsZero() {
			t.Fatalf("Finalize(first) = %+v, %v", first, err)
		}
		second, err := session.Finalize(ctx, input)
		if err != nil || second.Status != postgresjobs.TransitionRepeated {
			t.Fatalf("Finalize(repeat) = %+v, %v", second, err)
		}
		if second.Transition != first.Transition || second.FailureCode != first.FailureCode ||
			!second.RetryAt.Equal(first.RetryAt) || !second.FinalizedAt.Equal(first.FinalizedAt) {
			t.Fatalf("replayed transition = %+v, want exact %+v", second, first)
		}
		var state string
		var availableAt, retryAt time.Time
		var attemptsUsed int
		if err := pool.PGX().QueryRow(ctx, `SELECT job.state, job.available_at, attempt.retry_at, attempt.attempts_used
			FROM postgres_jobs AS job JOIN postgres_job_attempts AS attempt USING (logical_job_id, attempt_generation)
			WHERE job.logical_job_id = $1`, string(claimed.Attempt.LogicalJobID)).Scan(&state, &availableAt, &retryAt, &attemptsUsed); err != nil {
			t.Fatalf("read finalized retry: %v", err)
		}
		if state != string(jobs.StateRetryWait) || attemptsUsed != int(claimed.AttemptNumber) ||
			!availableAt.Equal(retryAt) || !retryAt.Equal(first.RetryAt) {
			t.Fatalf("stored retry = state %q attempts %d available/retry %s/%s, want exact result %+v", state, attemptsUsed, availableAt, retryAt, first)
		}
	})

	t.Run("lease expiry alone does not defeat the current finalizer", func(t *testing.T) {
		_, claimed := claimPostgresJob(ctx, t, pool, store, "finalize-expired", "worker-expired-finalize", time.Minute)
		if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_jobs SET lease_expires_at = clock_timestamp() - interval '1 second' WHERE logical_job_id = $1`, string(claimed.Attempt.LogicalJobID)); err != nil {
			t.Fatalf("expire current job lease: %v", err)
		}
		if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_job_attempts SET lease_expires_at = clock_timestamp() - interval '1 second' WHERE logical_job_id = $1`, string(claimed.Attempt.LogicalJobID)); err != nil {
			t.Fatalf("expire current attempt lease: %v", err)
		}
		session := acquirePostgresJobsSession(ctx, t, store)
		defer session.Release(ctx)
		result, err := session.Finalize(ctx, postgresjobs.FinalizeInput{
			Attempt: claimed.Attempt,
			Transition: jobs.Transition{
				State: jobs.StateSucceeded, AttemptsUsed: claimed.AttemptNumber,
				ElapsedUsed: time.Second, Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted,
			},
		})
		if err != nil || result.Status != postgresjobs.TransitionApplied || result.Transition.State != jobs.StateSucceeded {
			t.Fatalf("Finalize(expired current attempt) = %+v, %v", result, err)
		}
	})

	t.Run("attempt budget mismatch is fenced", func(t *testing.T) {
		_, claimed := claimPostgresJob(ctx, t, pool, store, "finalize-budget-mismatch", "worker-budget-mismatch", time.Minute)
		session := acquirePostgresJobsSession(ctx, t, store)
		defer session.Release(ctx)
		result, err := session.Finalize(ctx, postgresjobs.FinalizeInput{
			Attempt: claimed.Attempt,
			Transition: jobs.Transition{
				State: jobs.StateSucceeded, AttemptsUsed: claimed.AttemptNumber + 1,
				ElapsedUsed: time.Second, Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted,
			},
		})
		if err != nil || result.Status != postgresjobs.TransitionStale {
			t.Fatalf("Finalize(counter mismatch) = %+v, %v, want stale", result, err)
		}
	})
}
