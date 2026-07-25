package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

// Budget owned by the PostgreSQL startup stage. It lives here so the
// DATABASE=none profile removes it together with this file.
const postgresStartupBudget = 15 * time.Second

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
	})
	if err != nil {
		return nil, fmt.Errorf("%w: postgres init failed: %w", errDependencyInit, err)
	}
	return pg, nil
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
