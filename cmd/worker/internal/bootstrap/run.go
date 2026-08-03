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

	"github.com/example/go-service-template-rest/internal/background"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/observability/logctx"
)

const (
	startupTimeout   = 30 * time.Second
	diagnosticsClose = 2 * time.Second
	backgroundClose  = 5 * time.Second
	handlerClose     = 5 * time.Second
	telemetryClose   = 5 * time.Second
)

// HandlerBuilder constructs the binary-local transport adapter that invokes
// feature-owned behavior after messaging topology admission. It receives the
// same concrete producer owned by this worker process. Any returned cleanup
// normally runs before Run returns, including when the builder returns an
// invalid result.
// If forced shutdown leaves an uncooperative handler running, process exit owns
// its resources instead of racing that cleanup with the handler.
type HandlerBuilder func(context.Context, config.Config, *slog.Logger, *natsjs.Producer) (handler natsjs.Handler, cleanup func(context.Context), err error)

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
	if err := validateWorkerShutdownBudget(cfg.HTTP.GracePeriod, cfg.Messaging.Worker.DrainTimeout); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.Observability.Metrics.Addr) == "" {
		return fmt.Errorf("%w: worker diagnostics address is required", natsjs.ErrRejected)
	}
	log := newWorkerLogger(os.Stdout, cfg)
	metrics := telemetry.New()
	telemetryCleanup, err := setupTelemetry(startupCtx, cfg, metrics, log)
	if err != nil {
		return err
	}
	var cleanupDeadline time.Time
	defer func() {
		cleanupCtx, cleanupCancel := workerCleanupContext(signalCtx, cleanupDeadline, telemetryClose)
		defer cleanupCancel()
		telemetryCleanup(cleanupCtx)
	}()
	client, err := natsjs.Connect(startupCtx, messagingClientConfig(cfg.Messaging), natsjs.RoleWorker, natsjs.Observability{Logger: log})
	if err != nil {
		return fmt.Errorf("initialize worker messaging: %w", err)
	}
	defer client.Close()
	handler, handlerCleanup, err := buildHandler(startupCtx, cfg, log, client.Producer())
	defer func() {
		if handlerCleanup != nil {
			cleanupCtx, cleanupCancel := workerCleanupContext(signalCtx, cleanupDeadline, handlerClose)
			defer cleanupCancel()
			handlerCleanup(cleanupCtx)
		}
	}()
	if err != nil {
		return fmt.Errorf("initialize worker feature handler: %w", err)
	}
	if handler == nil {
		return fmt.Errorf("%w: worker feature handler is not registered", natsjs.ErrRejected)
	}
	worker, err := client.NewWorker(startupCtx, workerCfg, handler)
	if err != nil {
		return fmt.Errorf("initialize durable consumer: %w", err)
	}
	cleanupSafe, deadline, err := runWorkerLifecycle(signalCtx, startupCtx, cfg, log, metrics, client, worker)
	cleanupDeadline = deadline
	if !cleanupSafe {
		handlerCleanup = nil
	}
	return err
}

func newWorkerLogger(out io.Writer, cfg config.Config) *slog.Logger {
	return slog.New(logctx.New(slog.NewJSONHandler(out, &slog.HandlerOptions{Level: cfg.Log.Level}))).With(
		"service.name", cfg.Observability.OTel.ServiceName,
		"service.version", cfg.App.Version,
		"deployment.environment.name", cfg.App.Env,
	)
}

func workerCleanupContext(base context.Context, deadline time.Time, limit time.Duration) (context.Context, context.CancelFunc) {
	base = context.WithoutCancel(base)
	if deadline.IsZero() {
		return context.WithTimeout(base, limit)
	}
	deadlineCtx, deadlineCancel := context.WithDeadline(base, deadline)
	stageCtx, stageCancel := context.WithTimeout(deadlineCtx, limit)
	return stageCtx, func() {
		stageCancel()
		deadlineCancel()
	}
}

func runWorkerLifecycle(
	signalCtx context.Context,
	startupCtx context.Context,
	cfg config.Config,
	log *slog.Logger,
	metrics *telemetry.Metrics,
	client *natsjs.Client,
	worker *natsjs.Worker,
) (bool, time.Time, error) {
	healthSvc := health.New(client)
	if err := healthSvc.Refresh(startupCtx, cfg.HTTP.ReadinessTimeout, cfg.Health.FailureThreshold); err != nil {
		return true, time.Time{}, fmt.Errorf("admit worker readiness: %w", err)
	}
	diagnostics := newDiagnosticsServer(cfg.Observability.Metrics.Addr, healthSvc, client.Ready, metrics)
	listener, err := listenDiagnostics(startupCtx, cfg.Observability.Metrics.Addr)
	if err != nil {
		return true, time.Time{}, err
	}
	defer func() { _ = listener.Close() }()

	runtimeCtx := context.WithoutCancel(signalCtx)
	supervisor := background.New(runtimeCtx, log)
	supervisor.Go(background.Task{Name: "messaging_connection", Run: client.Run})
	supervisor.Go(background.Task{
		Name: "messaging_readiness",
		Run: func(ctx context.Context) error {
			return healthSvc.Watch(ctx, cfg.Health.RefreshInterval, cfg.HTTP.ReadinessTimeout, cfg.Health.FailureThreshold, nil)
		},
	})
	workerResult := make(chan error, 1)
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		workerResult <- worker.Run(runtimeCtx)
	}()
	diagnosticsResult := make(chan error, 1)
	go func() {
		err := diagnostics.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		diagnosticsResult <- err
	}()

	var triggerErr error
	workerResultRead := false
	diagnosticsResultRead := false
	select {
	case <-signalCtx.Done():
	case triggerErr = <-supervisor.Failures():
	case triggerErr = <-workerResult:
		workerResultRead = true
	case triggerErr = <-diagnosticsResult:
		diagnosticsResultRead = true
	}
	if signalCtx.Err() == nil {
		if triggerErr == nil {
			triggerErr = fmt.Errorf("worker runtime stopped unexpectedly")
		}
	}
	healthSvc.StartDrain()
	worker.StartDrain()
	shutdownBase := context.WithoutCancel(signalCtx)
	processCtx, processCancel := context.WithTimeout(shutdownBase, cfg.HTTP.GracePeriod)
	shutdownDeadline, _ := processCtx.Deadline()
	defer processCancel()
	workerCtx, workerCancel := context.WithTimeout(processCtx, cfg.Messaging.Worker.DrainTimeout)
	workerErr := worker.Shutdown(workerCtx)
	workerCancel()
	diagnosticsCtx, diagnosticsCancel := context.WithTimeout(processCtx, diagnosticsClose)
	diagnosticsErr := diagnostics.Shutdown(diagnosticsCtx)
	diagnosticsErr = errors.Join(diagnosticsErr, diagnostics.Close())
	if !diagnosticsResultRead {
		select {
		case runErr := <-diagnosticsResult:
			diagnosticsErr = errors.Join(diagnosticsErr, runErr)
		case <-diagnosticsCtx.Done():
			diagnosticsErr = errors.Join(diagnosticsErr, fmt.Errorf("join worker diagnostics: %w", diagnosticsCtx.Err()))
		}
	}
	diagnosticsCancel()
	backgroundCtx, backgroundCancel := context.WithTimeout(processCtx, backgroundClose)
	backgroundErr := supervisor.Shutdown(backgroundCtx)
	backgroundCancel()
	cleanupSafe := handlerStoppedBeforeReturn(workerErr, workerDone)
	if !workerResultRead {
		select {
		case runErr := <-workerResult:
			if triggerErr == nil {
				triggerErr = runErr
			}
		default:
		}
	}
	return cleanupSafe, shutdownDeadline, errors.Join(triggerErr, diagnosticsErr, workerErr, backgroundErr)
}

func handlerStoppedBeforeReturn(workerErr error, workerDone <-chan struct{}) bool {
	if workerErr == nil {
		<-workerDone
		return true
	}
	select {
	case <-workerDone:
		return true
	default:
		// A handler that ignored forced cancellation may still use its
		// dependencies. Process exit owns their cleanup in this path.
		return false
	}
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

func validateWorkerShutdownBudget(gracePeriod, drainTimeout time.Duration) error {
	cleanupReserve := diagnosticsClose + backgroundClose + handlerClose + telemetryClose
	required := drainTimeout + cleanupReserve
	if gracePeriod >= required {
		return nil
	}
	return fmt.Errorf(
		"%w: http.grace_period must be >= messaging worker drain timeout plus the post-drain teardown budget (%s + %s = %s)",
		config.ErrValidate,
		drainTimeout,
		cleanupReserve,
		required,
	)
}
