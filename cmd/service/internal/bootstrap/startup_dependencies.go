package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var errDependencyInit = errors.New("dependency init")

const (
	startupDependencyPostgres     = "postgres"
	startupPostgresProbeOperation = "postgres_probe"
	startupPostgresResolveStage   = "startup.resolve.postgres"
	startupPostgresProbeStage     = "startup.probe.postgres"
)

const (
	startupDependencyTelemetry       = "telemetry"
	startupDependencyNetworkPolicy   = "network_policy"
	startupDependencyIngressPolicy   = "ingress_policy"
	startupDependencyMetricsExposure = "metrics_exposure"
	startupDependencyEgressException = "egress_exception"
	startupDependencyModeFeatureOff  = "feature_off"
)

const (
	startupLogComponentStartupProbes = "startup_probes"
	startupLogComponentShutdown      = "shutdown"

	startupOperationTelemetryInit  = "telemetry_init"
	startupOperationTelemetryFlush = "telemetry_flush"
)

type (
	postgresConnectFunc func(context.Context, postgres.Options) (*postgres.Pool, error)
	startupDelayFunc    func(int) time.Duration
	startupSleepFunc    func(context.Context, time.Duration) error
)

type dependencyProbeRuntime struct {
	tracer        trace.Tracer
	bootstrapSpan trace.Span
	cfg           config.Config
	log           *slog.Logger
	networkPolicy networkPolicy
}

func postgresStartupProbeAddress(cfg config.PostgresConfig) (string, error) {
	address, err := postgres.ProbeAddress(cfg.DSN)
	if err != nil {
		return "", fmt.Errorf("%w: resolve postgres probe address: %w", errDependencyInit, err)
	}
	return address, nil
}

func initPostgresWithRetry(ctx context.Context, cfg config.PostgresConfig) (*postgres.Pool, error) {
	return initPostgresWithRetryFunc(ctx, cfg, postgres.New, fullJitterDelay, sleepWithContext)
}

func initPostgresWithRetryFunc(
	ctx context.Context,
	cfg config.PostgresConfig,
	connect postgresConnectFunc,
	delayFor startupDelayFunc,
	sleep startupSleepFunc,
) (*postgres.Pool, error) {
	options := postgres.Options{
		DSN:                cfg.DSN,
		ConnectTimeout:     cfg.ConnectTimeout,
		HealthcheckTimeout: cfg.HealthcheckTimeout,
		MaxOpenConns:       cfg.MaxOpenConns,
		ConnMaxLifetime:    cfg.ConnMaxLifetime,
	}

	var lastErr error
	for attempt := 1; attempt <= postgresStartupAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("%w: postgres init canceled: %w", errDependencyInit, err)
		}

		pg, err := connect(ctx, options)
		if err == nil {
			return pg, nil
		}

		lastErr = err
		if !shouldRetryPostgresStartup(err, attempt) {
			break
		}

		delay := delayFor(attempt)
		if err := sleep(ctx, delay); err != nil {
			return nil, fmt.Errorf("%w: postgres retry wait canceled: %w", errDependencyInit, err)
		}
	}

	return nil, fmt.Errorf("%w: postgres init failed after retries: %w", errDependencyInit, lastErr)
}

func shouldRetryPostgresStartup(err error, attempt int) bool {
	if attempt >= postgresStartupAttempts {
		return false
	}
	return errors.Is(err, postgres.ErrConnect) || errors.Is(err, postgres.ErrHealthcheck)
}

func fullJitterDelay(attempt int) time.Duration {
	backoff := startupRetryBaseDelay << (attempt - 1)
	backoff = min(backoff, startupRetryMaxDelay)
	if backoff <= 0 {
		return 0
	}

	return rand.N(backoff + 1) // #nosec G404 -- startup retry jitter is not security-sensitive.
}

type dependencyProbeOutcome struct {
	probes       []health.Probe
	postgresPool *postgres.Pool
}

type postgresReadinessProbe struct {
	probe  health.Probe
	budget time.Duration
}

func newPostgresReadinessProbe(probe health.Probe, budget time.Duration) postgresReadinessProbe {
	return postgresReadinessProbe{probe: probe, budget: budget}
}

func (p postgresReadinessProbe) Name() string {
	return p.probe.Name()
}

func (p postgresReadinessProbe) Check(ctx context.Context) error {
	probeCtx, probeCancel := withStageBudget(ctx, p.budget)
	defer probeCancel()
	if err := p.probe.Check(probeCtx); err != nil {
		return fmt.Errorf("postgres readiness probe: %w", err)
	}
	if err := probeCtx.Err(); err != nil {
		return fmt.Errorf("postgres readiness probe context: %w", err)
	}
	return nil
}

type dependencyProbeSpec struct {
	stage        string
	dep          string
	budget       time.Duration
	minRemaining time.Duration
	probe        func(context.Context) error
}

type probeExecutionResult struct {
	budgetBlocked bool
	parentErr     error
	err           error
}

func initStartupDependencies(startupCtx context.Context, bootstrapCtx context.Context, runtime dependencyProbeRuntime) (dependencyProbeOutcome, error) {
	dependencyProbeCtx, dependencyProbeCancel := withStageBudget(startupCtx, startupProbeBudget)
	defer dependencyProbeCancel()

	outcome := dependencyProbeOutcome{probes: make([]health.Probe, 0, 1)}
	pg, err := initPostgresDependency(bootstrapCtx, dependencyProbeCtx, runtime)
	if err != nil {
		return outcome, err
	}
	if pg != nil {
		outcome.postgresPool = pg
		outcome.probes = append(outcome.probes, newPostgresReadinessProbe(pg, runtime.cfg.Postgres.HealthcheckTimeout))
	}

	return outcome, nil
}

func initPostgresDependency(bootstrapCtx context.Context, dependencyProbeCtx context.Context, runtime dependencyProbeRuntime) (*postgres.Pool, error) {
	if !runtime.cfg.Postgres.Enabled {
		return nil, nil //nolint:nilnil // Disabled dependency intentionally has no pool and no startup error.
	}

	postgresProbeAddress, addressErr := postgresStartupProbeAddress(runtime.cfg.Postgres)
	if addressErr != nil {
		return nil, rejectStartupForDependencyInit(
			bootstrapCtx,
			runtime.bootstrapSpan,
			runtime.log,
			startupDependencyPostgres,
			startupPostgresResolveStage,
			addressErr,
		)
	}
	if err := runtime.networkPolicy.EnforceEgressTarget(postgresProbeAddress, "tcp"); err != nil {
		return nil, rejectStartupForPolicyViolation(
			bootstrapCtx,
			runtime.bootstrapSpan,
			runtime.log,
			startupDependencyPostgres,
			err,
		)
	}

	var pg *postgres.Pool
	pgReturned := false
	defer func() {
		if !pgReturned && pg != nil {
			pg.Close()
		}
	}()

	probeResult := runDependencyProbe(dependencyProbeCtx, runtime.tracer, dependencyProbeSpec{
		stage:        startupPostgresProbeStage,
		dep:          startupDependencyPostgres,
		budget:       postgresProbeBudget,
		minRemaining: startupFailFastThreshold + startupReserveBudget,
		probe: func(probeCtx context.Context) error {
			var err error
			pg, err = initPostgresWithRetry(probeCtx, runtime.cfg.Postgres)
			return err
		},
	})
	if probeResult.err != nil {
		if probeResult.budgetBlocked {
			rejectErr := dependencyInitAbortFailure(startupDependencyPostgres, probeResult)
			recordDependencyProbeRejection(bootstrapCtx, runtime, rejectErr)
			return nil, rejectErr
		}

		sanitizedErr := dependencyInitFailure(startupDependencyPostgres, probeResult.err)
		recordDependencyProbeRejection(bootstrapCtx, runtime, sanitizedErr)
		return nil, sanitizedErr
	}

	pgReturned = true
	return pg, nil
}

func runDependencyProbe(dependencyProbeCtx context.Context, tracer trace.Tracer, spec dependencyProbeSpec) probeExecutionResult {
	if err := ensureRemainingStartupBudget(dependencyProbeCtx, spec.minRemaining, spec.stage); err != nil {
		return probeExecutionResult{budgetBlocked: true, parentErr: dependencyProbeCtx.Err(), err: err}
	}

	probeCtx, probeCancel := withStageBudget(dependencyProbeCtx, spec.budget)
	probeCtx, probeSpan := tracer.Start(probeCtx, spec.stage)
	err := spec.probe(probeCtx)
	parentErr := dependencyProbeCtx.Err()
	stageErr := probeCtx.Err()
	if err == nil {
		if parentErr != nil {
			err = parentErr
		} else if stageErr != nil {
			err = stageErr
		}
	}
	probeCancel()

	attrs := []attribute.KeyValue{attribute.String("dep", spec.dep)}
	if err != nil {
		probeSpan.RecordError(err)
		attrs = append(attrs, attribute.String("result", "error"))
	} else {
		attrs = append(attrs, attribute.String("result", "success"))
	}
	probeSpan.SetAttributes(attrs...)
	probeSpan.End()

	return probeExecutionResult{budgetBlocked: false, parentErr: parentErr, err: err}
}

func dependencyInitFailure(dep string, err error) error {
	if err == nil {
		return fmt.Errorf("%w: %s init failed", errDependencyInit, dep)
	}
	if errors.Is(err, errDependencyInit) {
		return fmt.Errorf("%s init failed: %w", dep, err)
	}
	return fmt.Errorf("%w: %s init failed: %w", errDependencyInit, dep, err)
}

func dependencyInitAbortFailure(dep string, result probeExecutionResult) error {
	if result.budgetBlocked {
		if errors.Is(result.err, errDependencyInit) {
			return fmt.Errorf("%s init skipped: %w", dep, result.err)
		}
		return fmt.Errorf("%w: %s init skipped: %w", errDependencyInit, dep, result.err)
	}
	return dependencyInitFailure(dep, result.err)
}
