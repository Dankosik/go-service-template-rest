package bootstrap

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/observability/logctx"
)

const (
	startupTimeout   = 15 * time.Second
	diagnosticsClose = 2 * time.Second
	publisherClose   = 5 * time.Second
	telemetryClose   = 5 * time.Second
	forcedJoin       = 2 * time.Second
)

// PublisherRuntime is one publisher and the lifecycle of the client that owns
// it. Its fields stay private so only the validating constructor can create a
// value admitted by the relay process. It remains available to outbox-only
// services after a selected transport builder is removed.
type PublisherRuntime struct {
	publisher postgresoutbox.Publisher
	run       func(context.Context) error
	ready     func() bool
	shutdown  func(context.Context) error
}

func NewPublisherRuntime(
	publisher postgresoutbox.Publisher,
	run func(context.Context) error,
	ready func() bool,
	shutdown func(context.Context) error,
) (PublisherRuntime, error) {
	runtime := PublisherRuntime{publisher: publisher, run: run, ready: ready, shutdown: shutdown}
	if err := validatePublisherRuntime(runtime); err != nil {
		return PublisherRuntime{}, err
	}
	return runtime, nil
}

func validatePublisherRuntime(runtime PublisherRuntime) error {
	if err := postgresoutbox.ValidatePublisher(runtime.publisher); err != nil {
		return fmt.Errorf("validate publisher: %w", err)
	}
	if runtime.run == nil || runtime.ready == nil || runtime.shutdown == nil {
		return fmt.Errorf("%w: publisher runtime lifecycle is incomplete", postgresoutbox.ErrConfig)
	}
	return nil
}

type PublisherBuilder func(context.Context, config.Config, *slog.Logger) (PublisherRuntime, error)

type relayRunner interface {
	Ready() bool
	StartDrain()
	Run(ctx context.Context) postgresoutbox.RelayResult
}

func Run(args []string, buildPublisher PublisherBuilder) error {
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return run(signalCtx, args, buildPublisher)
}

func run(signalCtx context.Context, args []string, buildPublisher PublisherBuilder) (runErr error) {
	loadOptions, classifyLegacy, err := parseLoadOptions(args)
	if err != nil {
		return err
	}
	if !classifyLegacy && buildPublisher == nil {
		return fmt.Errorf("%w: outbox publisher builder is not registered", postgresoutbox.ErrConfig)
	}
	startupCtx, startupCancel := context.WithTimeout(signalCtx, startupTimeout)
	defer startupCancel()
	cfg, _, err := config.LoadDetailedWithContext(startupCtx, loadOptions)
	if err != nil {
		return fmt.Errorf("load outbox relay config: %w", err)
	}
	if classifyLegacy {
		if err := validateClassificationConfig(cfg); err != nil {
			return err
		}
		return runLegacyClassification(signalCtx, startupCtx, cfg, newLogger(os.Stdout, cfg))
	}
	if err := validateRuntimeConfig(cfg); err != nil {
		return err
	}
	log := newLogger(os.Stdout, cfg)

	// Telemetry is installed before any dependency it has to outlive. Defers run
	// last-registered-first, so everything registered below this point releases
	// while telemetry can still export what that cleanup records.
	metrics := telemetry.New()
	telemetryCleanup := setupTelemetry(startupCtx, cfg, metrics, log)
	defer func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(signalCtx), telemetryClose)
		defer cancel()
		telemetryCleanup(ctx)
	}()
	outboxTelemetry, err := postgresoutbox.NewTelemetry(
		metrics.MeterProvider().Meter(postgresoutbox.TelemetryScope), log)
	if err != nil {
		return fmt.Errorf("initialize outbox telemetry: %w", err)
	}
	defer outboxTelemetry.Close()

	publisherRuntime, err := buildPublisher(startupCtx, cfg, log)
	// One registration, covering every exit from here on. teardown's fields fill
	// in as each dependency appears, so a startup failure releases exactly what
	// had been built by then.
	teardown := relayTeardown{publisher: &publisherRuntime}
	defer func() { runErr = errors.Join(runErr, teardown.release(signalCtx)) }()
	if err != nil {
		return fmt.Errorf("build outbox publisher: %w", err)
	}
	if err := validatePublisherRuntime(publisherRuntime); err != nil {
		return fmt.Errorf("admit outbox publisher runtime: %w", err)
	}

	pool, err := postgres.New(startupCtx, postgresOptions(cfg.Postgres))
	if err != nil {
		return fmt.Errorf("initialize outbox postgres: %w", err)
	}
	teardown.pool = pool.Close
	store, err := postgresoutbox.NewStore(pool, outboxTelemetry)
	if err != nil {
		return fmt.Errorf("initialize outbox store: %w", err)
	}
	relay, err := postgresoutbox.NewRelay(store, publisherRuntime.publisher, outboxTelemetry, relayConfig(cfg.Outbox))
	if err != nil {
		return fmt.Errorf("initialize outbox relay: %w", err)
	}
	result := runRelayLifecycle(signalCtx, startupCtx, cfg, metrics, relay, &publisherRuntime)
	teardown.unsafe = result.CleanupUnsafe
	return result.Err
}

func runLegacyClassification(
	ctx context.Context,
	startupCtx context.Context,
	cfg config.Config,
	log *slog.Logger,
) error {
	pool, err := postgres.New(startupCtx, postgresOptions(cfg.Postgres))
	if err != nil {
		return fmt.Errorf("initialize outbox postgres for legacy classification: %w", err)
	}
	defer pool.Close()
	store, err := postgresoutbox.NewStore(pool, nil)
	if err != nil {
		return fmt.Errorf("initialize outbox store for legacy classification: %w", err)
	}
	return classifyLegacyUntilZero(
		ctx, cfg.Outbox.MaxAttempts, cfg.Outbox.CleanupBatchSize, log,
		store.ClassifyLegacyUncertainty,
	)
}

func classifyLegacyUntilZero(
	ctx context.Context,
	maxAttempts, batchSize int,
	log *slog.Logger,
	classify func(context.Context, int, int) (int, error),
) error {
	for {
		classified, err := classify(ctx, maxAttempts, batchSize)
		if err != nil {
			return fmt.Errorf("classify legacy outbox uncertainty: %w", err)
		}
		if classified == 0 {
			log.InfoContext(ctx, "outbox_legacy_uncertainty_classification_complete")
			return nil
		}
		log.InfoContext(ctx, "outbox_legacy_uncertainty_classified", "count", classified)
	}
}

// relayTeardown releases the publisher and then the pool, and only while
// releasing them is safe. Its fields are filled in as each dependency appears,
// so a startup failure releases exactly what had been built by then.
//
// Publisher before pool is deliberate and is not reverse construction order:
// adapter cleanup can still be draining sends, and the pool is what a
// last-moment durable write would need.
//
// unsafe is the relay's own report that a publisher goroutine outlived
// cancellation. That goroutine can still reach both the adapter and the pool, so
// nothing is released and process exit owns them instead.
//
// release carries no idempotence latch because run defers it exactly once, on
// the goroutine that registered it. A second call site would need one: the
// cleanup half belongs to the adapter, and PublisherBuilder does not promise it
// tolerates being called twice.
type relayTeardown struct {
	publisher *PublisherRuntime
	pool      func()
	unsafe    bool
}

func (t *relayTeardown) release(signalCtx context.Context) error {
	if t.unsafe {
		return nil
	}
	var err error
	if t.publisher != nil {
		var unsafe bool
		err, unsafe = closePublisher(signalCtx, t.publisher.shutdown)
		if unsafe {
			return err
		}
		t.publisher.shutdown = nil
	}
	if t.pool != nil {
		t.pool()
	}
	return err
}

func closePublisher(parent context.Context, cleanup func(context.Context) error) (error, bool) {
	if cleanup == nil {
		return nil, false
	}
	base := context.WithoutCancel(parent)
	if deadline, ok := parent.Deadline(); ok {
		var cancelDeadline context.CancelFunc
		base, cancelDeadline = context.WithDeadline(base, deadline)
		defer cancelDeadline()
	}
	ctx, cancel := context.WithTimeout(base, publisherClose)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		var runErr error
		defer func() {
			if recover() != nil {
				runErr = errors.New("outbox publisher cleanup panicked")
			}
			result <- runErr
		}()
		runErr = cleanup(ctx)
	}()
	select {
	case err := <-result:
		return err, false
	case <-ctx.Done():
		return fmt.Errorf("close outbox publisher: %w", ctx.Err()), true
	}
}

func runRelayLifecycle(
	signalCtx context.Context,
	startupCtx context.Context,
	cfg config.Config,
	metrics *telemetry.Metrics,
	relay relayRunner,
	publisher *PublisherRuntime,
) postgresoutbox.RelayResult {
	ready := func() bool { return relay.Ready() && publisher.ready() }
	diag, err := startDiagnostics(startupCtx, cfg.Observability.Metrics.Addr, ready, metrics)
	if err != nil {
		return postgresoutbox.RelayResult{Err: err}
	}
	relayCtx, relayCancel := context.WithCancel(context.WithoutCancel(signalCtx))
	defer relayCancel()
	publisherCtx, publisherCancel := context.WithCancel(context.WithoutCancel(signalCtx))
	defer publisherCancel()
	relayResult := make(chan postgresoutbox.RelayResult, 1)
	go func() { relayResult <- relay.Run(relayCtx) }()
	publisherResult := make(chan error, 1)
	go func() { publisherResult <- publisher.run(publisherCtx) }()

	// Three ways out, and only the signal is an ordinary one. A relay or a
	// diagnostics server that returned by itself is a fault, so each supplies
	// the reason the other two would otherwise leave unexplained.
	var result postgresoutbox.RelayResult
	var triggerErr error
	relayStopped := false
	publisherStopped := false
	select {
	case <-signalCtx.Done():
	case result = <-relayResult:
		relayStopped = true
		if result.Err == nil {
			result.Err = errors.New("outbox relay stopped unexpectedly")
		}
	case <-diag.watch():
		// diag.stop below carries whatever Serve reported.
		triggerErr = errors.New("outbox diagnostics stopped unexpectedly")
	case triggerErr = <-publisherResult:
		publisherStopped = true
		if triggerErr == nil {
			triggerErr = errors.New("outbox publisher supervisor stopped unexpectedly")
		}
	}

	processCtx, processCancel := context.WithTimeout(context.WithoutCancel(signalCtx), cfg.HTTP.GracePeriod)
	defer processCancel()
	relay.StartDrain()
	if !relayStopped {
		result = drainRelay(processCtx, cfg.Outbox.DrainTimeout, relayCancel, relayResult)
	}

	publisherErr, publisherUnsafe := stopPublisherSupervisor(
		processCtx, publisherCancel, publisherResult, publisherStopped,
	)
	result.Err = errors.Join(result.Err, publisherErr)
	result.CleanupUnsafe = result.CleanupUnsafe || publisherUnsafe

	diagnosticsErr := diag.stop(processCtx)
	var shutdownErr error
	if !result.CleanupUnsafe {
		var unsafe bool
		shutdownErr, unsafe = closePublisher(processCtx, publisher.shutdown)
		if !unsafe {
			publisher.shutdown = nil
		} else {
			result.CleanupUnsafe = true
		}
	}
	result.Err = errors.Join(triggerErr, result.Err, diagnosticsErr, shutdownErr)
	return result
}

func stopPublisherSupervisor(
	processCtx context.Context,
	cancel context.CancelFunc,
	result <-chan error,
	alreadyStopped bool,
) (error, bool) {
	cancel()
	if alreadyStopped {
		return nil, false
	}
	joinCtx, joinCancel := context.WithTimeout(processCtx, forcedJoin)
	defer joinCancel()
	select {
	case runErr := <-result:
		if errors.Is(runErr, context.Canceled) {
			return nil, false
		}
		return runErr, false
	case <-joinCtx.Done():
		return fmt.Errorf("join outbox publisher supervisor: %w", joinCtx.Err()), true
	}
}

// drainRelay waits out a relay that has not reported yet and returns its
// result. It escalates in two steps: the drain budget lets the current batch
// finish, then cancellation forces its publications to stop. Only the final
// step can fail to produce a result, and the one it synthesizes reports cleanup
// as unsafe because the relay goroutine is still running.
//
// The caller owns StartDrain and the already-reported case, so this is only
// ever entered with a result still outstanding.
func drainRelay(
	processCtx context.Context,
	drainTimeout time.Duration,
	runtimeCancel context.CancelFunc,
	relayResult <-chan postgresoutbox.RelayResult,
) postgresoutbox.RelayResult {
	drainCtx, drainCancel := context.WithTimeout(processCtx, drainTimeout)
	defer drainCancel()
	select {
	case result := <-relayResult:
		return result
	case <-drainCtx.Done():
	}

	runtimeCancel()
	joinCtx, joinCancel := context.WithTimeout(processCtx, forcedJoin)
	defer joinCancel()
	select {
	case result := <-relayResult:
		return result
	case <-joinCtx.Done():
		return postgresoutbox.RelayResult{
			CleanupUnsafe: true,
			Err:           fmt.Errorf("join outbox relay: %w", joinCtx.Err()),
		}
	}
}

func validateRuntimeConfig(cfg config.Config) error {
	if !cfg.Outbox.Enabled {
		return fmt.Errorf("%w: outbox must be enabled for relay", postgresoutbox.ErrConfig)
	}
	if !cfg.Postgres.Enabled {
		return fmt.Errorf("%w: postgres must be enabled for relay", postgresoutbox.ErrConfig)
	}
	if strings.TrimSpace(cfg.Observability.Metrics.Addr) == "" {
		return fmt.Errorf("%w: outbox diagnostics address is required", postgresoutbox.ErrConfig)
	}
	// Relay and publisher supervisors have distinct joins and can both consume
	// their bound before dependency shutdown begins.
	cleanupReserve := 2*forcedJoin + diagnosticsClose + publisherClose + telemetryClose
	if cfg.HTTP.GracePeriod < cfg.Outbox.DrainTimeout ||
		cfg.HTTP.GracePeriod-cfg.Outbox.DrainTimeout < cleanupReserve {
		return fmt.Errorf(
			"%w: http.grace_period must fit outbox drain and post-drain cleanup (%s + %s)",
			config.ErrValidate,
			cfg.Outbox.DrainTimeout,
			cleanupReserve,
		)
	}
	return nil
}

func validateClassificationConfig(cfg config.Config) error {
	if !cfg.Outbox.Enabled {
		return fmt.Errorf("%w: outbox must be enabled for legacy classification", postgresoutbox.ErrConfig)
	}
	if !cfg.Postgres.Enabled {
		return fmt.Errorf("%w: postgres must be enabled for legacy classification", postgresoutbox.ErrConfig)
	}
	return nil
}

func parseLoadOptions(args []string) (config.LoadOptions, bool, error) {
	var configPath string
	var overlays []string
	var classifyLegacy bool
	flags := flag.NewFlagSet("outbox-relay", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.BoolVar(&classifyLegacy, "classify-legacy-uncertainty", false,
		"classify legacy outbox uncertainty and exit")
	flags.Func("config", "path to base config file", func(value string) error {
		configPath = strings.TrimSpace(value)
		if configPath == "" {
			return errors.New("config path cannot be empty")
		}
		return nil
	})
	flags.Func("config-overlay", "path to config overlay file (repeatable)", func(value string) error {
		value = strings.TrimSpace(value)
		if value == "" {
			return errors.New("config overlay path cannot be empty")
		}
		overlays = append(overlays, value)
		return nil
	})
	if err := flags.Parse(args); err != nil {
		return config.LoadOptions{}, false, fmt.Errorf("parse flags: %w", err)
	}
	if len(flags.Args()) != 0 {
		return config.LoadOptions{}, false, errors.New("parse flags: unexpected positional arguments")
	}
	return config.LoadOptions{ConfigPath: configPath, ConfigOverlays: overlays}, classifyLegacy, nil
}

func newLogger(out io.Writer, cfg config.Config) *slog.Logger {
	return slog.New(logctx.New(slog.NewJSONHandler(out, &slog.HandlerOptions{Level: cfg.Log.Level}))).With(
		"service.name", cfg.Observability.OTel.ServiceName,
		"service.version", cfg.App.Version,
		"deployment.environment.name", cfg.App.Env,
		"component", "outbox_relay",
	)
}

func postgresOptions(cfg config.PostgresConfig) postgres.Options {
	return postgres.Options{
		DSN: cfg.DSN, ConnectTimeout: cfg.ConnectTimeout, HealthcheckTimeout: cfg.HealthcheckTimeout,
		MaxOpenConns: cfg.MaxOpenConns, MinIdleConns: cfg.MinIdleConns, AcquireTimeout: cfg.AcquireTimeout,
		ConnMaxLifetime: cfg.ConnMaxLifetime, StatementTimeout: cfg.StatementTimeout,
	}
}

func relayConfig(cfg config.OutboxConfig) postgresoutbox.RelayConfig {
	return postgresoutbox.RelayConfig{
		PollInterval: cfg.PollInterval, BatchSize: cfg.BatchSize, PublishConcurrency: cfg.PublishConcurrency,
		PublishTimeout: cfg.PublishTimeout, LeaseDuration: cfg.LeaseDuration,
		MaxAttempts: cfg.MaxAttempts, RetryBase: cfg.RetryBase, RetryMax: cfg.RetryMax,
		ObservationInterval: cfg.ObservationInterval, CleanupInterval: cfg.CleanupInterval,
		PublishedRetention: cfg.PublishedRetention, CleanupBatchSize: cfg.CleanupBatchSize,
	}
}
