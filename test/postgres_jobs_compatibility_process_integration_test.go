//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgresjobs"
	"github.com/example/go-service-template-rest/internal/jobs"
	"github.com/example/go-service-template-rest/internal/waittest"
)

func TestPostgresJobsCompatibilityProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGTERM process lifecycle is Unix-specific")
	}
	ctx, pool, store := newPostgresJobsFixture(t)
	oldOnly, nMinusOne, n := buildPostgresJobsCompatibilityArtifacts(t)
	createPostgresJobsEffectLedger(ctx, t, pool)

	expanded := startPostgresJobsTestWorker(t, nMinusOne.repositoryRoot, nMinusOne.binary, postgresJobsTestWorkerOptions{
		Pool: pool, AppName: "jobs-compat-expand", Handler: "recovery", Files: jobsWorkerLeaseFiles(t),
	})
	waitForPostgresJobsWorkerReady(t, expanded.addr)
	// N may emit v2 only after the exact v1+v2 rollback artifact is ready.
	expandedV1 := stageDuePostgresJobsCompatibilityJob(ctx, t, pool, store, "v1", "compat-expand-v1")
	expandedV2 := stageDuePostgresJobsCompatibilityJob(ctx, t, pool, store, "v2", "compat-expand-v2")
	waitForPostgresJobsCompatibilitySuccess(ctx, t, pool, expandedV1)
	waitForPostgresJobsCompatibilitySuccess(ctx, t, pool, expandedV2)
	stopPostgresJobsCompatibilityWorker(t, expanded)

	retained := seedPostgresJobsCompatibilityRetainedV1(ctx, t, pool, store)
	current := startPostgresJobsTestWorker(t, n.repositoryRoot, n.binary, postgresJobsTestWorkerOptions{
		Pool: pool, AppName: "jobs-compat-n", Handler: "recovery", Files: jobsWorkerLeaseFiles(t),
	})
	waitForPostgresJobsWorkerReady(t, current.addr)
	assertPostgresJobsCompatibilityRetained(ctx, t, pool, retained)
	stopPostgresJobsCompatibilityWorker(t, current)
	setPostgresJobsCompatibilityPaused(ctx, t, pool, false)
	retained = retained[:3]
	current = startPostgresJobsTestWorker(t, n.repositoryRoot, n.binary, postgresJobsTestWorkerOptions{
		Pool: pool, AppName: "jobs-compat-n-work", Handler: "recovery", Files: jobsWorkerLeaseFiles(t),
	})
	waitForPostgresJobsWorkerReady(t, current.addr)
	currentV1 := stageDuePostgresJobsCompatibilityJob(ctx, t, pool, store, "v1", "compat-n-v1")
	currentV2 := stageDuePostgresJobsCompatibilityJob(ctx, t, pool, store, "v2", "compat-n-v2")
	waitForPostgresJobsCompatibilitySuccess(ctx, t, pool, currentV1)
	waitForPostgresJobsCompatibilitySuccess(ctx, t, pool, currentV2)
	assertPostgresJobsCompatibilityRetained(ctx, t, pool, retained)
	stopPostgresJobsCompatibilityWorker(t, current)

	liveV1 := stageDuePostgresJobsCompatibilityJob(ctx, t, pool, store, "v1", "compat-rollback-v1")
	liveV2 := stageDuePostgresJobsCompatibilityJob(ctx, t, pool, store, "v2", "compat-rollback-v2")
	old := startPostgresJobsTestWorker(t, oldOnly.repositoryRoot, oldOnly.binary, postgresJobsTestWorkerOptions{
		Pool: pool, AppName: "jobs-compat-old", Handler: "recovery", Files: jobsWorkerLeaseFiles(t),
	})
	waitForPostgresJobsCompatibilityRejection(t, old)
	assertPostgresJobsCompatibilityVisible(ctx, t, pool, liveV1, jobs.StateReady, 0)
	assertPostgresJobsCompatibilityVisible(ctx, t, pool, liveV2, jobs.StateReady, 0)
	assertPostgresJobsCompatibilityRetained(ctx, t, pool, retained)

	rollbackPaused := stageDuePostgresJobsCompatibilityJob(ctx, t, pool, store, "v1", "compat-rollback-paused")
	setPostgresJobsCompatibilityPaused(ctx, t, pool, true)
	rollback := startPostgresJobsTestWorker(t, nMinusOne.repositoryRoot, nMinusOne.binary, postgresJobsTestWorkerOptions{
		Pool: pool, AppName: "jobs-compat-rollback", Handler: "recovery", Files: jobsWorkerLeaseFiles(t),
	})
	waitForPostgresJobsWorkerReady(t, rollback.addr)
	assertPostgresJobsCompatibilityVisible(ctx, t, pool, rollbackPaused, jobs.StateReady, 0)
	assertPostgresJobsCompatibilityRetained(ctx, t, pool, retained)
	setPostgresJobsCompatibilityPaused(ctx, t, pool, false)
	waitForPostgresJobsCompatibilitySuccess(ctx, t, pool, liveV1)
	waitForPostgresJobsCompatibilitySuccess(ctx, t, pool, liveV2)
	waitForPostgresJobsCompatibilitySuccess(ctx, t, pool, rollbackPaused)
	assertPostgresJobsCompatibilityRetained(ctx, t, pool, retained)
	stopPostgresJobsCompatibilityWorker(t, rollback)

	v3 := stageDuePostgresJobsCompatibilityJob(ctx, t, pool, store, "v3", "compat-retained-v3")
	unknown := startPostgresJobsTestWorker(t, n.repositoryRoot, n.binary, postgresJobsTestWorkerOptions{
		Pool: pool, AppName: "jobs-compat-v3", Handler: "recovery", Files: jobsWorkerLeaseFiles(t),
	})
	waitForPostgresJobsCompatibilityRejection(t, unknown)
	assertPostgresJobsCompatibilityVisible(ctx, t, pool, v3, jobs.StateReady, 0)
	assertPostgresJobsCompatibilityRetained(ctx, t, pool, retained)

	var actions int
	if err := pool.PGX().QueryRow(ctx, `SELECT count(*) FROM postgres_job_actions`).Scan(&actions); err != nil {
		t.Fatalf("count compatibility actions: %v", err)
	}
	if actions != 0 {
		t.Fatalf("compatibility actions = %d, want no contract/delete action", actions)
	}
}

func stageDuePostgresJobsCompatibilityJob(ctx context.Context, t *testing.T, pool *postgres.Pool, store *postgresjobs.Store, argsVersion, suffix string) jobs.Prepared {
	t.Helper()
	definition, err := jobs.NewDefinition(jobs.DefinitionInput[map[string]string]{
		Revision:        jobs.Revision{Kind: "email", ArgsVersion: argsVersion, PolicyVersion: "p1"},
		MaxPayloadBytes: 1024,
		Validate: func(args map[string]string) error {
			if len(args) == 0 {
				return fmt.Errorf("arguments are required")
			}
			return nil
		},
		Policy: jobs.Policy{
			Producer: jobs.ProducerPolicy{Scope: "feature-operation", RecognitionPeriod: time.Hour},
			Effect:   jobs.EffectPolicy{Authority: jobs.EffectConditionalWrite, DuplicateTolerance: "same key is harmless", LateResultPrecedence: "effect ledger wins", AmbiguousAction: jobs.AmbiguousEffectRetry, ReadbackAuthority: "effect ledger"},
			Retry:    jobs.RetryPolicy{MaxAttempts: 3, MaxElapsed: time.Hour, InitialBackoff: 20 * time.Millisecond, MaxBackoff: 20 * time.Millisecond, HintPolicy: jobs.RetryHintIgnore, Jitter: jobs.JitterNone, MaxRecoveryWave: 8},
			Recovery: jobs.RecoveryPolicy{Mode: jobs.RecoveryUnavailable, Attempts: jobs.BudgetPreserved, Elapsed: jobs.BudgetPreserved},
			Schedule: jobs.ScheduleOneOff, MaxAttemptDuration: time.Minute, MaxAttemptCost: 1, MaxUsefulDuration: time.Hour, TerminationEnvelope: time.Minute,
			Data:     jobs.DataPolicy{Classification: "private", Redaction: "omit payload", Retention: "explicit deletion only", Deletion: "disabled", OperatorRoles: "none"},
			Operator: jobs.OperatorUnavailable, WorkClass: jobs.WorkClassNeutral,
		},
	})
	if err != nil {
		t.Fatalf("build %s definition: %v", argsVersion, err)
	}
	prepared, err := definition.Prepare(map[string]string{"task": suffix}, postgresJobsAcceptanceIdentity(suffix), time.Now())
	if err != nil {
		t.Fatalf("prepare %s job: %v", argsVersion, err)
	}
	mustStagePostgresJob(ctx, t, pool, store, prepared)
	if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_jobs SET state = 'ready', available_at = clock_timestamp() - interval '1 second' WHERE logical_job_id = $1`, string(prepared.Identity().LogicalJobID)); err != nil {
		t.Fatalf("make %s job due: %v", argsVersion, err)
	}
	return prepared
}

func waitForPostgresJobsCompatibilitySuccess(ctx context.Context, t *testing.T, pool *postgres.Pool, prepared jobs.Prepared) {
	t.Helper()
	waittest.Until(t, 5*time.Second, func() bool {
		var state, argsVersion string
		err := pool.PGX().QueryRow(ctx, `SELECT state, args_version FROM postgres_jobs WHERE logical_job_id = $1`, string(prepared.Identity().LogicalJobID)).Scan(&state, &argsVersion)
		if err != nil {
			t.Fatal(err)
		}
		return state == string(jobs.StateSucceeded) && argsVersion == prepared.Revision().ArgsVersion
	}, "compatible worker execution")
}

func seedPostgresJobsCompatibilityRetainedV1(ctx context.Context, t *testing.T, pool *postgres.Pool, store *postgresjobs.Store) []struct {
	prepared jobs.Prepared
	state    jobs.State
} {
	t.Helper()
	retained := []struct {
		prepared jobs.Prepared
		state    jobs.State
	}{
		{prepared: stageDuePostgresJobsCompatibilityJob(ctx, t, pool, store, "v1", "compat-scheduled"), state: jobs.StateScheduled},
		{prepared: stageDuePostgresJobsCompatibilityJob(ctx, t, pool, store, "v1", "compat-retry"), state: jobs.StateRetryWait},
		{prepared: stageDuePostgresJobsCompatibilityJob(ctx, t, pool, store, "v1", "compat-redriveable"), state: jobs.StatePermanent},
		{prepared: stageDuePostgresJobsCompatibilityJob(ctx, t, pool, store, "v1", "compat-paused"), state: jobs.StateReady},
	}
	for _, row := range retained[:3] {
		terminal := row.state == jobs.StatePermanent
		if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_jobs
SET state = $2, available_at = clock_timestamp() + interval '1 hour', terminal_at = CASE WHEN $3::boolean THEN clock_timestamp() ELSE NULL END
WHERE logical_job_id = $1`, string(row.prepared.Identity().LogicalJobID), string(row.state), terminal); err != nil {
			t.Fatalf("seed retained %s job: %v", row.state, err)
		}
	}
	setPostgresJobsCompatibilityPaused(ctx, t, pool, true)
	return retained
}

func setPostgresJobsCompatibilityPaused(ctx context.Context, t *testing.T, pool *postgres.Pool, paused bool) {
	t.Helper()
	if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_job_claim_scopes SET paused = $1, scope_generation = scope_generation + 1 WHERE work_class = 'neutral'`, paused); err != nil {
		t.Fatalf("set compatibility scope pause=%t: %v", paused, err)
	}
}

func assertPostgresJobsCompatibilityRetained(ctx context.Context, t *testing.T, pool *postgres.Pool, retained []struct {
	prepared jobs.Prepared
	state    jobs.State
}) {
	t.Helper()
	for _, row := range retained {
		assertPostgresJobsCompatibilityVisible(ctx, t, pool, row.prepared, row.state, 0)
	}
}

func assertPostgresJobsCompatibilityVisible(ctx context.Context, t *testing.T, pool *postgres.Pool, prepared jobs.Prepared, wantState jobs.State, wantAttempts int) {
	t.Helper()
	var argsVersion, state string
	var attempts int
	err := pool.PGX().QueryRow(ctx, `
SELECT j.args_version, j.state, count(a.logical_job_id)
FROM postgres_jobs AS j
LEFT JOIN postgres_job_attempts AS a USING (logical_job_id)
WHERE j.logical_job_id = $1
GROUP BY j.args_version, j.state`, string(prepared.Identity().LogicalJobID)).Scan(&argsVersion, &state, &attempts)
	if err != nil {
		t.Fatalf("read retained compatibility row: %v", err)
	}
	if argsVersion != prepared.Revision().ArgsVersion || state != string(wantState) || attempts != wantAttempts {
		t.Fatalf("retained compatibility row = revision=%s state=%s attempts=%d, want revision=%s state=%s attempts=%d", argsVersion, state, attempts, prepared.Revision().ArgsVersion, wantState, wantAttempts)
	}
}

func stopPostgresJobsCompatibilityWorker(t *testing.T, worker *postgresJobsTestWorker) {
	t.Helper()
	if err := worker.process.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("stop compatibility worker: %v", err)
	}
	if err, exited := waitForPostgresJobsWorkerExit(worker, 12*time.Second); !exited || err != nil {
		t.Fatalf("compatibility worker exit = %v, exited=%t\n%s", err, exited, worker.output.String())
	}
}

func waitForPostgresJobsCompatibilityRejection(t *testing.T, worker *postgresJobsTestWorker) {
	t.Helper()
	if err, exited := waitForPostgresJobsWorkerExit(worker, 12*time.Second); !exited || err == nil {
		t.Fatalf("incompatible worker exit = %v, exited=%t\n%s", err, exited, worker.output.String())
	}
}
