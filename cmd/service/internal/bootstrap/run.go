package bootstrap

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Dankosik/search-service/internal/app/health"
	"github.com/Dankosik/search-service/internal/app/ping"
	"github.com/Dankosik/search-service/internal/config"
	httpx "github.com/Dankosik/search-service/internal/infra/http"
	"github.com/Dankosik/search-service/internal/infra/telemetry"
)

const (
	telemetryShutdownTimeout    = 5 * time.Second
	shutdownReadinessDelay      = 15 * time.Second
	startupBudget               = 30 * time.Second
	startupReserveBudget        = 3 * time.Second
	startupFailFastThreshold    = 150 * time.Millisecond
	startupConfigLoadBudget     = 10 * time.Second
	startupConfigValidateBudget = 2 * time.Second
	startupProbeBudget          = 15 * time.Second
	startupTelemetryBudget      = 2 * time.Second
	startupAdmissionBudget      = 2 * time.Second

	postgresProbeBudget = 5 * time.Second
	redisProbeBudget    = 3 * time.Second
	mongoProbeBudget    = 5 * time.Second

	startupRetryBaseDelay   = 50 * time.Millisecond
	startupRetryMaxDelay    = 250 * time.Millisecond
	postgresStartupAttempts = 2
	redisStoreProbeAttempts = 2
	mongoProbeAttempts      = 2
)

type overlayPathsFlag []string

func (f *overlayPathsFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *overlayPathsFlag) Set(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("config overlay path cannot be empty")
	}
	*f = append(*f, trimmed)
	return nil
}

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
	startupLifecycleStartedAt := time.Now()

	loadOptions.LoadBudget = startupConfigLoadBudget
	loadOptions.ValidateBudget = startupConfigValidateBudget

	bootstrap, err := bootstrapRuntime(startupCtx, loadOptions, metrics, startupLifecycleStartedAt)
	if err != nil {
		return err
	}
	bootstrapSpan := newStartupSpanController(bootstrap.bootstrapSpan, bootstrap.telemetryCleanup)
	defer bootstrapSpan.Close(startupCtx)

	bootstrapCtx := startupBootstrapContext(startupCtx, bootstrap.bootstrapSpan)

	probeOutcome, err := initStartupDependencies(startupCtx, bootstrapCtx, dependencyProbeRuntime{
		tracer:                    bootstrap.tracer,
		bootstrapSpan:             bootstrap.bootstrapSpan,
		cfg:                       bootstrap.cfg,
		metrics:                   metrics,
		log:                       bootstrap.log,
		networkPolicy:             bootstrap.networkPolicy,
		startupLifecycleStartedAt: startupLifecycleStartedAt,
	})
	if err != nil {
		return err
	}
	if probeOutcome.postgresPool != nil {
		defer probeOutcome.postgresPool.Close()
	}

	healthSvc := health.New(probeOutcome.probes...)
	pingSvc := ping.New()
	startupAdmission := newStartupAdmissionController(bootstrapSpan, metrics)
	ingressGuard := newRuntimeIngressAdmissionGuard(bootstrap.networkPolicy)
	readinessCheck := func(ctx context.Context) error {
		if err := ingressGuard.Check(ctx); err != nil {
			return err
		}
		return healthSvc.Ready(ctx)
	}

	handler := httpx.NewRouter(
		bootstrap.log,
		httpx.Handlers{
			Health:      healthSvc,
			Ping:        pingSvc,
			BeforeReady: ingressGuard.Check,
		},
		metrics,
		httpx.RouterConfig{MaxBodyBytes: bootstrap.cfg.HTTP.MaxBodyBytes},
	)

	srv := httpx.New(httpx.Config{
		Addr:              bootstrap.cfg.HTTP.Addr,
		ReadHeaderTimeout: bootstrap.cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       bootstrap.cfg.HTTP.ReadTimeout,
		WriteTimeout:      bootstrap.cfg.HTTP.WriteTimeout,
		IdleTimeout:       bootstrap.cfg.HTTP.IdleTimeout,
		MaxHeaderBytes:    bootstrap.cfg.HTTP.MaxHeaderBytes,
	}, handler)

	return serveHTTPRuntime(
		signalCtx,
		bootstrapCtx,
		bootstrap.bootstrapSpan,
		bootstrap.cfg,
		bootstrap.log,
		metrics,
		healthSvc,
		srv,
		readinessCheck,
		startupAdmission,
		shutdownReadinessDelay,
	)
}

func parseLoadOptions(args []string) (config.LoadOptions, error) {
	var overlays overlayPathsFlag

	flags := flag.NewFlagSet("service", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	configPath := flags.String("config", "", "path to base config file")
	flags.Var(&overlays, "config-overlay", "path to config overlay file (repeatable)")
	configStrict := flags.Bool("config-strict", false, "enable strict unknown-key validation")

	if err := flags.Parse(args); err != nil {
		return config.LoadOptions{}, fmt.Errorf("parse flags: %w", err)
	}

	return config.LoadOptions{
		ConfigPath:     strings.TrimSpace(*configPath),
		ConfigOverlays: overlays,
		Strict:         *configStrict,
	}, nil
}
