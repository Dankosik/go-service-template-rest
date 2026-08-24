//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

type jobsWorkerProcessArgs struct {
	Value string `json:"value"`
}

func (jobsWorkerProcessArgs) Kind() string { return "jobs_worker_test" }

func TestPostgresJobsWorkerProcess(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	dsn := pgtest.Migrated(t, os.DirFS(".."), "migrations")
	pool, err := postgres.Open(ctx, postgres.Options{DSN: dsn, MaxOpenConns: 4})
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	t.Cleanup(pool.Close)

	producer, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		t.Fatalf("create River producer: %v", err)
	}
	if _, err := producer.Insert(ctx, jobsWorkerProcessArgs{Value: "wait-for-cancel"}, nil); err != nil {
		t.Fatalf("insert River job: %v", err)
	}

	binary := filepath.Join(t.TempDir(), "jobs-worker")
	build := exec.CommandContext(ctx, "go", "build", "-tags", "jobs_test_worker", "-o", binary, "./cmd/jobs-worker")
	build.Dir = ".."
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build jobs worker: %v\n%s", err, output)
	}

	marker := filepath.Join(t.TempDir(), "worked")
	var output bytes.Buffer
	process := exec.CommandContext(ctx, binary)
	process.Stdout = &output
	process.Stderr = &output
	process.Env = append(os.Environ(),
		"APP__APP__ENV=local",
		"APP__POSTGRES__ENABLED=true",
		"APP__POSTGRES__DSN="+dsn,
		"APP__POSTGRES__MAX_OPEN_CONNS=4",
		"APP__JOBS__MAX_WORKERS=1",
		"APP__HTTP__GRACE_PERIOD=22s",
		"APP__HTTP__SHUTDOWN_TIMEOUT=10s",
		"APP__HTTP__READINESS_PROPAGATION_DELAY=0s",
		"APP__OBSERVABILITY__METRICS__ADDR="+waittest.FreeTCPAddr(t, "jobs diagnostics"),
		"JOBS_WORKER_TEST_MARKER="+marker,
	)
	if err := process.Start(); err != nil {
		t.Fatalf("start jobs worker: %v", err)
	}
	waitErr := make(chan error, 1)
	go func() { waitErr <- process.Wait() }()
	t.Cleanup(func() {
		if process.ProcessState == nil {
			_ = process.Process.Kill()
		}
	})

	var earlyExit bool
	var earlyErr error
	waittest.Until(t, 30*time.Second, func(context.Context) bool {
		select {
		case earlyErr = <-waitErr:
			earlyExit = true
			return true
		default:
		}
		value, err := os.ReadFile(marker)
		return err == nil && string(value) == "wait-for-cancel"
	}, "jobs worker effect")
	if earlyExit {
		t.Fatalf("jobs worker exited before handling work: %v\n%s", earlyErr, output.String())
	}
	if err := process.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signal jobs worker: %v", err)
	}
	select {
	case err := <-waitErr:
		if err != nil {
			t.Fatalf("jobs worker exit: %v\n%s", err, output.String())
		}
	case <-time.After(20 * time.Second):
		t.Fatalf("jobs worker did not stop\n%s", output.String())
	}
}
