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

type PublisherBuilder func(
	context.Context,
	config.Config,
	*slog.Logger,
) (postgresoutbox.Publisher, func(context.Context), error)

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
	if buildPublisher == nil {
		return fmt.Errorf("%w: outbox publisher builder is not registered", postgresoutbox.ErrConfig)
	}
	loadOptions, err := parseLoadOptions(args)
	if err != nil {
		return err
	}
	startupCtx, startupCancel := context.WithTimeout(signalCtx, startupTimeout)
	defer startupCancel()
	cfg, _, err := config.LoadDetailedWithContext(startupCtx, loadOptions)
	if err != nil {
		return fmt.Errorf("load outbox relay config: %w", err)
	}
	if err := validateRuntimeConfig(cfg); err != nil {
		return err
	}
	log := newLogger(os.Stdout, cfg)
	publisher, publisherCleanup, err := buildPublisher(startupCtx, cfg, log)
	// Registered here so a startup failure before the pool exists still closes
	// the publisher. It stays registered for the rest of the function, so it is
	// the backstop rather than the normal path — see the second registration
	// after the pool, which is what actually runs.
	var teardown relayTeardown
	teardown.publisher = publisherCleanup
	defer func() { runErr = errors.Join(runErr, teardown.release(signalCtx)) }()
	if err != nil {
		return fmt.Errorf("build outbox publisher: %w", err)
	}
	if err := postgresoutbox.ValidatePublisher(publisher); err != nil {
		return fmt.Errorf("admit outbox publisher: %w", err)
	}

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

	pool, err := postgres.New(startupCtx, postgresOptions(cfg.Postgres))
	if err != nil {
		return fmt.Errorf("initialize outbox postgres: %w", err)
	}
	// The same teardown again, and this registration is the load-bearing one.
	// Defers run last-registered-first, so this one runs before the telemetry
	// defers above it and the publisher and pool are released while telemetry
	// can still export what their cleanup records. The earlier registration
	// then finds released set and does nothing. Ordering is the only reason
	// this exists; releasing twice is already prevented by the latch.
	teardown.pool = pool.Close
	defer func() { runErr = errors.Join(runErr, teardown.release(signalCtx)) }()
	store, err := postgresoutbox.NewStore(pool, outboxTelemetry)
	if err != nil {
		return fmt.Errorf("initialize outbox store: %w", err)
	}
	relay, err := postgresoutbox.NewRelay(store, publisher, outboxTelemetry, relayConfig(cfg.Outbox))
	if err != nil {
		return fmt.Errorf("initialize outbox relay: %w", err)
	}
	result := runRelayLifecycle(signalCtx, startupCtx, cfg, metrics, relay)
	teardown.unsafe = result.CleanupUnsafe
	return result.Err
}

// relayTeardown releases the publisher and then the pool, once, and only while
// releasing them is safe. Its fields are filled in as each dependency appears,
// so a startup failure releases exactly what had been built by then.
//
// Publisher before pool is deliberate and is not reverse construction order:
// adapter cleanup can still be draining sends, and the pool is what a
// last-moment durable write would need.
//
// unsafe is the relay's own report that a publisher goroutine outlived
// cancellation. That goroutine can still reach both the adapter and the pool, so
// nothing is released and process exit owns them instead. Every method runs on
// the goroutine that deferred it, so released needs no synchronization.
type relayTeardown struct {
	publisher func(context.Context)
	pool      func()
	unsafe    bool
	released  bool
}

func (t *relayTeardown) release(signalCtx context.Context) error {
	if t.unsafe || t.released {
		return nil
	}
	t.released = true
	err := closePublisher(signalCtx, t.publisher)
	if t.pool != nil {
		t.pool()
	}
	return err
}

func closePublisher(parent context.Context, cleanup func(context.Context)) error {
	if cleanup == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), publisherClose)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		var err error
		defer func() {
			if recover() != nil {
				err = errors.New("outbox publisher cleanup panicked")
			}
			result <- err
		}()
		cleanup(ctx)
	}()
	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return fmt.Errorf("close outbox publisher: %w", ctx.Err())
	}
}

func runRelayLifecycle(
	signalCtx context.Context,
	startupCtx context.Context,
	cfg config.Config,
	metrics *telemetry.Metrics,
	relay relayRunner,
) postgresoutbox.RelayResult {
	served, err := startDiagnostics(startupCtx, cfg.Observability.Metrics.Addr, relay.Ready, metrics)
	if err != nil {
		return postgresoutbox.RelayResult{Err: err}
	}
	runtimeCtx, runtimeCancel := context.WithCancel(context.WithoutCancel(signalCtx))
	defer runtimeCancel()
	relayResult := make(chan postgresoutbox.RelayResult, 1)
	go func() { relayResult <- relay.Run(runtimeCtx) }()

	// Three ways out, and only the signal is an ordinary one. A relay or a
	// diagnostics server that returned by itself is a fault, so each supplies
	// the reason the other two would otherwise leave unexplained.
	var result postgresoutbox.RelayResult
	var triggerErr error
	relayStopped := false
	select {
	case <-signalCtx.Done():
	case result = <-relayResult:
		relayStopped = true
		if result.Err == nil {
			result.Err = errors.New("outbox relay stopped unexpectedly")
		}
	case <-served.watch():
		// served.stop below carries whatever Serve reported.
		triggerErr = errors.New("outbox diagnostics stopped unexpectedly")
	}

	processCtx, processCancel := context.WithTimeout(context.WithoutCancel(signalCtx), cfg.HTTP.GracePeriod)
	defer processCancel()
	relay.StartDrain()
	if !relayStopped {
		result = drainRelay(processCtx, cfg.Outbox.DrainTimeout, runtimeCancel, relayResult)
	}
	result.Err = errors.Join(triggerErr, result.Err, served.stop(processCtx))
	return result
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
	cleanupReserve := forcedJoin + diagnosticsClose + publisherClose + telemetryClose
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

func parseLoadOptions(args []string) (config.LoadOptions, error) {
	var configPath string
	var overlays []string
	flags := flag.NewFlagSet("outbox-relay", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
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
		return config.LoadOptions{}, fmt.Errorf("parse flags: %w", err)
	}
	if len(flags.Args()) != 0 {
		return config.LoadOptions{}, errors.New("parse flags: unexpected positional arguments")
	}
	return config.LoadOptions{ConfigPath: configPath, ConfigOverlays: overlays}, nil
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
