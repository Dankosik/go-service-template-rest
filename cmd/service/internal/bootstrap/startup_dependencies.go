package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
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

// Budgets owned by the PostgreSQL startup stage. They live here so the
// DATABASE=none profile removes them together with this file.
const (
	postgresStartupBudget = 15 * time.Second

	// postgresProbeBudget bounds the pool open and first ping, and is the
	// ceiling validateStartupBudgetCompatibility enforces on
	// postgres.connect_timeout and postgres.healthcheck_timeout, so a
	// configured timeout can never exceed the stage that runs it.
	postgresProbeBudget = 5 * time.Second

	// startupReadinessHeadroom is the margin required between the health-check
	// budget and http.readiness_timeout, so a probe that spends its whole
	// budget still leaves room to answer the request.
	startupReadinessHeadroom = 150 * time.Millisecond
)

// errDependencyInit classifies a failure to bring up a runtime dependency. It
// belongs to this file because dependencies are what it describes: the
// DATABASE=none profile has none, so its stub declares no sentinel.
var errDependencyInit = errors.New("dependency init")

type postgresStartupRuntime struct {
	tracer        trace.Tracer
	bootstrapSpan trace.Span
	cfg           config.Config
	log           *slog.Logger
}

type runtimeDependencies struct {
	health   *health.Service
	postgres *postgres.Pool
	closed   *sync.Once
}

// Close releases pooled dependencies, bounded by ctx, and is safe to call twice.
//
// The bound is the point. pgxpool.Close blocks until every acquired connection is
// returned and destroyed and accepts no context of its own, so a handler that
// outlived the HTTP drain while holding a connection would park the process here
// until the platform SIGKILLs it — taking the shutdown telemetry with it. A
// connection still held at this point is a leaked handler, not a slow close, and
// reporting it beats waiting for it.
func (d runtimeDependencies) Close(ctx context.Context) {
	if d.closed == nil {
		d.postgres.Close()
		return
	}

	d.closed.Do(func() {
		done := make(chan struct{})
		go func() {
			defer close(done)
			d.postgres.Close()
		}()

		select {
		case <-done:
		case <-ctx.Done():
		}
	})
}

// dependencyCloseContext bounds a dependency release with its own budget,
// detached from the signal context that is already canceled by this point.
func dependencyCloseContext(base context.Context) context.Context {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(base), dependencyCloseTimeout)
	// The cancel is deliberately not deferred to the caller: the context is
	// consumed synchronously by Close, and releasing the timer immediately after
	// would defeat the bound.
	context.AfterFunc(ctx, cancel)
	return ctx
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
		closed:   new(sync.Once),
	}, nil
}

// initPostgres opens the pool once. There is deliberately no retry loop here:
// postgres.ConnectTimeout already bounds a slow dependency, and a bounded
// in-process retry cannot survive the failure it would be for — a database
// restart takes seconds to minutes, far beyond any startup budget. Restarting
// the process is the platform's job, and every supported deployment target
// already has a restart policy for it.
func initPostgres(ctx context.Context, cfg config.PostgresConfig) (*postgres.Pool, error) {
	pg, err := postgres.New(ctx, postgres.Options{
		DSN:                cfg.DSN,
		ConnectTimeout:     cfg.ConnectTimeout,
		HealthcheckTimeout: cfg.HealthcheckTimeout,
		MaxOpenConns:       cfg.MaxOpenConns,
		ConnMaxLifetime:    cfg.ConnMaxLifetime,
		StatementTimeout:   cfg.StatementTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: postgres init failed: %w", errDependencyInit, err)
	}
	return pg, nil
}

func validateStartupBudgetCompatibility(cfg config.Config) error {
	if cfg.Postgres.Enabled {
		if err := validateStartupTimeoutBudget("postgres.connect_timeout", cfg.Postgres.ConnectTimeout, postgresProbeBudget); err != nil {
			return err
		}
		if err := validateStartupTimeoutBudget("postgres.healthcheck_timeout", cfg.Postgres.HealthcheckTimeout, postgresProbeBudget); err != nil {
			return err
		}
	}
	return validateStartupReadinessHeadroom(cfg)
}

func validateStartupTimeoutBudget(name string, value time.Duration, budget time.Duration) error {
	if value <= budget {
		return nil
	}
	return fmt.Errorf(
		"%w: %s must be <= startup probe budget %s",
		config.ErrValidate,
		name,
		budget,
	)
}

func validateStartupReadinessHeadroom(cfg config.Config) error {
	if !cfg.Postgres.Enabled {
		return nil
	}

	required := cfg.Postgres.HealthcheckTimeout + startupReadinessHeadroom
	if cfg.HTTP.ReadinessTimeout >= required {
		return nil
	}
	return fmt.Errorf(
		"%w: http.readiness_timeout must be >= postgres.healthcheck_timeout readiness budget plus startup headroom (%s + %s = %s)",
		config.ErrValidate,
		cfg.Postgres.HealthcheckTimeout,
		startupReadinessHeadroom,
		required,
	)
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

	probeCtx, probeCancel := withStageBudget(dependencyCtx, postgresProbeBudget)
	probeCtx, probeSpan := runtime.tracer.Start(probeCtx, startupPostgresProbeStage)

	pg, probeErr := initPostgres(probeCtx, runtime.cfg.Postgres)
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
