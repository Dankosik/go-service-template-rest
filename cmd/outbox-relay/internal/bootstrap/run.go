package bootstrap

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"reflect"
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
	publisherCleanupBeforePool := true
	if publisherCleanup != nil {
		defer func() {
			if !publisherCleanupBeforePool {
				return
			}
			runErr = errors.Join(runErr, closePublisher(signalCtx, publisherCleanup))
		}()
	}
	if err != nil {
		return fmt.Errorf("build outbox publisher: %w", err)
	}
	if publisherMissing(publisher) {
		return fmt.Errorf("%w: outbox publisher is not registered", postgresoutbox.ErrConfig)
	}

	metrics := telemetry.New()
	telemetryCleanup, err := setupTelemetry(startupCtx, cfg, metrics, log)
	if err != nil {
		return err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(signalCtx), telemetryClose)
		defer cancel()
		telemetryCleanup(ctx)
	}()
	outboxTelemetry, err := postgresoutbox.NewTelemetry(metrics.MeterProvider().Meter("service.outbox.postgres"), log)
	if err != nil {
		return fmt.Errorf("initialize outbox telemetry: %w", err)
	}
	defer outboxTelemetry.Close()

	pool, err := postgres.New(startupCtx, postgresOptions(cfg.Postgres))
	if err != nil {
		return fmt.Errorf("initialize outbox postgres: %w", err)
	}
	publisherCleanupBeforePool = false
	cleanupSafe := true
	defer func() {
		runErr = errors.Join(runErr, cleanupRelayDependencies(signalCtx, cleanupSafe, publisherCleanup, pool.Close))
	}()
	store, err := postgresoutbox.NewStore(pool, outboxTelemetry)
	if err != nil {
		return fmt.Errorf("initialize outbox store: %w", err)
	}
	relay, err := postgresoutbox.NewRelay(store, publisher, outboxTelemetry, relayConfig(cfg.Outbox))
	if err != nil {
		return fmt.Errorf("initialize outbox relay: %w", err)
	}
	result := runRelayLifecycle(signalCtx, startupCtx, cfg, metrics, relay)
	cleanupSafe = result.CleanupSafe
	return result.Err
}

func cleanupRelayDependencies(
	signalCtx context.Context,
	cleanupSafe bool,
	publisherCleanup func(context.Context),
	poolClose func(),
) error {
	if !cleanupSafe {
		return nil
	}
	err := closePublisher(signalCtx, publisherCleanup)
	poolClose()
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
	listener, err := listenDiagnostics(startupCtx, cfg.Observability.Metrics.Addr)
	if err != nil {
		return postgresoutbox.RelayResult{CleanupSafe: true, Err: err}
	}
	diagnostics := newDiagnosticsServer(cfg.Observability.Metrics.Addr, relay.Ready, metrics)
	runtimeCtx, runtimeCancel := context.WithCancel(context.WithoutCancel(signalCtx))
	defer runtimeCancel()
	relayResult := make(chan postgresoutbox.RelayResult, 1)
	go func() { relayResult <- relay.Run(runtimeCtx) }()
	diagnosticsResult := make(chan error, 1)
	go func() {
		err := diagnostics.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		diagnosticsResult <- err
	}()

	result := postgresoutbox.RelayResult{CleanupSafe: true}
	var triggerErr error
	relayRead := false
	diagnosticsRead := false
	select {
	case <-signalCtx.Done():
	case result = <-relayResult:
		relayRead = true
		if result.Err == nil {
			result.Err = errors.New("outbox relay stopped unexpectedly")
		}
	case err := <-diagnosticsResult:
		diagnosticsRead = true
		triggerErr = err
		if triggerErr == nil {
			triggerErr = errors.New("outbox diagnostics stopped unexpectedly")
		}
	}

	processCtx, processCancel := context.WithTimeout(context.WithoutCancel(signalCtx), cfg.HTTP.GracePeriod)
	defer processCancel()
	result, _ = drainRelay(
		processCtx,
		cfg.Outbox.DrainTimeout,
		runtimeCancel,
		relay,
		relayResult,
		result,
		relayRead,
	)

	diagnosticsCtx, diagnosticsCancel := context.WithTimeout(processCtx, diagnosticsClose)
	diagnosticsErr := diagnostics.Shutdown(diagnosticsCtx)
	if !diagnosticsRead {
		select {
		case err := <-diagnosticsResult:
			diagnosticsErr = errors.Join(diagnosticsErr, err)
		case <-diagnosticsCtx.Done():
			diagnosticsErr = errors.Join(diagnosticsErr, fmt.Errorf("join outbox diagnostics: %w", diagnosticsCtx.Err()))
		}
	}
	diagnosticsCancel()
	result.Err = errors.Join(triggerErr, result.Err, diagnosticsErr)
	return result
}

func drainRelay(
	processCtx context.Context,
	drainTimeout time.Duration,
	runtimeCancel context.CancelFunc,
	relay relayRunner,
	relayResult <-chan postgresoutbox.RelayResult,
	result postgresoutbox.RelayResult,
	relayRead bool,
) (postgresoutbox.RelayResult, bool) {
	relay.StartDrain()
	if relayRead {
		return result, true
	}
	drainCtx, drainCancel := context.WithTimeout(processCtx, drainTimeout)
	defer drainCancel()
	select {
	case result = <-relayResult:
		return result, true
	case <-drainCtx.Done():
	}

	runtimeCancel()
	joinCtx, joinCancel := context.WithTimeout(processCtx, forcedJoin)
	defer joinCancel()
	select {
	case result = <-relayResult:
		return result, true
	case <-joinCtx.Done():
		return postgresoutbox.RelayResult{
			CleanupSafe: false,
			Err:         fmt.Errorf("join outbox relay: %w", joinCtx.Err()),
		}, false
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

func publisherMissing(publisher postgresoutbox.Publisher) bool {
	if publisher == nil {
		return true
	}
	value := reflect.ValueOf(publisher)
	//nolint:exhaustive // Only kinds that can contain a typed nil are relevant.
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
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
