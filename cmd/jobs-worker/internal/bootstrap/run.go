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

	// profile:inbound-webhooks-standard:start
	"github.com/jackc/pgx/v5/pgxpool"
	// profile:inbound-webhooks-standard:end
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
	"github.com/riverqueue/rivercontrib/otelriver"

	// profile:inbound-webhooks-standard:start
	"go.opentelemetry.io/otel/metric"
	// profile:inbound-webhooks-standard:end
)

// WorkersRuntime is the builder result: validated workers, cleanup, and an
// optional post-pool binder.
type WorkersRuntime struct {
	Workers *river.Workers
	Cleanup func(context.Context)
	// profile:inbound-webhooks-standard:start
	Bind func(context.Context, *pgxpool.Pool, metric.MeterProvider) error
	// profile:inbound-webhooks-standard:end
}

// WorkersBuilder is binary-local business composition. Derived services add
// their typed River workers here; the reusable binary has no default job kind.
type WorkersBuilder func(context.Context, config.Config, *slog.Logger) (WorkersRuntime, error)

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
	cleanupSafe := true
	var cleanupDeadline time.Time
	defer func() {
		if cleanupSafe {
			cleanupCtx, cleanupCancel := runtimeopts.TeardownStage(signalCtx, cleanupDeadline, telemetryClose)
			defer cleanupCancel()
			telemetryCleanup(cleanupCtx)
		}
	}()
	if err != nil {
		return err
	}

	runtime, err := buildWorkers(startupCtx, cfg, log)
	if err != nil {
		return fmt.Errorf("build jobs workers: %w", err)
	}
	if runtime.Workers == nil {
		return errors.New("jobs workers are not registered")
	}
	defer func() {
		if runtime.Cleanup != nil && cleanupSafe {
			cleanupCtx, cancel := runtimeopts.TeardownStage(signalCtx, cleanupDeadline, telemetryClose)
			defer cancel()
			runtime.Cleanup(cleanupCtx)
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
	// profile:inbound-webhooks-standard:start
	if runtime.Bind != nil {
		if err := runtime.Bind(startupCtx, pool, metrics.MeterProvider()); err != nil {
			return fmt.Errorf("bind jobs workers: %w", err)
		}
	}
	// profile:inbound-webhooks-standard:end

	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		CancelledJobRetentionPeriod: 24 * time.Hour,
		CompletedJobRetentionPeriod: 24 * time.Hour,
		DiscardedJobRetentionPeriod: 7 * 24 * time.Hour,
		JobTimeout:                  river.JobTimeoutDefault,
		Logger:                      log,
		MaxAttempts:                 river.MaxAttemptsDefault,
		PollOnly:                    true,
		SoftStopTimeout:             cfg.HTTP.ShutdownTimeout,
		Plugins: []rivertype.Plugin{
			otelriver.NewMiddleware(&otelriver.MiddlewareConfig{EnableTracePropagation: true}),
		},
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: cfg.Jobs.MaxWorkers},
		},
		Workers: runtime.Workers,
	})
	if err != nil {
		return fmt.Errorf("initialize River client: %w", err)
	}
	stopStartedRiver := func(trigger error) error {
		processCtx, cancelProcess, deadline := runtimeopts.ArmTeardown(signalCtx, cfg.HTTP.GracePeriod)
		defer cancelProcess()
		cleanupDeadline = deadline
		stopCtx, cancelStop := context.WithTimeout(processCtx, cfg.HTTP.ShutdownTimeout)
		defer cancelStop()
		stopErr := client.StopAndCancel(stopCtx)
		cleanupSafe = riverStoppedBeforeReturn(stopErr, client.Stopped())
		if !cleanupSafe {
			stopErr = errors.Join(stopErr, fmt.Errorf("join River client: %w", stopCtx.Err()))
		}
		return errors.Join(trigger, stopErr)
	}

	runCtx, cancelRun := context.WithCancel(context.WithoutCancel(signalCtx))
	defer cancelRun()
	started, err := runtimeopts.StartRuntime(startupCtx, runCtx, cancelRun, client.Start)
	if err != nil {
		startErr := fmt.Errorf("start River client: %w", err)
		if started {
			return stopStartedRiver(startErr)
		}
		return startErr
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
		return stopStartedRiver(err)
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
	cleanupSafe = riverStoppedBeforeReturn(stopErr, client.Stopped())
	if !cleanupSafe {
		stopErr = errors.Join(stopErr, fmt.Errorf("join River client: %w", processCtx.Err()))
	}
	if cleanupSafe {
		cancelRun()
	}
	diagnosticsErr := diagnostics.Stop(processCtx, diagnosticsClose)
	return errors.Join(trigger, stopErr, diagnosticsErr)
}

func riverStoppedBeforeReturn(stopErr error, stopped <-chan struct{}) bool {
	if stopErr == nil {
		<-stopped
		return true
	}
	select {
	case <-stopped:
		return true
	default:
		return false
	}
}
