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
	tracer                    trace.Tracer
	bootstrapSpan             trace.Span
	cfg                       config.Config
	metrics                   *telemetry.Metrics
	log                       *slog.Logger
	deployTelemetry           *deployTelemetryRecorder
	networkPolicy             networkPolicy
	startupLifecycleStartedAt time.Time
}

type dependencyProbeOutcome struct {
	probes       []health.Probe
	postgresPool *postgres.Pool
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

func initStartupDependencies(startupCtx context.Context, bootstrapCtx context.Context, runtime dependencyProbeRuntime) (dependencyProbeOutcome, error) {
	dependencyProbeCtx, dependencyProbeCancel := withStageBudget(startupCtx, startupProbeBudget)
	defer dependencyProbeCancel()

	outcome := dependencyProbeOutcome{probes: make([]health.Probe, 0, 1)}

	pg, err := initPostgresDependency(bootstrapCtx, runtime, dependencyProbeCtx)
	if err != nil {
		return outcome, err
	}
	if pg != nil {
		outcome.postgresPool = pg
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
	if !runtime.cfg.Postgres.Enabled {
		runtime.metrics.SetStartupDependencyStatus("postgres", "disabled", true)
		return nil, nil
	}

	runtime.metrics.SetStartupDependencyStatus("postgres", "critical_fail_closed", false)
	postgresProbeAddress, addressErr := postgresStartupProbeAddress(runtime.cfg.Postgres)
	if addressErr != nil {
		return nil, rejectStartupForDependencyInit(
			bootstrapCtx,
			runtime.bootstrapSpan,
			runtime.metrics,
			runtime.log,
			runtime.deployTelemetry,
			runtime.startupLifecycleStartedAt,
			"postgres",
			"startup.resolve.postgres",
			addressErr,
		)
	}
	if err := runtime.networkPolicy.EnforceEgressTarget(bootstrapCtx, runtime.deployTelemetry, postgresProbeAddress, "tcp"); err != nil {
		return nil, rejectStartupForPolicyViolation(
			bootstrapCtx,
			runtime.bootstrapSpan,
			runtime.metrics,
			runtime.log,
			runtime.deployTelemetry,
			runtime.startupLifecycleStartedAt,
			"postgres",
			err,
		)
	}

	var pg *postgres.Pool
	probeResult := runDependencyProbe(dependencyProbeCtx, runtime.tracer, dependencyProbeSpec{
		stage:        "postgres_startup_probe",
		spanName:     "startup.probe.postgres",
		dep:          "postgres",
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
			runtime.bootstrapSpan.RecordError(probeResult.err)
			runtime.bootstrapSpan.SetAttributes(
				attribute.String("result", "error"),
				attribute.String("error.type", "dependency_init"),
				attribute.String("failed.stage", "startup.probe.postgres"),
			)
			runtime.metrics.IncConfigValidationFailure("dependency_init")
			runtime.metrics.IncConfigStartupOutcome("rejected")
			runtime.log.Error(
				"startup_blocked",
				startupLogArgs(
					bootstrapCtx,
					"startup_probes",
					"postgres_probe",
					"error",
					"error.type", "dependency_init",
					"dependency", "postgres",
				)...,
			)
			recordAdmissionFailure(bootstrapCtx, runtime.deployTelemetry, "dependency_init", "postgres")
			return nil, fmt.Errorf("%w: postgres init skipped: %w", config.ErrDependencyInit, probeResult.err)
		}

		sanitizedErr := dependencyInitFailure("postgres", probeResult.err)
		runtime.metrics.SetStartupDependencyStatus("postgres", "critical_fail_closed", false)
		runtime.metrics.IncConfigValidationFailure("dependency_init")
		runtime.metrics.IncConfigStartupOutcome("rejected")
		runtime.bootstrapSpan.RecordError(sanitizedErr)
		runtime.bootstrapSpan.SetAttributes(
			attribute.String("result", "error"),
			attribute.String("error.type", "dependency_init"),
			attribute.String("failed.stage", "startup.probe.postgres"),
		)
		runtime.log.Error(
			"startup_blocked",
			startupLogArgs(
				bootstrapCtx,
				"startup_probes",
				"postgres_probe",
				"error",
				"error.type", "dependency_init",
				"dependency", "postgres",
			)...,
		)
		recordAdmissionFailure(bootstrapCtx, runtime.deployTelemetry, "dependency_init", "postgres")
		return nil, sanitizedErr
	}

	runtime.metrics.SetStartupDependencyStatus("postgres", "critical_fail_closed", true)
	return pg, nil
}

func initRedisDependency(bootstrapCtx context.Context, runtime dependencyProbeRuntime, dependencyProbeCtx context.Context) (health.Probe, error) {
	if !runtime.cfg.Redis.Enabled {
		runtime.metrics.SetStartupDependencyStatus("redis", "disabled", true)
		return nil, nil
	}

	redisMode := redisStartupMode(runtime.cfg.Redis.Mode)
	redisCriticality := "optional_fail_open"
	if redisMode == "store" {
		redisCriticality = "critical_fail_closed"
	}
	runtime.metrics.SetStartupDependencyStatus("redis", redisCriticality, false)
	if redisMode == "cache" {
		runtime.metrics.SetStartupDependencyStatus("redis", "feature_off", false)
	}

	redisProbeAddress, addressErr := redisStartupProbeAddress(runtime.cfg.Redis)
	if addressErr != nil {
		return nil, rejectStartupForDependencyInit(
			bootstrapCtx,
			runtime.bootstrapSpan,
			runtime.metrics,
			runtime.log,
			runtime.deployTelemetry,
			runtime.startupLifecycleStartedAt,
			"redis",
			"startup.resolve.redis",
			addressErr,
		)
	}
	if err := runtime.networkPolicy.EnforceEgressTarget(bootstrapCtx, runtime.deployTelemetry, redisProbeAddress, "tcp"); err != nil {
		return nil, rejectStartupForPolicyViolation(
			bootstrapCtx,
			runtime.bootstrapSpan,
			runtime.metrics,
			runtime.log,
			runtime.deployTelemetry,
			runtime.startupLifecycleStartedAt,
			"redis",
			err,
		)
	}

	probeResult := runDependencyProbe(dependencyProbeCtx, runtime.tracer, dependencyProbeSpec{
		stage:        "redis_startup_probe",
		spanName:     "startup.probe.redis",
		dep:          "redis",
		mode:         redisMode,
		budget:       redisProbeBudget,
		minRemaining: startupFailFastThreshold,
		probe: func(probeCtx context.Context) error {
			if redisMode == "store" {
				return probeRedisWithRetry(probeCtx, runtime.cfg.Redis)
			}
			return probeRedisWithContext(probeCtx, runtime.cfg.Redis)
		},
	})
	if probeResult.failed {
		if shouldAbortDegradedDependencyStartup(probeResult) {
			runtime.metrics.IncConfigValidationFailure("dependency_init")
			runtime.metrics.IncConfigStartupOutcome("rejected")
			runtime.bootstrapSpan.RecordError(probeResult.err)
			runtime.bootstrapSpan.SetAttributes(
				attribute.String("result", "error"),
				attribute.String("error.type", "dependency_init"),
				attribute.String("failed.stage", "startup.probe.redis"),
			)
			runtime.log.Error(
				"startup_blocked",
				startupLogArgs(
					bootstrapCtx,
					"startup_probes",
					"redis_probe",
					"error",
					"error.type", "dependency_init",
					"dependency", "redis",
					"mode", redisMode,
				)...,
			)
			recordAdmissionFailure(bootstrapCtx, runtime.deployTelemetry, "dependency_init", "redis")
			return nil, dependencyInitAbortFailure("redis", probeResult)
		}
		if redisMode == "store" {
			rejectErr := dependencyInitFailure("redis", probeResult.err)
			runtime.bootstrapSpan.RecordError(rejectErr)
			runtime.bootstrapSpan.SetAttributes(
				attribute.String("result", "error"),
				attribute.String("error.type", "dependency_init"),
				attribute.String("failed.stage", "startup.probe.redis"),
			)
			runtime.metrics.IncConfigValidationFailure("dependency_init")
			runtime.metrics.IncConfigStartupOutcome("rejected")
			runtime.log.Error(
				"startup_blocked",
				startupLogArgs(
					bootstrapCtx,
					"startup_probes",
					"redis_probe",
					"error",
					"error.type", "dependency_init",
					"dependency", "redis",
					"mode", redisMode,
				)...,
			)
			recordAdmissionFailure(bootstrapCtx, runtime.deployTelemetry, "dependency_init", "redis")
			return nil, rejectErr
		}

		runtime.metrics.SetStartupDependencyStatus("redis", "feature_off", true)
		runtime.log.Warn(
			"startup_dependency_degraded",
			startupLogArgs(
				bootstrapCtx,
				"startup_probes",
				"redis_probe",
				"degraded",
				"dependency", "redis",
				"mode", "feature_off",
			)...,
		)
		return nil, nil
	}

	runtime.metrics.SetStartupDependencyStatus("redis", redisCriticality, true)
	if redisMode == "cache" {
		runtime.metrics.SetStartupDependencyStatus("redis", "feature_off", false)
	}
	if runtime.cfg.FeatureFlags.RedisReadinessProbe || redisMode == "store" {
		return startupNamedProbe{
			name: "redis",
			check: func(ctx context.Context) error {
				return probeRedisWithContext(ctx, runtime.cfg.Redis)
			},
		}, nil
	}
	return nil, nil
}

func initMongoDependency(bootstrapCtx context.Context, runtime dependencyProbeRuntime, dependencyProbeCtx context.Context) (health.Probe, error) {
	if !runtime.cfg.Mongo.Enabled {
		runtime.metrics.SetStartupDependencyStatus("mongo", "disabled", true)
		return nil, nil
	}

	runtime.metrics.SetStartupDependencyStatus("mongo", "critical_fail_degraded", false)
	mongoProbeAddress, addressErr := mongoStartupProbeAddress(runtime.cfg.Mongo)
	if addressErr != nil {
		return nil, rejectStartupForDependencyInit(
			bootstrapCtx,
			runtime.bootstrapSpan,
			runtime.metrics,
			runtime.log,
			runtime.deployTelemetry,
			runtime.startupLifecycleStartedAt,
			"mongo",
			"startup.resolve.mongo",
			addressErr,
		)
	}
	if err := runtime.networkPolicy.EnforceEgressTarget(bootstrapCtx, runtime.deployTelemetry, mongoProbeAddress, "tcp"); err != nil {
		return nil, rejectStartupForPolicyViolation(
			bootstrapCtx,
			runtime.bootstrapSpan,
			runtime.metrics,
			runtime.log,
			runtime.deployTelemetry,
			runtime.startupLifecycleStartedAt,
			"mongo",
			err,
		)
	}

	probeResult := runDependencyProbe(dependencyProbeCtx, runtime.tracer, dependencyProbeSpec{
		stage:        "mongo_startup_probe",
		spanName:     "startup.probe.mongo",
		dep:          "mongo",
		budget:       mongoProbeBudget,
		minRemaining: startupFailFastThreshold,
		probe: func(probeCtx context.Context) error {
			return probeMongoWithRetry(probeCtx, runtime.cfg.Mongo)
		},
	})
	if probeResult.failed {
		if shouldAbortDegradedDependencyStartup(probeResult) {
			runtime.metrics.IncConfigValidationFailure("dependency_init")
			runtime.metrics.IncConfigStartupOutcome("rejected")
			runtime.bootstrapSpan.RecordError(probeResult.err)
			runtime.bootstrapSpan.SetAttributes(
				attribute.String("result", "error"),
				attribute.String("error.type", "dependency_init"),
				attribute.String("failed.stage", "startup.probe.mongo"),
			)
			runtime.log.Error(
				"startup_blocked",
				startupLogArgs(
					bootstrapCtx,
					"startup_probes",
					"mongo_probe",
					"error",
					"error.type", "dependency_init",
					"dependency", "mongo",
					"mode", "degraded_read_only_or_stale",
				)...,
			)
			recordAdmissionFailure(bootstrapCtx, runtime.deployTelemetry, "dependency_init", "mongo")
			return nil, dependencyInitAbortFailure("mongo", probeResult)
		}
		runtime.log.Warn(
			"startup_dependency_degraded",
			startupLogArgs(
				bootstrapCtx,
				"startup_probes",
				"mongo_probe",
				"degraded",
				"dependency", "mongo",
				"mode", "degraded_read_only_or_stale",
			)...,
		)
		return nil, nil
	}

	runtime.metrics.SetStartupDependencyStatus("mongo", "critical_fail_degraded", true)
	if runtime.cfg.FeatureFlags.MongoReadinessProbe {
		return startupNamedProbe{
			name: "mongo",
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
