package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

func Run(args []string) error {
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return run(signalCtx, args)
}

func run(signalCtx context.Context, args []string) (runErr error) {
	loadOptions, err := parseLoadOptions(args)
	if err != nil {
		return err
	}
	startupCtx, cancelStartup := context.WithTimeout(signalCtx, startupTimeout)
	defer cancelStartup()
	cfg, _, err := config.LoadDetailedWithContext(startupCtx, loadOptions)
	if err != nil {
		return fmt.Errorf("load webhook worker config: %w", err)
	}
	if err := validateRuntimeConfig(cfg); err != nil {
		return err
	}
	manifest, err := postgreswebhook.ParseSecretManifest(cfg.Webhooks.StaticSecrets)
	if err != nil {
		return fmt.Errorf("load webhook secret manifest: %w", err)
	}
	cfg.Webhooks.StaticSecrets = ""
	log := runtimeopts.Logger(os.Stdout, cfg, "component", "webhook_worker")
	metrics := telemetry.New()
	telemetryCleanup := setupTelemetry(startupCtx, cfg, metrics, log)
	defer func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(signalCtx), telemetryClose)
		defer cancel()
		telemetryCleanup(ctx)
	}()
	pool, err := postgres.New(startupCtx, runtimeopts.Postgres(cfg.Postgres))
	if err != nil {
		return fmt.Errorf("initialize webhook worker postgres: %w", err)
	}
	cleanupSafe := true
	defer func() {
		if cleanupSafe {
			pool.Close()
		}
	}()
	store, err := postgreswebhook.NewStore(pool, postgreswebhook.StoreOptions{OperationTimeout: cfg.Webhooks.StoreOperationTimeout, CapacityRevision: cfg.Webhooks.CapacityRevision, GlobalConcurrency: cfg.Webhooks.GlobalConcurrency, ManifestRevision: manifest.Revision()})
	if err != nil {
		return fmt.Errorf("initialize webhook store: %w", err)
	}
	if err := store.InitializeOrTransitionCapacity(startupCtx); err != nil {
		return fmt.Errorf("initialize webhook capacity: %w", err)
	}
	workerCfg, err := workerConfig(cfg)
	if err != nil {
		return err
	}
	worker, err := postgreswebhook.NewWorker(store, manifest, workerCfg)
	if err != nil {
		return fmt.Errorf("initialize webhook worker: %w", err)
	}
	defer func() { runErr = errors.Join(runErr, worker.CloseTelemetry()) }()
	result := runLifecycle(signalCtx, startupCtx, cfg, metrics, worker)
	cleanupSafe = result.CleanupSafe
	return result.Err
}
