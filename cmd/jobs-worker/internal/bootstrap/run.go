package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
	"github.com/riverqueue/rivercontrib/otelriver"
)

// WorkersBuilder is binary-local business composition. Derived services add
// their typed River workers here; the reusable binary has no default job kind.
type WorkersBuilder func(context.Context, config.Config, *slog.Logger) (*river.Workers, func(context.Context), error)

func Run(args []string, buildWorkers WorkersBuilder) error {
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return run(signalCtx, args, buildWorkers)
}

//nolint:cyclop // This is the single process composition and teardown owner.
func run(signalCtx context.Context, args []string, buildWorkers WorkersBuilder) error {
	if buildWorkers == nil {
		return errors.New("jobs worker builder is not registered")
	}
	loadOptions, err := parseLoadOptions(args)
	if err != nil {
		return err
	}
	startupCtx, cancelStartup := context.WithTimeout(signalCtx, startupTimeout)
	defer cancelStartup()
	cfg, _, err := config.LoadJobsWorkerDetailedWithContext(startupCtx, loadOptions)
	if err != nil {
		return fmt.Errorf("load jobs worker config: %w", err)
	}
	if err := validateRuntimeConfig(cfg); err != nil {
		return err
	}

	log := runtimeopts.Logger(os.Stdout, cfg, "component", "jobs_worker")
	metrics := telemetry.New()
	telemetryCleanup, err := runtimeopts.InstallTelemetry(startupCtx, cfg, metrics, log, "jobs_worker")
	if err != nil {
		return err
	}
	cleanupSafe := true
	var cleanupDeadline time.Time
	defer func() {
		if cleanupSafe {
			cleanupCtx, cleanupCancel := runtimeopts.TeardownStage(signalCtx, cleanupDeadline, telemetryClose)
			defer cleanupCancel()
			telemetryCleanup(cleanupCtx)
		}
	}()

	workers, cleanup, err := buildWorkers(startupCtx, cfg, log)
	if err != nil {
		return fmt.Errorf("build jobs workers: %w", err)
	}
	if workers == nil {
		return errors.New("jobs workers are not registered")
	}
	defer func() {
		if cleanup != nil && cleanupSafe {
			cleanupCtx, cancel := runtimeopts.TeardownStage(signalCtx, cleanupDeadline, telemetryClose)
			defer cancel()
			cleanup(cleanupCtx)
		}
	}()

	pool, err := postgres.Open(startupCtx, runtimeopts.Postgres(cfg.Postgres))
	if err != nil {
		return fmt.Errorf("initialize jobs worker postgres: %w", err)
	}
	defer func() {
		if cleanupSafe {
			pool.Close()
		}
	}()

	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		CancelledJobRetentionPeriod: 24 * time.Hour,
		CompletedJobRetentionPeriod: 24 * time.Hour,
		DiscardedJobRetentionPeriod: 7 * 24 * time.Hour,
		JobTimeout:                  river.JobTimeoutDefault,
		Logger:                      log,
		MaxAttempts:                 river.MaxAttemptsDefault,
		SoftStopTimeout:             cfg.HTTP.ShutdownTimeout,
		Plugins: []rivertype.Plugin{
			otelriver.NewMiddleware(&otelriver.MiddlewareConfig{EnableTracePropagation: true}),
		},
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: cfg.Jobs.MaxWorkers},
		},
		Workers: workers,
	})
	if err != nil {
		return fmt.Errorf("initialize River client: %w", err)
	}

	runCtx, cancelRun := context.WithCancel(context.WithoutCancel(signalCtx))
	defer cancelRun()
	if err := client.Start(runCtx); err != nil {
		return fmt.Errorf("start River client: %w", err)
	}

	var ready atomic.Bool
	ready.Store(true)
	diagnostics, err := runtimeopts.ListenDiagnostics(
		startupCtx,
		cfg.Observability.Metrics.Addr,
		"jobs-worker",
		ready.Load,
		metrics,
	)
	if err != nil {
		stopCtx, cancel := context.WithTimeout(context.WithoutCancel(signalCtx), cfg.HTTP.ShutdownTimeout)
		defer cancel()
		return errors.Join(err, client.StopAndCancel(stopCtx))
	}

	var trigger error
	select {
	case <-signalCtx.Done():
	case <-client.Stopped():
		trigger = errors.New("river client stopped unexpectedly")
	case <-diagnostics.Stopped():
		trigger = errors.New("jobs worker diagnostics stopped unexpectedly")
	}
	ready.Store(false)

	processCtx, cancelProcess, deadline := runtimeopts.ArmTeardown(signalCtx, cfg.HTTP.GracePeriod)
	defer cancelProcess()
	cleanupDeadline = deadline
	stopErr := client.Stop(processCtx)
	select {
	case <-client.Stopped():
	case <-processCtx.Done():
		cleanupSafe = false
		stopErr = errors.Join(stopErr, fmt.Errorf("join River client: %w", processCtx.Err()))
	}
	if cleanupSafe {
		cancelRun()
	}
	diagnosticsErr := diagnostics.Stop(processCtx, diagnosticsClose)
	return errors.Join(trigger, stopErr, diagnosticsErr)
}
