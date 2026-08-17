//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/jobs"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresJobsLeaseSafetyProcess(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("SIGTERM process lifecycle is Unix-specific")
	}
	ctx, pool, store := newPostgresJobsFixture(t)
	repositoryRoot, binary := buildPostgresJobsTestWorker(t)
	createPostgresJobsEffectLedger(ctx, t, pool)
	prepared := stageDuePostgresRecoveryJob(ctx, t, pool, store, "lease-safety")
	logicalJobID := prepared.Identity().LogicalJobID

	firstAppName := "jobs-lease-first"
	firstFiles := jobsWorkerLeaseFiles(t)
	first := startPostgresJobsTestWorker(t, repositoryRoot, binary, postgresJobsTestWorkerOptions{
		Pool: pool, AppName: firstAppName, Handler: "lease", Files: firstFiles,
	})
	waitForPostgresJobsWorkerReady(t, first.addr)
	waitForPostgresJobsAttempt(ctx, t, pool, logicalJobID, 1)
	waitForPostgresJobsTestFile(t, firstFiles.entered, "handler start")

	locker, lockTx := lockPostgresJobsRow(ctx, t, pool, logicalJobID)
	defer locker.Release()
	defer func() { _ = lockTx.Rollback(ctx) }()
	firstPID := waitForPostgresJobsBlockedWorkerPID(ctx, t, pool, firstAppName, locker.Conn().PgConn().PID())
	firstPIDs := postgresJobsWorkerPIDs(ctx, t, pool, firstAppName)
	pids, pidsReady := monitorPostgresJobsTerminatedWorkerPID(ctx, pool, firstAppName, firstPIDs, firstPID, first.finished)
	waitForPostgresJobsMonitorStart(t, pidsReady, "terminated worker Session PID")
	var terminated bool
	if err := pool.PGX().QueryRow(ctx, `SELECT pg_terminate_backend($1)`, firstPID).Scan(&terminated); err != nil {
		t.Fatalf("terminate reserved Session %d: %v", firstPID, err)
	}
	if !terminated {
		t.Fatalf("terminate reserved Session %d = false", firstPID)
	}
	if err := lockTx.Rollback(ctx); err != nil {
		t.Fatalf("release blocked control operation: %v", err)
	}
	waitForPostgresJobsTestFile(t, firstFiles.cancelled, "attempt cancellation")
	if !postgresJobsLeaseIsFuture(ctx, t, pool, logicalJobID) {
		t.Fatal("attempt cancellation happened after lease expiry")
	}
	waitForPostgresJobsReadinessClosed(t, first.addr)
	waitForPostgresJobsWorkerPIDGone(ctx, t, pool, firstAppName, firstPID)
	waittest.Until(t, 15*time.Second, func() bool {
		marker, err := os.ReadFile(firstFiles.unsafeDrain)
		return err == nil && string(marker) == "unsafe drain\n"
	}, "complete unsafe drain marker")
	drainMarkerRead := time.Now()
	firstErr, firstExited := waitForPostgresJobsWorkerExit(first, 2*time.Second)
	if !firstExited {
		t.Fatalf("first worker did not exit within two seconds of unsafe drain\n%s", first.output.String())
	}
	if firstErr == nil {
		t.Fatalf("first worker exit = nil, want terminal failure\n%s", first.output.String())
	}
	if elapsed := time.Since(drainMarkerRead); elapsed > 2*time.Second {
		t.Fatalf("first worker unsafe-drain exit elapsed = %s, want at most 2s", elapsed)
	}
	if _, err := os.Stat(firstFiles.unsafeCleanup); !os.IsNotExist(err) {
		t.Fatalf("first worker unsafe cleanup marker error = %v, want absent", err)
	}
	if !strings.Contains(first.output.String(), "postgres jobs control Session terminal") || !strings.Contains(first.output.String(), "SQLSTATE 57P01") {
		t.Fatalf("first worker did not report terminal Session 57P01\n%s", first.output.String())
	}
	if strings.Contains(first.output.String(), "join jobs worker coordinator") || strings.Contains(first.output.String(), "join jobs-worker diagnostics") {
		t.Fatalf("first worker performed forbidden post-drain cleanup\n%s", first.output.String())
	}

	postClosure := stageDuePostgresJob(ctx, t, pool, store, "lease-safety-post-closure")
	claims := monitorPostgresJobsPostFaultClaims(ctx, pool, postClosure.Identity().LogicalJobID, first.finished)
	assertPostgresJobsMonitor(t, pids, "replacement Session PID")
	assertPostgresJobsMonitor(t, claims, "post-fault claim")
	assertPostgresJobsAttemptCount(ctx, t, pool, postClosure.Identity().LogicalJobID, 0)
	if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_jobs SET state = 'scheduled', available_at = clock_timestamp() + interval '1 hour' WHERE logical_job_id = $1`, string(postClosure.Identity().LogicalJobID)); err != nil {
		t.Fatalf("isolate post-closure job: %v", err)
	}
	if drains := strings.Count(first.output.String(), "join drained job attempts"); drains != 1 {
		t.Fatalf("terminal worker drain events = %d, want 1\n%s", drains, first.output.String())
	}
	if _, err := os.Stat(firstFiles.cleanup); !os.IsNotExist(err) {
		t.Fatalf("first worker cleanup marker error = %v, want unsafe cleanup absent", err)
	}

	waitForPostgresJobsLeaseExpiry(ctx, t, pool, logicalJobID, "first attempt lease expiry")
	secondAppName := "jobs-lease-second"
	secondFiles := jobsWorkerLeaseFiles(t)
	secondLocker, secondLockTx := lockPostgresJobsRow(ctx, t, pool, logicalJobID)
	defer secondLocker.Release()
	defer func() { _ = secondLockTx.Rollback(ctx) }()
	second := startPostgresJobsTestWorker(t, repositoryRoot, binary, postgresJobsTestWorkerOptions{
		Pool: pool, AppName: secondAppName, Handler: "recovery", Files: secondFiles, EffectGate: secondFiles.effectGate,
	})
	secondPID := waitForPostgresJobsBlockedWorkerPID(ctx, t, pool, secondAppName, secondLocker.Conn().PgConn().PID())
	if secondPID == firstPID {
		t.Fatalf("replacement Session PID = %d, want distinct from terminated PID", secondPID)
	}
	if err := secondLockTx.Rollback(ctx); err != nil {
		t.Fatalf("release replacement control operation: %v", err)
	}
	waitForPostgresJobsWorkerReady(t, second.addr)
	waitForPostgresJobsFencedRecovery(ctx, t, pool, logicalJobID)
	waitForPostgresJobsTestFile(t, secondFiles.entered, "fenced recovery handler start")
	if err := os.WriteFile(secondFiles.effectGate, []byte("release\n"), 0o600); err != nil {
		t.Fatalf("release fenced recovery effect: %v", err)
	}
	waitForPostgresJobsTerminalRecovery(ctx, t, pool, prepared)
	assertPostgresJobsEffectCount(ctx, t, pool, prepared.Identity(), 1)
	if err := second.process.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signal replacement worker: %v", err)
	}
	secondErr, secondExited := waitForPostgresJobsWorkerExit(second, 12*time.Second)
	if !secondExited {
		t.Fatalf("replacement worker did not exit inside the hard bound\n%s", second.output.String())
	}
	if secondErr != nil {
		t.Fatalf("replacement worker exit = %v\n%s", secondErr, second.output.String())
	}
}

type postgresJobsTestWorkerOptions struct {
	Pool         *postgres.Pool
	AppName      string
	Handler      string
	Files        jobsWorkerLeaseFileSet
	EffectGate   string
	CompleteGate string
	Result       string
}

type jobsWorkerLeaseFileSet struct {
	entered, cancelled, cleanup, unsafeDrain, unsafeCleanup, effect, effectGate, completeGate string
}

func jobsWorkerLeaseFiles(t *testing.T) jobsWorkerLeaseFileSet {
	t.Helper()
	dir := t.TempDir()
	return jobsWorkerLeaseFileSet{
		entered: filepath.Join(dir, "entered"), cancelled: filepath.Join(dir, "cancelled"),
		cleanup: filepath.Join(dir, "cleanup"), unsafeDrain: filepath.Join(dir, "unsafe-drain"), unsafeCleanup: filepath.Join(dir, "unsafe-cleanup"), effect: filepath.Join(dir, "effect"),
		effectGate: filepath.Join(dir, "effect-gate"), completeGate: filepath.Join(dir, "complete-gate"),
	}
}

type postgresJobsTestWorker struct {
	process  *exec.Cmd
	addr     string
	output   bytes.Buffer
	waited   chan error
	finished chan struct{}
}

type postgresJobsWorkerArtifact struct {
	repositoryRoot string
	binary         string
	hash           string
}

func buildPostgresJobsTestWorker(t *testing.T) (string, string) {
	t.Helper()
	repositoryRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "jobs-worker")
	build := exec.CommandContext(t.Context(), "go", "build", "-tags", "jobs_test_worker", "-o", binary, "./cmd/jobs-worker")
	build.Dir = repositoryRoot
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build jobs worker: %v\n%s", err, output)
	}
	return repositoryRoot, binary
}

func buildPostgresJobsCompatibilityArtifacts(t *testing.T) (postgresJobsWorkerArtifact, postgresJobsWorkerArtifact, postgresJobsWorkerArtifact) {
	t.Helper()
	repositoryRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	build := func(name string, includeV2 bool) postgresJobsWorkerArtifact {
		t.Helper()
		artifactRoot := filepath.Join(t.TempDir(), name)
		copyCurrentRepository(t, repositoryRoot, artifactRoot)
		if includeV2 {
			builderPath := filepath.Join(artifactRoot, "cmd", "jobs-worker", "builder_testworker.go")
			builder, readErr := os.ReadFile(builderPath)
			if readErr != nil {
				t.Fatalf("read %s builder: %v", name, readErr)
			}
			const anchor = "\tacceptanceInput := definitionInput\n"
			const registration = "\tcompatibilityInput := definitionInput\n" +
				"\tcompatibilityInput.Revision.ArgsVersion = \"v2\"\n" +
				"\tcompatibility, err := jobs.NewDefinition(compatibilityInput)\n" +
				"\tif err != nil {\n\t\treturn nil, nil, err\n\t}\n" +
				"\tif err := jobs.Register(registry, compatibility, handler); err != nil {\n\t\treturn nil, nil, err\n\t}\n\n" +
				anchor
			updated := strings.Replace(string(builder), anchor, registration, 1)
			if updated == string(builder) {
				t.Fatalf("inject %s v2 registry: builder anchor not found", name)
			}
			if writeErr := os.WriteFile(builderPath, []byte(updated), 0o600); writeErr != nil {
				t.Fatalf("write %s builder: %v", name, writeErr)
			}
		}
		binary := filepath.Join(artifactRoot, "jobs-worker")
		command := exec.CommandContext(t.Context(), "go", "build", "-trimpath", "-tags", "jobs_test_worker", "-o", binary, "./cmd/jobs-worker")
		command.Dir = artifactRoot
		if output, buildErr := command.CombinedOutput(); buildErr != nil {
			t.Fatalf("build %s jobs worker: %v\n%s", name, buildErr, output)
		}
		contents, readErr := os.ReadFile(binary)
		if readErr != nil {
			t.Fatalf("read %s jobs worker: %v", name, readErr)
		}
		digest := sha256.Sum256(contents)
		return postgresJobsWorkerArtifact{repositoryRoot: artifactRoot, binary: binary, hash: hex.EncodeToString(digest[:])}
	}
	oldOnly := build("old-only", false)
	nMinusOne := build("n-minus-one", true)
	n := build("n", true)
	if oldOnly.hash == nMinusOne.hash || nMinusOne.hash != n.hash {
		t.Fatalf("compatibility artifact hashes old=%s n-1=%s n=%s", oldOnly.hash, nMinusOne.hash, n.hash)
	}
	return oldOnly, nMinusOne, n
}

func startPostgresJobsTestWorker(t *testing.T, repositoryRoot, binary string, options postgresJobsTestWorkerOptions) *postgresJobsTestWorker {
	t.Helper()
	worker := &postgresJobsTestWorker{addr: waittest.FreeTCPAddr(t, "jobs worker diagnostics"), waited: make(chan error, 1), finished: make(chan struct{})}
	worker.process = exec.Command(binary)
	worker.process.Dir = repositoryRoot
	worker.process.Env = append(cleanServiceEnvironment(os.Environ()),
		"APP__APP__ENV=integration",
		"APP__AUTHN__ISSUER=",
		"APP__OBSERVABILITY__METRICS__ADDR="+worker.addr,
		"APP__POSTGRES__ENABLED=true",
		"APP__POSTGRES__DSN="+postgresJobsDSNParam(t, postgresJobsDSN(options.Pool), "application_name", options.AppName),
		"APP__POSTGRES__MAX_OPEN_CONNS=8",
		"APP__POSTGRES__STATEMENT_TIMEOUT=1s",
		"APP__POSTGRES__ACQUIRE_TIMEOUT=500ms",
		"APP__JOBS__ENABLED=true",
		"APP__JOBS__POLL_INTERVAL=10ms",
		"APP__JOBS__MAX_CONCURRENCY=1",
		"APP__JOBS__LEASE_DURATION=6s",
		"APP__JOBS__STORE_OPERATION_TIMEOUT=1s",
		"APP__JOBS__OBSERVATION_INTERVAL=10ms",
		"APP__JOBS__DRAIN_TIMEOUT=3s",
		"APP__HTTP__GRACE_PERIOD=12s",
		"APP__HTTP__SHUTDOWN_TIMEOUT=1s",
		"APP__HTTP__WRITE_TIMEOUT=1s",
		"APP__HTTP__READINESS_TIMEOUT=1s",
		"APP__HTTP__REQUEST_TIMEOUT=1s",
		"APP__HTTP__READINESS_PROPAGATION_DELAY=0s",
		"JOBS_WORKER_TEST_CLEANUP_FILE="+options.Files.cleanup,
		"JOBS_WORKER_TEST_UNSAFE_DRAIN_FILE="+options.Files.unsafeDrain,
		"JOBS_WORKER_TEST_UNSAFE_CLEANUP_FILE="+options.Files.unsafeCleanup,
	)
	if options.Handler != "" {
		worker.process.Env = append(worker.process.Env,
			"JOBS_WORKER_TEST_HANDLER="+options.Handler,
			"JOBS_WORKER_TEST_ENTERED_FILE="+options.Files.entered,
			"JOBS_WORKER_TEST_CANCELLED_FILE="+options.Files.cancelled,
			"JOBS_WORKER_TEST_RESULT="+options.Result,
		)
	}
	if options.Handler == "recovery" {
		worker.process.Env = append(worker.process.Env,
			"JOBS_WORKER_TEST_EFFECT_DSN="+postgresJobsDSN(options.Pool),
			"JOBS_WORKER_TEST_EFFECT_FILE="+options.Files.effect,
			"JOBS_WORKER_TEST_EFFECT_GATE="+options.EffectGate,
			"JOBS_WORKER_TEST_COMPLETE_GATE="+options.CompleteGate,
		)
	}
	worker.process.Stdout, worker.process.Stderr = &worker.output, &worker.output
	t.Cleanup(func() { t.Logf("jobs worker output:\n%s", worker.output.String()) })
	if err := worker.process.Start(); err != nil {
		t.Fatalf("start jobs worker: %v", err)
	}
	go func() {
		worker.waited <- worker.process.Wait()
		close(worker.finished)
	}()
	t.Cleanup(func() {
		select {
		case <-worker.finished:
		default:
			_ = worker.process.Process.Kill()
			<-worker.finished
		}
	})
	return worker
}

func waitForPostgresJobsBlockedWorkerPID(ctx context.Context, t *testing.T, pool *postgres.Pool, appName string, blockerPID uint32) uint32 {
	t.Helper()
	var pid uint32
	waittest.UntilFunc(t, 5*time.Second, func() bool {
		if err := pool.PGX().QueryRow(ctx, `
SELECT coalesce(min(pid), 0)
FROM pg_stat_activity
WHERE datname = current_database() AND application_name = $1
  AND $2::integer = ANY(pg_blocking_pids(pid))`, appName, blockerPID).Scan(&pid); err != nil {
			t.Fatal(err)
		}
		return pid != 0
	}, func() string { return "blocked reserved worker Session PID; " + postgresJobsSessionSnapshot(ctx, pool) })
	return pid
}

func postgresJobsWorkerPIDs(ctx context.Context, t *testing.T, pool *postgres.Pool, appName string) []int32 {
	t.Helper()
	rows, err := pool.PGX().Query(ctx, `SELECT pid FROM pg_stat_activity WHERE datname = current_database() AND application_name = $1`, appName)
	if err != nil {
		t.Fatalf("list worker Session PIDs: %v", err)
	}
	defer rows.Close()
	var pids []int32
	for rows.Next() {
		var pid int32
		if err := rows.Scan(&pid); err != nil {
			t.Fatal(err)
		}
		pids = append(pids, pid)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return pids
}

func postgresJobsSessionSnapshot(ctx context.Context, pool *postgres.Pool) string {
	var snapshot string
	if err := pool.PGX().QueryRow(ctx, `
SELECT coalesce(string_agg(pid::text || ':' || application_name, ',' ORDER BY pid), '')
FROM pg_stat_activity
WHERE datname = current_database()`).Scan(&snapshot); err != nil {
		return err.Error()
	}
	return snapshot
}

func waitForPostgresJobsWorkerPIDGone(ctx context.Context, t *testing.T, pool *postgres.Pool, appName string, pid uint32) {
	t.Helper()
	waittest.Until(t, 5*time.Second, func() bool {
		var present bool
		if err := pool.PGX().QueryRow(ctx, `SELECT exists(SELECT 1 FROM pg_stat_activity WHERE datname = current_database() AND application_name = $1 AND pid = $2)`, appName, pid).Scan(&present); err != nil {
			t.Fatal(err)
		}
		return !present
	}, "terminated worker Session disappearance")
}

func monitorPostgresJobsTerminatedWorkerPID(ctx context.Context, pool *postgres.Pool, appName string, initialPIDs []int32, terminatedPID uint32, finished <-chan struct{}) (<-chan error, <-chan error) {
	result := make(chan error, 1)
	ready := make(chan error, 1)
	go func() {
		defer close(result)
		defer close(ready)
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		started := false
		for {
			var replacement bool
			var terminatedPresent bool
			err := pool.PGX().QueryRow(ctx, `
SELECT coalesce(bool_or(pid <> ALL($2::integer[])), false), coalesce(bool_or(pid = $3), false)
FROM pg_stat_activity
WHERE datname = current_database() AND application_name = $1`, appName, initialPIDs, terminatedPID).Scan(&replacement, &terminatedPresent)
			if err != nil {
				err = fmt.Errorf("observe terminated worker Session PIDs: %w", err)
				if !started {
					ready <- err
				}
				result <- err
				return
			}
			if replacement {
				err = errors.New("terminated worker opened a replacement Session PID")
				if !started {
					ready <- err
				}
				result <- err
				return
			}
			if !started {
				ready <- nil
				started = true
			}
			if !terminatedPresent {
				select {
				case <-finished:
					result <- nil
					return
				case <-ticker.C:
				}
				continue
			}
			select {
			case <-finished:
				result <- nil
				return
			case <-ticker.C:
			}
		}
	}()
	return result, ready
}

func waitForPostgresJobsMonitorStart(t *testing.T, ready <-chan error, description string) {
	t.Helper()
	select {
	case err := <-ready:
		if err != nil {
			t.Fatalf("%s monitor start: %v", description, err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("%s monitor did not start", description)
	}
}

func monitorPostgresJobsPostFaultClaims(ctx context.Context, pool *postgres.Pool, logicalJobID jobs.LogicalJobID, finished <-chan struct{}) <-chan error {
	result := make(chan error, 1)
	go func() {
		defer close(result)
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for {
			var attempts int
			err := pool.PGX().QueryRow(ctx, `SELECT count(*) FROM postgres_job_attempts WHERE logical_job_id = $1`, string(logicalJobID)).Scan(&attempts)
			if err != nil {
				result <- fmt.Errorf("observe post-fault claims: %w", err)
				return
			}
			if attempts != 0 {
				result <- fmt.Errorf("post-fault attempts = %d", attempts)
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

func assertPostgresJobsMonitor(t *testing.T, result <-chan error, description string) {
	t.Helper()
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("%s monitor: %v", description, err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("%s monitor did not stop", description)
	}
}

func waitForPostgresJobsWorkerReady(t *testing.T, addr string) {
	t.Helper()
	client := &http.Client{Timeout: 100 * time.Millisecond}
	waittest.Until(t, 5*time.Second, func() bool {
		response, err := client.Get("http://" + addr + "/health/ready")
		if err != nil {
			return false
		}
		defer response.Body.Close()
		return response.StatusCode == http.StatusOK
	}, "jobs worker readiness")
}

func waitForPostgresJobsReadinessClosed(t *testing.T, addr string) {
	t.Helper()
	client := &http.Client{Timeout: 100 * time.Millisecond}
	waittest.Until(t, 5*time.Second, func() bool {
		response, err := client.Get("http://" + addr + "/health/ready")
		if err != nil {
			return false
		}
		defer response.Body.Close()
		return response.StatusCode == http.StatusServiceUnavailable
	}, "jobs worker readiness withdrawal")
}

func waitForPostgresJobsAttempt(ctx context.Context, t *testing.T, pool *postgres.Pool, logicalJobID jobs.LogicalJobID, want int) {
	t.Helper()
	waittest.Until(t, 5*time.Second, func() bool {
		var attempts int
		if err := pool.PGX().QueryRow(ctx, `SELECT count(*) FROM postgres_job_attempts WHERE logical_job_id = $1`, string(logicalJobID)).Scan(&attempts); err != nil {
			t.Fatal(err)
		}
		return attempts == want
	}, fmt.Sprintf("%d durable attempts", want))
}

func waitForPostgresJobsTestFile(t *testing.T, path, description string) {
	t.Helper()
	waittest.Until(t, 5*time.Second, func() bool {
		contents, err := os.ReadFile(path)
		return err == nil && len(contents) > 0
	}, description)
}

func lockPostgresJobsRow(ctx context.Context, t *testing.T, pool *postgres.Pool, logicalJobID jobs.LogicalJobID) (*pgxpool.Conn, pgx.Tx) {
	t.Helper()
	locker, err := pool.PGX().Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire control-operation blocker: %v", err)
	}
	lockTx, err := locker.Begin(ctx)
	if err != nil {
		locker.Release()
		t.Fatalf("begin control-operation blocker: %v", err)
	}
	if _, err := lockTx.Exec(ctx, `SELECT 1 FROM postgres_jobs WHERE logical_job_id = $1 FOR UPDATE`, string(logicalJobID)); err != nil {
		_ = lockTx.Rollback(ctx)
		locker.Release()
		t.Fatalf("block control operation: %v", err)
	}
	return locker, lockTx
}

func postgresJobsLeaseIsFuture(ctx context.Context, t *testing.T, pool *postgres.Pool, logicalJobID jobs.LogicalJobID) bool {
	t.Helper()
	var beforeExpiry bool
	if err := pool.PGX().QueryRow(ctx, `SELECT lease_expires_at > clock_timestamp() FROM postgres_jobs WHERE logical_job_id = $1`, string(logicalJobID)).Scan(&beforeExpiry); err != nil {
		t.Fatalf("compare cancellation with lease expiry: %v", err)
	}
	return beforeExpiry
}

func postgresJobsLeaseExpired(ctx context.Context, t *testing.T, pool *postgres.Pool, logicalJobID jobs.LogicalJobID) bool {
	t.Helper()
	var expired bool
	if err := pool.PGX().QueryRow(ctx, `SELECT lease_expires_at < clock_timestamp() FROM postgres_jobs WHERE logical_job_id = $1`, string(logicalJobID)).Scan(&expired); err != nil {
		t.Fatal(err)
	}
	return expired
}

func waitForPostgresJobsLeaseExpiry(
	ctx context.Context,
	t *testing.T,
	pool *postgres.Pool,
	logicalJobID jobs.LogicalJobID,
	description string,
) {
	t.Helper()
	var remainingMilliseconds int64
	if err := pool.PGX().QueryRow(ctx, `
SELECT greatest(coalesce((extract(epoch FROM (lease_expires_at - clock_timestamp())) * 1000)::bigint, 0), 0)
FROM postgres_jobs
WHERE logical_job_id = $1`, logicalJobID).Scan(&remainingMilliseconds); err != nil {
		t.Fatalf("read jobs lease deadline: %v", err)
	}
	waittest.Until(t, time.Duration(remainingMilliseconds)*time.Millisecond+2*time.Second, func() bool {
		return postgresJobsLeaseExpired(ctx, t, pool, logicalJobID)
	}, description)
}

func waitForPostgresJobsFencedRecovery(ctx context.Context, t *testing.T, pool *postgres.Pool, logicalJobID jobs.LogicalJobID) {
	t.Helper()
	waittest.Until(t, 5*time.Second, func() bool {
		var state, outcome, effect string
		err := pool.PGX().QueryRow(ctx, `
		SELECT coalesce(final_state, ''), coalesce(outcome, ''), coalesce(effect_status, '')
		FROM postgres_job_attempts
		WHERE logical_job_id = $1 AND attempt_generation = 1`, string(logicalJobID)).Scan(&state, &outcome, &effect)
		if err != nil {
			t.Fatal(err)
		}
		return state == string(jobs.StateRetryWait) && outcome == string(jobs.OutcomeLost) && effect == string(jobs.EffectUnknown)
	}, "fenced lease-loss recovery")
}

func waitForPostgresJobsWorkerExit(worker *postgresJobsTestWorker, timeout time.Duration) (error, bool) {
	select {
	case err := <-worker.waited:
		return err, true
	case <-time.After(timeout):
		return nil, false
	}
}
