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
	"github.com/example/go-service-template-rest/cmd/outbox-relay/outboxworker"
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
	startupTimeout             = 30 * time.Second
	defaultOutboxWorkers       = 16
	outboxDrain                = 25 * time.Second
	diagnosticsShutdownTimeout = 2 * time.Second
	backgroundShutdownTimeout  = 5 * time.Second
	telemetryShutdownTimeout   = 5 * time.Second
	outboxTailBudget           = diagnosticsShutdownTimeout + backgroundShutdownTimeout + telemetryShutdownTimeout
)

func Run(args []string) error {
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return run(signalCtx, args)
}

func run(signalCtx context.Context, args []string) error {
	loadOptions, err := config.ParseLoadOptions(args)
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
	cleanupWindow := runtimeopts.UnarmedTeardown(signalCtx)
	defer func() {
		cleanupCtx, cancel := runtimeopts.TeardownStage(cleanupWindow, telemetryShutdownTimeout)
		defer cancel()
		_ = telemetryCleanup(cleanupCtx)
	}()

	pool, err := postgres.Open(startupCtx, runtimeopts.Postgres(cfg.Postgres))
	if err != nil {
		return fmt.Errorf("initialize outbox postgres: %w", err)
	}
	// False means bounded shutdown returned without joining an owner. Keep its
	// dependencies alive until process exit instead of closing them underneath it.
	cleanupSafe := true
	defer func() {
		if cleanupSafe {
			pool.Close()
		}
	}()
	messaging, err := natsjs.Connect(
		startupCtx,
		runtimeopts.Messaging(cfg.Messaging),
		natsjs.Observability{Logger: log},
	)
	if err != nil {
		return fmt.Errorf("initialize outbox messaging: %w", err)
	}
	defer func() {
		if cleanupSafe {
			messaging.Close()
		}
	}()

	workers := river.NewWorkers()
	outboxWorker, err := outboxworker.New(messaging.Producer())
	if err != nil {
		return fmt.Errorf("initialize NATS outbox worker: %w", err)
	}
	if err := river.AddWorkerSafely(workers, outboxWorker); err != nil {
		return fmt.Errorf("register outbox worker: %w", err)
	}
	riverClient, err := river.NewClient(
		riverpgxv5.New(pool),
		riverClientConfig(workers, log),
	)
	if err != nil {
		return fmt.Errorf("initialize River outbox worker: %w", err)
	}
	cleanupSafe, cleanupWindow, err = runLifecycle(
		signalCtx, startupCtx, cfg, log, metrics, pool, messaging, riverClient,
	)
	return err
}

func validateRuntimeConfig(cfg config.Config) error {
	if !cfg.Postgres.Enabled {
		return fmt.Errorf("%w: postgres must be enabled for outbox relay", config.ErrValidate)
	}
	if strings.TrimSpace(cfg.Messaging.URLs) == "" {
		return fmt.Errorf("%w: messaging must be enabled for outbox relay", config.ErrValidate)
	}
	if err := runtimeopts.RequireDiagnosticsAddr(cfg.Observability.Metrics.Addr, "outbox"); err != nil {
		return err
	}
	return runtimeopts.ValidateGracePeriod(
		cfg.HTTP.GracePeriod,
		"the code-owned outbox relay drain",
		outboxDrain,
		outboxTailBudget,
	)
}

func riverClientConfig(workers *river.Workers, log *slog.Logger) *river.Config {
	plugin := otelriver.NewMiddleware(&otelriver.MiddlewareConfig{
		EnableSemanticMetrics:  true,
		EnableTracePropagation: true,
	})
	return &river.Config{
		// -1 keeps cancelled and discarded jobs indefinitely, so unpublished
		// intent cannot disappear through cleanup.
		CancelledJobRetentionPeriod: -1,
		DiscardedJobRetentionPeriod: -1,
		Logger:                      log,
		// The pool's finite statement_timeout would cancel a long-lived LISTEN.
		PollOnly: true,
		Plugins:  []rivertype.Plugin{plugin},
		Queues: map[string]river.QueueConfig{
			postgresoutbox.Queue: {MaxWorkers: defaultOutboxWorkers},
		},
		SoftStopTimeout: outboxDrain,
		Workers:         workers,
	}
}

// runLifecycle admits, serves, and drains the relay. It reports whether River
// and the background tasks joined, so run may close the pool and NATS. The
// returned window is already canceled and serves only as the parent for
// runtimeopts.TeardownStage in deferred cleanup.
func runLifecycle(
	signalCtx context.Context,
	startupCtx context.Context,
	cfg config.Config,
	log *slog.Logger,
	metrics *telemetry.Metrics,
	pool postgresPinger,
	messaging messagingRuntime,
	riverClient riverRuntime,
) (bool, context.Context, error) {
	unarmed := runtimeopts.UnarmedTeardown(signalCtx)
	var admitted atomic.Bool
	readiness := health.New(postgresReadinessProbe{pool: pool}, messaging)
	if err := readiness.Refresh(startupCtx, cfg.HTTP.ReadinessTimeout, cfg.Health.FailureThreshold); err != nil {
		return true, unarmed, fmt.Errorf("admit outbox readiness: %w", err)
	}
	diagnostics, err := runtimeopts.ListenDiagnostics(
		startupCtx,
		cfg.Observability.Metrics.Addr,
		"outbox",
		func() bool { return relayReady(admitted.Load(), messaging.Ready(), readiness.Cached()) },
		metrics,
		cfg.Observability.Pprof.Enabled,
	)
	if err != nil {
		return true, unarmed, err
	}
	runtimeCtx, cancelRuntime := context.WithCancel(context.WithoutCancel(signalCtx))
	defer cancelRuntime()
	supervisor := background.New(runtimeCtx, log)
	supervisor.Go(background.Task{Name: "messaging_connection", Run: messaging.Run})
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
	// stopTail stops diagnostics, then the background tasks. Messaging drains
	// under the background stage only once River has joined, because River's
	// jobs publish through it.
	stopTail := func(window context.Context, shutdownMessaging bool) (diagnosticsErr, backgroundErr, messagingErr error) {
		diagnosticsErr = diagnostics.Stop(window, diagnosticsShutdownTimeout)
		backgroundCtx, cancelBackground := runtimeopts.TeardownStage(window, backgroundShutdownTimeout)
		defer cancelBackground()
		backgroundErr = supervisor.Shutdown(backgroundCtx)
		if shutdownMessaging {
			messagingErr = messaging.Shutdown(backgroundCtx)
		}
		return diagnosticsErr, backgroundErr, messagingErr
	}

	started, err := runtimeopts.StartRuntime(startupCtx, runtimeCtx, cancelRuntime, riverClient.Start)
	if err != nil {
		window, cancelProcess := runtimeopts.ArmTeardown(signalCtx, cfg.HTTP.GracePeriod)
		defer cancelProcess()
		var riverErr error
		if started {
			riverCtx, cancelRiver := runtimeopts.TeardownStage(window, outboxDrain)
			riverErr = riverClient.StopAndCancel(riverCtx)
			cancelRiver()
		}
		diagnosticsErr, backgroundErr, _ := stopTail(window, false)
		return relayCleanupSafe(riverErr == nil, backgroundErr), window, errors.Join(
			fmt.Errorf("start River outbox worker: %w", err),
			riverErr,
			diagnosticsErr,
			backgroundErr,
		)
	}
	admitted.Store(true)

	var trigger error
	select {
	case <-signalCtx.Done():
	case trigger = <-supervisor.Failures():
	case <-riverClient.Stopped():
		trigger = errors.New("river outbox worker stopped unexpectedly")
	case <-diagnostics.Stopped():
		trigger = errors.New("outbox diagnostics stopped unexpectedly")
	}
	admitted.Store(false)
	readiness.StartDrain()
	window, cancelProcess := runtimeopts.ArmTeardown(signalCtx, cfg.HTTP.GracePeriod)
	defer cancelProcess()
	riverCtx, cancelRiver := runtimeopts.TeardownStage(window, outboxDrain)
	riverErr := riverClient.Stop(riverCtx)
	cancelRiver()
	riverStopped := riverErr == nil
	if riverStopped {
		messaging.StopPublish()
	}
	diagnosticsErr, backgroundErr, messagingErr := stopTail(window, riverStopped)
	return relayCleanupSafe(riverStopped, backgroundErr), window,
		errors.Join(trigger, riverErr, messagingErr, diagnosticsErr, backgroundErr)
}

// relayCleanupSafe reports whether the pool and NATS may be closed: River has
// joined, so no job still publishes, and the background tasks that use both
// finished within their stage.
func relayCleanupSafe(riverStopped bool, backgroundErr error) bool {
	return riverStopped && !errors.Is(backgroundErr, context.DeadlineExceeded)
}

type postgresReadinessProbe struct {
	pool postgresPinger
}

type postgresPinger interface {
	Ping(ctx context.Context) error
}

type messagingRuntime interface {
	health.Probe
	Ready() bool
	Run(ctx context.Context) error
	StopPublish()
	Shutdown(ctx context.Context) error
}

func relayReady(admitted, messagingReady bool, dependencyErr error) bool {
	return admitted && messagingReady && dependencyErr == nil
}

type riverRuntime interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	StopAndCancel(ctx context.Context) error
	Stopped() <-chan struct{}
}

func (postgresReadinessProbe) Name() string { return "postgres" }

func (p postgresReadinessProbe) Check(ctx context.Context) error {
	if err := p.pool.Ping(ctx); err != nil {
		return fmt.Errorf("check PostgreSQL readiness: %w", err)
	}
	return nil
}
