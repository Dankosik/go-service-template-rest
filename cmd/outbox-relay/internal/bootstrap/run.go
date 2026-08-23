package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/background"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
	"github.com/riverqueue/rivercontrib/otelriver"
)

const (
	startupTimeout       = 30 * time.Second
	defaultOutboxWorkers = 16
	diagnosticsClose     = 2 * time.Second
	backgroundClose      = 5 * time.Second
	telemetryClose       = 5 * time.Second
	outboxTailBudget     = diagnosticsClose + backgroundClose + telemetryClose
)

func Run(args []string) error {
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return run(signalCtx, args)
}

func run(signalCtx context.Context, args []string) (runErr error) {
	loadOptions, err := config.ParseLoadOptions("outbox-relay", args, nil)
	if err != nil {
		return err
	}
	startupCtx, cancelStartup := context.WithTimeout(signalCtx, startupTimeout)
	defer cancelStartup()
	cfg, _, err := config.LoadDetailedWithContext(startupCtx, loadOptions)
	if err != nil {
		return fmt.Errorf("load outbox relay config: %w", err)
	}
	if err := validateRuntimeConfig(cfg); err != nil {
		return err
	}
	log := runtimeopts.Logger(os.Stdout, cfg, "component", "outbox_relay")
	metrics := telemetry.New()
	telemetryCleanup, metricsErr := runtimeopts.InstallTelemetry(startupCtx, cfg, metrics, log, "outbox")
	if metricsErr != nil {
		log.WarnContext(startupCtx, "outbox_metrics_degraded", "reason", telemetry.FailureReason(metricsErr))
	}
	var cleanupDeadline time.Time
	defer func() {
		cleanupCtx, cancel := runtimeopts.TeardownStage(signalCtx, cleanupDeadline, telemetryClose)
		defer cancel()
		telemetryCleanup(cleanupCtx)
	}()

	pool, err := postgres.Open(startupCtx, runtimeopts.Postgres(cfg.Postgres))
	if err != nil {
		return fmt.Errorf("initialize outbox postgres: %w", err)
	}
	cleanupSafe := true
	defer func() {
		if cleanupSafe {
			pool.Close()
		}
	}()
	client, err := natsjs.Connect(
		startupCtx,
		runtimeopts.Messaging(cfg.Messaging),
		natsjs.Observability{Logger: log},
	)
	if err != nil {
		return fmt.Errorf("initialize outbox messaging: %w", err)
	}
	defer func() {
		if cleanupSafe {
			client.Close()
		}
	}()

	workers := river.NewWorkers()
	outboxWorker, err := natsjs.NewOutboxWorker(client.Producer())
	if err != nil {
		return fmt.Errorf("initialize NATS outbox worker: %w", err)
	}
	if err := river.AddWorkerSafely(workers, outboxWorker); err != nil {
		return fmt.Errorf("register outbox worker: %w", err)
	}
	riverClient, err := river.NewClient(
		riverpgxv5.New(pool),
		riverClientConfig(cfg, workers, log),
	)
	if err != nil {
		return fmt.Errorf("initialize River outbox worker: %w", err)
	}
	cleanupSafe, cleanupDeadline, runErr = runLifecycle(
		signalCtx, startupCtx, cfg, log, metrics, pool, client, riverClient,
	)
	return runErr
}

func validateRuntimeConfig(cfg config.Config) error {
	if !cfg.Postgres.Enabled {
		return fmt.Errorf("%w: postgres must be enabled for outbox relay", postgresoutbox.ErrConfig)
	}
	if strings.TrimSpace(cfg.Messaging.URLs) == "" {
		return fmt.Errorf("%w: messaging must be enabled for outbox relay", postgresoutbox.ErrConfig)
	}
	if strings.TrimSpace(cfg.Observability.Metrics.Addr) == "" {
		return fmt.Errorf("%w: outbox diagnostics address is required", postgresoutbox.ErrConfig)
	}
	return runtimeopts.ValidateGracePeriod(
		cfg.HTTP.GracePeriod,
		"http.shutdown_timeout",
		cfg.HTTP.ShutdownTimeout,
		outboxTailBudget,
	)
}

func riverClientConfig(cfg config.Config, workers *river.Workers, log *slog.Logger) *river.Config {
	plugin := otelriver.NewMiddleware(&otelriver.MiddlewareConfig{
		EnableSemanticMetrics:  true,
		EnableTracePropagation: true,
	})
	return &river.Config{
		CancelledJobRetentionPeriod: -1,
		DiscardedJobRetentionPeriod: -1,
		Logger:                      log,
		PollOnly:                    true,
		Plugins:                     []rivertype.Plugin{plugin},
		Queues: map[string]river.QueueConfig{
			postgresoutbox.Queue: {MaxWorkers: defaultOutboxWorkers},
		},
		SoftStopTimeout: cfg.HTTP.ShutdownTimeout,
		Workers:         workers,
	}
}

func runLifecycle[TTx any](
	signalCtx context.Context,
	startupCtx context.Context,
	cfg config.Config,
	log *slog.Logger,
	metrics *telemetry.Metrics,
	pool postgresPinger,
	client *natsjs.Client,
	riverClient *river.Client[TTx],
) (cleanupSafe bool, deadline time.Time, result error) {
	var ready atomic.Bool
	readiness := health.New(postgresReadinessProbe{pool: pool}, client)
	if err := readiness.Refresh(startupCtx, cfg.HTTP.ReadinessTimeout, cfg.Health.FailureThreshold); err != nil {
		return true, time.Time{}, fmt.Errorf("admit outbox readiness: %w", err)
	}
	diagnostics, err := runtimeopts.ListenDiagnostics(
		startupCtx,
		cfg.Observability.Metrics.Addr,
		"outbox",
		func() bool { return ready.Load() && client.Ready() && readiness.Cached() == nil },
		metrics,
	)
	if err != nil {
		return true, time.Time{}, err
	}
	runtimeCtx, cancelRuntime := context.WithCancel(context.WithoutCancel(signalCtx))
	defer cancelRuntime()
	supervisor := background.New(runtimeCtx, log)
	supervisor.Go(background.Task{Name: "messaging_connection", Run: client.Run})
	supervisor.Go(background.Task{
		Name: "dependency_readiness",
		Run: func(ctx context.Context) error {
			return readiness.Watch(
				ctx,
				cfg.Health.RefreshInterval,
				cfg.HTTP.ReadinessTimeout,
				cfg.Health.FailureThreshold,
				nil,
			)
		},
	})
	started, err := runtimeopts.StartRuntime(startupCtx, runtimeCtx, cancelRuntime, riverClient.Start)
	if err != nil {
		processCtx, cancelProcess, shutdownDeadline := runtimeopts.ArmTeardown(signalCtx, cfg.HTTP.GracePeriod)
		defer cancelProcess()
		var riverErr error
		if started {
			riverCtx, cancelRiver := runtimeopts.TeardownStage(
				processCtx, shutdownDeadline, cfg.HTTP.ShutdownTimeout,
			)
			riverErr = riverClient.StopAndCancel(riverCtx)
			cancelRiver()
		}
		diagnosticsErr := diagnostics.Stop(processCtx, diagnosticsClose)
		backgroundCtx, cancelBackground := runtimeopts.TeardownStage(
			processCtx, shutdownDeadline, backgroundClose,
		)
		backgroundErr := supervisor.Shutdown(backgroundCtx)
		cancelBackground()
		cleanupSafe := riverErr == nil && !errors.Is(backgroundErr, context.DeadlineExceeded)
		return cleanupSafe, shutdownDeadline, errors.Join(
			fmt.Errorf("start River outbox worker: %w", err),
			riverErr,
			diagnosticsErr,
			backgroundErr,
		)
	}
	ready.Store(true)

	var trigger error
	select {
	case <-signalCtx.Done():
	case trigger = <-supervisor.Failures():
	case <-riverClient.Stopped():
		trigger = errors.New("river outbox worker stopped unexpectedly")
	case <-diagnostics.Stopped():
		trigger = errors.New("outbox diagnostics stopped unexpectedly")
	}
	ready.Store(false)
	readiness.StartDrain()
	processCtx, cancelProcess, shutdownDeadline := runtimeopts.ArmTeardown(signalCtx, cfg.HTTP.GracePeriod)
	defer cancelProcess()
	riverCtx, cancelRiver := runtimeopts.TeardownStage(
		processCtx, shutdownDeadline, cfg.HTTP.ShutdownTimeout,
	)
	riverErr := riverClient.Stop(riverCtx)
	cancelRiver()
	cleanupSafe = riverErr == nil
	client.StopPublish()
	diagnosticsErr := diagnostics.Stop(processCtx, diagnosticsClose)
	backgroundCtx, cancelBackground := runtimeopts.TeardownStage(
		processCtx, shutdownDeadline, backgroundClose,
	)
	backgroundErr := supervisor.Shutdown(backgroundCtx)
	messagingErr := client.Shutdown(backgroundCtx)
	cancelBackground()
	cleanupSafe = cleanupSafe && !errors.Is(backgroundErr, context.DeadlineExceeded)
	return cleanupSafe, shutdownDeadline, errors.Join(trigger, riverErr, messagingErr, diagnosticsErr, backgroundErr)
}

type postgresReadinessProbe struct {
	pool postgresPinger
}

type postgresPinger interface {
	Ping(ctx context.Context) error
}

func (postgresReadinessProbe) Name() string { return "postgres" }

func (p postgresReadinessProbe) Check(ctx context.Context) error {
	if err := p.pool.Ping(ctx); err != nil {
		return fmt.Errorf("check PostgreSQL readiness: %w", err)
	}
	return nil
}
