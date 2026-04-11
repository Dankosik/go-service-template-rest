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

type dependencyProbeSpec struct {
	stage        string
	spanName     string
	dep          string
	mode         string
	budget       time.Duration
	minRemaining time.Duration
	probe        func(context.Context) error
}

type probeExecutionResult struct {
	budgetBlocked bool
	failed        bool
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
		if runtime.cfg.FeatureFlags.PostgresReadinessProbe {
			outcome.probes = append(outcome.probes, pg)
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
		runtime.metrics.SetStartupDependencyStatus(labels.dependency, startupDependencyModeDisabled, true)
		return nil, nil
	}

	runtime.metrics.SetStartupDependencyStatus(labels.dependency, startupDependencyModeCriticalFailClosed, false)
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
	probeResult := runDependencyProbe(dependencyProbeCtx, runtime.tracer, dependencyProbeSpec{
		stage:        labels.probeName,
		spanName:     labels.probeStage,
		dep:          labels.dependency,
		budget:       postgresProbeBudget,
		minRemaining: startupFailFastThreshold + startupReserveBudget,
		probe: func(probeCtx context.Context) error {
			var err error
			pg, err = initPostgresWithRetry(probeCtx, runtime.cfg.Postgres)
			return err
		},
	})
	if probeResult.failed {
		if probeResult.budgetBlocked {
			recordDependencyProbeRejection(
				bootstrapCtx,
				runtime,
				labels.dependency,
				labels.operation,
				labels.probeStage,
				"",
				probeResult.err,
			)
			return nil, fmt.Errorf("%w: postgres init skipped: %w", config.ErrDependencyInit, probeResult.err)
		}

		sanitizedErr := dependencyInitFailure(labels.dependency, probeResult.err)
		runtime.metrics.SetStartupDependencyStatus(labels.dependency, startupDependencyModeCriticalFailClosed, false)
		recordDependencyProbeRejection(
			bootstrapCtx,
			runtime,
			labels.dependency,
			labels.operation,
			labels.probeStage,
			"",
			sanitizedErr,
		)
		return nil, sanitizedErr
	}

	runtime.metrics.SetStartupDependencyStatus(labels.dependency, startupDependencyModeCriticalFailClosed, true)
	return pg, nil
}

func initRedisDependency(bootstrapCtx context.Context, runtime dependencyProbeRuntime, dependencyProbeCtx context.Context) (health.Probe, error) {
	labels := startupRedisDependencyLabels
	if !runtime.cfg.Redis.Enabled {
		runtime.metrics.SetStartupDependencyStatus(labels.dependency, startupDependencyModeDisabled, true)
		return nil, nil
	}

	redisMode := redisStartupMode(runtime.cfg.Redis.Mode)
	redisCriticality := startupDependencyModeOptionalFailOpen
	if redisMode == startupDependencyModeStore {
		redisCriticality = startupDependencyModeCriticalFailClosed
	}
	runtime.metrics.SetStartupDependencyStatus(labels.dependency, redisCriticality, false)
	if redisMode == startupDependencyModeCache {
		runtime.metrics.SetStartupDependencyStatus(labels.dependency, startupDependencyModeFeatureOff, false)
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
		stage:        labels.probeName,
		spanName:     labels.probeStage,
		dep:          labels.dependency,
		mode:         redisMode,
		budget:       redisProbeBudget,
		minRemaining: startupFailFastThreshold,
		probe: func(probeCtx context.Context) error {
			if redisMode == startupDependencyModeStore {
				return probeRedisWithRetry(probeCtx, runtime.cfg.Redis)
			}
			return probeRedisWithContext(probeCtx, runtime.cfg.Redis)
		},
	})
	if probeResult.failed {
		if shouldAbortDegradedDependencyStartup(probeResult) {
			recordDependencyProbeRejection(
				bootstrapCtx,
				runtime,
				labels.dependency,
				labels.operation,
				labels.probeStage,
				redisMode,
				probeResult.err,
			)
			return nil, dependencyInitAbortFailure(labels.dependency, probeResult)
		}
		if redisMode == startupDependencyModeStore {
			rejectErr := dependencyInitFailure(labels.dependency, probeResult.err)
			recordDependencyProbeRejection(
				bootstrapCtx,
				runtime,
				labels.dependency,
				labels.operation,
				labels.probeStage,
				redisMode,
				rejectErr,
			)
			return nil, rejectErr
		}

		runtime.metrics.SetStartupDependencyStatus(labels.dependency, startupDependencyModeFeatureOff, true)
		runtime.log.Warn(
			"startup_dependency_degraded",
			startupLogArgs(
				bootstrapCtx,
				startupLogComponentStartupProbes,
				labels.operation,
				"degraded",
				"dependency", labels.dependency,
				"mode", startupDependencyModeFeatureOff,
			)...,
		)
		return nil, nil
	}

	runtime.metrics.SetStartupDependencyStatus(labels.dependency, redisCriticality, true)
	if redisMode == startupDependencyModeCache {
		runtime.metrics.SetStartupDependencyStatus(labels.dependency, startupDependencyModeFeatureOff, false)
	}
	if runtime.cfg.FeatureFlags.RedisReadinessProbe || redisMode == startupDependencyModeStore {
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
		runtime.metrics.SetStartupDependencyStatus(labels.dependency, startupDependencyModeDisabled, true)
		return nil, nil
	}

	runtime.metrics.SetStartupDependencyStatus(labels.dependency, startupDependencyModeCriticalFailDegraded, false)
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
		stage:        labels.probeName,
		spanName:     labels.probeStage,
		dep:          labels.dependency,
		budget:       mongoProbeBudget,
		minRemaining: startupFailFastThreshold,
		probe: func(probeCtx context.Context) error {
			return probeMongoWithRetry(probeCtx, runtime.cfg.Mongo)
		},
	})
	if probeResult.failed {
		if shouldAbortDegradedDependencyStartup(probeResult) {
			recordDependencyProbeRejection(
				bootstrapCtx,
				runtime,
				labels.dependency,
				labels.operation,
				labels.probeStage,
				startupDependencyModeDegradedReadOnlyOrStale,
				probeResult.err,
			)
			return nil, dependencyInitAbortFailure(labels.dependency, probeResult)
		}
		runtime.log.Warn(
			"startup_dependency_degraded",
			startupLogArgs(
				bootstrapCtx,
				startupLogComponentStartupProbes,
				labels.operation,
				"degraded",
				"dependency", labels.dependency,
				"mode", startupDependencyModeDegradedReadOnlyOrStale,
			)...,
		)
		return nil, nil
	}

	runtime.metrics.SetStartupDependencyStatus(labels.dependency, startupDependencyModeCriticalFailDegraded, true)
	if runtime.cfg.FeatureFlags.MongoReadinessProbe {
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
		return probeExecutionResult{budgetBlocked: true, failed: true, err: err}
	}

	probeCtx, probeCancel := withStageBudget(dependencyProbeCtx, spec.budget)
	probeCtx, probeSpan := tracer.Start(probeCtx, spec.spanName)
	err := spec.probe(probeCtx)
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

	return probeExecutionResult{budgetBlocked: false, failed: err != nil, err: err}
}

func shouldAbortDegradedDependencyStartup(result probeExecutionResult) bool {
	if result.budgetBlocked {
		return true
	}
	return errors.Is(result.err, context.Canceled) || errors.Is(result.err, context.DeadlineExceeded)
}

func dependencyInitFailure(dep string, err error) error {
	if err == nil {
		return fmt.Errorf("%w: %s init failed", config.ErrDependencyInit, dep)
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %s init failed: %w", config.ErrDependencyInit, dep, err)
	}
	return fmt.Errorf("%w: %s init failed: %w", config.ErrDependencyInit, dep, err)
}

func dependencyInitAbortFailure(dep string, result probeExecutionResult) error {
	if result.budgetBlocked {
		return fmt.Errorf("%w: %s init skipped: %w", config.ErrDependencyInit, dep, result.err)
	}
	return dependencyInitFailure(dep, result.err)
}
