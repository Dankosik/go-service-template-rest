package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

const startupTimeout = 15 * time.Second

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

	pool, err := postgres.New(startupCtx, runtimeopts.Postgres(cfg.Postgres))
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
