package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	startupDependencyPostgres     = "postgres"
	startupPostgresProbeOperation = "postgres_probe"
)

// Budgets owned by the PostgreSQL startup stage. They live here so the
// DATABASE=none profile removes them together with this file.
const (
	postgresStartupBudget = 15 * time.Second

	// postgresProbeBudget bounds the pool open and first ping, and is the
	// ceiling around the adapter's fixed connection and ping budgets.
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
	cfg config.Config
	log *slog.Logger
}

type runtimeDependencies struct {
	readiness health.Probe
	postgres  *pgxpool.Pool
	closed    *sync.Once
}

// ReadinessProbes returns what this profile's dependencies contribute to the
// readiness verdict.
//
// The health service is built by the caller rather than here, because the
// supervisor is a probe too and it does not exist yet at this point in startup;
// see run.go. Returning probes instead of a service is what lets both profiles
// share that composition.
func (d runtimeDependencies) ReadinessProbes() []health.Probe {
	if d.readiness == nil {
		return nil
	}
	return []health.Probe{d.readiness}
}

// readinessProbeBudget is how long one steady-state readiness evaluation may
// take. It covers the complete serial probe set, so the configured aggregate
// budget remains the owner when optional probes are added.
//
// It is passed to health.Watch separately from the refresh interval. The two used
// to be one argument, which clamped this budget to the interval: with the shipped
// defaults a 3s PostgreSQL probe became 2s in steady state while startup
// admission still granted the full budget, so a database answering in between
// passed admission and then flapped out of rotation.
func readinessProbeBudget(cfg config.Config) time.Duration {
	return cfg.HTTP.ReadinessTimeout
}

// Close releases pooled dependencies, bounded by ctx, and is safe to call twice.
//
// The bound is the point. pgxpool.Close blocks until every acquired connection is
// returned and destroyed and accepts no context of its own, so a handler that
// outlived the HTTP drain while holding a connection would park the process here
// until the platform SIGKILLs it — taking the shutdown telemetry with it. A
// connection still held at this point is a leaked handler, not a slow close, and
// reporting it beats waiting for it. If ctx wins, the close goroutine may still
// be blocked; returning is safe only because this process exits next and never
// reuses the pool.
func (d runtimeDependencies) Close(ctx context.Context) {
	if d.postgres == nil {
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

func initRuntimeDependencies(
	startupCtx context.Context,
	bootstrap startupBootstrap,
) (runtimeDependencies, error) {
	postgresCtx, postgresCancel := withStageBudget(startupCtx, postgresStartupBudget)
	pg, err := initPostgresDependency(startupCtx, postgresCtx, postgresStartupRuntime{
		cfg: bootstrap.cfg,
		log: bootstrap.log,
	})
	postgresCancel()
	if err != nil {
		return runtimeDependencies{}, err
	}

	return runtimeDependencies{
		readiness: newPostgresReadinessProbe(pg, postgres.DefaultHealthcheckTimeout),
		postgres:  pg,
		closed:    new(sync.Once),
	}, nil
}

// initPostgres opens the pool once. There is deliberately no retry loop here:
// the shared connection timeout already bounds a slow dependency, and a bounded
// in-process retry cannot survive the failure it would be for — a database
// restart takes seconds to minutes, far beyond any startup budget. Restarting
// the process is the platform's job, and every supported deployment target
// already has a restart policy for it.
func initPostgres(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	pg, err := postgres.Open(ctx, runtimeopts.Postgres(cfg))
	if err != nil {
		return nil, fmt.Errorf("%w: postgres init failed: %w", errDependencyInit, err)
	}
	return pg, nil
}

func validateStartupBudgetCompatibility(cfg config.Config) error {
	return validateStartupReadinessHeadroom(cfg)
}

func validateStartupReadinessHeadroom(cfg config.Config) error {
	if !cfg.Postgres.Enabled {
		return nil
	}

	required := postgres.DefaultHealthcheckTimeout + startupReadinessHeadroom
	if cfg.HTTP.ReadinessTimeout >= required {
		return nil
	}
	return fmt.Errorf(
		"%w: http.readiness_timeout must be >= postgres readiness budget plus startup headroom (%s + %s = %s)",
		config.ErrValidate,
		postgres.DefaultHealthcheckTimeout,
		startupReadinessHeadroom,
		required,
	)
}

type postgresPinger interface {
	Ping(ctx context.Context) error
}

type postgresReadinessProbe struct {
	pool   postgresPinger
	budget time.Duration
}

func newPostgresReadinessProbe(pool postgresPinger, budget time.Duration) postgresReadinessProbe {
	return postgresReadinessProbe{pool: pool, budget: budget}
}

func (p postgresReadinessProbe) Name() string {
	return startupDependencyPostgres
}

func (p postgresReadinessProbe) Check(ctx context.Context) error {
	if p.pool == nil {
		return fmt.Errorf("postgres readiness probe: %w", postgres.ErrHealthcheck)
	}
	probeCtx, probeCancel := withStageBudget(ctx, p.budget)
	defer probeCancel()
	if err := p.pool.Ping(probeCtx); err != nil {
		return fmt.Errorf("postgres readiness probe: %w", err)
	}
	if err := probeCtx.Err(); err != nil {
		return fmt.Errorf("postgres readiness probe context: %w", err)
	}
	return nil
}

func initPostgresDependency(bootstrapCtx context.Context, dependencyCtx context.Context, runtime postgresStartupRuntime) (*pgxpool.Pool, error) {
	if !runtime.cfg.Postgres.Enabled {
		return nil, rejectPostgresStartupForDependencyInit(
			bootstrapCtx,
			runtime.log,
			errors.New("postgres is required by the DATABASE=postgres profile"),
		)
	}

	probeCtx, probeCancel := withStageBudget(dependencyCtx, postgresProbeBudget)
	probeStarted := time.Now()

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
	probeDuration := time.Since(probeStarted)

	pgReturned := false
	defer func() {
		if !pgReturned && pg != nil {
			pg.Close()
		}
	}()

	if probeErr != nil {
		sanitizedErr := postgresDependencyInitFailure(probeErr)
		recordDependencyProbeRejection(bootstrapCtx, runtime, probeDuration, sanitizedErr)
		return nil, sanitizedErr
	}

	// The probe duration is reported here because nothing else measures how long
	// a dependency took to become usable, and a startup that is slow rather than
	// broken is otherwise indistinguishable from one that is merely starting.
	runtime.log.InfoContext(
		bootstrapCtx,
		"startup_dependency_ready",
		startupLogArgs(
			startupLogComponentStartupProbes,
			startupPostgresProbeOperation,
			"success",
			"dependency", startupDependencyPostgres,
			"probe.duration_ms", probeDuration.Milliseconds(),
		)...,
	)

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
	log *slog.Logger,
	err error,
) error {
	rejectErr := postgresDependencyInitFailure(err)
	log.ErrorContext(
		ctx,
		"startup_blocked",
		startupLogArgs(
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

func recordDependencyProbeRejection(
	ctx context.Context,
	runtime postgresStartupRuntime,
	probeDuration time.Duration,
	err error,
) {
	runtime.log.ErrorContext(
		ctx,
		"startup_blocked",
		startupLogArgs(
			startupLogComponentStartupProbes,
			startupPostgresProbeOperation,
			"error",
			"error.type", "dependency_init",
			"dependency", startupDependencyPostgres,
			"probe.duration_ms", probeDuration.Milliseconds(),
			"err", err,
		)...,
	)
}
