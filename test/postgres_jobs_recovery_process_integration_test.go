//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgresjobs"
	"github.com/example/go-service-template-rest/internal/jobs"
	"github.com/example/go-service-template-rest/internal/waittest"
)

func TestPostgresJobsRecoveryProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGKILL/SIGSTOP process lifecycle is Unix-specific")
	}
	repositoryRoot, binary := buildPostgresJobsTestWorker(t)
	for _, scenario := range []struct {
		name         string
		beforeEffect bool
		overlap      bool
	}{
		{name: "before_effect_crash", beforeEffect: true},
		{name: "after_effect_before_completion_crash"},
		{name: "stale_overlap_after_rescue", overlap: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			testPostgresJobsRecoveryProcess(t, repositoryRoot, binary, scenario.beforeEffect, scenario.overlap)
		})
	}
	for _, terminal := range []struct {
		result string
		state  jobs.State
	}{
		{result: "permanent", state: jobs.StatePermanent},
		{result: "poison", state: jobs.StatePoison},
		{result: "unknown", state: jobs.StateOutcomeUnknown},
		{result: "exhausted", state: jobs.StateExhausted},
	} {
		t.Run("stored_revision_"+terminal.result, func(t *testing.T) {
			testPostgresJobsStoredRevisionTerminal(t, repositoryRoot, binary, terminal.result, terminal.state)
		})
	}
}

func testPostgresJobsRecoveryProcess(t *testing.T, repositoryRoot, binary string, beforeEffect, overlap bool) {
	t.Helper()
	ctx, pool, store := newPostgresJobsFixture(t)
	createPostgresJobsEffectLedger(ctx, t, pool)
	prepared := stageDuePostgresRecoveryJob(ctx, t, pool, store, "recovery-"+t.Name())
	logicalJobID := prepared.Identity().LogicalJobID
	retainPostgresJobsAction(ctx, t, pool, logicalJobID)

	firstFiles := jobsWorkerLeaseFiles(t)
	first := startPostgresJobsTestWorker(t, repositoryRoot, binary, postgresJobsTestWorkerOptions{
		Pool: pool, AppName: "jobs-recovery-first", Handler: "recovery", Files: firstFiles,
		EffectGate:   gateForRecovery(beforeEffect, firstFiles.effectGate),
		CompleteGate: gateForRecovery(!beforeEffect, firstFiles.completeGate),
	})
	waitForPostgresJobsWorkerReady(t, first.addr)
	waitForPostgresJobsAttempt(ctx, t, pool, logicalJobID, 1)
	waitForPostgresJobsTestFile(t, firstFiles.entered, "recovery handler start")
	if beforeEffect {
		assertPostgresJobsEffectCount(ctx, t, pool, prepared.Identity(), 0)
	} else {
		waitForPostgresJobsTestFile(t, firstFiles.effect, "conditional effect commit")
		assertPostgresJobsEffectCount(ctx, t, pool, prepared.Identity(), 1)
	}

	if overlap {
		if err := first.process.Process.Signal(syscall.SIGSTOP); err != nil {
			t.Fatalf("stop stale worker: %v", err)
		}
	} else if err := first.process.Process.Kill(); err != nil {
		t.Fatalf("kill crashed worker: %v", err)
	}
	if !overlap {
		if err, exited := waitForPostgresJobsWorkerExit(first, 5*time.Second); !exited || err == nil {
			t.Fatalf("crashed worker exit = %v, exited=%t", err, exited)
		}
	}
	waittest.Until(t, 5*time.Second, func() bool {
		return postgresJobsLeaseExpired(ctx, t, pool, logicalJobID)
	}, "crashed attempt lease expiry")

	secondFiles := jobsWorkerLeaseFiles(t)
	second := startPostgresJobsTestWorker(t, repositoryRoot, binary, postgresJobsTestWorkerOptions{
		Pool: pool, AppName: "jobs-recovery-second", Handler: "recovery", Files: secondFiles,
	})
	waitForPostgresJobsWorkerReady(t, second.addr)
	waitForPostgresJobsTerminalRecovery(ctx, t, pool, prepared)
	assertPostgresJobsEffectCount(ctx, t, pool, prepared.Identity(), 1)
	assertPostgresJobsRetainedRecoveryFacts(ctx, t, pool, prepared)

	if overlap {
		if err := first.process.Process.Signal(syscall.SIGCONT); err != nil {
			t.Fatalf("resume stale worker: %v", err)
		}
		if err := os.WriteFile(firstFiles.completeGate, []byte("release\n"), 0o600); err != nil {
			t.Fatalf("release stale finalizer: %v", err)
		}
		waitPostgresJobsStaleFinalizer(ctx, t, pool, logicalJobID)
	}
	if err := second.process.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("stop recovered worker: %v", err)
	}
	if err, exited := waitForPostgresJobsWorkerExit(second, 12*time.Second); !exited || err != nil {
		t.Fatalf("recovered worker exit = %v, exited=%t\n%s", err, exited, second.output.String())
	}
	if overlap {
		select {
		case <-first.finished:
		default:
			if err := first.process.Process.Signal(syscall.SIGTERM); err != nil {
				t.Fatalf("stop stale worker: %v", err)
			}
			if _, exited := waitForPostgresJobsWorkerExit(first, 12*time.Second); !exited {
				t.Fatalf("stale worker did not exit\n%s", first.output.String())
			}
		}
	}
}

func gateForRecovery(enabled bool, path string) string {
	if enabled {
		return path
	}
	return ""
}

func createPostgresJobsEffectLedger(ctx context.Context, t *testing.T, pool *postgres.Pool) {
	t.Helper()
	if _, err := pool.PGX().Exec(ctx, `
CREATE TABLE test_postgres_jobs_effect_ledger (
  effect_scope text NOT NULL,
  effect_key text NOT NULL,
  logical_job_id text NOT NULL,
  PRIMARY KEY (effect_scope, effect_key)
)`); err != nil {
		t.Fatalf("create test effect ledger: %v", err)
	}
}

func retainPostgresJobsAction(ctx context.Context, t *testing.T, pool *postgres.Pool, logicalJobID jobs.LogicalJobID) {
	t.Helper()
	if _, err := pool.PGX().Exec(ctx, `
INSERT INTO postgres_job_actions (action_id, request_fingerprint, actor_id, action_kind, logical_job_id, expected_state, expected_generation, reason, result)
VALUES ('recovery-action', decode(repeat('00', 32), 'hex'), 'recovery-test', 'cancel', $1, 'running', 1, 'recovery-test', 'applied')`, string(logicalJobID)); err != nil {
		t.Fatalf("retain action fact: %v", err)
	}
}

func assertPostgresJobsEffectCount(ctx context.Context, t *testing.T, pool *postgres.Pool, identity jobs.AcceptanceIdentity, want int) {
	t.Helper()
	var got int
	if err := pool.PGX().QueryRow(ctx, `SELECT count(*) FROM test_postgres_jobs_effect_ledger WHERE effect_scope = $1 AND effect_key = $2 AND logical_job_id = $3`, identity.EffectScope, identity.EffectKey, identity.LogicalJobID).Scan(&got); err != nil {
		t.Fatalf("count conditional effects: %v", err)
	}
	if got != want {
		t.Fatalf("conditional effects = %d, want %d", got, want)
	}
}

func waitForPostgresJobsTerminalRecovery(ctx context.Context, t *testing.T, pool *postgres.Pool, prepared jobs.Prepared) {
	t.Helper()
	waittest.Until(t, 8*time.Second, func() bool {
		var state string
		var attempts int
		err := pool.PGX().QueryRow(ctx, `SELECT j.state, count(*) OVER () FROM postgres_jobs AS j JOIN postgres_job_attempts AS a USING (logical_job_id) WHERE j.logical_job_id = $1 ORDER BY a.attempt_generation DESC LIMIT 1`, prepared.Identity().LogicalJobID).Scan(&state, &attempts)
		if err != nil {
			t.Fatal(err)
		}
		return state == string(jobs.StateSucceeded) && attempts == 2
	}, "rescued second attempt completion")
}

func assertPostgresJobsRetainedRecoveryFacts(ctx context.Context, t *testing.T, pool *postgres.Pool, prepared jobs.Prepared) {
	t.Helper()
	var producerScope, producerKey, occurrenceScope, occurrenceID, effectScope, effectKey, kind, argsVersion, policyVersion, state string
	var attemptsUsed, generation, recoveryGeneration, actions int
	err := pool.PGX().QueryRow(ctx, `
SELECT producer_scope, producer_key, occurrence_scope, occurrence_id, effect_scope, effect_key, kind, args_version, policy_version, state, attempts_used, attempt_generation, recovery_generation,
  (SELECT count(*) FROM postgres_job_actions WHERE logical_job_id = postgres_jobs.logical_job_id)
FROM postgres_jobs WHERE logical_job_id = $1`, prepared.Identity().LogicalJobID).Scan(&producerScope, &producerKey, &occurrenceScope, &occurrenceID, &effectScope, &effectKey, &kind, &argsVersion, &policyVersion, &state, &attemptsUsed, &generation, &recoveryGeneration, &actions)
	if err != nil {
		t.Fatalf("read retained recovery facts: %v", err)
	}
	identity := prepared.Identity()
	if producerScope != string(identity.ProducerScope) || producerKey != string(identity.ProducerKey) || occurrenceScope != string(identity.OccurrenceScope) || occurrenceID != string(identity.OccurrenceID) || effectScope != string(identity.EffectScope) || effectKey != string(identity.EffectKey) {
		t.Fatalf("retained identities changed: producer=%q/%q occurrence=%q/%q effect=%q/%q", producerScope, producerKey, occurrenceScope, occurrenceID, effectScope, effectKey)
	}
	revision := prepared.Revision()
	if kind != revision.Kind || argsVersion != revision.ArgsVersion || policyVersion != revision.PolicyVersion || state != string(jobs.StateSucceeded) || attemptsUsed != 2 || generation != 2 || recoveryGeneration != 0 || actions != 1 {
		t.Fatalf("retained recovery facts = revision=%s/%s/%s state=%s attempts=%d generation=%d recovery=%d actions=%d", kind, argsVersion, policyVersion, state, attemptsUsed, generation, recoveryGeneration, actions)
	}
}

func waitPostgresJobsStaleFinalizer(ctx context.Context, t *testing.T, pool *postgres.Pool, logicalJobID jobs.LogicalJobID) {
	t.Helper()
	waittest.Until(t, 5*time.Second, func() bool {
		var state, outcome, effect string
		err := pool.PGX().QueryRow(ctx, `
SELECT j.state, coalesce(a.outcome, ''), coalesce(a.effect_status, '')
FROM postgres_jobs AS j JOIN postgres_job_attempts AS a ON a.logical_job_id = j.logical_job_id
WHERE j.logical_job_id = $1 AND a.attempt_generation = 1`, logicalJobID).Scan(&state, &outcome, &effect)
		if err != nil {
			t.Fatal(err)
		}
		return state == string(jobs.StateSucceeded) && outcome == string(jobs.OutcomeLost) && effect == string(jobs.EffectUnknown)
	}, "stale finalizer no-op")
}

func testPostgresJobsStoredRevisionTerminal(t *testing.T, repositoryRoot, binary, result string, want jobs.State) {
	t.Helper()
	ctx, pool, store := newPostgresJobsFixture(t)
	prepared := stageDuePostgresRecoveryJob(ctx, t, pool, store, "recovery-"+result)
	wantAttempts := 1
	if result == "exhausted" {
		wantAttempts = 3
	}
	first := startPostgresJobsTestWorker(t, repositoryRoot, binary, postgresJobsTestWorkerOptions{
		Pool: pool, AppName: "jobs-recovery-" + result, Handler: "recovery", Files: jobsWorkerLeaseFiles(t), Result: result,
	})
	waitForPostgresJobsWorkerReady(t, first.addr)
	waittest.Until(t, 5*time.Second, func() bool {
		var state string
		var attempts int
		err := pool.PGX().QueryRow(ctx, `SELECT j.state, count(*) OVER () FROM postgres_jobs AS j JOIN postgres_job_attempts AS a USING (logical_job_id) WHERE j.logical_job_id = $1 ORDER BY a.attempt_generation DESC LIMIT 1`, prepared.Identity().LogicalJobID).Scan(&state, &attempts)
		if err != nil {
			t.Fatal(err)
		}
		return state == string(want) && attempts == wantAttempts
	}, "stored revision terminal classification")
	if err := first.process.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("stop terminal worker: %v", err)
	}
	if err, exited := waitForPostgresJobsWorkerExit(first, 12*time.Second); !exited || err != nil {
		t.Fatalf("terminal worker exit = %v, exited=%t", err, exited)
	}
	second := startPostgresJobsTestWorker(t, repositoryRoot, binary, postgresJobsTestWorkerOptions{
		Pool: pool, AppName: "jobs-recovery-restart-" + result, Handler: "recovery", Files: jobsWorkerLeaseFiles(t),
	})
	claims := monitorPostgresJobsAttemptCount(ctx, pool, prepared.Identity().LogicalJobID, wantAttempts, second.finished)
	waitForPostgresJobsWorkerReady(t, second.addr)
	if err := second.process.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("stop restart worker: %v", err)
	}
	if err, exited := waitForPostgresJobsWorkerExit(second, 12*time.Second); !exited || err != nil {
		t.Fatalf("restart worker exit = %v, exited=%t", err, exited)
	}
	assertPostgresJobsMonitor(t, claims, "restart claim")
}

func stageDuePostgresRecoveryJob(ctx context.Context, t *testing.T, pool *postgres.Pool, store *postgresjobs.Store, suffix string) jobs.Prepared {
	t.Helper()
	definition, err := jobs.NewDefinition(jobs.DefinitionInput[map[string]string]{
		Revision:        jobs.Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "p1"},
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
		t.Fatalf("build recovery definition: %v", err)
	}
	prepared, err := definition.Prepare(map[string]string{"task": suffix}, postgresJobsAcceptanceIdentity(suffix), time.Now())
	if err != nil {
		t.Fatalf("prepare recovery job: %v", err)
	}
	mustStagePostgresJob(ctx, t, pool, store, prepared)
	if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_jobs SET state = 'ready', available_at = clock_timestamp() - interval '1 second' WHERE logical_job_id = $1`, string(prepared.Identity().LogicalJobID)); err != nil {
		t.Fatalf("make recovery job due: %v", err)
	}
	return prepared
}

func monitorPostgresJobsAttemptCount(ctx context.Context, pool *postgres.Pool, logicalJobID jobs.LogicalJobID, want int, finished <-chan struct{}) <-chan error {
	result := make(chan error, 1)
	go func() {
		defer close(result)
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for {
			var got int
			if err := pool.PGX().QueryRow(ctx, `SELECT count(*) FROM postgres_job_attempts WHERE logical_job_id = $1`, string(logicalJobID)).Scan(&got); err != nil {
				result <- err
				return
			}
			if got != want {
				result <- fmt.Errorf("attempts = %d, want %d", got, want)
				return
			}
			select {
			case <-finished:
				result <- nil
				return
			case <-ticker.C:
			}
		}
	}()
	return result
}
