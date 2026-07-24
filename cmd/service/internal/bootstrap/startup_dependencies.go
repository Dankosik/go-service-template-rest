package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var errDependencyInit = errors.New("dependency init")

const (
	startupDependencyPostgres     = "postgres"
	startupPostgresProbeOperation = "postgres_probe"
	startupPostgresResolveStage   = "startup.resolve.postgres"
	startupPostgresProbeStage     = "startup.probe.postgres"
)

const (
	startupDependencyTelemetry       = "telemetry"
	startupDependencyNetworkPolicy   = "network_policy"
	startupDependencyIngressPolicy   = "ingress_policy"
	startupDependencyEgressException = "egress_exception"
	startupDependencyModeFeatureOff  = "feature_off"
)

const (
	startupLogComponentStartupProbes = "startup_probes"
	startupLogComponentShutdown      = "shutdown"

	startupOperationTelemetryInit  = "telemetry_init"
	startupOperationTelemetryFlush = "telemetry_flush"
)

type (
	postgresConnectFunc    func(context.Context, postgres.Options) (*postgres.Pool, error)
	postgresRetryDelayFunc func(int) time.Duration
	postgresRetrySleepFunc func(context.Context, time.Duration) error
)

type postgresStartupRuntime struct {
	tracer        trace.Tracer
	bootstrapSpan trace.Span
	cfg           config.Config
	log           *slog.Logger
	networkPolicy networkPolicy
}

func postgresStartupProbeAddress(cfg config.PostgresConfig) (string, error) {
	address, err := postgres.ProbeAddress(cfg.DSN)
	if err != nil {
		return "", fmt.Errorf("%w: resolve postgres probe address: %w", errDependencyInit, err)
	}
	return address, nil
}

func initPostgresWithRetry(ctx context.Context, cfg config.PostgresConfig) (*postgres.Pool, error) {
	return initPostgresWithRetryFunc(ctx, cfg, postgres.New, fullJitterDelay, sleepWithContext)
}

func initPostgresWithRetryFunc(
	ctx context.Context,
	cfg config.PostgresConfig,
	connect postgresConnectFunc,
	delayFor postgresRetryDelayFunc,
	sleep postgresRetrySleepFunc,
) (*postgres.Pool, error) {
	options := postgres.Options{
		DSN:                cfg.DSN,
		ConnectTimeout:     cfg.ConnectTimeout,
		HealthcheckTimeout: cfg.HealthcheckTimeout,
		MaxOpenConns:       cfg.MaxOpenConns,
		ConnMaxLifetime:    cfg.ConnMaxLifetime,
	}

	var lastErr error
	for attempt := 1; attempt <= postgresStartupAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("%w: postgres init canceled: %w", errDependencyInit, err)
		}

		pg, err := connect(ctx, options)
		if err == nil {
			return pg, nil
		}

		lastErr = err
		if !shouldRetryPostgresStartup(err, attempt) {
			break
		}

		delay := delayFor(attempt)
		if err := sleep(ctx, delay); err != nil {
			return nil, fmt.Errorf("%w: postgres retry wait canceled: %w", errDependencyInit, err)
		}
	}

	return nil, fmt.Errorf("%w: postgres init failed after retries: %w", errDependencyInit, lastErr)
}

func shouldRetryPostgresStartup(err error, attempt int) bool {
	if attempt >= postgresStartupAttempts {
		return false
	}
	return errors.Is(err, postgres.ErrConnect) || errors.Is(err, postgres.ErrHealthcheck)
}

func fullJitterDelay(attempt int) time.Duration {
	backoff := startupRetryBaseDelay << (attempt - 1)
	backoff = min(backoff, startupRetryMaxDelay)
	if backoff <= 0 {
		return 0
	}

	return rand.N(backoff + 1) // #nosec G404 -- startup retry jitter is not security-sensitive.
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
		return nil, nil //nolint:nilnil // Disabled dependency intentionally has no pool and no startup error.
	}

	postgresProbeAddress, addressErr := postgresStartupProbeAddress(runtime.cfg.Postgres)
	if addressErr != nil {
		return nil, rejectPostgresStartupForDependencyInit(
			bootstrapCtx,
			runtime.bootstrapSpan,
			runtime.log,
			addressErr,
		)
	}
	if err := runtime.networkPolicy.EnforceEgressTarget(postgresProbeAddress, "tcp"); err != nil {
		return nil, rejectStartupForPolicyViolation(
			bootstrapCtx,
			runtime.bootstrapSpan,
			runtime.log,
			startupDependencyPostgres,
			err,
		)
	}

	if err := ensureRemainingStartupBudget(
		dependencyCtx,
		startupFailFastThreshold+startupReserveBudget,
		startupPostgresProbeStage,
	); err != nil {
		rejectErr := fmt.Errorf("%s init skipped: %w", startupDependencyPostgres, err)
		recordDependencyProbeRejection(bootstrapCtx, runtime, rejectErr)
		return nil, rejectErr
	}

	probeCtx, probeCancel := withStageBudget(dependencyCtx, postgresProbeBudget)
	probeCtx, probeSpan := runtime.tracer.Start(probeCtx, startupPostgresProbeStage)

	pg, probeErr := initPostgresWithRetry(probeCtx, runtime.cfg.Postgres)
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
