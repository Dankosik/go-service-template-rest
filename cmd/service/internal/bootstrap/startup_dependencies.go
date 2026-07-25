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

const (
	startupDependencyPostgres     = "postgres"
	startupPostgresProbeOperation = "postgres_probe"
	startupPostgresResolveStage   = "startup.resolve.postgres"
	startupPostgresProbeStage     = "startup.probe.postgres"
)

// Budgets and retry bounds owned by the PostgreSQL startup stage. They live
// here so the DATABASE=none profile removes them together with this file.
const (
	startupReserveBudget  = 3 * time.Second
	postgresStartupBudget = 15 * time.Second

	startupRetryBaseDelay   = 50 * time.Millisecond
	startupRetryMaxDelay    = 250 * time.Millisecond
	postgresStartupAttempts = 2
)

type (
	postgresConnectFunc    func(context.Context, postgres.Options) (*postgres.Pool, error)
	postgresRetryDelayFunc func(int) time.Duration
	postgresRetrySleepFunc func(context.Context, time.Duration) error
)

type postgresStartupRuntime struct {
	tracer        trace.Tracer
	bootstrapSpan trace.Span
	cfg           config.Config
	log           *slog.Logger
}

type runtimeDependencies struct {
	health   *health.Service
	postgres *postgres.Pool
}

func (d runtimeDependencies) Close() {
	d.postgres.Close()
}

func initRuntimeDependencies(
	bootstrapCtx context.Context,
	startupCtx context.Context,
	bootstrap startupBootstrap,
) (runtimeDependencies, error) {
	postgresCtx, postgresCancel := withStageBudget(startupCtx, postgresStartupBudget)
	pg, err := initPostgresDependency(bootstrapCtx, postgresCtx, postgresStartupRuntime{
		tracer:        bootstrap.tracer,
		bootstrapSpan: bootstrap.bootstrapSpan,
		cfg:           bootstrap.cfg,
		log:           bootstrap.log,
	})
	postgresCancel()
	if err != nil {
		return runtimeDependencies{}, err
	}
	return runtimeDependencies{
		health:   health.New(newPostgresReadinessProbe(pg, bootstrap.cfg.Postgres.HealthcheckTimeout)),
		postgres: pg,
	}, nil
}

func initPostgresWithRetry(ctx context.Context, cfg config.PostgresConfig) (*postgres.Pool, error) {
	return initPostgresWithRetryFunc(ctx, cfg, postgres.New, fullJitterDelay, sleepWithContext)
}

func initPostgresWithRetryFunc(
	ctx context.Context,
	cfg config.PostgresConfig,
	connect postgresConnectFunc,
	delayFor postgresRetryDelayFunc,
	sleep postgresRetrySleepFunc,
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

func ensureRemainingStartupBudget(ctx context.Context, minRemaining time.Duration, stage string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %s aborted before probe: %w", errDependencyInit, stage, err)
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return nil
	}
	remaining := time.Until(deadline)
	if remaining < minRemaining {
		return fmt.Errorf(
			"%w: %s aborted due to low remaining startup budget (%s < %s)",
			errDependencyInit,
			stage,
			remaining,
			minRemaining,
		)
	}
	return nil
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

func initPostgresDependency(bootstrapCtx context.Context, dependencyCtx context.Context, runtime postgresStartupRuntime) (*postgres.Pool, error) {
	if !runtime.cfg.Postgres.Enabled {
		return nil, rejectPostgresStartupForDependencyInit(
			bootstrapCtx,
			runtime.bootstrapSpan,
			runtime.log,
			errors.New("postgres is required by the DATABASE=postgres profile"),
		)
	}

	if err := ensureRemainingStartupBudget(
		dependencyCtx,
		startupFailFastThreshold+startupReserveBudget,
		startupPostgresProbeStage,
	); err != nil {
		rejectErr := fmt.Errorf("%s init skipped: %w", startupDependencyPostgres, err)
		recordDependencyProbeRejection(bootstrapCtx, runtime, rejectErr)
		return nil, rejectErr
	}

	probeCtx, probeCancel := withStageBudget(dependencyCtx, postgresProbeBudget)
	probeCtx, probeSpan := runtime.tracer.Start(probeCtx, startupPostgresProbeStage)

	pg, probeErr := initPostgresWithRetry(probeCtx, runtime.cfg.Postgres)
	parentErr := dependencyCtx.Err()
	stageErr := probeCtx.Err()
	if probeErr == nil {
		if parentErr != nil {
			probeErr = parentErr
		} else if stageErr != nil {
			probeErr = stageErr
		}
	}
	probeCancel()

	attrs := []attribute.KeyValue{attribute.String("dep", startupDependencyPostgres)}
	if probeErr != nil {
		probeSpan.RecordError(probeErr)
		attrs = append(attrs, attribute.String("result", "error"))
	} else {
		attrs = append(attrs, attribute.String("result", "success"))
	}
	probeSpan.SetAttributes(attrs...)
	probeSpan.End()

	pgReturned := false
	defer func() {
		if !pgReturned && pg != nil {
			pg.Close()
		}
	}()

	if probeErr != nil {
		sanitizedErr := postgresDependencyInitFailure(probeErr)
		recordDependencyProbeRejection(bootstrapCtx, runtime, sanitizedErr)
		return nil, sanitizedErr
	}

	pgReturned = true
	return pg, nil
}

func postgresDependencyInitFailure(err error) error {
	if err == nil {
		return fmt.Errorf("%w: %s init failed", errDependencyInit, startupDependencyPostgres)
	}
	if errors.Is(err, errDependencyInit) {
		return fmt.Errorf("%s init failed: %w", startupDependencyPostgres, err)
	}
	return fmt.Errorf("%w: %s init failed: %w", errDependencyInit, startupDependencyPostgres, err)
}

func rejectPostgresStartupForDependencyInit(
	ctx context.Context,
	bootstrapSpan trace.Span,
	log *slog.Logger,
	err error,
) error {
	rejectErr := postgresDependencyInitFailure(err)
	recordStartupRejection(bootstrapSpan, "dependency_init", startupPostgresResolveStage, rejectErr)
	log.Error(
		"startup_blocked",
		startupLogArgs(
			ctx,
			startupLogComponentStartupProbes,
			"postgres_config",
			"error",
			"error.type", "dependency_init",
			"dependency", startupDependencyPostgres,
			"err", rejectErr,
		)...,
	)
	return rejectErr
}

func recordDependencyProbeRejection(ctx context.Context, runtime postgresStartupRuntime, err error) {
	recordStartupRejection(runtime.bootstrapSpan, "dependency_init", startupPostgresProbeStage, err)
	runtime.log.Error(
		"startup_blocked",
		startupLogArgs(
			ctx,
			startupLogComponentStartupProbes,
			startupPostgresProbeOperation,
			"error",
			"error.type", "dependency_init",
			"dependency", startupDependencyPostgres,
			"err", err,
		)...,
	)
}
