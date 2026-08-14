//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgresjobs"
	"github.com/example/go-service-template-rest/internal/jobs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const postgresJobsOperationBudget = 150 * time.Millisecond

func testPostgresJobsOperationTimerHeadroom(t *testing.T) {
	ctx, pool, store := newPostgresJobsOperationBudgetFixture(t)
	prepared := stageDuePostgresJob(ctx, t, pool, store, "operation-timer-headroom")
	probePool, probeStore := newPostgresJobsStatementStore(ctx, t, pool)
	defer probePool.Close()
	installPostgresJobsTimerProbe(ctx, t, pool)

	session := acquirePostgresJobsSession(ctx, t, probeStore)
	defer session.Release(ctx)
	_, err := session.Claim(ctx, postgresjobs.ClaimOptions{
		RegistryKeys: []jobs.Revision{prepared.Revision()}, WorkerID: "worker-operation-timer-headroom", Limit: 1, LeaseDuration: time.Minute,
	})
	postgresErr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok || postgresErr.Message != "statement_timeout=140ms lock_timeout=140ms" {
		t.Fatalf("operation timer probe error = %v, want server timers with 10ms headroom", err)
	}
}

func TestPostgresJobsOperationBudget(t *testing.T) {
	t.Run("server timer headroom", testPostgresJobsOperationTimerHeadroom)

	for _, testCase := range postgresJobsOperationBudgetCases() {
		t.Run(testCase.name+" client cancellation", func(t *testing.T) {
			ctx, pool, store := newPostgresJobsOperationBudgetFixture(t)
			logicalJobID, operation := testCase.prepare(ctx, t, pool, store)
			session := acquirePostgresJobsSession(ctx, t, store)
			defer session.Release(ctx)
			before := postgresJobsOperationState(ctx, t, pool, logicalJobID)

			locker, lockTx := lockPostgresJobsTable(ctx, t, pool)
			operationCtx, cancel := context.WithCancel(ctx)
			done := make(chan error, 1)
			go func() { done <- operation(operationCtx, session) }()
			waitForPostgresJobsBlocker(ctx, t, pool, session.BackendPID(), locker.Conn().PgConn().PID())
			cancel()
			if err := <-done; !errors.Is(err, context.Canceled) {
				t.Fatalf("%s cancellation error = %v, want context.Canceled", testCase.name, err)
			}
			unlockPostgresJobsTable(t, locker, lockTx)
			assertPostgresJobsOperationCleanup(ctx, t, pool, session, logicalJobID, before)
		})

		t.Run(testCase.name+" lock timeout", func(t *testing.T) {
			ctx, pool, store := newPostgresJobsOperationBudgetFixture(t)
			logicalJobID, operation := testCase.prepare(ctx, t, pool, store)
			session := acquirePostgresJobsSession(ctx, t, store)
			defer session.Release(ctx)
			before := postgresJobsOperationState(ctx, t, pool, logicalJobID)

			locker, lockTx := lockPostgresJobsTable(ctx, t, pool)
			done := make(chan error, 1)
			go func() { done <- operation(ctx, session) }()
			waitForPostgresJobsBlocker(ctx, t, pool, session.BackendPID(), locker.Conn().PgConn().PID())
			assertPostgresJobsTimeout(t, <-done, "55P03", "57014")
			unlockPostgresJobsTable(t, locker, lockTx)
			assertPostgresJobsOperationCleanup(ctx, t, pool, session, logicalJobID, before)
		})

		t.Run(testCase.name+" statement timeout", func(t *testing.T) {
			ctx, pool, store := newPostgresJobsOperationBudgetFixture(t)
			logicalJobID, operation := testCase.prepare(ctx, t, pool, store)
			before := postgresJobsOperationState(ctx, t, pool, logicalJobID)
			statementPool, statementStore := newPostgresJobsStatementStore(ctx, t, pool)
			defer statementPool.Close()
			installPostgresJobsStatementBlocker(ctx, t, pool)
			session := acquirePostgresJobsSession(ctx, t, statementStore)
			defer session.Release(ctx)
			assertPostgresJobsTimeout(t, operation(ctx, session), "57014")
			assertPostgresJobsOperationCleanup(ctx, t, pool, session, logicalJobID, before)
		})
	}
}

type postgresJobsOperationBudgetCase struct {
	name    string
	prepare func(context.Context, *testing.T, *postgres.Pool, *postgresjobs.Store) (jobs.LogicalJobID, func(context.Context, *postgresjobs.Session) error)
}

func postgresJobsOperationBudgetCases() []postgresJobsOperationBudgetCase {
	return []postgresJobsOperationBudgetCase{
		{
			name: "claim",
			prepare: func(ctx context.Context, t *testing.T, pool *postgres.Pool, store *postgresjobs.Store) (jobs.LogicalJobID, func(context.Context, *postgresjobs.Session) error) {
				prepared := stageDuePostgresJob(ctx, t, pool, store, "operation-claim")
				return prepared.Identity().LogicalJobID, func(ctx context.Context, session *postgresjobs.Session) error {
					_, err := session.Claim(ctx, postgresjobs.ClaimOptions{
						RegistryKeys: []jobs.Revision{prepared.Revision()}, WorkerID: "worker-operation-claim", Limit: 1, LeaseDuration: time.Minute,
					})
					return err
				}
			},
		},
		{
			name: "claim resolution",
			prepare: func(ctx context.Context, t *testing.T, pool *postgres.Pool, store *postgresjobs.Store) (jobs.LogicalJobID, func(context.Context, *postgresjobs.Session) error) {
				_, claimed := claimPostgresJob(ctx, t, pool, store, "operation-resolve", "worker-operation-resolve", time.Minute)
				return claimed.Attempt.LogicalJobID, func(ctx context.Context, session *postgresjobs.Session) error {
					_, err := session.ResolveClaims(ctx, []postgresjobs.AttemptIdentity{claimed.Attempt})
					return err
				}
			},
		},
		{
			name: "renew and cancellation observation",
			prepare: func(ctx context.Context, t *testing.T, pool *postgres.Pool, store *postgresjobs.Store) (jobs.LogicalJobID, func(context.Context, *postgresjobs.Session) error) {
				_, claimed := claimPostgresJob(ctx, t, pool, store, "operation-renew", "worker-operation-renew", time.Minute)
				return claimed.Attempt.LogicalJobID, func(ctx context.Context, session *postgresjobs.Session) error {
					_, err := session.Renew(ctx, []postgresjobs.AttemptIdentity{claimed.Attempt}, time.Minute)
					return err
				}
			},
		},
		{
			name: "finalize",
			prepare: func(ctx context.Context, t *testing.T, pool *postgres.Pool, store *postgresjobs.Store) (jobs.LogicalJobID, func(context.Context, *postgresjobs.Session) error) {
				_, claimed := claimPostgresJob(ctx, t, pool, store, "operation-finalize", "worker-operation-finalize", time.Minute)
				input := postgresjobs.FinalizeInput{Attempt: claimed.Attempt, Transition: postgresJobsSucceededTransition(claimed.AttemptNumber)}
				return claimed.Attempt.LogicalJobID, func(ctx context.Context, session *postgresjobs.Session) error {
					_, err := session.Finalize(ctx, input)
					return err
				}
			},
		},
		{
			name: "rescue read",
			prepare: func(ctx context.Context, t *testing.T, pool *postgres.Pool, store *postgresjobs.Store) (jobs.LogicalJobID, func(context.Context, *postgresjobs.Session) error) {
				_, claimed := claimPostgresJob(ctx, t, pool, store, "operation-rescue-read", "worker-operation-rescue-read", time.Minute)
				expirePostgresJobsAttempt(ctx, t, pool, claimed.Attempt)
				return claimed.Attempt.LogicalJobID, func(ctx context.Context, session *postgresjobs.Session) error {
					_, err := session.RescueCandidates(ctx, 1)
					return err
				}
			},
		},
		{
			name: "rescue write",
			prepare: func(ctx context.Context, t *testing.T, pool *postgres.Pool, store *postgresjobs.Store) (jobs.LogicalJobID, func(context.Context, *postgresjobs.Session) error) {
				_, claimed := claimPostgresJob(ctx, t, pool, store, "operation-rescue-write", "worker-operation-rescue-write", time.Minute)
				expirePostgresJobsAttempt(ctx, t, pool, claimed.Attempt)
				input := postgresjobs.RescueInput{Attempt: claimed.Attempt, Transition: postgresJobsLostTransition(claimed.AttemptNumber), FailureCode: "lease_lost"}
				return claimed.Attempt.LogicalJobID, func(ctx context.Context, session *postgresjobs.Session) error {
					_, err := session.Rescue(ctx, input)
					return err
				}
			},
		},
		{
			name: "observation",
			prepare: func(ctx context.Context, t *testing.T, pool *postgres.Pool, store *postgresjobs.Store) (jobs.LogicalJobID, func(context.Context, *postgresjobs.Session) error) {
				prepared := stageDuePostgresJob(ctx, t, pool, store, "operation-observe")
				return prepared.Identity().LogicalJobID, func(ctx context.Context, session *postgresjobs.Session) error {
					_, err := session.Observe(ctx, []jobs.Revision{prepared.Revision()})
					return err
				}
			},
		},
	}
}

func newPostgresJobsOperationBudgetFixture(t *testing.T) (context.Context, *postgres.Pool, *postgresjobs.Store) {
	t.Helper()
	ctx, pool, _ := newPostgresJobsFixture(t)
	store, err := postgresjobs.NewStore(pool, postgresjobs.StoreOptions{
		OperationTimeout: postgresJobsOperationBudget,
		StatementTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("postgresjobs.NewStore(operation budget): %v", err)
	}
	return ctx, pool, store
}

func newPostgresJobsStatementStore(ctx context.Context, t *testing.T, pool *postgres.Pool) (*postgres.Pool, *postgresjobs.Store) {
	t.Helper()
	var database string
	if err := pool.PGX().QueryRow(ctx, "SELECT current_database()").Scan(&database); err != nil {
		t.Fatalf("read operation-budget database: %v", err)
	}
	role := "jobs_operation_budget_" + database
	roleIdentifier := pgx.Identifier{role}.Sanitize()
	if _, err := pool.PGX().Exec(ctx, fmt.Sprintf("CREATE ROLE %s LOGIN PASSWORD 'jobs-operation-budget'", roleIdentifier)); err != nil {
		t.Fatalf("create operation-budget role: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if _, err := pool.PGX().Exec(cleanupCtx, "DROP OWNED BY "+roleIdentifier); err != nil {
			t.Errorf("drop operation-budget role grants: %v", err)
		}
		if _, err := pool.PGX().Exec(cleanupCtx, "DROP ROLE "+roleIdentifier); err != nil {
			t.Errorf("drop operation-budget role: %v", err)
		}
	})
	statements := []string{
		"GRANT CONNECT ON DATABASE " + pgx.Identifier{database}.Sanitize() + " TO " + roleIdentifier,
		"GRANT USAGE ON SCHEMA public TO " + roleIdentifier,
		"GRANT SELECT, INSERT, UPDATE ON postgres_job_actions, postgres_job_attempts, postgres_job_claim_scopes, postgres_jobs TO " + roleIdentifier,
	}
	for _, statement := range statements {
		if _, err := pool.PGX().Exec(ctx, statement); err != nil {
			t.Fatalf("grant operation-budget role with %q: %v", statement, err)
		}
	}
	rolePool, _ := newPostgresJobsStore(
		ctx, t, postgresJobsProbeDSNUser(t, postgresJobsDSN(pool), role, "jobs-operation-budget"), 2, postgresJobsOperationBudget,
	)
	roleStore, err := postgresjobs.NewStore(rolePool, postgresjobs.StoreOptions{
		OperationTimeout: postgresJobsOperationBudget,
		StatementTimeout: 5 * time.Second,
	})
	if err != nil {
		rolePool.Close()
		t.Fatalf("postgresjobs.NewStore(operation-budget role): %v", err)
	}
	return rolePool, roleStore
}

func postgresJobsSucceededTransition(attemptsUsed uint32) jobs.Transition {
	return jobs.Transition{
		State: jobs.StateSucceeded, AttemptsUsed: attemptsUsed,
		ElapsedUsed: time.Second, Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted,
	}
}

func postgresJobsLostTransition(attemptsUsed uint32) jobs.Transition {
	return jobs.Transition{
		State: jobs.StateOutcomeUnknown, AttemptsUsed: attemptsUsed,
		ElapsedUsed: time.Second, Outcome: jobs.OutcomeLost, Effect: jobs.EffectUnknown,
	}
}

type postgresJobsPersistedOperationState struct {
	state              string
	attemptGeneration  int64
	recoveryGeneration int64
	workerID           string
	attempts           int
}

func postgresJobsOperationState(ctx context.Context, t *testing.T, pool *postgres.Pool, logicalJobID jobs.LogicalJobID) postgresJobsPersistedOperationState {
	t.Helper()
	var state postgresJobsPersistedOperationState
	if err := pool.PGX().QueryRow(ctx, `
SELECT state, attempt_generation, recovery_generation, coalesce(current_worker_id, ''),
       (SELECT count(*) FROM postgres_job_attempts WHERE logical_job_id = $1)
FROM postgres_jobs
WHERE logical_job_id = $1`, string(logicalJobID)).Scan(
		&state.state, &state.attemptGeneration, &state.recoveryGeneration, &state.workerID, &state.attempts,
	); err != nil {
		t.Fatalf("read operation state: %v", err)
	}
	return state
}

func lockPostgresJobsTable(ctx context.Context, t *testing.T, pool *postgres.Pool) (*pgxpool.Conn, pgx.Tx) {
	t.Helper()
	locker, err := pool.PGX().Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire operation blocker: %v", err)
	}
	tx, err := locker.Begin(ctx)
	if err != nil {
		locker.Release()
		t.Fatalf("begin operation blocker: %v", err)
	}
	if _, err := tx.Exec(ctx, "LOCK TABLE postgres_jobs IN ACCESS EXCLUSIVE MODE"); err != nil {
		_ = tx.Rollback(context.Background())
		locker.Release()
		t.Fatalf("lock postgres jobs table: %v", err)
	}
	return locker, tx
}

func unlockPostgresJobsTable(t *testing.T, locker *pgxpool.Conn, tx pgx.Tx) {
	t.Helper()
	if err := tx.Rollback(context.Background()); err != nil {
		t.Fatalf("release operation blocker: %v", err)
	}
	locker.Release()
}

func installPostgresJobsStatementBlocker(ctx context.Context, t *testing.T, pool *postgres.Pool) {
	t.Helper()
	if _, err := pool.PGX().Exec(ctx, `
CREATE FUNCTION test_postgres_jobs_statement_blocker() RETURNS boolean
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM pg_sleep(1);
    RETURN true;
END;
$$;
ALTER TABLE postgres_job_actions ENABLE ROW LEVEL SECURITY;
ALTER TABLE postgres_job_actions FORCE ROW LEVEL SECURITY;
CREATE POLICY test_postgres_jobs_statement_blocker ON postgres_job_actions
    USING (test_postgres_jobs_statement_blocker())
    WITH CHECK (test_postgres_jobs_statement_blocker());
ALTER TABLE postgres_job_attempts ENABLE ROW LEVEL SECURITY;
ALTER TABLE postgres_job_attempts FORCE ROW LEVEL SECURITY;
CREATE POLICY test_postgres_jobs_statement_blocker ON postgres_job_attempts
    USING (test_postgres_jobs_statement_blocker())
    WITH CHECK (test_postgres_jobs_statement_blocker());
ALTER TABLE postgres_job_claim_scopes ENABLE ROW LEVEL SECURITY;
ALTER TABLE postgres_job_claim_scopes FORCE ROW LEVEL SECURITY;
CREATE POLICY test_postgres_jobs_statement_blocker ON postgres_job_claim_scopes
    USING (test_postgres_jobs_statement_blocker())
    WITH CHECK (test_postgres_jobs_statement_blocker());
ALTER TABLE postgres_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE postgres_jobs FORCE ROW LEVEL SECURITY;
CREATE POLICY test_postgres_jobs_statement_blocker ON postgres_jobs
    USING (test_postgres_jobs_statement_blocker())
    WITH CHECK (test_postgres_jobs_statement_blocker());`); err != nil {
		t.Fatalf("install statement blocker: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.PGX().Exec(context.Background(), `
DROP POLICY IF EXISTS test_postgres_jobs_statement_blocker ON postgres_job_actions;
ALTER TABLE postgres_job_actions NO FORCE ROW LEVEL SECURITY;
ALTER TABLE postgres_job_actions DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS test_postgres_jobs_statement_blocker ON postgres_job_attempts;
ALTER TABLE postgres_job_attempts NO FORCE ROW LEVEL SECURITY;
ALTER TABLE postgres_job_attempts DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS test_postgres_jobs_statement_blocker ON postgres_job_claim_scopes;
ALTER TABLE postgres_job_claim_scopes NO FORCE ROW LEVEL SECURITY;
ALTER TABLE postgres_job_claim_scopes DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS test_postgres_jobs_statement_blocker ON postgres_jobs;
ALTER TABLE postgres_jobs NO FORCE ROW LEVEL SECURITY;
ALTER TABLE postgres_jobs DISABLE ROW LEVEL SECURITY;
DROP FUNCTION IF EXISTS test_postgres_jobs_statement_blocker();`)
	})
}

func installPostgresJobsTimerProbe(ctx context.Context, t *testing.T, pool *postgres.Pool) {
	t.Helper()
	if _, err := pool.PGX().Exec(ctx, `
CREATE FUNCTION test_postgres_jobs_timer_probe() RETURNS boolean
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'statement_timeout=% lock_timeout=%', current_setting('statement_timeout'), current_setting('lock_timeout');
END;
$$;
ALTER TABLE postgres_job_claim_scopes ENABLE ROW LEVEL SECURITY;
ALTER TABLE postgres_job_claim_scopes FORCE ROW LEVEL SECURITY;
CREATE POLICY test_postgres_jobs_timer_probe ON postgres_job_claim_scopes
    USING (test_postgres_jobs_timer_probe())
    WITH CHECK (test_postgres_jobs_timer_probe());`); err != nil {
		t.Fatalf("install operation timer probe: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.PGX().Exec(context.Background(), `
DROP POLICY IF EXISTS test_postgres_jobs_timer_probe ON postgres_job_claim_scopes;
ALTER TABLE postgres_job_claim_scopes NO FORCE ROW LEVEL SECURITY;
ALTER TABLE postgres_job_claim_scopes DISABLE ROW LEVEL SECURITY;
DROP FUNCTION IF EXISTS test_postgres_jobs_timer_probe();`)
	})
}

func assertPostgresJobsTimeout(t *testing.T, err error, codes ...string) {
	t.Helper()
	if !errors.Is(err, postgresjobs.ErrOperationTimeout) {
		t.Fatalf("operation error = %v, want ErrOperationTimeout", err)
	}
	postgresErr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok || !slices.Contains(codes, postgresErr.Code) {
		t.Fatalf("operation PostgreSQL error = %v, want SQLSTATE in %v", err, codes)
	}
}

func assertPostgresJobsOperationCleanup(
	ctx context.Context,
	t *testing.T,
	pool *postgres.Pool,
	session *postgresjobs.Session,
	logicalJobID jobs.LogicalJobID,
	want postgresJobsPersistedOperationState,
) {
	t.Helper()
	if got := postgresJobsOperationState(ctx, t, pool, logicalJobID); got != want {
		t.Fatalf("operation state = %+v, want %+v", got, want)
	}
	pid := session.BackendPID()
	if pid == 0 {
		t.Fatal("operation Session lost its backend instead of remaining usable")
	}
	var state string
	if err := pool.PGX().QueryRow(ctx, "SELECT state FROM pg_stat_activity WHERE pid = $1", pid).Scan(&state); err != nil {
		t.Fatalf("read operation backend state: %v", err)
	}
	if state != "idle" {
		t.Fatalf("operation backend state = %q, want idle", state)
	}
	var locks int
	if err := pool.PGX().QueryRow(ctx, `
SELECT count(*)
FROM pg_locks
WHERE pid = $1
  AND relation IN ('postgres_jobs'::regclass, 'postgres_job_attempts'::regclass)`, pid).Scan(&locks); err != nil {
		t.Fatalf("read operation locks: %v", err)
	}
	if locks != 0 {
		t.Fatalf("operation backend retained %d job locks", locks)
	}
}
