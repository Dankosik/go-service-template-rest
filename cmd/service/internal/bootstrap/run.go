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
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

// Budgets owned by every profile. Dependency-specific budgets and retry bounds
// live with their dependency stage so a profile that drops the dependency drops
// them in the same file.
const (
	telemetryShutdownTimeout = 5 * time.Second
	startupBudget            = 30 * time.Second
	startupFailFastThreshold = 150 * time.Millisecond
	startupTelemetryBudget   = 2 * time.Second

	// postgresProbeBudget also bounds config compatibility validation, which
	// every profile keeps.
	postgresProbeBudget = 5 * time.Second

	startupReadinessHeadroom = startupFailFastThreshold
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

	dependencies, err := initRuntimeDependencies(bootstrapCtx, startupCtx, bootstrap)
	if err != nil {
		return err
	}
	defer dependencies.Close()

	healthSvc := dependencies.health
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
			LogHealthProbes:  bootstrap.cfg.HTTP.AccessLogHealthProbes,
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

	var metricsSrv runtimeServer
	if bootstrap.cfg.Observability.Metrics.Addr != "" {
		metricsMux := http.NewServeMux()
		metricsMux.Handle("GET /metrics", metrics.Handler())
		metricsSrv = &http.Server{
			Handler:           metricsMux,
			ReadHeaderTimeout: bootstrap.cfg.HTTP.ReadHeaderTimeout,
			ReadTimeout:       bootstrap.cfg.HTTP.ReadTimeout,
			WriteTimeout:      bootstrap.cfg.HTTP.WriteTimeout,
			IdleTimeout:       bootstrap.cfg.HTTP.IdleTimeout,
			MaxHeaderBytes:    bootstrap.cfg.HTTP.MaxHeaderBytes,
		}
	}

	return serveHTTPRuntime(signalCtx, bootstrapCtx, serveHTTPRuntimeArgs{
		bootstrapSpan:  bootstrap.bootstrapSpan,
		cfg:            bootstrap.cfg,
		log:            bootstrap.log,
		healthSvc:      healthSvc,
		srv:            srv,
		metricsSrv:     metricsSrv,
		readinessCheck: healthSvc.Ready,
		admission:      startupAdmission,
		shutdownDelay:  bootstrap.cfg.HTTP.ReadinessPropagationDelay,
	})
}

func parseLoadOptions(args []string) (config.LoadOptions, error) {
	var configPath string
	var overlays []string

	flags := flag.NewFlagSet("service", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	flags.Func("config", "path to base config file", func(value string) error {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return fmt.Errorf("config path cannot be empty")
		}
		configPath = trimmed
		return nil
	})
	flags.Func("config-overlay", "path to config overlay file (repeatable)", func(value string) error {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return fmt.Errorf("config overlay path cannot be empty")
		}
		overlays = append(overlays, trimmed)
		return nil
	})
	if err := flags.Parse(args); err != nil {
		return config.LoadOptions{}, fmt.Errorf("parse flags: %w", err)
	}
	if positional := flags.Args(); len(positional) > 0 {
		return config.LoadOptions{}, fmt.Errorf("parse flags: unexpected positional arguments: %v", positional)
	}

	return config.LoadOptions{
		ConfigPath:     configPath,
		ConfigOverlays: overlays,
	}, nil
}
