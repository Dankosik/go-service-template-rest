package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgresjobs"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/jobs"
)

// RegistryBuilder is binary-local feature composition. The reusable binary has
// no default job kind, so a nil builder fails before loading configuration or
// acquiring PostgreSQL.
type RegistryBuilder func(context.Context, config.Config, *slog.Logger) (*jobs.Registry, func(context.Context), error)

func Run(args []string, buildRegistry RegistryBuilder) error {
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return run(signalCtx, args, buildRegistry)
}

//nolint:cyclop // Startup and ownership cleanup remain explicit in their single composition owner.
func run(signalCtx context.Context, args []string, buildRegistry RegistryBuilder) (runErr error) {
	if buildRegistry == nil {
		return fmt.Errorf("%w: jobs worker registry builder is not registered", postgresjobs.ErrConfig)
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
	instanceID := telemetry.ResolveInstanceID(cfg.App.InstanceID)
	cfg.App.InstanceID = instanceID
	cleanupSafe := true
	metrics := telemetry.New()
	telemetryCleanup, err := runtimeopts.InstallTelemetry(startupCtx, cfg, metrics, log, "jobs_worker")
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
	registry, cleanup, err := buildRegistry(startupCtx, cfg, log)
	if err != nil {
		return fmt.Errorf("build jobs registry: %w", err)
	}
	defer func() {
		if cleanup != nil && cleanupSafe {
			cleanupCtx, cancel := runtimeopts.TeardownStage(signalCtx, cleanupDeadline, telemetryClose)
			defer cancel()
			cleanup(cleanupCtx)
		}
	}()
	if registry == nil {
		return fmt.Errorf("%w: jobs worker registry is not registered", postgresjobs.ErrConfig)
	}
	if len(registry.Keys()) == 0 {
		return fmt.Errorf("%w: jobs worker registry has no definitions", postgresjobs.ErrConfig)
	}
	if err := validateTerminationEnvelope(cfg.HTTP.GracePeriod, registry.TerminationEnvelope()); err != nil {
		return err
	}
	engineCfg, err := engineConfig(cfg.Jobs, instanceID)
	if err != nil {
		return err
	}

	pool, err := postgres.New(startupCtx, runtimeopts.Postgres(cfg.Postgres))
	if err != nil {
		return fmt.Errorf("initialize jobs worker postgres: %w", err)
	}
	defer func() {
		if cleanupSafe {
			pool.Close()
		}
	}()
	store, err := postgresjobs.NewStore(pool, postgresjobs.StoreOptions{OperationTimeout: cfg.Jobs.StoreOperationTimeout, StatementTimeout: cfg.Postgres.StatementTimeout})
	if err != nil {
		return fmt.Errorf("initialize jobs store: %w", err)
	}
	session, err := store.AcquireSession(startupCtx)
	if err != nil {
		return fmt.Errorf("acquire jobs Session: %w", err)
	}
	defer func() {
		if cleanupSafe {
			releaseCtx, cancel := runtimeopts.TeardownStage(signalCtx, cleanupDeadline, cfg.Jobs.StoreOperationTimeout)
			defer cancel()
			session.Release(releaseCtx)
		}
	}()
	if err := session.CheckSchema(startupCtx); err != nil {
		return fmt.Errorf("admit jobs schema: %w", err)
	}
	engine, err := postgresjobs.NewEngine(session, registry, engineCfg)
	if err != nil {
		return fmt.Errorf("initialize jobs engine: %w", err)
	}
	defer func() {
		if cleanupSafe {
			runErr = errors.Join(runErr, engine.Close())
		}
	}()
	result := runLifecycle(signalCtx, startupCtx, cfg, metrics, engine)
	cleanupSafe = result.CleanupSafe
	cleanupDeadline = result.Deadline
	return result.Err
}
