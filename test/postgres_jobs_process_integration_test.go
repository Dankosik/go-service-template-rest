//go:build integration

package integration_test

import (
	"bytes"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/jobs"
	"github.com/example/go-service-template-rest/internal/waittest"
)

func TestPostgresJobsProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGTERM process lifecycle is Unix-specific")
	}
	ctx, pool, store := newPostgresJobsFixture(t)
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

	for _, test := range []struct {
		name       string
		handler    string
		stage      bool
		wantFailed bool
	}{
		{name: "no_work"},
		{name: "cooperative", handler: "cooperative", stage: true},
		{name: "noncooperative", handler: "noncooperative", stage: true, wantFailed: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var claimedJobID string
			var postSignalJobID string
			if test.stage {
				claimedJobID = string(stageDuePostgresJob(ctx, t, pool, store, "process-"+test.name).Identity().LogicalJobID)
			}
			addr := waittest.FreeTCPAddr(t, "jobs worker diagnostics")
			cleanupFile := filepath.Join(t.TempDir(), "cleanup")
			var output bytes.Buffer
			process := exec.Command(binary)
			process.Dir = repositoryRoot
			process.Env = append(cleanServiceEnvironment(os.Environ()),
				"APP__APP__ENV=integration",
				"APP__AUTHN__ISSUER=",
				"APP__OBSERVABILITY__METRICS__ADDR="+addr,
				"APP__POSTGRES__ENABLED=true",
				"APP__POSTGRES__DSN="+postgresJobsDSN(pool),
				"APP__POSTGRES__MAX_OPEN_CONNS=8",
				"APP__POSTGRES__STATEMENT_TIMEOUT=1s",
				"APP__POSTGRES__ACQUIRE_TIMEOUT=500ms",
				"APP__JOBS__ENABLED=true",
				"APP__JOBS__POLL_INTERVAL=10ms",
				"APP__JOBS__MAX_CONCURRENCY=1",
				"APP__JOBS__LEASE_DURATION=1s",
				"APP__JOBS__STORE_OPERATION_TIMEOUT=100ms",
				"APP__JOBS__OBSERVATION_INTERVAL=10ms",
				"APP__JOBS__DRAIN_TIMEOUT=100ms",
				"APP__HTTP__GRACE_PERIOD=8s",
				"APP__HTTP__SHUTDOWN_TIMEOUT=1s",
				"APP__HTTP__WRITE_TIMEOUT=1s",
				"APP__HTTP__READINESS_TIMEOUT=1s",
				"APP__HTTP__REQUEST_TIMEOUT=1s",
				"APP__HTTP__READINESS_PROPAGATION_DELAY=0s",
				"JOBS_WORKER_TEST_CLEANUP_FILE="+cleanupFile,
			)
			if test.handler != "" {
				process.Env = append(process.Env, "JOBS_WORKER_TEST_HANDLER="+test.handler)
			}
			process.Stdout, process.Stderr = &output, &output
			t.Cleanup(func() { t.Logf("jobs worker output:\n%s", output.String()) })
			if err := process.Start(); err != nil {
				t.Fatalf("start jobs worker: %v", err)
			}
			waited := make(chan error, 1)
			finished := make(chan struct{})
			go func() {
				waited <- process.Wait()
				close(finished)
			}()
			t.Cleanup(func() {
				select {
				case <-finished:
				default:
					_ = process.Process.Kill()
					<-finished
				}
			})

			client := &http.Client{Timeout: 100 * time.Millisecond}
			waittest.Until(t, 5*time.Second, func() bool {
				response, err := client.Get("http://" + addr + "/health/ready")
				if err != nil {
					return false
				}
				_ = response.Body.Close()
				return response.StatusCode == http.StatusOK
			}, "jobs worker readiness")
			if test.stage {
				waittest.Until(t, 5*time.Second, func() bool {
					var attempts int
					if err := pool.PGX().QueryRow(ctx, `SELECT count(*) FROM postgres_job_attempts WHERE logical_job_id = $1`, claimedJobID).Scan(&attempts); err != nil {
						t.Fatal(err)
					}
					return attempts == 1
				}, "durable claimed attempt")
			}
			if err := process.Process.Signal(syscall.SIGTERM); err != nil {
				t.Fatalf("signal jobs worker: %v", err)
			}
			if test.stage {
				waittest.Until(t, 5*time.Second, func() bool {
					response, err := client.Get("http://" + addr + "/health/ready")
					if err != nil {
						return false
					}
					defer response.Body.Close()
					return response.StatusCode == http.StatusServiceUnavailable
				}, "readiness withdrawal before claim quiescence")
				postSignal := stageDuePostgresJob(ctx, t, pool, store, "process-post-signal-"+test.name)
				postSignalJobID = string(postSignal.Identity().LogicalJobID)
			}
			select {
			case err := <-waited:
				if (err != nil) != test.wantFailed {
					t.Fatalf("jobs worker exit = %v, want failed=%t\n%s", err, test.wantFailed, output.String())
				}
			case <-time.After(12 * time.Second):
				t.Fatalf("jobs worker did not exit\n%s", output.String())
			}
			if _, err := os.Stat(cleanupFile); (err == nil) != !test.wantFailed {
				t.Fatalf("cleanup marker error = %v, want cleanup=%t", err, !test.wantFailed)
			}
			if test.stage {
				assertPostgresJobsAttemptCount(ctx, t, pool, jobs.LogicalJobID(postSignalJobID), 0)
				if _, err := pool.PGX().Exec(ctx, `UPDATE postgres_jobs SET state = 'scheduled', available_at = clock_timestamp() + interval '1 hour' WHERE logical_job_id = $1`, postSignalJobID); err != nil {
					t.Fatalf("isolate post-quiescence job: %v", err)
				}
			}
			if test.wantFailed {
				waittest.Until(t, 3*time.Second, func() bool {
					var state string
					var leaseExpired bool
					if err := pool.PGX().QueryRow(ctx, `
SELECT state, lease_expires_at < clock_timestamp()
FROM postgres_jobs
WHERE logical_job_id = $1`, claimedJobID).Scan(&state, &leaseExpired); err == nil {
						return state == "running" && leaseExpired
					}
					return false
				}, "durable unjoined attempt recoverability")
			}
		})
	}
}
