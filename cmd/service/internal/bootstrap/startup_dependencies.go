package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/example/go-service-template-rest/internal/app/health"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type dependencyProbeRuntime struct {
	tracer        trace.Tracer
	bootstrapSpan trace.Span
	cfg           config.Config
	log           *slog.Logger
	networkPolicy networkPolicy
}

type dependencyProbeOutcome struct {
	probes       []health.Probe
	postgresPool *postgres.Pool
}

type dependencyCleanupStack struct {
	cleanups []func()
}

func (s *dependencyCleanupStack) add(cleanup func()) {
	if cleanup == nil {
		return
	}
	s.cleanups = append(s.cleanups, cleanup)
}

func (s *dependencyCleanupStack) run() {
	for _, cleanup := range slices.Backward(s.cleanups) {
		cleanup()
	}
	s.cleanups = nil
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

func initStartupDependencies(startupCtx context.Context, bootstrapCtx context.Context, runtime dependencyProbeRuntime) (outcome dependencyProbeOutcome, err error) {
	dependencyProbeCtx, dependencyProbeCancel := withStageBudget(startupCtx, startupProbeBudget)
	defer dependencyProbeCancel()

	outcome = dependencyProbeOutcome{probes: make([]health.Probe, 0, 1)}
	cleanupStack := dependencyCleanupStack{}
	defer func() {
		if err != nil {
			cleanupStack.run()
		}
	}()

	pg, err := initPostgresDependency(bootstrapCtx, dependencyProbeCtx, runtime)
	if err != nil {
		return outcome, err
	}
	if pg != nil {
		outcome.postgresPool = pg
		cleanupStack.add(pg.Close)
		if runtime.cfg.PostgresReadinessProbeRequired() {
			outcome.probes = append(outcome.probes, newPostgresReadinessProbe(pg, runtime.cfg.Postgres.HealthcheckTimeout))
		}
	}

	return outcome, nil
}

func initPostgresDependency(bootstrapCtx context.Context, dependencyProbeCtx context.Context, runtime dependencyProbeRuntime) (*postgres.Pool, error) {
	labels := startupPostgresDependencyLabels
	if !runtime.cfg.Postgres.Enabled {
		return nil, nil //nolint:nilnil // Disabled dependency intentionally has no pool and no startup error.
	}

	postgresProbeAddress, addressErr := postgresStartupProbeAddress(runtime.cfg.Postgres)
	if addressErr != nil {
		return nil, rejectStartupForDependencyInit(
			bootstrapCtx,
			runtime.bootstrapSpan,
			runtime.log,
			labels.dependency,
			labels.resolveStage,
			addressErr,
		)
	}
	if err := runtime.networkPolicy.EnforceEgressTarget(postgresProbeAddress, "tcp"); err != nil {
		return nil, rejectStartupForPolicyViolation(
			bootstrapCtx,
			runtime.bootstrapSpan,
			runtime.log,
			labels.dependency,
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
		stage:        labels.probeStage,
		dep:          labels.dependency,
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
			rejectErr := dependencyInitAbortFailure(labels.dependency, probeResult)
			recordDependencyProbeRejection(
				bootstrapCtx,
				runtime,
				labels,
				"",
				rejectErr,
			)
			return nil, rejectErr
		}

		sanitizedErr := dependencyInitFailure(labels.dependency, probeResult.err)
		recordDependencyProbeRejection(
			bootstrapCtx,
			runtime,
			labels,
			"",
			sanitizedErr,
		)
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
