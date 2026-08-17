//go:build integration

package integration_test

import (
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgresjobs"
)

func TestPostgresJobsRenew(t *testing.T) {
	t.Parallel()
	ctx, pool, store := newPostgresJobsFixture(t)
	_, claimed := claimPostgresJob(ctx, t, pool, store, "renew", "worker-renew", 30*time.Second)
	session := acquirePostgresJobsSession(ctx, t, store)
	defer session.Release(ctx)

	if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_jobs SET state = 'cancel_requested' WHERE logical_job_id = $1 AND attempt_generation = $2`, string(claimed.Attempt.LogicalJobID), claimed.Attempt.AttemptGeneration); err != nil {
		t.Fatalf("request durable cancellation: %v", err)
	}
	renewed, err := session.Renew(ctx, []postgresjobs.AttemptIdentity{claimed.Attempt}, time.Minute)
	if err != nil || len(renewed) != 1 {
		t.Fatalf("Renew(matching) = %+v, %v", renewed, err)
	}
	if !renewed[0].CancelRequested || !renewed[0].LeaseExpiresAt.After(renewed[0].ObservedAt) {
		t.Fatalf("matching renewal = %+v, want cancellation and future lease", renewed[0])
	}
	var jobLease, attemptLease time.Time
	if err := pool.PGX().QueryRow(ctx, `SELECT job.lease_expires_at, attempt.lease_expires_at
		FROM postgres_jobs AS job JOIN postgres_job_attempts AS attempt USING (logical_job_id, attempt_generation)
		WHERE job.logical_job_id = $1`, string(claimed.Attempt.LogicalJobID)).Scan(&jobLease, &attemptLease); err != nil {
		t.Fatalf("read renewed leases: %v", err)
	}
	if !jobLease.Equal(attemptLease) || !jobLease.Equal(renewed[0].LeaseExpiresAt) {
		t.Fatalf("job/attempt/result leases = %s/%s/%s, want exact equality", jobLease, attemptLease, renewed[0].LeaseExpiresAt)
	}

	stale := claimed.Attempt
	stale.AttemptGeneration++
	staleRenewal, err := session.Renew(ctx, []postgresjobs.AttemptIdentity{stale}, 2*time.Minute)
	if err != nil || len(staleRenewal) != 0 {
		t.Fatalf("Renew(stale) = %+v, %v; want no matching attempt", staleRenewal, err)
	}
	var afterStale time.Time
	if err := pool.PGX().QueryRow(ctx, `SELECT lease_expires_at FROM postgres_jobs WHERE logical_job_id = $1`, string(claimed.Attempt.LogicalJobID)).Scan(&afterStale); err != nil {
		t.Fatalf("read lease after stale renewal: %v", err)
	}
	if !afterStale.Equal(jobLease) {
		t.Fatalf("lease after stale renewal = %s, want unchanged %s", afterStale, jobLease)
	}
}
