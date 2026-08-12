package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

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
	registry, cleanup, err := buildRegistry(startupCtx, cfg, log)
	if err != nil {
		return fmt.Errorf("build jobs registry: %w", err)
	}
	cleanupSafe := true
	defer func() {
		if cleanup != nil && cleanupSafe {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(signalCtx), telemetryClose)
			defer cancel()
			cleanup(cleanupCtx)
		}
	}()
	if registry == nil {
		return fmt.Errorf("%w: jobs worker registry is not registered", postgresjobs.ErrConfig)
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
			session.Release(context.WithoutCancel(signalCtx))
		}
	}()
	if err := session.CheckSchema(startupCtx); err != nil {
		return fmt.Errorf("admit jobs schema: %w", err)
	}
	engineCfg, err := engineConfig(cfg.Jobs)
	if err != nil {
		return err
	}
	engine, err := postgresjobs.NewEngine(session, registry, engineCfg)
	if err != nil {
		return fmt.Errorf("initialize jobs engine: %w", err)
	}
	metrics := telemetry.New()
	result := runLifecycle(signalCtx, startupCtx, cfg, metrics, engine)
	cleanupSafe = result.CleanupSafe
	return result.Err
}
