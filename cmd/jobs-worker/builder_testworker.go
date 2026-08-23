//go:build jobs_test_worker

package main

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/example/go-service-template-rest/cmd/jobs-worker/internal/bootstrap"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/riverqueue/river"
)

type jobsWorkerTestArgs struct {
	Value string `json:"value"`
}

func (jobsWorkerTestArgs) Kind() string { return "jobs_worker_test" }

type jobsWorkerTestWorker struct {
	river.WorkerDefaults[jobsWorkerTestArgs]
}

func (*jobsWorkerTestWorker) Work(_ context.Context, job *river.Job[jobsWorkerTestArgs]) error {
	path := os.Getenv("JOBS_WORKER_TEST_MARKER")
	if path == "" {
		return errors.New("JOBS_WORKER_TEST_MARKER is required")
	}
	return os.WriteFile(path, []byte(job.Args.Value), 0o600)
}

var buildWorkers bootstrap.WorkersBuilder = func(
	context.Context,
	config.Config,
	*slog.Logger,
) (bootstrap.WorkersRuntime, error) {
	workers := river.NewWorkers()
	if err := river.AddWorkerSafely(workers, &jobsWorkerTestWorker{}); err != nil {
		return bootstrap.WorkersRuntime{}, err
	}
	return bootstrap.WorkersRuntime{Workers: workers}, nil
}
