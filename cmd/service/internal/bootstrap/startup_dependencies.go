package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/app/health"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type dependencyProbeRuntime struct {
	tracer        trace.Tracer
	bootstrapSpan trace.Span
	cfg           config.Config
	metrics       *telemetry.Metrics
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
	for i := len(s.cleanups) - 1; i >= 0; i-- {
		s.cleanups[i]()
	}
	s.cleanups = nil
}

type startupNamedProbe struct {
	name  string
	check func(context.Context) error
}

func (p startupNamedProbe) Name() string {
	return p.name
}

func (p startupNamedProbe) Check(ctx context.Context) error {
	return p.check(ctx)
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
		return err
	}
	return probeCtx.Err()
}

type dependencyProbeSpec struct {
	stage        string
	dep          string
	mode         string
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

	pg, err := initPostgresDependency(bootstrapCtx, runtime, dependencyProbeCtx)
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

	redisProbe, err := initRedisDependency(bootstrapCtx, runtime, dependencyProbeCtx)
	if err != nil {
		return outcome, err
	}
	if redisProbe != nil {
		outcome.probes = append(outcome.probes, redisProbe)
	}

	mongoProbe, err := initMongoDependency(bootstrapCtx, runtime, dependencyProbeCtx)
	if err != nil {
		return outcome, err
	}
	if mongoProbe != nil {
		outcome.probes = append(outcome.probes, mongoProbe)
	}

	return outcome, nil
}

func initPostgresDependency(bootstrapCtx context.Context, runtime dependencyProbeRuntime, dependencyProbeCtx context.Context) (*postgres.Pool, error) {
	labels := startupPostgresDependencyLabels
	if !runtime.cfg.Postgres.Enabled {
		runtime.metrics.MarkStartupDependencyReady(labels.dependency, startupDependencyModeDisabled)
		return nil, nil
	}

	runtime.metrics.MarkStartupDependencyBlocked(labels.dependency, startupDependencyModeCriticalFailClosed)
	postgresProbeAddress, addressErr := postgresStartupProbeAddress(runtime.cfg.Postgres)
	if addressErr != nil {
		return nil, rejectStartupForDependencyInit(
			bootstrapCtx,
			runtime.bootstrapSpan,
			runtime.metrics,
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
			runtime.metrics,
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
		runtime.metrics.MarkStartupDependencyBlocked(labels.dependency, startupDependencyModeCriticalFailClosed)
		recordDependencyProbeRejection(
			bootstrapCtx,
			runtime,
			labels,
			"",
			sanitizedErr,
		)
		return nil, sanitizedErr
	}

	runtime.metrics.MarkStartupDependencyReady(labels.dependency, startupDependencyModeCriticalFailClosed)
	pgReturned = true
	return pg, nil
}

func initRedisDependency(bootstrapCtx context.Context, runtime dependencyProbeRuntime, dependencyProbeCtx context.Context) (health.Probe, error) {
	labels := startupRedisDependencyLabels
	if !runtime.cfg.Redis.Enabled {
		runtime.metrics.MarkStartupDependencyReady(labels.dependency, startupDependencyModeDisabled)
		return nil, nil
	}

	redisMode := runtime.cfg.Redis.ModeValue()
	redisCriticality := startupDependencyModeOptionalFailOpen
	if redisMode == config.RedisModeStore {
		redisCriticality = startupDependencyModeCriticalFailClosed
	}
	runtime.metrics.MarkStartupDependencyBlocked(labels.dependency, redisCriticality)
	if redisMode == config.RedisModeCache {
		runtime.metrics.MarkStartupDependencyBlocked(labels.dependency, startupDependencyModeFeatureOff)
	}

	redisProbeAddress, addressErr := redisStartupProbeAddress(runtime.cfg.Redis)
	if addressErr != nil {
		return nil, rejectStartupForDependencyInit(
			bootstrapCtx,
			runtime.bootstrapSpan,
			runtime.metrics,
			runtime.log,
			labels.dependency,
			labels.resolveStage,
			addressErr,
		)
	}
	if err := runtime.networkPolicy.EnforceEgressTarget(redisProbeAddress, "tcp"); err != nil {
		return nil, rejectStartupForPolicyViolation(
			bootstrapCtx,
			runtime.bootstrapSpan,
			runtime.metrics,
			runtime.log,
			labels.dependency,
			err,
		)
	}

	probeResult := runDependencyProbe(dependencyProbeCtx, runtime.tracer, dependencyProbeSpec{
		stage:        labels.probeStage,
		dep:          labels.dependency,
		mode:         redisMode,
		budget:       redisProbeBudget,
		minRemaining: startupFailFastThreshold,
		probe: func(probeCtx context.Context) error {
			if redisMode == config.RedisModeStore {
				return probeRedisWithRetry(probeCtx, runtime.cfg.Redis)
			}
			return probeRedisWithContext(probeCtx, runtime.cfg.Redis)
		},
	})
	if probeResult.err != nil {
		if err := handleDegradedDependencyProbeFailure(
			bootstrapCtx,
			runtime,
			labels,
			probeResult,
			runtime.cfg.RedisReadinessProbeRequired(),
			redisMode,
			startupDependencyModeFeatureOff,
		); err != nil {
			return nil, err
		}
		return nil, nil
	}

	runtime.metrics.MarkStartupDependencyReady(labels.dependency, redisCriticality)
	if redisMode == config.RedisModeCache {
		runtime.metrics.MarkStartupDependencyBlocked(labels.dependency, startupDependencyModeFeatureOff)
	}
	if runtime.cfg.RedisReadinessProbeRequired() {
		return startupNamedProbe{
			name: labels.dependency,
			check: func(ctx context.Context) error {
				return probeRedisWithContext(ctx, runtime.cfg.Redis)
			},
		}, nil
	}
	return nil, nil
}

func initMongoDependency(bootstrapCtx context.Context, runtime dependencyProbeRuntime, dependencyProbeCtx context.Context) (health.Probe, error) {
	labels := startupMongoDependencyLabels
	if !runtime.cfg.Mongo.Enabled {
		runtime.metrics.MarkStartupDependencyReady(labels.dependency, startupDependencyModeDisabled)
		return nil, nil
	}

	runtime.metrics.MarkStartupDependencyBlocked(labels.dependency, startupDependencyModeCriticalFailDegraded)
	mongoProbeAddress, addressErr := mongoStartupProbeAddress(runtime.cfg.Mongo)
	if addressErr != nil {
		return nil, rejectStartupForDependencyInit(
			bootstrapCtx,
			runtime.bootstrapSpan,
			runtime.metrics,
			runtime.log,
			labels.dependency,
			labels.resolveStage,
			addressErr,
		)
	}
	if err := runtime.networkPolicy.EnforceEgressTarget(mongoProbeAddress, "tcp"); err != nil {
		return nil, rejectStartupForPolicyViolation(
			bootstrapCtx,
			runtime.bootstrapSpan,
			runtime.metrics,
			runtime.log,
			labels.dependency,
			err,
		)
	}

	probeResult := runDependencyProbe(dependencyProbeCtx, runtime.tracer, dependencyProbeSpec{
		stage:        labels.probeStage,
		dep:          labels.dependency,
		budget:       mongoProbeBudget,
		minRemaining: startupFailFastThreshold,
		probe: func(probeCtx context.Context) error {
			return probeMongoWithRetry(probeCtx, runtime.cfg.Mongo)
		},
	})
	if probeResult.err != nil {
		if err := handleDegradedDependencyProbeFailure(
			bootstrapCtx,
			runtime,
			labels,
			probeResult,
			runtime.cfg.MongoReadinessProbeRequired(),
			startupDependencyModeDegradedReadOnlyOrStale,
			startupDependencyModeDegradedReadOnlyOrStale,
		); err != nil {
			return nil, err
		}
		return nil, nil
	}

	runtime.metrics.MarkStartupDependencyReady(labels.dependency, startupDependencyModeCriticalFailDegraded)
	if runtime.cfg.MongoReadinessProbeRequired() {
		return startupNamedProbe{
			name: labels.dependency,
			check: func(ctx context.Context) error {
				return probeMongoWithContext(ctx, runtime.cfg.Mongo)
			},
		}, nil
	}
	return nil, nil
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
	if mode := strings.TrimSpace(spec.mode); mode != "" {
		attrs = append(attrs, attribute.String("mode", mode))
	}
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

func shouldAbortDegradedDependencyStartup(result probeExecutionResult) bool {
	if result.budgetBlocked {
		return true
	}
	return result.parentErr != nil
}

func handleDegradedDependencyProbeFailure(
	ctx context.Context,
	runtime dependencyProbeRuntime,
	labels startupDependencyProbeLabels,
	result probeExecutionResult,
	readinessRequired bool,
	rejectionMode string,
	degradedMode string,
) error {
	if shouldAbortDegradedDependencyStartup(result) {
		rejectErr := dependencyInitAbortFailure(labels.dependency, result)
		recordDependencyProbeRejection(ctx, runtime, labels, rejectionMode, rejectErr)
		return rejectErr
	}
	if readinessRequired {
		rejectErr := dependencyInitFailure(labels.dependency, result.err)
		recordDependencyProbeRejection(ctx, runtime, labels, rejectionMode, rejectErr)
		return rejectErr
	}

	recordDegradedDependencyStartup(ctx, runtime, labels.dependency, labels.operation, degradedMode)
	return nil
}

func recordDegradedDependencyStartup(ctx context.Context, runtime dependencyProbeRuntime, dependency, operation, mode string) {
	runtime.metrics.MarkStartupDependencyReady(dependency, mode)
	runtime.log.Warn(
		"startup_dependency_degraded",
		startupLogArgs(
			ctx,
			startupLogComponentStartupProbes,
			operation,
			"degraded",
			"dependency", dependency,
			"mode", mode,
		)...,
	)
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
