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
	"strings"
	"syscall"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

const (
	startupTimeout   = 30 * time.Second
	diagnosticsClose = 2 * time.Second
	telemetryClose   = 5 * time.Second
)

// HandlerBuilder constructs the feature-owned handler after the worker has
// loaded and validated its runtime configuration. Any returned cleanup runs
// before Run returns, including when the builder returns an invalid result.
type HandlerBuilder func(context.Context, config.Config, *slog.Logger) (handler natsjs.Handler, cleanup func(), err error)

func Run(args []string, buildHandler HandlerBuilder) error {
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return run(signalCtx, args, buildHandler)
}

func run(signalCtx context.Context, args []string, buildHandler HandlerBuilder) error {
	if buildHandler == nil {
		return fmt.Errorf("%w: worker feature handler builder is not registered", natsjs.ErrRejected)
	}
	loadOptions, err := parseLoadOptions(args)
	if err != nil {
		return err
	}
	startupCtx, startupCancel := context.WithTimeout(signalCtx, startupTimeout)
	defer startupCancel()
	cfg, _, err := config.LoadDetailedWithContext(startupCtx, loadOptions)
	if err != nil {
		return fmt.Errorf("load worker config: %w", err)
	}
	if !cfg.Messaging.Enabled {
		return fmt.Errorf("%w: messaging must be enabled for worker", natsjs.ErrRejected)
	}
	workerCfg, err := messagingWorkerConfig(cfg.Messaging)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.Observability.Metrics.Addr) == "" {
		return fmt.Errorf("%w: worker diagnostics address is required", natsjs.ErrRejected)
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.Log.Level}))
	metrics := telemetry.New()
	telemetryCleanup, err := setupTelemetry(startupCtx, cfg, metrics, log)
	if err != nil {
		return err
	}
	defer telemetryCleanup()
	handler, handlerCleanup, err := buildHandler(startupCtx, cfg, log)
	if handlerCleanup != nil {
		defer handlerCleanup()
	}
	if err != nil {
		return fmt.Errorf("initialize worker feature handler: %w", err)
	}
	if handler == nil {
		return fmt.Errorf("%w: worker feature handler is not registered", natsjs.ErrRejected)
	}
	client, err := natsjs.Connect(startupCtx, messagingClientConfig(cfg.Messaging), natsjs.RoleWorker, natsjs.Observability{Logger: log})
	if err != nil {
		return fmt.Errorf("initialize worker messaging: %w", err)
	}
	defer client.Close()
	worker, err := client.NewWorker(startupCtx, workerCfg, handler)
	if err != nil {
		return fmt.Errorf("initialize durable consumer: %w", err)
	}
	healthSvc := health.New(client)
	if err := healthSvc.Refresh(startupCtx, cfg.HTTP.ReadinessTimeout, cfg.Health.FailureThreshold); err != nil {
		return fmt.Errorf("admit worker readiness: %w", err)
	}
	diagnostics := newDiagnosticsServer(cfg.Observability.Metrics.Addr, healthSvc, client.Ready, metrics)
	listener, err := listenDiagnostics(startupCtx, cfg.Observability.Metrics.Addr)
	if err != nil {
		return err
	}
	defer func() { _ = listener.Close() }()

	runCtx, cancelRun := context.WithCancel(context.WithoutCancel(signalCtx))
	defer cancelRun()
	errCh := make(chan error, 4)
	go func() { errCh <- client.Run(runCtx) }()
	go func() { errCh <- worker.Run(runCtx) }()
	go func() {
		errCh <- healthSvc.Watch(runCtx, cfg.Health.RefreshInterval, cfg.HTTP.ReadinessTimeout, cfg.Health.FailureThreshold, nil)
	}()
	go func() {
		err := diagnostics.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		errCh <- err
	}()

	var triggerErr error
	select {
	case <-signalCtx.Done():
	case triggerErr = <-errCh:
		if triggerErr == nil {
			triggerErr = fmt.Errorf("worker runtime stopped unexpectedly")
		}
	}
	healthSvc.StartDrain()
	worker.StartDrain()
	shutdownBase := context.WithoutCancel(signalCtx)
	shutdownCtx, shutdownCancel := context.WithTimeout(shutdownBase, cfg.Messaging.Worker.DrainTimeout)
	workerErr := worker.Shutdown(shutdownCtx)
	shutdownCancel()
	diagnosticsCtx, diagnosticsCancel := context.WithTimeout(shutdownBase, diagnosticsClose)
	diagnosticsErr := diagnostics.Shutdown(diagnosticsCtx)
	diagnosticsCancel()
	cancelRun()
	return errors.Join(triggerErr, diagnosticsErr, workerErr)
}

func parseLoadOptions(args []string) (config.LoadOptions, error) {
	var configPath string
	var overlays []string
	flags := flag.NewFlagSet("worker", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Func("config", "path to base config file", func(value string) error {
		configPath = strings.TrimSpace(value)
		if configPath == "" {
			return fmt.Errorf("config path cannot be empty")
		}
		return nil
	})
	flags.Func("config-overlay", "path to config overlay file (repeatable)", func(value string) error {
		value = strings.TrimSpace(value)
		if value == "" {
			return fmt.Errorf("config overlay path cannot be empty")
		}
		overlays = append(overlays, value)
		return nil
	})
	if err := flags.Parse(args); err != nil {
		return config.LoadOptions{}, fmt.Errorf("parse flags: %w", err)
	}
	if len(flags.Args()) != 0 {
		return config.LoadOptions{}, fmt.Errorf("parse flags: unexpected positional arguments")
	}
	return config.LoadOptions{ConfigPath: configPath, ConfigOverlays: overlays}, nil
}

func messagingClientConfig(cfg config.MessagingConfig) natsjs.Config {
	urls := make([]string, 0)
	for value := range strings.SplitSeq(cfg.URLs, ",") {
		urls = append(urls, strings.TrimSpace(value))
	}
	return natsjs.Config{
		URLs:                 urls,
		CredentialsFile:      cfg.CredentialsFile,
		RootCAFile:           cfg.RootCAFile,
		AllowPlaintext:       cfg.AllowPlaintext,
		AllowUnauthenticated: cfg.AllowUnauthenticated,
		Stream:               cfg.Stream,
		MaxPayloadBytes:      cfg.MaxPayloadBytes,
		MaxPendingPublishes:  cfg.MaxPendingPublishes,
	}
}

func messagingWorkerConfig(cfg config.MessagingConfig) (natsjs.WorkerConfig, error) {
	delays := make([]time.Duration, 0)
	for value := range strings.SplitSeq(cfg.Worker.RetryDelays, ",") {
		delay, err := time.ParseDuration(strings.TrimSpace(value))
		if err != nil {
			return natsjs.WorkerConfig{}, fmt.Errorf("%w: invalid messaging worker retry delay", natsjs.ErrRejected)
		}
		delays = append(delays, delay)
	}
	result := natsjs.WorkerConfig{
		Consumer:             strings.TrimSpace(cfg.Worker.Consumer),
		FilterSubject:        strings.TrimSpace(cfg.Worker.FilterSubject),
		DeadLetterSubject:    strings.TrimSpace(cfg.Worker.DeadLetterSubject),
		MaxConcurrency:       cfg.Worker.MaxConcurrency,
		MaxDeliveryBytes:     cfg.Worker.MaxDeliveryBytes,
		HandlerTimeout:       cfg.Worker.HandlerTimeout,
		RetryDelays:          delays,
		DeadLetterRetryDelay: cfg.Worker.DeadLetterRetryDelay,
	}
	if cfg.Worker.DrainTimeout <= 0 {
		return natsjs.WorkerConfig{}, fmt.Errorf("%w: messaging worker drain timeout must be positive", natsjs.ErrRejected)
	}
	if err := natsjs.ValidateWorkerConfig(result, cfg.MaxPayloadBytes); err != nil {
		return natsjs.WorkerConfig{}, fmt.Errorf("validate messaging worker config: %w", err)
	}
	return result, nil
}
