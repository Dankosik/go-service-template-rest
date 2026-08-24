//go:build integration

package integration_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

type riverIntegrationArgs struct {
	Value string `json:"value"`
}

func (riverIntegrationArgs) Kind() string { return "river_integration" }

type riverIntegrationWorker struct {
	river.WorkerDefaults[riverIntegrationArgs]
	worked chan<- string
}

func (w *riverIntegrationWorker) Work(_ context.Context, job *river.Job[riverIntegrationArgs]) error {
	w.worked <- job.Args.Value
	return nil
}

func TestPostgresJobsTransactionalInsertionAndWorkerLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	dsn := pgtest.Migrated(t, os.DirFS(".."), "migrations")
	pool, err := postgres.Open(ctx, postgres.Options{DSN: dsn, MaxOpenConns: 4})
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	t.Cleanup(pool.Close)

	worked := make(chan string, 1)
	workers := river.NewWorkers()
	if err := river.AddWorkerSafely(workers, &riverIntegrationWorker{worked: worked}); err != nil {
		t.Fatalf("register River worker: %v", err)
	}
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		t.Fatalf("create River migrator: %v", err)
	}
	validation, err := migrator.Validate(ctx, nil)
	if err != nil {
		t.Fatalf("validate River migrations: %v", err)
	}
	if !validation.OK {
		t.Fatalf("River migrations are stale: %v", validation.Messages)
	}
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		JobTimeout:  time.Minute,
		Logger:      slog.Default(),
		MaxAttempts: 1,
		PollOnly:    true,
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 1},
		},
		Workers: workers,
	})
	if err != nil {
		t.Fatalf("create River client: %v", err)
	}

	rollbackTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback transaction: %v", err)
	}
	if _, err := client.InsertTx(ctx, rollbackTx, riverIntegrationArgs{Value: "rolled-back"}, nil); err != nil {
		t.Fatalf("insert rolled-back job: %v", err)
	}
	if err := rollbackTx.Rollback(ctx); err != nil {
		t.Fatalf("roll back job transaction: %v", err)
	}
	assertRiverJobCount(t, ctx, pool, "river_integration", 0)

	commitTx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatalf("begin commit transaction: %v", err)
	}
	if _, err := client.InsertTx(ctx, commitTx, riverIntegrationArgs{Value: "committed"}, nil); err != nil {
		t.Fatalf("insert committed job: %v", err)
	}
	if err := commitTx.Commit(ctx); err != nil {
		t.Fatalf("commit job transaction: %v", err)
	}
	assertRiverJobCount(t, ctx, pool, "river_integration", 1)

	events, unsubscribe := client.Subscribe(river.EventKindJobCompleted)
	defer unsubscribe()
	runCtx, stopRun := context.WithCancel(t.Context())
	defer stopRun()
	if err := client.Start(runCtx); err != nil {
		t.Fatalf("start River client: %v", err)
	}

	select {
	case value := <-worked:
		if value != "committed" {
			t.Fatalf("worked value = %q, want committed", value)
		}
	case <-ctx.Done():
		t.Fatalf("wait for River worker: %v", ctx.Err())
	}
	select {
	case event := <-events:
		if event.Job.Kind != "river_integration" {
			t.Fatalf("completed kind = %q, want river_integration", event.Job.Kind)
		}
	case <-ctx.Done():
		t.Fatalf("wait for River completion: %v", ctx.Err())
	}
	if err := client.Stop(ctx); err != nil {
		t.Fatalf("stop River client: %v", err)
	}
}

func assertRiverJobCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, kind string, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM river_job WHERE kind = $1", kind).Scan(&got); err != nil {
		t.Fatalf("count River jobs: %v", err)
	}
	if got != want {
		t.Fatalf("River job count = %d, want %d", got, want)
	}
}
