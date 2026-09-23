package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

const startupTimeout = 30 * time.Second

// HandlerBuilder registers binary-local typed event handlers. Any returned
// cleanup normally runs before Run returns, including when the builder returns
// an invalid result.
// If forced shutdown leaves an uncooperative handler running, process exit owns
// its resources instead of racing that cleanup with the handler.
type HandlerBuilder func(context.Context, config.Config, *slog.Logger) (registry *natsjs.Registry, cleanup func(context.Context), err error)

func Run(args []string, buildHandler HandlerBuilder) error {
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return run(signalCtx, args, buildHandler)
}

func run(signalCtx context.Context, args []string, buildHandler HandlerBuilder) error {
	if buildHandler == nil {
		return fmt.Errorf("%w: worker feature handler builder is not registered", natsjs.ErrRejected)
	}
	loadOptions, err := config.ParseLoadOptions(args)
	if err != nil {
		return err
	}
	startupCtx, startupCancel := context.WithTimeout(signalCtx, startupTimeout)
	defer startupCancel()
	cfg, _, err := config.Load(startupCtx, loadOptions)
	if err != nil {
		return fmt.Errorf("load worker config: %w", err)
	}
	if err := validateRuntimeConfig(cfg); err != nil {
		return err
	}
	workerCfg, err := messagingWorkerConfig(cfg.Messaging)
	if err != nil {
		return err
	}
	log := runtimeopts.Logger(os.Stdout, cfg)
	metrics := telemetry.New()
	// A metrics provider that could not be built stops this binary, which is this
	// composition root's own answer rather than InstallTelemetry's: a worker with
	// no meter cannot report what it consumed, so nothing would notice it stopped
	// consuming, while a worker with no exporter for spans still records every
	// count an alert is built on.
	flushTelemetry, err := runtimeopts.InstallTelemetry(startupCtx, cfg, metrics, log, "worker")
	cleanupWindow := runtimeopts.UnarmedTeardown(signalCtx)
	// False means the handler did not join within the shutdown budget. Its
	// dependencies stay intact and process exit owns them.
	cleanupSafe := true
	// Registered before the error is read, because tracing is installed whether
	// or not metrics were: a worker that refuses to start over its meter still
	// owes the span exporter the flush that lets its goroutine end.
	defer func() {
		cleanupCtx, cleanupCancel := runtimeopts.TeardownStage(cleanupWindow, telemetryShutdownTimeout)
		defer cleanupCancel()
		_ = flushTelemetry(cleanupCtx)
	}()
	if err != nil {
		return err
	}
	client, err := natsjs.Connect(startupCtx, runtimeopts.Messaging(cfg.Messaging), natsjs.Observability{Logger: log})
	if err != nil {
		return fmt.Errorf("initialize worker messaging: %w", err)
	}
	defer client.Close()
	registry, handlerCleanup, err := buildHandler(startupCtx, cfg, log)
	defer func() {
		if handlerCleanup != nil && cleanupSafe {
			cleanupCtx, cleanupCancel := runtimeopts.TeardownStage(cleanupWindow, handlerShutdownTimeout)
			defer cleanupCancel()
			handlerCleanup(cleanupCtx)
		}
	}()
	if err != nil {
		return fmt.Errorf("initialize worker feature handler: %w", err)
	}
	if registry == nil {
		return fmt.Errorf("%w: worker feature handler is not registered", natsjs.ErrRejected)
	}
	handler, err := registry.Handler()
	if err != nil {
		return fmt.Errorf("initialize typed event handlers: %w", err)
	}
	worker, err := client.NewWorker(startupCtx, workerCfg, handler)
	if err != nil {
		return fmt.Errorf("initialize durable consumer: %w", err)
	}
	cleanupSafe, cleanupWindow, err = runWorkerLifecycle(signalCtx, startupCtx, cfg, log, metrics, client, worker)
	return err
}
