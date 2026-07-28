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
	"sync"
	"syscall"
	"time"

	"github.com/example/go-service-template-rest/internal/background"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
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
//	diagnosticsShutdownTimeout 2s   close /metrics, after the drain it measures
//	backgroundShutdownTimeout  5s   cancel and join supervised background tasks
//	dependencyCloseTimeout     5s   release pooled dependencies
//	telemetryShutdownTimeout   5s   span and metric flush, last so it records the above
//
// Every one of them is a ceiling, not a reservation, and all of them draw from
// one process-wide deadline; see shutdownBudget for why that is the only thing
// that makes the order above worth anything.
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

	// diagnosticsShutdownTimeout bounds closing the private metrics listener, which
	// serveHTTPRuntime does after the API drain rather than with it, so a scraper can
	// still collect what the drain recorded. It is short because the only thing it
	// can be waiting for is one in-flight scrape.
	diagnosticsShutdownTimeout = 2 * time.Second

	// shutdownTailBudget is everything the teardown can still spend once the HTTP
	// drain returns. http.shutdown_timeout bounds the drain and nothing else, so
	// this is the part an operator comparing a config against a platform grace
	// period cannot see; validateShutdownGraceBudget relates the two for them.
	shutdownTailBudget = diagnosticsShutdownTimeout +
		backgroundShutdownTimeout +
		dependencyCloseTimeout +
		telemetryShutdownTimeout
)

// shutdownBudget is the one deadline every teardown stage draws from.
//
// Without it the stages were four independent ceilings that happened to run in
// sequence, so the worst case was their sum: 30s of drain plus 17s of teardown
// against a platform that had promised 45s, and against the 30s Kubernetes grants
// by default. The stage that lost was always the same one — the telemetry flush
// runs last precisely so it can record what the others did, which makes it the
// first thing a SIGKILL takes. Every rolling deploy that actually used its drain
// threw away the evidence of how the drain went.
//
// The clock starts when teardown begins rather than at startup, because that is
// when the platform's grace period starts. All uses are on the goroutine running
// Run, in sequence; the Once is for the several callers that may each be the
// first to observe that serving has ended.
type shutdownBudget struct {
	grace    time.Duration
	started  sync.Once
	deadline time.Time
}

func newShutdownBudget(grace time.Duration) *shutdownBudget {
	return &shutdownBudget{grace: grace}
}

// start fixes the deadline, and is safe to call from every point that can begin
// a teardown: a signal, a server that stopped on its own, or a failed startup.
func (b *shutdownBudget) start() {
	b.started.Do(func() { b.deadline = time.Now().Add(b.grace) })
}

// stage bounds one teardown stage by the smaller of its own budget and what is
// left of the grace period.
//
// It is detached from the signal context, which is already canceled by the time
// any stage runs, so the stage gets the bound rather than an instant expiry.
func (b *shutdownBudget) stage(base context.Context, want time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(base), b.clamp(want))
	// Consumed synchronously by the stage. Releasing the timer at the call site
	// would defeat the bound.
	context.AfterFunc(ctx, cancel)
	return ctx
}

// clamp reports how long a stage asking for want may actually take.
//
// A spent grace period yields a floor rather than zero: the process is about to
// be killed either way, and a stage given no time at all cannot even report that
// it was cut short.
func (b *shutdownBudget) clamp(want time.Duration) time.Duration {
	b.start()
	remaining := time.Until(b.deadline)
	if remaining < shutdownStageFloor {
		return shutdownStageFloor
	}
	return min(want, remaining)
}

// shutdownStageFloor is the smallest budget a teardown stage is given once the
// grace period is spent. It is enough for an already-idle stage to finish and to
// record that it did, and short enough not to extend a shutdown that is over.
const shutdownStageFloor = 100 * time.Millisecond

// validateShutdownGraceBudget rejects a drain budget that cannot fit inside the
// platform's grace period alongside the teardown that follows it.
//
// It lives here rather than in internal/config because shutdownTailBudget is
// owned by this package: the four stage ceilings are process structure, not
// configuration, and an operator can only reason about them against the one
// number the platform gave them.
func validateShutdownGraceBudget(cfg config.Config) error {
	required := cfg.HTTP.ShutdownTimeout + shutdownTailBudget
	if cfg.HTTP.GracePeriod >= required {
		return nil
	}
	return fmt.Errorf(
		"%w: http.grace_period must be >= http.shutdown_timeout plus the post-drain teardown budget (%s + %s = %s)",
		config.ErrValidate,
		cfg.HTTP.ShutdownTimeout,
		shutdownTailBudget,
		required,
	)
}

func Run(args []string) (runErr error) {
	loadOptions, err := parseLoadOptions(args)
	if err != nil {
		return err
	}

	// Correlation-carrying from the first record, because config rejection and a
	// failed exit are logged through this one before bootstrapLoggerStage replaces
	// it with the configured logger.
	bootstrapLog := newProcessLogger(os.Stdout, slog.LevelInfo).With(
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
			slog.ErrorContext(
				signalCtx,
				"process_exit",
				startupLogArgs(
					"lifecycle",
					"process_exit",
					"error",
					"err", runErr,
				)...,
			)
			return
		}
		slog.InfoContext(
			signalCtx,
			"process_exit",
			startupLogArgs(
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
	// One deadline for the whole teardown, armed when serving ends. Every stage
	// below draws from it, so their ceilings can no longer add up past what the
	// platform actually granted.
	shutdown := newShutdownBudget(bootstrap.cfg.HTTP.GracePeriod)
	// Telemetry is flushed last, so it can carry a record of everything the
	// shutdown path below it did. The closure matters: a deferred call evaluates
	// its arguments at registration, so building the context here rather than
	// inside would start its clock at startup and hand the flush a spent budget.
	defer func() { bootstrap.telemetryCleanup(shutdown.stage(signalCtx, telemetryShutdownTimeout)) }()

	// The GC limit is published before any dependency allocates, so the first
	// large allocation is already collected against the container's real ceiling
	// rather than against math.MaxInt64.
	memoryLimit := applyMemoryLimit(bootstrap.log, bootstrap.cfg.Runtime.MemoryLimitRatio)
	// Reported against the same number, because http.max_in_flight and
	// http.max_body_bytes bound a heap the GC was just handed a ceiling for and
	// nothing else multiplies the two.
	reportRequestBufferBudget(bootstrap.log, bootstrap.cfg, memoryLimit)

	dependencies, err := initRuntimeDependencies(startupCtx, bootstrap)
	if err != nil {
		return err
	}
	// Close is idempotent. The deferred call is the safety net for the early
	// returns below; the ordered shutdown path closes it explicitly, before the
	// telemetry flush and after background work has been joined.
	defer func() { dependencies.Close(shutdown.stage(signalCtx, dependencyCloseTimeout)) }()

	startupAdmission := newStartupAdmissionController()

	supervisor := newSupervisedBackground(signalCtx, bootstrap.log)
	// The ordered teardown below owns the real stop. This is the safety net for
	// the early returns between here and there: with cancellation detached from
	// the signal, a return that skipped Shutdown would leave supervised
	// goroutines running past Run.
	defer func() {
		_ = supervisor.Shutdown(shutdown.stage(signalCtx, backgroundShutdownTimeout))
	}()

	// The failure channel below terminates serving; the readiness probe makes the
	// same failure visible during the short interval before the drain begins.
	healthSvc := health.New(append(dependencies.ReadinessProbes(), supervisor)...)

	// Readiness is served from cached state, so something has to keep that state
	// current. Registering it as an ordinary supervised task means a refresher
	// that dies takes the same reported path as any other failed background work;
	// health.Cached expiring a verdict nothing is refreshing is what covers the
	// ways it can die without reporting.
	supervisor.Go(background.Task{
		Name: "readiness_refresh",
		Run: func(ctx context.Context) error {
			return healthSvc.Watch(
				ctx,
				bootstrap.cfg.Health.RefreshInterval,
				readinessProbeBudget(bootstrap.cfg),
				bootstrap.cfg.Health.FailureThreshold,
				func(err error) {
					logReadinessTransition(ctx, bootstrap.log, err)
				},
			)
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
			// The active profile's dependency failures, classified once here
			// rather than in every operation. A service appends its own domain
			// mappers to this slice; see httpx.DomainErrorMapper.
			DomainErrors: dependencies.DomainErrors(),
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
		cfg:                bootstrap.cfg,
		log:                bootstrap.log,
		healthSvc:          healthSvc,
		srv:                srv,
		metricsSrv:         metricsSrv,
		backgroundFailures: supervisor.Failures(),
		// Admission refreshes rather than probing separately, so the verdict it
		// admits on is the same one the probe route will serve. Without that, the
		// first probe after admission could still answer 503 from an unevaluated
		// cache and have the instance pulled straight back out of rotation.
		readinessCheck: func(ctx context.Context) error {
			return healthSvc.Refresh(ctx, bootstrap.cfg.HTTP.ReadinessTimeout, bootstrap.cfg.Health.FailureThreshold)
		},
		admission:     startupAdmission,
		shutdownDelay: bootstrap.cfg.HTTP.ReadinessPropagationDelay,
		shutdown:      shutdown,
	})

	// Ordered teardown. HTTP is already drained, so this is the first moment
	// nothing can still be depending on background work; it is canceled and
	// joined here, before the dependencies it uses are released, and both happen
	// before the deferred telemetry flush so the flush can carry a record of how
	// they went. Shutdown is idempotent, so the deferred safety net above is a
	// no-op once this has run.
	backgroundErr := supervisor.Shutdown(shutdown.stage(signalCtx, backgroundShutdownTimeout))

	dependencies.Close(shutdown.stage(signalCtx, dependencyCloseTimeout))

	return errors.Join(serveErr, backgroundErr)
}

func logReadinessTransition(ctx context.Context, log *slog.Logger, err error) {
	if err != nil {
		log.WarnContext(
			ctx,
			"readiness_transition",
			"component", "readiness",
			"operation", "refresh",
			"outcome", "unhealthy",
			"err", err,
		)
		return
	}
	log.InfoContext(
		ctx,
		"readiness_transition",
		"component", "readiness",
		"operation", "refresh",
		"outcome", "healthy",
	)
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
