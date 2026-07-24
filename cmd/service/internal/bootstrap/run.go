package bootstrap

import (
	"context"
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
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

const (
	telemetryShutdownTimeout = 5 * time.Second
	startupBudget            = 30 * time.Second
	startupReserveBudget     = 3 * time.Second
	startupFailFastThreshold = 150 * time.Millisecond
	startupProbeBudget       = 15 * time.Second
	startupTelemetryBudget   = 2 * time.Second

	postgresProbeBudget = 5 * time.Second

	startupReadinessHeadroom = startupFailFastThreshold

	startupRetryBaseDelay   = 50 * time.Millisecond
	startupRetryMaxDelay    = 250 * time.Millisecond
	postgresStartupAttempts = 2
)

func Run(args []string) (runErr error) {
	loadOptions, err := parseLoadOptions(args)
	if err != nil {
		return err
	}

	bootstrapLog := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})).With(
		"service.name", "service",
		"service.version", "unknown",
		"deployment.environment.name", "unknown",
	)
	slog.SetDefault(bootstrapLog)

	metrics := telemetry.New()
	// NotifyContext already unregisters on signal delivery, and stop is idempotent.
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	defer func() {
		if runErr != nil {
			slog.Error(
				"process_exit",
				startupLogArgs(
					signalCtx,
					"lifecycle",
					"process_exit",
					"error",
					"err", runErr,
				)...,
			)
			return
		}
		slog.Info(
			"process_exit",
			startupLogArgs(
				signalCtx,
				"lifecycle",
				"process_exit",
				"success",
			)...,
		)
	}()

	startupCtx, startupCancel := context.WithTimeout(signalCtx, startupBudget)
	defer startupCancel()

	bootstrap, err := bootstrapRuntime(startupCtx, loadOptions, metrics)
	if err != nil {
		return err
	}
	bootstrapSpan := newStartupSpanController(bootstrap.bootstrapSpan, bootstrap.telemetryCleanup)
	defer bootstrapSpan.Close(startupCtx)

	bootstrapCtx := startupBootstrapContext(startupCtx, bootstrap.bootstrapSpan)

	probeOutcome, err := initStartupDependencies(startupCtx, bootstrapCtx, postgresStartupRuntime{
		tracer:        bootstrap.tracer,
		bootstrapSpan: bootstrap.bootstrapSpan,
		cfg:           bootstrap.cfg,
		log:           bootstrap.log,
		networkPolicy: bootstrap.networkPolicy,
	})
	if err != nil {
		return err
	}
	if probeOutcome.postgresPool != nil {
		defer probeOutcome.postgresPool.Close()
	}

	healthSvc := health.New(probeOutcome.probes...)
	startupAdmission := newStartupAdmissionController(bootstrapSpan)

	handler, err := httpx.NewRouter(
		bootstrap.log,
		httpx.Handlers{
			Health:        healthSvc,
			ReadinessGate: startupAdmission.CheckReady,
		},
		metrics,
		httpx.RouterConfig{
			MaxBodyBytes:     bootstrap.cfg.HTTP.MaxBodyBytes,
			ReadinessTimeout: bootstrap.cfg.HTTP.ReadinessTimeout,
			OTelServerName:   bootstrap.cfg.Observability.OTel.ServiceName,
		},
	)
	if err != nil {
		return fmt.Errorf("build http router: %w", err)
	}

	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: bootstrap.cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       bootstrap.cfg.HTTP.ReadTimeout,
		WriteTimeout:      bootstrap.cfg.HTTP.WriteTimeout,
		IdleTimeout:       bootstrap.cfg.HTTP.IdleTimeout,
		MaxHeaderBytes:    bootstrap.cfg.HTTP.MaxHeaderBytes,
	}

	return serveHTTPRuntime(signalCtx, bootstrapCtx, serveHTTPRuntimeArgs{
		bootstrapSpan:  bootstrap.bootstrapSpan,
		cfg:            bootstrap.cfg,
		log:            bootstrap.log,
		healthSvc:      healthSvc,
		srv:            srv,
		readinessCheck: healthSvc.Ready,
		admission:      startupAdmission,
		shutdownDelay:  bootstrap.cfg.HTTP.ReadinessPropagationDelay,
	})
}

func parseLoadOptions(args []string) (config.LoadOptions, error) {
	var overlays []string

	flags := flag.NewFlagSet("service", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	configPath := flags.String("config", "", "path to base config file")
	flags.Func("config-overlay", "path to config overlay file (repeatable)", func(value string) error {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return fmt.Errorf("config overlay path cannot be empty")
		}
		overlays = append(overlays, trimmed)
		return nil
	})
	configStrict := flags.Bool("config-strict", false, "enable strict unknown-key validation")

	if err := flags.Parse(args); err != nil {
		return config.LoadOptions{}, fmt.Errorf("parse flags: %w", err)
	}
	if positional := flags.Args(); len(positional) > 0 {
		return config.LoadOptions{}, fmt.Errorf("parse flags: unexpected positional arguments: %v", positional)
	}

	return config.LoadOptions{
		ConfigPath:     strings.TrimSpace(*configPath),
		ConfigOverlays: overlays,
		Strict:         *configStrict,
	}, nil
}
