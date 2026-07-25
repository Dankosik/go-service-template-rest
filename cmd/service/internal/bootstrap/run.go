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
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

// Budgets owned by every profile. How they nest, outermost first:
//
//	startupBudget             30s   flag parsing through readiness admission
//	 ├─ startupTelemetryBudget 2s   metrics setup, then tracing setup
//	 ├─ postgresStartupBudget 15s   PostgreSQL profile only (startup_dependencies.go)
//	 │   └─ postgresProbeBudget 5s  pool open and first ping
//	 └─ http.readiness_timeout      startup admission (typed config)
//
// Shutdown budgets, in the order they are spent after the HTTP drain:
//
//	backgroundShutdownTimeout  5s   cancel and join supervised background tasks
//	dependencyCloseTimeout     5s   release pooled dependencies
//	telemetryShutdownTimeout   5s   span and metric flush, last so it records the above
//
// Background work is canceled here rather than on the signal, because the HTTP
// drain above it is still serving requests that depend on it; see
// newSupervisedBackground.
//
// Dependency-specific budgets live with their dependency stage so a profile that
// drops the dependency drops them in the same file.
const (
	telemetryShutdownTimeout = 5 * time.Second
	startupBudget            = 30 * time.Second
	startupTelemetryBudget   = 2 * time.Second

	// backgroundShutdownTimeout bounds the join. A task that ignores its
	// cancellation is reported rather than waited on, so it cannot hold up the
	// telemetry flush that has to record that it did.
	backgroundShutdownTimeout = 5 * time.Second

	// dependencyCloseTimeout bounds releasing pooled dependencies. pgxpool.Close
	// blocks until every acquired connection is returned and destroyed, and takes
	// no context of its own — so a handler that outlived the drain while holding a
	// connection would otherwise block the process here until the platform
	// SIGKILLs it, discarding the shutdown telemetry on the way out.
	dependencyCloseTimeout = 5 * time.Second
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
	// Telemetry is flushed last, so it can carry a record of everything the
	// shutdown path below it did.
	defer bootstrap.telemetryCleanup(startupCtx)

	// The GC limit is published before any dependency allocates, so the first
	// large allocation is already collected against the container's real ceiling
	// rather than against math.MaxInt64.
	applyMemoryLimit(bootstrap.log, bootstrap.cfg.Runtime.MemoryLimitRatio)

	dependencies, err := initRuntimeDependencies(startupCtx, bootstrap)
	if err != nil {
		return err
	}
	// Close is idempotent. The deferred call is the safety net for the early
	// returns below; the ordered shutdown path closes it explicitly, before the
	// telemetry flush and after background work has been joined.
	defer dependencies.Close(dependencyCloseContext(signalCtx))

	healthSvc := dependencies.health
	startupAdmission := newStartupAdmissionController()

	supervisor := newSupervisedBackground(signalCtx, bootstrap.log)
	// The ordered teardown below owns the real stop. This is the safety net for
	// the early returns between here and there: with cancellation detached from
	// the signal, a return that skipped Shutdown would leave supervised
	// goroutines running past Run.
	defer func() {
		_ = supervisor.Shutdown(backgroundShutdownContext(signalCtx))
	}()
	// Readiness is served from cached state, so something has to keep that state
	// current. Registering it as an ordinary supervised task means a refresher
	// that dies takes the same reported path as any other failed background work
	// instead of leaving a stale verdict behind.
	supervisor.Go(background.Task{
		Name: "readiness_refresh",
		Run: func(ctx context.Context) error {
			return healthSvc.Watch(ctx, bootstrap.cfg.Health.RefreshInterval, bootstrap.cfg.Health.FailureThreshold)
		},
	})

	handler, err := httpx.NewRouter(
		bootstrap.log,
		httpx.Handlers{
			Health:        healthSvc,
			ReadinessGate: startupAdmission.CheckReady,
		},
		metrics,
		httpx.RouterConfig{
			MaxBodyBytes:    bootstrap.cfg.HTTP.MaxBodyBytes,
			RequestTimeout:  bootstrap.cfg.HTTP.RequestTimeout,
			MaxInFlight:     bootstrap.cfg.HTTP.MaxInFlight,
			OTelServerName:  bootstrap.cfg.Observability.OTel.ServiceName,
			LogHealthProbes: bootstrap.cfg.HTTP.AccessLogHealthProbes,
			// Authenticate is deliberately unset. This contract declares no
			// security requirement, so nothing reaches the seam; an operation
			// that adds one gets a fail-closed 401 until a service supplies its
			// own AuthenticationFunc here.
		},
	)
	if err != nil {
		return fmt.Errorf("build http router: %w", err)
	}

	// ErrorLog routes net/http's own reporting through the service logger. With it
	// unset, a malformed request line, a TLS handshake failure, or a panic that
	// escapes the handler chain is printed as unstructured text to stderr by the
	// standard log package — which a pipeline configured for JSON on stdout either
	// drops or files under parse failures, hiding exactly the class of error that
	// diagnoses a bad client or a broken proxy.
	errorLog := slog.NewLogLogger(bootstrap.log.Handler(), slog.LevelError)

	srv := &http.Server{
		Handler:           handler,
		ErrorLog:          errorLog,
		ReadHeaderTimeout: bootstrap.cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       bootstrap.cfg.HTTP.ReadTimeout,
		WriteTimeout:      bootstrap.cfg.HTTP.WriteTimeout,
		IdleTimeout:       bootstrap.cfg.HTTP.IdleTimeout,
		MaxHeaderBytes:    bootstrap.cfg.HTTP.MaxHeaderBytes,
	}

	var metricsSrv runtimeServer
	if bootstrap.cfg.Observability.Metrics.Addr != "" {
		metricsSrv = newDiagnosticsServer(bootstrap.cfg, metrics, errorLog)
	}

	serveErr := serveHTTPRuntime(signalCtx, startupCtx, serveHTTPRuntimeArgs{
		cfg:        bootstrap.cfg,
		log:        bootstrap.log,
		healthSvc:  healthSvc,
		srv:        srv,
		metricsSrv: metricsSrv,
		// Admission refreshes rather than probing separately, so the verdict it
		// admits on is the same one the probe route will serve. Without that, the
		// first probe after admission could still answer 503 from an unevaluated
		// cache and have the instance pulled straight back out of rotation.
		readinessCheck: func(ctx context.Context) error {
			return healthSvc.Refresh(ctx, bootstrap.cfg.HTTP.ReadinessTimeout, bootstrap.cfg.Health.FailureThreshold)
		},
		admission:     startupAdmission,
		shutdownDelay: bootstrap.cfg.HTTP.ReadinessPropagationDelay,
	})

	// Ordered teardown. HTTP is already drained, so this is the first moment
	// nothing can still be depending on background work; it is canceled and
	// joined here, before the dependencies it uses are released, and both happen
	// before the deferred telemetry flush so the flush can carry a record of how
	// they went. Shutdown is idempotent, so the deferred safety net above is a
	// no-op once this has run.
	backgroundErr := supervisor.Shutdown(backgroundShutdownContext(signalCtx))

	dependencies.Close(dependencyCloseContext(signalCtx))

	return errors.Join(serveErr, backgroundErr)
}

// newSupervisedBackground builds the supervisor for runtime background work,
// deliberately detached from the signal context so that Shutdown is the only
// thing that stops it.
//
// Deriving it from signalCtx would cancel every task the instant SIGTERM
// arrived, while the HTTP drain keeps serving for up to
// readiness_propagation_delay plus the remaining shutdown_timeout — 30s with the
// shipped defaults. Every request admitted in that window would run without the
// work it depends on: an outbox publisher, a write batcher, a token refresher,
// or a lease renewer would already be gone. Nothing would report it either, so
// the only artifact is one INFO line saying the task was canceled, in the middle
// of a shutdown that otherwise looks orderly.
func newSupervisedBackground(signalCtx context.Context, log *slog.Logger) *background.Supervisor {
	return background.New(context.WithoutCancel(signalCtx), log)
}

// backgroundShutdownContext bounds the cancel-and-join, detached from the signal
// context that is already canceled by the time it is spent.
func backgroundShutdownContext(base context.Context) context.Context {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(base), backgroundShutdownTimeout)
	// Consumed synchronously by Shutdown; releasing the timer at the call site
	// would defeat the bound. dependencyCloseContext does the same.
	context.AfterFunc(ctx, cancel)
	return ctx
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
