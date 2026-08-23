package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/go-service-template-rest/internal/background"
	// profile:bootstrap-config:start
	"github.com/example/go-service-template-rest/internal/config"
	// profile:bootstrap-config:end
	// profile:http-idempotency-postgres:start
	"github.com/example/go-service-template-rest/internal/httpidempotency"
	// profile:http-idempotency-postgres:end
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/observability/logctx"
)

// Budgets owned by every profile. How they nest, outermost first:
//
//	startupBudget             30s   flag parsing through readiness admission
//	 ├─ startupTelemetryBudget 2s   metrics setup, then tracing setup
//	 ├─ dependency stage            connect budget and first-contact probe, only
//	 │                              in a profile that has a dependency
//	 └─ http.readiness_timeout      startup admission (typed config)
//
// Shutdown budgets, in the order they are spent after the HTTP drain:
//
//	diagnosticsShutdownTimeout 2s   close /metrics, after the drain it measures
//	backgroundShutdownTimeout  5s   cancel and join supervised background tasks
//	dependencyCloseTimeout     5s   release pooled dependencies
//	telemetryShutdownTimeout   5s   span and metric flush, last so it records the above
//
// Every one is a ceiling, not a reservation, and all draw from one process-wide
// deadline; see shutdownBudget. Dependency-specific budgets live with their
// dependency stage, so a profile that drops the dependency drops them too.
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
	// serveRuntime does after the API drain rather than with it, so a scraper can
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

// runtimeWiring carries the one process edge that tests must stop without
// binding sockets.
type runtimeWiring struct {
	dependencies func(context.Context, startupBootstrap) (runtimeDependencies, error)
	serve        func(context.Context, context.Context, serveRuntimeArgs) error
	lifecycle    func(runtimeLifecycleStage)
	// profile:object-storage:start
	initObjectStorage func(context.Context, config.ObjectStorageConfig) (objectStorageRuntime, error)
	// profile:object-storage:end
	// profile:authn-bearer:start
	initAuthn  func(context.Context, config.Config, *telemetry.Metrics, *slog.Logger) (authnRuntime, error)
	authnStage func(authnBootstrapStage)
	// profile:authn-bearer:end
}

type runtimeLifecycleStage string

const (
	runtimeLifecycleMemoryPublished runtimeLifecycleStage = "memory_published"
	// profile:object-storage:start
	runtimeLifecycleObjectStorageConstructed runtimeLifecycleStage = "object_storage_constructed"
	// profile:object-storage:end
	runtimeLifecycleHTTPDrained      runtimeLifecycleStage = "http_drained"
	runtimeLifecycleBackgroundJoined runtimeLifecycleStage = "background_joined"
	// profile:object-storage:start
	runtimeLifecycleObjectStorageClosed runtimeLifecycleStage = "object_storage_closed"
	// profile:object-storage:end
	runtimeLifecycleDependenciesClosed runtimeLifecycleStage = "dependencies_closed"
	runtimeLifecycleTelemetryFlushed   runtimeLifecycleStage = "telemetry_flushed"
)

func productionRuntimeWiring() runtimeWiring {
	return runtimeWiring{
		dependencies: initRuntimeDependencies,
		serve:        serveRuntime,
		lifecycle:    func(runtimeLifecycleStage) {},
		// profile:object-storage:start
		initObjectStorage: initObjectStorage,
		// profile:object-storage:end
		// profile:authn-bearer:start
		initAuthn: func(
			ctx context.Context,
			cfg config.Config,
			metrics *telemetry.Metrics,
			log *slog.Logger,
		) (authnRuntime, error) {
			return initAuthn(ctx, cfg, metrics, log)
		},
		authnStage: func(authnBootstrapStage) {},
		// profile:authn-bearer:end
	}
}

func checkRuntimeReadiness(
	ctx context.Context,
	startupAdmission *startupAdmissionController,
	// profile:messaging-nats-jetstream:start
	messaging messagingRuntime,
	// profile:messaging-nats-jetstream:end
) error {
	if err := startupAdmission.CheckReady(ctx); err != nil {
		return err
	}
	// profile:messaging-nats-jetstream:start
	if !messaging.Ready() {
		return errors.New("messaging is not ready")
	}
	// profile:messaging-nats-jetstream:end
	return nil
}

func Run(args []string) error {
	return runWithRuntime(args, productionRuntimeWiring())
}

func runWithRuntime(args []string, wiring runtimeWiring) (runErr error) {
	loadOptions, err := parseLoadOptions(args)
	if err != nil {
		return err
	}

	// Correlation-carrying from the first record, because config rejection and a
	// failed exit are logged through this one before bootstrapLoggerStage replaces
	// it with the configured logger.
	bootstrapLog := logctx.NewProcessLogger(os.Stdout, slog.LevelInfo).With(
		"service.name", "service",
		"service.version", "unknown",
		"deployment.environment.name", "unknown",
	)
	slog.SetDefault(bootstrapLog)

	metrics := telemetry.New()
	// NotifyContext already unregisters on signal delivery, and stop is idempotent.
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	defer func() { logProcessExit(signalCtx, runErr) }()

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
	defer func() {
		bootstrap.telemetryCleanup(shutdown.stage(signalCtx, telemetryShutdownTimeout))
		wiring.lifecycle(runtimeLifecycleTelemetryFlushed)
	}()

	// The GC limit is published before any dependency allocates, so the first
	// large allocation is already collected against the container's real ceiling
	// rather than against math.MaxInt64.
	memoryLimit := applyMemoryLimit(bootstrap.log, bootstrap.cfg.Runtime.MemoryLimitRatio)
	wiring.lifecycle(runtimeLifecycleMemoryPublished)
	// Reported against the same number, because http.max_in_flight and
	// http.max_body_bytes bound a heap the GC was just handed a ceiling for and
	// nothing else multiplies the two.
	reportRequestBufferBudget(bootstrap.log, bootstrap.cfg, memoryLimit)

	// profile:object-storage:start
	objectStorage, err := wiring.initObjectStorage(startupCtx, bootstrap.cfg.ObjectStorage)
	if err != nil {
		return err
	}
	wiring.lifecycle(runtimeLifecycleObjectStorageConstructed)
	objectStorageClosed := false
	defer func() {
		if !objectStorageClosed {
			objectStorage.Close()
			wiring.lifecycle(runtimeLifecycleObjectStorageClosed)
		}
	}()
	// profile:object-storage:end

	dependencies, err := wiring.dependencies(startupCtx, bootstrap)
	if err != nil {
		return err
	}
	// Close is idempotent. The deferred call is the safety net for the early
	// returns below; the ordered shutdown path closes it explicitly, before the
	// telemetry flush and after background work has been joined.
	dependenciesClosed := false
	closeOwners := func() {
		if dependenciesClosed {
			return
		}

		runtimeCloseCtx := shutdown.stage(signalCtx, dependencyCloseTimeout)
		// profile:object-storage:start
		if !objectStorageClosed {
			objectStorage.Close()
			objectStorageClosed = true
			wiring.lifecycle(runtimeLifecycleObjectStorageClosed)
		}
		// profile:object-storage:end
		dependencies.Close(runtimeCloseCtx)
		dependenciesClosed = true
		wiring.lifecycle(runtimeLifecycleDependenciesClosed)
	}
	defer closeOwners()

	// profile:authn-bearer:start
	authnVerifier, err := wiring.initAuthn(startupCtx, bootstrap.cfg, metrics, bootstrap.log)
	if err != nil {
		return err
	}
	wiring.authnStage(authnStageTrustEstablished)
	authnClosed := false
	defer func() {
		if !authnClosed {
			authnVerifier.Close()
		}
	}()
	// profile:authn-bearer:end

	startupAdmission := new(startupAdmissionController)

	supervisor := newSupervisedBackground(signalCtx, bootstrap.log)
	// The ordered teardown below owns the real stop. This is the safety net for
	// the early returns between here and there: with cancellation detached from
	// the signal, a return that skipped Shutdown would leave supervised
	// goroutines running past Run.
	defer func() {
		backgroundCtx := shutdown.stage(signalCtx, backgroundShutdownTimeout)
		_ = supervisor.Shutdown(backgroundCtx)
	}()

	// profile:messaging-nats-jetstream:start
	messaging, err := initMessagingRuntime(startupCtx, bootstrap.cfg.Messaging, bootstrap.log)
	if err != nil {
		return err
	}
	defer messaging.Close()
	if connectionRun := messaging.ConnectionRun(); connectionRun != nil {
		supervisor.Go(background.Task{Name: "messaging_connection", Run: connectionRun})
	}
	// profile:messaging-nats-jetstream:end

	// profile:http-idempotency-postgres:start
	idempotency, err := initDeclaredHTTPIdempotency(
		startupCtx,
		bootstrap.cfg,
		dependencies.postgres,
		bootstrap.log,
	)
	if err != nil {
		return err
	}
	idempotency.Supervise(supervisor)
	// profile:http-idempotency-postgres:end

	// The failure channel below terminates serving; the readiness probe makes the
	// same failure visible during the short interval before the drain begins.
	readinessProbes := dependencies.ReadinessProbes()
	// profile:messaging-nats-jetstream:start
	readinessProbes = append(readinessProbes, messaging.ReadinessProbes()...)
	// profile:messaging-nats-jetstream:end
	healthSvc := newReadinessService(readinessProbes, supervisor)

	domainErrors := dependencies.DomainErrors()
	// profile:http-idempotency-postgres:start
	domainErrors = append(domainErrors, httpidempotency.ClassifyError)
	// profile:http-idempotency-postgres:end
	// profile:inbound-webhooks-standard:start
	inboundReceiver, err := initInboundWebhookReceiver(bootstrap.cfg, dependencies.postgres, metrics, bootstrap.log)
	if err != nil {
		return err
	}
	// profile:inbound-webhooks-standard:end
	handler, err := newHTTPHandler(
		bootstrap.cfg,
		bootstrap.log,
		metrics,
		domainErrors,
		httpRuntimeBindings{
			Handlers: httpx.Handlers{
				Health: healthSvc,
				ReadinessGate: func(ctx context.Context) error {
					return checkRuntimeReadiness(
						ctx,
						startupAdmission,
						// profile:messaging-nats-jetstream:start
						messaging,
						// profile:messaging-nats-jetstream:end
					)
				},
				// profile:inbound-webhooks-standard:start
				InboundWebhook: inboundReceiver,
				// profile:inbound-webhooks-standard:end
			},
			// profile:authn-bearer:start
			Authenticate:          httpx.Authenticated(authnVerifier.ResolveHTTP),
			AuthenticateChallenge: "Bearer",
			// profile:authn-bearer:end
		},
	)
	if err != nil {
		return err
	}
	// profile:authn-bearer:start
	wiring.authnStage(authnStageHTTPRouterBuilt)
	// profile:authn-bearer:end

	// Shared with the diagnostics listener below, so both publish net/http's own
	// reporting through the service logger; newHTTPServer owns why that matters.
	errorLog := slog.NewLogLogger(bootstrap.log.Handler(), slog.LevelError)
	srv := newHTTPServer(bootstrap.cfg.HTTP, handler, errorLog)
	// profile:authn-bearer:start
	wiring.authnStage(authnStageHTTPServerBuilt)
	// profile:authn-bearer:end

	// profile:grpc:start
	var grpcSrv grpcRuntimeServer
	if bootstrap.cfg.GRPC.Server.Enabled {
		builtGRPC, buildErr := newGRPCRuntime(
			bootstrap.cfg,
			bootstrap.log,
			metrics,
			domainErrors,
			serviceGRPCBindings(
				// profile:authn-bearer:start
				authnVerifier,
				// profile:authn-bearer:end
			),
		)
		if buildErr != nil {
			return buildErr
		}
		grpcSrv = builtGRPC
		// profile:authn-bearer:start
		wiring.authnStage(authnStageGRPCServerBuilt)
		// profile:authn-bearer:end
	}
	// profile:grpc:end

	superviseReadiness(
		bootstrap.cfg,
		bootstrap.log,
		healthSvc,
		supervisor,
		// profile:grpc:start
		grpcSrv,
		// profile:grpc:end
	)

	var metricsSrv runtimeServer
	if bootstrap.cfg.Observability.Metrics.Addr != "" {
		metricsSrv = newDiagnosticsServer(bootstrap.cfg, metrics, errorLog)
	}

	serveErr := wiring.serve(signalCtx, startupCtx, serveRuntimeArgs{
		cfg:       bootstrap.cfg,
		log:       bootstrap.log,
		healthSvc: healthSvc,
		httpSrv:   srv,
		// profile:grpc:start
		grpcSrv: grpcSrv,
		// profile:grpc:end
		metricsSrv:         metricsSrv,
		backgroundFailures: supervisor.Failures(),
		// Admission refreshes rather than probing separately, so the verdict it
		// admits on is the same one the probe route will serve. Without that, the
		// first probe after admission could still answer 503 from an unevaluated
		// cache and have the instance pulled straight back out of rotation.
		readinessCheck: func(ctx context.Context) error {
			if err := healthSvc.Refresh(ctx, bootstrap.cfg.HTTP.ReadinessTimeout, bootstrap.cfg.Health.FailureThreshold); err != nil {
				return fmt.Errorf("refresh initial readiness: %w", err)
			}
			return nil
		},
		admission:     startupAdmission,
		shutdownDelay: bootstrap.cfg.HTTP.ReadinessPropagationDelay,
		// profile:messaging-nats-jetstream:start
		preDrain: messaging.StartDrain,
		// profile:messaging-nats-jetstream:end
		shutdown: shutdown,
	})
	wiring.lifecycle(runtimeLifecycleHTTPDrained)

	// Ordered teardown. HTTP is already drained, so this is the first moment
	// nothing can still depend on background work: cancel and join it, then
	// release the dependencies it used, both before the deferred telemetry flush
	// so the flush can record how they went.
	backgroundCtx := shutdown.stage(signalCtx, backgroundShutdownTimeout)
	// profile:messaging-nats-jetstream:start
	messagingErr := messaging.Shutdown(backgroundCtx)
	// profile:messaging-nats-jetstream:end
	backgroundErr := supervisor.Shutdown(backgroundCtx)
	wiring.lifecycle(runtimeLifecycleBackgroundJoined)
	// profile:authn-bearer:start
	authnVerifier.Close()
	authnClosed = true
	// profile:authn-bearer:end
	closeOwners()

	return errors.Join(
		runErr,
		serveErr,
		// profile:messaging-nats-jetstream:start
		messagingErr,
		// profile:messaging-nats-jetstream:end
		backgroundErr,
	)
}

// newSupervisedBackground builds the supervisor for runtime background work,
// deliberately detached from the signal context so that Shutdown is the only
// thing that stops it.
//
// Deriving it from signalCtx would cancel every task the instant SIGTERM arrives,
// while the HTTP drain keeps serving for up to 30s with the shipped defaults —
// every request admitted in that window would run without the work it depends on.
func newSupervisedBackground(signalCtx context.Context, log *slog.Logger) *background.Supervisor {
	return background.New(context.WithoutCancel(signalCtx), log)
}
