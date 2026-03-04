package main

import (
	"context"
	crand "crypto/rand"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/example/go-service-template-rest/internal/app/health"
	"github.com/example/go-service-template-rest/internal/app/ping"
	"github.com/example/go-service-template-rest/internal/config"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	telemetryShutdownTimeout    = 5 * time.Second
	startupBudget               = 30 * time.Second
	startupReserveBudget        = 3 * time.Second
	startupFailFastThreshold    = 150 * time.Millisecond
	startupConfigLoadBudget     = 10 * time.Second
	startupConfigValidateBudget = 2 * time.Second
	startupProbeBudget          = 15 * time.Second
	startupTelemetryBudget      = 2 * time.Second

	postgresProbeBudget = 5 * time.Second
	redisProbeBudget    = 3 * time.Second
	mongoProbeBudget    = 5 * time.Second

	startupRetryBaseDelay   = 50 * time.Millisecond
	startupRetryMaxDelay    = 250 * time.Millisecond
	postgresStartupAttempts = 2
	redisStoreProbeAttempts = 2
	mongoProbeAttempts      = 2
)

type overlayPathsFlag []string

func (f *overlayPathsFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *overlayPathsFlag) Set(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("config overlay path cannot be empty")
	}
	*f = append(*f, trimmed)
	return nil
}

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run() (runErr error) {
	loadOptions, err := parseLoadOptions(os.Args[1:])
	if err != nil {
		return err
	}

	bootstrapLog := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})).With(
		"service.name", "service",
		"service.version", "unknown",
		"deployment.environment.name", "unknown",
	)
	slog.SetDefault(bootstrapLog)

	metrics := telemetry.New()
	deployTelemetry := newDeployTelemetryRecorder(bootstrapLog, metrics, "unknown")
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	defer func() {
		if runErr != nil {
			slog.Error(
				"process_exit",
				startupLogArgs(
					signalCtx,
					"lifecycle",
					"process_exit",
					"error",
					"err", runErr,
				)...,
			)
			return
		}
		slog.Info(
			"process_exit",
			startupLogArgs(
				signalCtx,
				"lifecycle",
				"process_exit",
				"success",
			)...,
		)
	}()

	startupCtx, startupCancel := context.WithTimeout(signalCtx, startupBudget)
	defer startupCancel()
	startupLifecycleStartedAt := time.Now()

	loadOptions.LoadBudget = startupConfigLoadBudget
	loadOptions.ValidateBudget = startupConfigValidateBudget

	slog.Info(
		"config_load_started",
		startupLogArgs(
			startupCtx,
			"config_loader",
			"load",
			"started",
			"config.strict", loadOptions.Strict,
			"config.file", loadOptions.ConfigPath,
			"config.overlay_count", len(loadOptions.ConfigOverlays),
		)...,
	)

	cfg, configReport, err := config.LoadDetailedWithContext(startupCtx, loadOptions)
	if err != nil {
		failedStage, failedDuration := failedStageDetails(configReport)
		errorType := config.ErrorType(err)
		metrics.ObserveConfigLoadDuration(failedStage, "error", failedDuration)
		metrics.IncConfigValidationFailure(errorType)
		metrics.IncConfigStartupOutcome("rejected")
		slog.Error(
			"config_load_failed",
			startupLogArgs(
				startupCtx,
				"config_loader",
				"load",
				"error",
				"stage", failedStage,
				"error.type", errorType,
			)...,
		)
		recordAdmissionFailureWithRollback(startupCtx, deployTelemetry, "startup_error", "startup", startupLifecycleStartedAt)
		return fmt.Errorf("load config (%s): %w", errorType, err)
	}

	recordConfigSuccessMetrics(metrics, configReport)
	if len(configReport.UnknownKeyWarnings) > 0 {
		metrics.AddConfigUnknownKeyWarnings(len(configReport.UnknownKeyWarnings))
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.Log.Level}))
	log = log.With(
		"service.name", cfg.Observability.OTel.ServiceName,
		"service.version", cfg.App.Version,
		"deployment.environment.name", cfg.App.Env,
	)
	slog.SetDefault(log)
	deployTelemetry.SetLogger(log)
	deployTelemetry.SetEnvironment(cfg.App.Env)

	metrics.SetStartupDependencyStatus("telemetry", "optional_fail_open", false)
	telemetryCtx, telemetryCancel := withStageBudget(startupCtx, startupTelemetryBudget)
	tracingShutdown, telemetryInitErr := telemetry.SetupTracing(telemetryCtx, telemetry.TracingConfig{
		ServiceName:      cfg.Observability.OTel.ServiceName,
		ServiceVersion:   cfg.App.Version,
		DeploymentEnv:    cfg.App.Env,
		TracesSampler:    cfg.Observability.OTel.TracesSampler,
		TracesSamplerArg: cfg.Observability.OTel.TracesSamplerArg,
		Exporter: telemetry.TraceExporterConfig{
			OTLPEndpoint:       cfg.Observability.OTel.Exporter.OTLPEndpoint,
			OTLPTracesEndpoint: cfg.Observability.OTel.Exporter.OTLPTracesEndpoint,
			OTLPHeaders:        cfg.Observability.OTel.Exporter.OTLPHeaders,
			OTLPProtocol:       cfg.Observability.OTel.Exporter.OTLPProtocol,
		},
	})
	telemetryCancel()
	if telemetryInitErr != nil {
		metrics.IncTelemetryInitFailure(telemetryInitFailureReason(telemetryInitErr))
		metrics.SetStartupDependencyStatus("telemetry", "feature_off", false)
	} else {
		metrics.SetStartupDependencyStatus("telemetry", "optional_fail_open", true)
		defer func() {
			log.Info(
				"telemetry_flush_started",
				startupLogArgs(
					startupCtx,
					"shutdown",
					"telemetry_flush",
					"started",
				)...,
			)
			shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(startupCtx), telemetryShutdownTimeout)
			defer cancel()
			if err := tracingShutdown(shutdownCtx); err != nil {
				log.Error(
					"tracing shutdown failed",
					startupLogArgs(
						startupCtx,
						"startup_probes",
						"telemetry_shutdown",
						"error",
						"error.type", "dependency_init",
						"err", err,
					)...,
				)
				return
			}
			log.Info(
				"telemetry_flush_completed",
				startupLogArgs(
					startupCtx,
					"shutdown",
					"telemetry_flush",
					"success",
				)...,
			)
		}()
	}

	// Emit config lifecycle spans after telemetry init attempt.
	// When telemetry runs in fail-open mode, logs/metrics remain the source of truth.
	tracer := otel.Tracer("service.startup")
	bootstrapCtx, bootstrapSpan := tracer.Start(startupCtx, "config.bootstrap")
	defer bootstrapSpan.End()

	recordConfigStageSpan(tracer, bootstrapCtx, config.StageLoadDefaults, configReport.LoadDefaultsDuration, "success", "")
	recordConfigStageSpan(tracer, bootstrapCtx, config.StageLoadFile, configReport.LoadFileDuration, "success", "")
	recordConfigStageSpan(tracer, bootstrapCtx, config.StageLoadEnv, configReport.LoadEnvDuration, "success", "")
	recordConfigStageSpan(tracer, bootstrapCtx, config.StageParse, configReport.ParseDuration, "success", "")
	recordConfigStageSpan(tracer, bootstrapCtx, config.StageValidate, configReport.ValidateDuration, "success", "")

	log.Info(
		"config_validated",
		startupLogArgs(
			bootstrapCtx,
			"config_validator",
			"validate",
			"success",
			"load.duration_ms", configReport.LoadDuration.Milliseconds(),
			"validate.duration_ms", configReport.ValidateDuration.Milliseconds(),
			"unknown_key_warnings", len(configReport.UnknownKeyWarnings),
		)...,
	)

	for _, key := range configReport.UnknownKeyWarnings {
		log.Warn(
			"unknown config key ignored (strict mode disabled)",
			startupLogArgs(
				bootstrapCtx,
				"config_validator",
				"unknown_key",
				"warning",
				"key", key,
			)...,
		)
	}
	if len(configReport.UnknownKeyWarnings) > 0 {
		deployTelemetry.RecordConfigDriftDetected(bootstrapCtx, "runtime", "", "startup-config")
	} else {
		deployTelemetry.RecordConfigDriftReconciled(bootstrapCtx, "success", "", "startup-config", 0)
	}
	if telemetryInitErr != nil {
		log.Warn(
			"startup_dependency_degraded",
			startupLogArgs(
				bootstrapCtx,
				"startup_probes",
				"telemetry_init",
				"degraded",
				"dependency", "telemetry",
				"mode", "feature_off",
				"reason", telemetryInitFailureReason(telemetryInitErr),
			)...,
		)
	}

	log.Info(
		"startup config summary",
		startupLogArgs(
			bootstrapCtx,
			"config_loader",
			"startup_summary",
			"success",
			"config.strict", loadOptions.Strict,
			"config.file", loadOptions.ConfigPath,
			"config.overlay_count", len(loadOptions.ConfigOverlays),
			"app.env", cfg.App.Env,
			"http.addr", cfg.HTTP.Addr,
			"postgres.enabled", cfg.Postgres.Enabled,
			"redis.enabled", cfg.Redis.Enabled,
			"redis.mode", cfg.Redis.Mode,
			"mongo.enabled", cfg.Mongo.Enabled,
		)...,
	)

	networkPolicy, err := loadNetworkPolicyFromEnv()
	if err != nil {
		policyClass, reasonClass := networkPolicyErrorLabels(err)
		if policyClass == "egress" {
			deployTelemetry.RecordNetworkEgressPolicyViolation(bootstrapCtx, reasonClass, "deny")
		} else {
			deployTelemetry.RecordNetworkIngressPolicyViolation(bootstrapCtx, reasonClass, "deny")
		}
		return rejectStartupForPolicyViolation(
			bootstrapCtx,
			bootstrapSpan,
			metrics,
			log,
			deployTelemetry,
			startupLifecycleStartedAt,
			"network_policy",
			fmt.Errorf("%w: invalid network policy configuration", config.ErrDependencyInit),
		)
	}

	if err := networkPolicy.EnforceIngress(bootstrapCtx, deployTelemetry); err != nil {
		return rejectStartupForPolicyViolation(
			bootstrapCtx,
			bootstrapSpan,
			metrics,
			log,
			deployTelemetry,
			startupLifecycleStartedAt,
			"ingress_policy",
			err,
		)
	}

	if err := networkPolicy.EmitEgressExceptionState(bootstrapCtx, deployTelemetry); err != nil {
		return rejectStartupForPolicyViolation(
			bootstrapCtx,
			bootstrapSpan,
			metrics,
			log,
			deployTelemetry,
			startupLifecycleStartedAt,
			"egress_exception",
			err,
		)
	}

	dependencyProbeCtx, dependencyProbeCancel := withStageBudget(startupCtx, startupProbeBudget)
	defer dependencyProbeCancel()

	probes := make([]health.Probe, 0, 1)
	if cfg.Postgres.Enabled {
		metrics.SetStartupDependencyStatus("postgres", "critical_fail_closed", false)
		postgresProbeAddress, addressErr := postgresStartupProbeAddress(cfg.Postgres)
		if addressErr != nil {
			return rejectStartupForPolicyViolation(
				bootstrapCtx,
				bootstrapSpan,
				metrics,
				log,
				deployTelemetry,
				startupLifecycleStartedAt,
				"postgres",
				addressErr,
			)
		}
		if err := networkPolicy.EnforceEgressTarget(bootstrapCtx, deployTelemetry, postgresProbeAddress, "tcp"); err != nil {
			return rejectStartupForPolicyViolation(
				bootstrapCtx,
				bootstrapSpan,
				metrics,
				log,
				deployTelemetry,
				startupLifecycleStartedAt,
				"postgres",
				err,
			)
		}
		if err := ensureRemainingStartupBudget(dependencyProbeCtx, startupFailFastThreshold+startupReserveBudget, "postgres_startup_probe"); err != nil {
			bootstrapSpan.RecordError(err)
			bootstrapSpan.SetAttributes(
				attribute.String("result", "error"),
				attribute.String("error.type", "dependency_init"),
				attribute.String("failed.stage", "startup.probe.postgres"),
			)
			metrics.IncConfigValidationFailure("dependency_init")
			metrics.IncConfigStartupOutcome("rejected")
			log.Error(
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
			recordAdmissionFailureWithRollback(bootstrapCtx, deployTelemetry, "dependency_init", "postgres", startupLifecycleStartedAt)
			return fmt.Errorf("%w: postgres init skipped: %w", config.ErrDependencyInit, err)
		}

		postgresProbeCtx, postgresProbeCancel := withStageBudget(dependencyProbeCtx, postgresProbeBudget)
		postgresProbeCtx, probeSpan := tracer.Start(postgresProbeCtx, "startup.probe.postgres")
		pg, err := initPostgresWithRetry(postgresProbeCtx, cfg.Postgres)
		postgresProbeCancel()
		if err != nil {
			sanitizedErr := fmt.Errorf("%w: postgres init failed", config.ErrDependencyInit)
			probeSpan.RecordError(sanitizedErr)
			probeSpan.SetAttributes(
				attribute.String("dep", "postgres"),
				attribute.String("result", "error"),
			)
			probeSpan.End()
			metrics.SetStartupDependencyStatus("postgres", "critical_fail_closed", false)
			metrics.IncConfigValidationFailure("dependency_init")
			metrics.IncConfigStartupOutcome("rejected")
			bootstrapSpan.RecordError(sanitizedErr)
			bootstrapSpan.SetAttributes(
				attribute.String("result", "error"),
				attribute.String("error.type", "dependency_init"),
				attribute.String("failed.stage", "startup.probe.postgres"),
			)
			log.Error(
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
			recordAdmissionFailureWithRollback(bootstrapCtx, deployTelemetry, "dependency_init", "postgres", startupLifecycleStartedAt)
			return sanitizedErr
		}
		probeSpan.SetAttributes(
			attribute.String("dep", "postgres"),
			attribute.String("result", "success"),
		)
		probeSpan.End()
		metrics.SetStartupDependencyStatus("postgres", "critical_fail_closed", true)
		defer pg.Close()
		probes = append(probes, pg)
	} else {
		metrics.SetStartupDependencyStatus("postgres", "disabled", true)
	}

	if cfg.Redis.Enabled {
		redisMode := redisStartupMode(cfg.Redis.Mode)
		redisCriticality := "optional_fail_open"
		if redisMode == "store" {
			redisCriticality = "critical_fail_closed"
		}
		metrics.SetStartupDependencyStatus("redis", redisCriticality, false)
		if redisMode == "cache" {
			metrics.SetStartupDependencyStatus("redis", "feature_off", false)
		}

		redisProbeAddress, addressErr := redisStartupProbeAddress(cfg.Redis)
		if addressErr != nil {
			return rejectStartupForPolicyViolation(
				bootstrapCtx,
				bootstrapSpan,
				metrics,
				log,
				deployTelemetry,
				startupLifecycleStartedAt,
				"redis",
				addressErr,
			)
		}
		if err := networkPolicy.EnforceEgressTarget(bootstrapCtx, deployTelemetry, redisProbeAddress, "tcp"); err != nil {
			return rejectStartupForPolicyViolation(
				bootstrapCtx,
				bootstrapSpan,
				metrics,
				log,
				deployTelemetry,
				startupLifecycleStartedAt,
				"redis",
				err,
			)
		}

		probeErr := ensureRemainingStartupBudget(dependencyProbeCtx, startupFailFastThreshold, "redis_startup_probe")
		if probeErr == nil {
			redisProbeCtx, redisProbeCancel := withStageBudget(dependencyProbeCtx, redisProbeBudget)
			redisProbeCtx, redisProbeSpan := tracer.Start(redisProbeCtx, "startup.probe.redis")
			if redisMode == "store" {
				probeErr = probeRedisWithRetry(redisProbeCtx, cfg.Redis)
			} else {
				probeErr = probeRedisWithContext(redisProbeCtx, cfg.Redis)
			}
			redisProbeCancel()
			if probeErr != nil {
				redisProbeSpan.RecordError(probeErr)
				redisProbeSpan.SetAttributes(
					attribute.String("dep", "redis"),
					attribute.String("mode", redisMode),
					attribute.String("result", "error"),
				)
			} else {
				redisProbeSpan.SetAttributes(
					attribute.String("dep", "redis"),
					attribute.String("mode", redisMode),
					attribute.String("result", "success"),
				)
			}
			redisProbeSpan.End()
		}

		if probeErr != nil {
			if redisMode == "store" {
				rejectErr := fmt.Errorf("%w: redis init failed", config.ErrDependencyInit)
				bootstrapSpan.RecordError(rejectErr)
				bootstrapSpan.SetAttributes(
					attribute.String("result", "error"),
					attribute.String("error.type", "dependency_init"),
					attribute.String("failed.stage", "startup.probe.redis"),
				)
				metrics.IncConfigValidationFailure("dependency_init")
				metrics.IncConfigStartupOutcome("rejected")
				log.Error(
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
				recordAdmissionFailureWithRollback(bootstrapCtx, deployTelemetry, "dependency_init", "redis", startupLifecycleStartedAt)
				return rejectErr
			}
			metrics.SetStartupDependencyStatus("redis", "feature_off", true)
			log.Warn(
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
		} else {
			metrics.SetStartupDependencyStatus("redis", redisCriticality, true)
			if redisMode == "cache" {
				metrics.SetStartupDependencyStatus("redis", "feature_off", false)
			}
		}
	} else {
		metrics.SetStartupDependencyStatus("redis", "disabled", true)
	}

	if cfg.Mongo.Enabled {
		metrics.SetStartupDependencyStatus("mongo", "critical_fail_degraded", false)
		mongoProbeAddress, addressErr := mongoStartupProbeAddress(cfg.Mongo)
		if addressErr != nil {
			return rejectStartupForPolicyViolation(
				bootstrapCtx,
				bootstrapSpan,
				metrics,
				log,
				deployTelemetry,
				startupLifecycleStartedAt,
				"mongo",
				addressErr,
			)
		}
		if err := networkPolicy.EnforceEgressTarget(bootstrapCtx, deployTelemetry, mongoProbeAddress, "tcp"); err != nil {
			return rejectStartupForPolicyViolation(
				bootstrapCtx,
				bootstrapSpan,
				metrics,
				log,
				deployTelemetry,
				startupLifecycleStartedAt,
				"mongo",
				err,
			)
		}

		probeErr := ensureRemainingStartupBudget(dependencyProbeCtx, startupFailFastThreshold, "mongo_startup_probe")
		if probeErr == nil {
			mongoProbeCtx, mongoProbeCancel := withStageBudget(dependencyProbeCtx, mongoProbeBudget)
			mongoProbeCtx, mongoProbeSpan := tracer.Start(mongoProbeCtx, "startup.probe.mongo")
			probeErr = probeMongoWithRetry(mongoProbeCtx, cfg.Mongo)
			mongoProbeCancel()
			if probeErr != nil {
				mongoProbeSpan.RecordError(probeErr)
				mongoProbeSpan.SetAttributes(
					attribute.String("dep", "mongo"),
					attribute.String("result", "error"),
				)
			} else {
				mongoProbeSpan.SetAttributes(
					attribute.String("dep", "mongo"),
					attribute.String("result", "success"),
				)
			}
			mongoProbeSpan.End()
		}

		if probeErr != nil {
			log.Warn(
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
		} else {
			metrics.SetStartupDependencyStatus("mongo", "critical_fail_degraded", true)
		}
	} else {
		metrics.SetStartupDependencyStatus("mongo", "disabled", true)
	}

	healthSvc := health.New(probes...)
	pingSvc := ping.New()

	handler := httpx.NewRouter(
		log,
		httpx.Handlers{
			Health: healthSvc,
			Ping:   pingSvc,
		},
		metrics,
		httpx.RouterConfig{MaxBodyBytes: cfg.HTTP.MaxBodyBytes},
	)

	srv := httpx.New(httpx.Config{
		Addr:              cfg.HTTP.Addr,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
		MaxHeaderBytes:    cfg.HTTP.MaxHeaderBytes,
	}, handler)

	listener, err := net.Listen("tcp", cfg.HTTP.Addr)
	if err != nil {
		bootstrapSpan.RecordError(err)
		bootstrapSpan.SetAttributes(
			attribute.String("result", "error"),
			attribute.String("error.type", "startup_error"),
			attribute.String("failed.stage", "startup.http_listen"),
		)
		metrics.IncConfigValidationFailure("startup_error")
		metrics.IncConfigStartupOutcome("rejected")
		log.Error(
			"startup_blocked",
			startupLogArgs(
				bootstrapCtx,
				"startup_probes",
				"http_listen",
				"error",
				"error.type", "startup_error",
				"err", err,
			)...,
		)
		recordAdmissionFailureWithRollback(bootstrapCtx, deployTelemetry, "startup_error", "startup", startupLifecycleStartedAt)
		return fmt.Errorf("listen http server: %w", err)
	}

	metrics.IncConfigStartupOutcome("ready")
	bootstrapSpan.SetAttributes(attribute.String("result", "success"))
	deployTelemetry.RecordAdmission(bootstrapCtx, "success", "ready", "")

	runErrCh := make(chan error, 1)
	serverStartedAt := time.Now()
	go func() {
		log.Info("http server started", "addr", listener.Addr().String(), "env", cfg.App.Env)
		runErrCh <- srv.Serve(listener)
	}()

	var serverErr error
	select {
	case <-signalCtx.Done():
		log.Info("shutdown signal received")
	case err := <-runErrCh:
		serverErr = err
		if serverErr != nil {
			log.Error("http server stopped with error", "err", serverErr)
			recordRollbackFailure(signalCtx, deployTelemetry, "runtime_error", serverStartedAt)
		}
	}

	shutdownStartedAt := time.Now()
	if err := drainAndShutdown(signalCtx, cfg.HTTP.ShutdownTimeout, healthSvc, srv); err != nil {
		recordRollbackFailure(signalCtx, deployTelemetry, "shutdown_error", shutdownStartedAt)
		return err
	}
	if serverErr != nil {
		return fmt.Errorf("http server stopped with error: %w", serverErr)
	}

	log.Info("shutdown complete")
	return nil
}

type startupDrainer interface {
	StartDrain()
}

type shutdownServer interface {
	Shutdown(context.Context) error
}

func drainAndShutdown(ctx context.Context, timeout time.Duration, drainer startupDrainer, srv shutdownServer) error {
	slog.Info(
		"drain_started",
		startupLogArgs(
			ctx,
			"shutdown",
			"drain",
			"started",
		)...,
	)
	drainer.StartDrain()
	slog.Info(
		"readiness_disabled",
		startupLogArgs(
			ctx,
			"shutdown",
			"readiness",
			"success",
		)...,
	)

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
		if errors.Is(err, context.DeadlineExceeded) {
			slog.Error(
				"shutdown_timeout",
				startupLogArgs(
					ctx,
					"shutdown",
					"drain",
					"error",
					"error.type", "deadline_exceeded",
				)...,
			)
		}
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	slog.Info(
		"drain_completed",
		startupLogArgs(
			ctx,
			"shutdown",
			"drain",
			"success",
		)...,
	)
	return nil
}

func recordConfigSuccessMetrics(metrics *telemetry.Metrics, report config.LoadReport) {
	if report.LoadDefaultsDuration > 0 {
		metrics.ObserveConfigLoadDuration(config.StageLoadDefaults, "success", report.LoadDefaultsDuration)
	}
	if report.LoadFileDuration > 0 {
		metrics.ObserveConfigLoadDuration(config.StageLoadFile, "success", report.LoadFileDuration)
	}
	if report.LoadEnvDuration > 0 {
		metrics.ObserveConfigLoadDuration(config.StageLoadEnv, "success", report.LoadEnvDuration)
	}
	if report.ParseDuration > 0 {
		metrics.ObserveConfigLoadDuration(config.StageParse, "success", report.ParseDuration)
	}
	if report.ValidateDuration > 0 {
		metrics.ObserveConfigLoadDuration(config.StageValidate, "success", report.ValidateDuration)
	}
}

func failedStageDetails(report config.LoadReport) (string, time.Duration) {
	stage := strings.TrimSpace(report.FailedStage)
	if stage == "" {
		stage = config.StageLoadDefaults
	}
	duration := report.FailedStageDuration
	if duration <= 0 {
		duration = report.LoadDuration
	}
	if duration <= 0 {
		duration = time.Millisecond
	}
	return stage, duration
}

func rejectStartupForPolicyViolation(
	ctx context.Context,
	bootstrapSpan trace.Span,
	metrics *telemetry.Metrics,
	log *slog.Logger,
	deployTelemetry *deployTelemetryRecorder,
	startupLifecycleStartedAt time.Time,
	dependency string,
	err error,
) error {
	bootstrapSpan.RecordError(err)
	bootstrapSpan.SetAttributes(
		attribute.String("result", "error"),
		attribute.String("error.type", "policy_violation"),
		attribute.String("failed.stage", "startup.policy."+strings.ToLower(strings.TrimSpace(dependency))),
	)
	metrics.IncConfigValidationFailure("policy_violation")
	metrics.IncConfigStartupOutcome("rejected")
	log.Error(
		"startup_blocked",
		startupLogArgs(
			ctx,
			"startup_probes",
			strings.ToLower(strings.TrimSpace(dependency))+"_policy",
			"error",
			"error.type", "policy_violation",
			"dependency", strings.ToLower(strings.TrimSpace(dependency)),
		)...,
	)
	recordAdmissionFailureWithRollback(ctx, deployTelemetry, "policy_violation", strings.ToLower(strings.TrimSpace(dependency)), startupLifecycleStartedAt)
	return fmt.Errorf("%w: startup blocked by network policy: %w", config.ErrDependencyInit, err)
}

func recordAdmissionFailureWithRollback(
	ctx context.Context,
	deployTelemetry *deployTelemetryRecorder,
	reasonClass string,
	probeType string,
	startedAt time.Time,
) {
	if deployTelemetry == nil {
		return
	}

	deployTelemetry.RecordAdmission(ctx, "failure", reasonClass, probeType)
	recordRollbackFailure(ctx, deployTelemetry, "admission_failed", startedAt)
}

func recordRollbackFailure(
	ctx context.Context,
	deployTelemetry *deployTelemetryRecorder,
	trigger string,
	startedAt time.Time,
) {
	if deployTelemetry == nil {
		return
	}

	deployTelemetry.RecordRollback(ctx, trigger, "failure", "", time.Since(startedAt))
	deployTelemetry.RecordRollbackPostcheck("/health/live", "failure")
	deployTelemetry.RecordRollbackPostcheck("/health/ready", "failure")
}

func parseLoadOptions(args []string) (config.LoadOptions, error) {
	var overlays overlayPathsFlag

	flags := flag.NewFlagSet("service", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	configPath := flags.String("config", "", "path to base config file")
	flags.Var(&overlays, "config-overlay", "path to config overlay file (repeatable)")
	configStrict := flags.Bool("config-strict", false, "enable strict unknown-key validation")

	if err := flags.Parse(args); err != nil {
		return config.LoadOptions{}, fmt.Errorf("parse flags: %w", err)
	}

	return config.LoadOptions{
		ConfigPath:     strings.TrimSpace(*configPath),
		ConfigOverlays: overlays,
		Strict:         *configStrict,
	}, nil
}

func startupLogArgs(ctx context.Context, component, operation, outcome string, extra ...any) []any {
	args := make([]any, 0, 6+len(extra))
	args = append(args,
		"component", component,
		"operation", operation,
		"outcome", outcome,
	)

	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		args = append(args,
			"trace_id", spanCtx.TraceID().String(),
			"span_id", spanCtx.SpanID().String(),
		)
	}

	args = append(args, extra...)
	return args
}

func telemetryInitFailureReason(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return "setup_error"
	}
}

func recordConfigStageSpan(tracer trace.Tracer, ctx context.Context, name string, duration time.Duration, result string, errorType string) {
	if duration <= 0 {
		return
	}
	_, span := tracer.Start(ctx, name)
	attrs := []attribute.KeyValue{
		attribute.Int64("duration_ms", duration.Milliseconds()),
		attribute.String("result", result),
	}
	if strings.TrimSpace(errorType) != "" {
		attrs = append(attrs, attribute.String("error.type", errorType))
	}
	span.SetAttributes(attrs...)
	span.End()
}

func initPostgresWithRetry(ctx context.Context, cfg config.PostgresConfig) (*postgres.Pool, error) {
	options := postgres.Options{
		DSN:                cfg.DSN,
		ConnectTimeout:     cfg.ConnectTimeout,
		HealthcheckTimeout: cfg.HealthcheckTimeout,
		MaxOpenConns:       cfg.MaxOpenConns,
		MaxIdleConns:       cfg.MaxIdleConns,
		ConnMaxLifetime:    cfg.ConnMaxLifetime,
	}

	var lastErr error
	for attempt := 1; attempt <= postgresStartupAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("%w: postgres init canceled: %w", config.ErrDependencyInit, err)
		}

		pg, err := postgres.New(ctx, options)
		if err == nil {
			return pg, nil
		}

		lastErr = err
		if !shouldRetryPostgresStartup(err, attempt) {
			break
		}

		delay := fullJitterDelay(attempt)
		if err := sleepWithContext(ctx, delay); err != nil {
			return nil, fmt.Errorf("%w: postgres retry wait canceled: %w", config.ErrDependencyInit, err)
		}
	}

	return nil, fmt.Errorf("%w: postgres init failed after retries: %w", config.ErrDependencyInit, lastErr)
}

func shouldRetryPostgresStartup(err error, attempt int) bool {
	if attempt >= postgresStartupAttempts {
		return false
	}
	return errors.Is(err, postgres.ErrConnect) || errors.Is(err, postgres.ErrHealthcheck)
}

func fullJitterDelay(attempt int) time.Duration {
	backoff := startupRetryBaseDelay << (attempt - 1)
	if backoff > startupRetryMaxDelay {
		backoff = startupRetryMaxDelay
	}
	if backoff <= 0 {
		return 0
	}

	jitter, err := crand.Int(crand.Reader, big.NewInt(int64(backoff)+1))
	if err != nil {
		return backoff
	}
	return time.Duration(jitter.Int64())
}

func withStageBudget(parent context.Context, stageBudget time.Duration) (context.Context, context.CancelFunc) {
	if stageBudget <= 0 {
		return context.WithCancel(parent) // #nosec G118 -- cancel function is returned to caller.
	}
	if deadline, ok := parent.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < stageBudget {
			stageBudget = remaining
		}
	}
	if stageBudget <= 0 {
		return context.WithCancel(parent) // #nosec G118 -- cancel function is returned to caller.
	}
	return context.WithTimeout(parent, stageBudget) // #nosec G118 -- cancel function is returned to caller.
}

func ensureRemainingStartupBudget(ctx context.Context, minRemaining time.Duration, stage string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return nil
	}
	remaining := time.Until(deadline)
	if remaining < minRemaining {
		return fmt.Errorf(
			"%w: %s aborted due to low remaining startup budget (%s < %s)",
			config.ErrDependencyInit,
			stage,
			remaining,
			minRemaining,
		)
	}
	return nil
}

func redisStartupMode(rawMode string) string {
	mode := strings.ToLower(strings.TrimSpace(rawMode))
	if mode == "store" {
		return "store"
	}
	return "cache"
}

func probeRedisWithContext(ctx context.Context, cfg config.RedisConfig) error {
	timeout := cfg.DialTimeout
	if timeout <= 0 {
		timeout = redisProbeBudget
	}
	return probeTCPDependency(ctx, cfg.Addr, timeout)
}

func probeRedisWithRetry(ctx context.Context, cfg config.RedisConfig) error {
	return probeWithRetry(ctx, redisStoreProbeAttempts, func(probeCtx context.Context) error {
		return probeRedisWithContext(probeCtx, cfg)
	})
}

func probeMongoWithContext(ctx context.Context, cfg config.MongoConfig) error {
	addr, err := config.MongoProbeAddress(cfg.URI)
	if err != nil {
		return fmt.Errorf("%w: resolve mongo probe address: %s", config.ErrDependencyInit, err.Error())
	}
	timeout := cfg.ConnectTimeout
	if timeout <= 0 {
		timeout = mongoProbeBudget
	}
	return probeTCPDependency(ctx, addr, timeout)
}

func probeMongoWithRetry(ctx context.Context, cfg config.MongoConfig) error {
	return probeWithRetry(ctx, mongoProbeAttempts, func(probeCtx context.Context) error {
		return probeMongoWithContext(probeCtx, cfg)
	})
}

func probeWithRetry(ctx context.Context, maxAttempts int, probe func(context.Context) error) error {
	if maxAttempts <= 1 {
		return probe(ctx)
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := probe(ctx)
		if err == nil {
			return nil
		}
		lastErr = err
		if !shouldRetryStartupProbe(err, attempt, maxAttempts) {
			break
		}

		delay := fullJitterDelay(attempt)
		if waitErr := sleepWithContext(ctx, delay); waitErr != nil {
			return waitErr
		}
	}

	return lastErr
}

func shouldRetryStartupProbe(err error, attempt int, maxAttempts int) bool {
	if attempt >= maxAttempts {
		return false
	}
	return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
}

func probeTCPDependency(ctx context.Context, address string, timeout time.Duration) error {
	trimmedAddress := strings.TrimSpace(address)
	if trimmedAddress == "" {
		return fmt.Errorf("%w: empty probe address", config.ErrDependencyInit)
	}

	dialCtx, dialCancel := withStageBudget(ctx, timeout)
	defer dialCancel()

	var dialer net.Dialer
	conn, err := dialer.DialContext(dialCtx, "tcp", trimmedAddress)
	if err != nil {
		return fmt.Errorf("%w: dial %s: %w", config.ErrDependencyInit, trimmedAddress, err)
	}
	_ = conn.Close()
	return nil
}

func sleepWithContext(ctx context.Context, wait time.Duration) error {
	if wait <= 0 {
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
