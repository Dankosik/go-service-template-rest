package main

import (
	"context"
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
		outcome.probes = append(outcome.probes, pg)
	}

	if err := initRedisDependency(bootstrapCtx, runtime, dependencyProbeCtx); err != nil {
		return outcome, err
	}

	if err := initMongoDependency(bootstrapCtx, runtime, dependencyProbeCtx); err != nil {
		return outcome, err
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
		return nil, rejectStartupForPolicyViolation(
			bootstrapCtx,
			runtime.bootstrapSpan,
			runtime.metrics,
			runtime.log,
			runtime.deployTelemetry,
			runtime.startupLifecycleStartedAt,
			"postgres",
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
			recordAdmissionFailureWithRollback(bootstrapCtx, runtime.deployTelemetry, "dependency_init", "postgres", runtime.startupLifecycleStartedAt)
			return nil, fmt.Errorf("%w: postgres init skipped: %w", config.ErrDependencyInit, probeResult.err)
		}

		sanitizedErr := fmt.Errorf("%w: postgres init failed", config.ErrDependencyInit)
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
		recordAdmissionFailureWithRollback(bootstrapCtx, runtime.deployTelemetry, "dependency_init", "postgres", runtime.startupLifecycleStartedAt)
		return nil, sanitizedErr
	}

	runtime.metrics.SetStartupDependencyStatus("postgres", "critical_fail_closed", true)
	return pg, nil
}

func initRedisDependency(bootstrapCtx context.Context, runtime dependencyProbeRuntime, dependencyProbeCtx context.Context) error {
	if !runtime.cfg.Redis.Enabled {
		runtime.metrics.SetStartupDependencyStatus("redis", "disabled", true)
		return nil
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
		return rejectStartupForPolicyViolation(
			bootstrapCtx,
			runtime.bootstrapSpan,
			runtime.metrics,
			runtime.log,
			runtime.deployTelemetry,
			runtime.startupLifecycleStartedAt,
			"redis",
			addressErr,
		)
	}
	if err := runtime.networkPolicy.EnforceEgressTarget(bootstrapCtx, runtime.deployTelemetry, redisProbeAddress, "tcp"); err != nil {
		return rejectStartupForPolicyViolation(
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
		if redisMode == "store" {
			rejectErr := fmt.Errorf("%w: redis init failed", config.ErrDependencyInit)
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
			recordAdmissionFailureWithRollback(bootstrapCtx, runtime.deployTelemetry, "dependency_init", "redis", runtime.startupLifecycleStartedAt)
			return rejectErr
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
		return nil
	}

	runtime.metrics.SetStartupDependencyStatus("redis", redisCriticality, true)
	if redisMode == "cache" {
		runtime.metrics.SetStartupDependencyStatus("redis", "feature_off", false)
	}
	return nil
}

func initMongoDependency(bootstrapCtx context.Context, runtime dependencyProbeRuntime, dependencyProbeCtx context.Context) error {
	if !runtime.cfg.Mongo.Enabled {
		runtime.metrics.SetStartupDependencyStatus("mongo", "disabled", true)
		return nil
	}

	runtime.metrics.SetStartupDependencyStatus("mongo", "critical_fail_degraded", false)
	mongoProbeAddress, addressErr := mongoStartupProbeAddress(runtime.cfg.Mongo)
	if addressErr != nil {
		return rejectStartupForPolicyViolation(
			bootstrapCtx,
			runtime.bootstrapSpan,
			runtime.metrics,
			runtime.log,
			runtime.deployTelemetry,
			runtime.startupLifecycleStartedAt,
			"mongo",
			addressErr,
		)
	}
	if err := runtime.networkPolicy.EnforceEgressTarget(bootstrapCtx, runtime.deployTelemetry, mongoProbeAddress, "tcp"); err != nil {
		return rejectStartupForPolicyViolation(
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
		return nil
	}

	runtime.metrics.SetStartupDependencyStatus("mongo", "critical_fail_degraded", true)
	return nil
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
