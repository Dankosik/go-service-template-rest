package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/example/go-service-template-rest/internal/background"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/problem"
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
	cfg config.Config
	log *slog.Logger
}

type runtimeDependencies struct {
	health      *health.Service
	postgres    *postgres.Pool
	idempotency *postgres.IdempotencyStore
	// idempotencySweepInterval is held rather than re-read from config, so the
	// supervised task and the store it prunes cannot be configured apart.
	idempotencySweepInterval time.Duration
	closed                   *sync.Once
}

// IdempotencyStore returns the store the HTTP chain replays completed attempts
// from, or nil when this profile has none.
//
// The explicit nil check is load-bearing. Returning a nil *postgres.IdempotencyStore
// as the interface would produce a non-nil interface holding a nil pointer, so the
// middleware would install itself and dereference nothing on the first POST
// carrying a key.
func (d runtimeDependencies) IdempotencyStore() httpx.IdempotencyStore {
	if d.idempotency == nil {
		return nil
	}
	return d.idempotency
}

// BackgroundTasks returns the supervised work this profile's dependencies need.
//
// It exists so run.go stays profile-independent: the DATABASE=none profile returns
// nothing and the shared startup path is unchanged.
func (d runtimeDependencies) BackgroundTasks() []background.Task {
	if d.idempotency == nil {
		return nil
	}

	store := d.idempotency
	interval := d.idempotencySweepInterval
	return []background.Task{{
		Name: "idempotency_sweep",
		Run: func(ctx context.Context) error {
			return sweepIdempotencyKeys(ctx, store, interval)
		},
	}}
}

// postgresSaturationRetryAfter is the hint on a request refused because every
// pooled connection was busy. It matches the load-shedding hint for the same
// reason: saturation means momentarily past capacity, not down, and a long hint
// turns a brief spike into a client-side outage.
const postgresSaturationRetryAfter = time.Second

// DomainErrors classifies the dependency failures a handler can surface but
// should not each have to translate.
//
// postgres.ErrSaturated is the one this profile owns, and the pool package has
// documented this exact mapping since it was written without anything
// implementing it. It is the database failure that is not the database's fault:
// every connection is busy serving, so the caller should back off and retry.
// Answering 500 instead told a client library not to retry the one failure that
// retrying fixes, and buried a moment of capacity pressure in the same error rate
// as a genuine bug.
func (d runtimeDependencies) DomainErrors() []problem.Mapper {
	return []problem.Mapper{classifyPostgresDomainError}
}

func classifyPostgresDomainError(err error) (problem.Mapped, bool) {
	if !errors.Is(err, postgres.ErrSaturated) {
		return problem.Mapped{}, false
	}
	return problem.Mapped{
		Code:       problem.CodeServiceUnavailable,
		Detail:     "the service is temporarily at capacity",
		RetryAfter: postgresSaturationRetryAfter,
	}, true
}

// idempotencySweeper is the one method the supervised task needs. Narrowing the
// parameter to it is what lets the task's timing and failure handling be proved
// without a database.
type idempotencySweeper interface {
	Sweep(ctx context.Context) (int64, error)
}

// sweepIdempotencyKeys deletes expired reservations until ctx is done.
//
// It returns nil on cancellation so the supervisor treats a drain as an ordinary
// stop, and it reports a failed sweep without returning: one failed pass is a
// transient database problem, and taking the process down for it would trade a
// slowly growing table for an outage. A pass that keeps failing shows up as a
// repeated error record rather than as silence.
func sweepIdempotencyKeys(ctx context.Context, store idempotencySweeper, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			removed, err := store.Sweep(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				slog.ErrorContext(ctx, "idempotency_sweep_failed", "component", "idempotency", "err", err)
				continue
			}
			if removed > 0 {
				slog.InfoContext(ctx, "idempotency_sweep_completed", "component", "idempotency", "removed", removed)
			}
		}
	}
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

	idempotencyStore, err := postgres.NewIdempotencyStore(pg, bootstrap.cfg.Postgres.IdempotencyRetention)
	if err != nil {
		pg.Close()
		return runtimeDependencies{}, fmt.Errorf("%w: idempotency store init failed: %w", errDependencyInit, err)
	}

	return runtimeDependencies{
		health:                   health.New(newPostgresReadinessProbe(pg, bootstrap.cfg.Postgres.HealthcheckTimeout)),
		postgres:                 pg,
		idempotency:              idempotencyStore,
		idempotencySweepInterval: bootstrap.cfg.Postgres.IdempotencySweepInterval,
		closed:                   new(sync.Once),
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
		MinIdleConns:       cfg.MinIdleConns,
		AcquireTimeout:     cfg.AcquireTimeout,
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
