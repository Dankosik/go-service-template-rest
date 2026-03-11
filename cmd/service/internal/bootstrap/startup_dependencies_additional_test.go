package bootstrap

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func TestRunDependencyProbe(t *testing.T) {
	t.Parallel()
	tracer := otel.Tracer("test")

	t.Run("budget blocked", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		defer cancel()
		res := runDependencyProbe(ctx, tracer, dependencyProbeSpec{
			stage:        "stage",
			spanName:     "span",
			dep:          "dep",
			budget:       time.Second,
			minRemaining: time.Hour,
			probe: func(context.Context) error {
				return nil
			},
		})
		if !res.budgetBlocked {
			t.Fatal("budgetBlocked = false, want true")
		}
		if !res.failed {
			t.Fatal("failed = false, want true")
		}
	})

	t.Run("probe success", func(t *testing.T) {
		res := runDependencyProbe(context.Background(), tracer, dependencyProbeSpec{
			stage:        "stage",
			spanName:     "span",
			dep:          "dep",
			mode:         "cache",
			budget:       time.Second,
			minRemaining: 0,
			probe: func(context.Context) error {
				return nil
			},
		})
		if res.budgetBlocked || res.failed || res.err != nil {
			t.Fatalf("unexpected result: %+v", res)
		}
	})
}

func TestDependencyInitFailurePreservesWrappedCause(t *testing.T) {
	t.Parallel()

	rootCause := errors.New("dial tcp 127.0.0.1:6379: connect refused")
	err := dependencyInitFailure("redis", rootCause)
	if err == nil {
		t.Fatal("dependencyInitFailure() error = nil, want non-nil")
	}
	if !errors.Is(err, config.ErrDependencyInit) {
		t.Fatalf("error = %v, want wrapped %v", err, config.ErrDependencyInit)
	}
	if !errors.Is(err, rootCause) {
		t.Fatalf("error = %v, want wrapped root cause", err)
	}
}

func TestInitRedisDependencyAddressErrorClassifiedAsDependencyInit(t *testing.T) {
	t.Parallel()

	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	runtime := dependencyProbeRuntime{
		tracer:                    otel.Tracer("test"),
		bootstrapSpan:             trace.SpanFromContext(context.Background()),
		metrics:                   metrics,
		log:                       logger,
		networkPolicy:             networkPolicy{egressAllowedSchemes: map[string]struct{}{"tcp": {}}},
		startupLifecycleStartedAt: time.Now(),
		cfg: config.Config{
			Redis: config.RedisConfig{
				Enabled: true,
				Mode:    "cache",
			},
		},
	}

	_, err := initRedisDependency(context.Background(), runtime, context.Background())
	if err == nil {
		t.Fatal("initRedisDependency() error = nil, want non-nil")
	}
	if !errors.Is(err, config.ErrDependencyInit) {
		t.Fatalf("initRedisDependency() error = %v, want wrapped %v", err, config.ErrDependencyInit)
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `config_validation_failures_total{reason="dependency_init"} 1`) {
		t.Fatalf("metrics output missing dependency_init classification:\n%s", metricsText)
	}
	if strings.Contains(metricsText, `config_validation_failures_total{reason="policy_violation"}`) {
		t.Fatalf("metrics output unexpectedly contains policy_violation classification:\n%s", metricsText)
	}
}

func TestInitRedisDependencyPolicyDenialRemainsPolicyViolation(t *testing.T) {
	t.Parallel()

	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	runtime := dependencyProbeRuntime{
		tracer:                    otel.Tracer("test"),
		bootstrapSpan:             trace.SpanFromContext(context.Background()),
		metrics:                   metrics,
		log:                       logger,
		networkPolicy:             networkPolicy{egressAllowedSchemes: map[string]struct{}{"tcp": {}}},
		startupLifecycleStartedAt: time.Now(),
		cfg: config.Config{
			Redis: config.RedisConfig{
				Enabled: true,
				Mode:    "cache",
				Addr:    "api.example.com:6379",
			},
		},
	}

	_, err := initRedisDependency(context.Background(), runtime, context.Background())
	if err == nil {
		t.Fatal("initRedisDependency() error = nil, want non-nil")
	}
	if !errors.Is(err, config.ErrDependencyInit) {
		t.Fatalf("initRedisDependency() error = %v, want wrapped %v", err, config.ErrDependencyInit)
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `config_validation_failures_total{reason="policy_violation"} 1`) {
		t.Fatalf("metrics output missing policy_violation classification:\n%s", metricsText)
	}
}

func TestInitRedisDependencyAddsRuntimeReadinessProbeForStoreMode(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	t.Cleanup(func() {
		_ = ln.Close()
	})

	go func() {
		for i := 0; i < 2; i++ {
			conn, acceptErr := ln.Accept()
			if acceptErr != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	runtime := dependencyProbeRuntime{
		tracer:                    otel.Tracer("test"),
		bootstrapSpan:             trace.SpanFromContext(context.Background()),
		metrics:                   metrics,
		log:                       logger,
		networkPolicy:             networkPolicy{egressAllowedSchemes: map[string]struct{}{"tcp": {}}},
		startupLifecycleStartedAt: time.Now(),
		cfg: config.Config{
			Redis: config.RedisConfig{
				Enabled:     true,
				Mode:        "store",
				Addr:        ln.Addr().String(),
				DialTimeout: 100 * time.Millisecond,
			},
		},
	}

	probe, err := initRedisDependency(context.Background(), runtime, context.Background())
	if err != nil {
		t.Fatalf("initRedisDependency() error = %v, want nil", err)
	}
	if probe == nil {
		t.Fatal("initRedisDependency() probe = nil, want runtime readiness probe")
	}
	if probe.Name() != "redis" {
		t.Fatalf("probe.Name() = %q, want %q", probe.Name(), "redis")
	}
	if err := probe.Check(context.Background()); err != nil {
		t.Fatalf("probe.Check() error = %v, want nil", err)
	}
}

func TestInitStartupDependenciesAllDisabled(t *testing.T) {
	t.Parallel()

	metrics := telemetry.New()
	runtime := dependencyProbeRuntime{
		tracer:                    otel.Tracer("test"),
		bootstrapSpan:             trace.SpanFromContext(context.Background()),
		cfg:                       config.Config{},
		metrics:                   metrics,
		log:                       slog.New(slog.NewJSONHandler(io.Discard, nil)),
		networkPolicy:             networkPolicy{},
		startupLifecycleStartedAt: time.Now(),
	}

	outcome, err := initStartupDependencies(context.Background(), context.Background(), runtime)
	if err != nil {
		t.Fatalf("initStartupDependencies() error = %v, want nil", err)
	}
	if len(outcome.probes) != 0 {
		t.Fatalf("probes len = %d, want 0", len(outcome.probes))
	}
	if outcome.postgresPool != nil {
		t.Fatal("postgresPool != nil, want nil")
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `startup_dependency_status{dep="postgres",mode="disabled"} 1`) {
		t.Fatalf("missing postgres disabled status:\n%s", metricsText)
	}
	if !strings.Contains(metricsText, `startup_dependency_status{dep="redis",mode="disabled"} 1`) {
		t.Fatalf("missing redis disabled status:\n%s", metricsText)
	}
	if !strings.Contains(metricsText, `startup_dependency_status{dep="mongo",mode="disabled"} 1`) {
		t.Fatalf("missing mongo disabled status:\n%s", metricsText)
	}
}

func TestDegradedDependenciesAbortOnCanceledStartup(t *testing.T) {
	t.Parallel()

	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	runtime := dependencyProbeRuntime{
		tracer:                    otel.Tracer("test"),
		bootstrapSpan:             trace.SpanFromContext(context.Background()),
		metrics:                   metrics,
		log:                       logger,
		networkPolicy:             networkPolicy{egressAllowedSchemes: map[string]struct{}{"tcp": {}}},
		startupLifecycleStartedAt: time.Now(),
	}

	t.Run("redis cache", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		runtime.cfg = config.Config{
			Redis: config.RedisConfig{
				Enabled: true,
				Mode:    "cache",
				Addr:    "127.0.0.1:6379",
			},
		}
		_, err := initRedisDependency(context.Background(), runtime, ctx)
		if err == nil {
			t.Fatal("initRedisDependency() error = nil, want non-nil")
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("initRedisDependency() error = %v, want wrapped %v", err, context.Canceled)
		}
	})

	t.Run("mongo degraded", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		runtime.cfg = config.Config{
			Mongo: config.MongoConfig{
				Enabled: true,
				URI:     "mongodb://127.0.0.1:27017/app",
			},
		}
		_, err := initMongoDependency(context.Background(), runtime, ctx)
		if err == nil {
			t.Fatal("initMongoDependency() error = nil, want non-nil")
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("initMongoDependency() error = %v, want wrapped %v", err, context.Canceled)
		}
	})
}

func TestDegradedDependenciesAbortOnExpiredStartupDeadline(t *testing.T) {
	t.Parallel()

	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	runtime := dependencyProbeRuntime{
		tracer:                    otel.Tracer("test"),
		bootstrapSpan:             trace.SpanFromContext(context.Background()),
		metrics:                   metrics,
		log:                       logger,
		networkPolicy:             networkPolicy{egressAllowedSchemes: map[string]struct{}{"tcp": {}}},
		startupLifecycleStartedAt: time.Now(),
	}

	expiredCtx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	t.Run("redis cache", func(t *testing.T) {
		runtime.cfg = config.Config{
			Redis: config.RedisConfig{
				Enabled: true,
				Mode:    "cache",
				Addr:    "127.0.0.1:6379",
			},
		}
		_, err := initRedisDependency(context.Background(), runtime, expiredCtx)
		if err == nil {
			t.Fatal("initRedisDependency() error = nil, want non-nil")
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("initRedisDependency() error = %v, want wrapped %v", err, context.DeadlineExceeded)
		}
	})

	t.Run("mongo degraded", func(t *testing.T) {
		runtime.cfg = config.Config{
			Mongo: config.MongoConfig{
				Enabled: true,
				URI:     "mongodb://127.0.0.1:27017/app",
			},
		}
		_, err := initMongoDependency(context.Background(), runtime, expiredCtx)
		if err == nil {
			t.Fatal("initMongoDependency() error = nil, want non-nil")
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("initMongoDependency() error = %v, want wrapped %v", err, context.DeadlineExceeded)
		}
	})
}

func TestDegradedDependenciesAbortOnLowRemainingStartupBudget(t *testing.T) {
	t.Parallel()

	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	runtime := dependencyProbeRuntime{
		tracer:                    otel.Tracer("test"),
		bootstrapSpan:             trace.SpanFromContext(context.Background()),
		metrics:                   metrics,
		log:                       logger,
		networkPolicy:             networkPolicy{egressAllowedSchemes: map[string]struct{}{"tcp": {}}},
		startupLifecycleStartedAt: time.Now(),
	}

	newLowBudgetCtx := func(t *testing.T) (context.Context, context.CancelFunc) {
		t.Helper()
		// Use a fresh per-subtest deadline that is below the fail-fast threshold
		// without already being expired under slower race/package-parallel runs.
		return context.WithDeadline(context.Background(), time.Now().Add(startupFailFastThreshold-time.Millisecond))
	}

	t.Run("redis cache", func(t *testing.T) {
		lowBudgetCtx, cancel := newLowBudgetCtx(t)
		defer cancel()

		runtime.cfg = config.Config{
			Redis: config.RedisConfig{
				Enabled: true,
				Mode:    "cache",
				Addr:    "127.0.0.1:6379",
			},
		}
		_, err := initRedisDependency(context.Background(), runtime, lowBudgetCtx)
		if err == nil {
			t.Fatal("initRedisDependency() error = nil, want non-nil")
		}
		if !errors.Is(err, config.ErrDependencyInit) {
			t.Fatalf("initRedisDependency() error = %v, want wrapped %v", err, config.ErrDependencyInit)
		}
		if !strings.Contains(err.Error(), "low remaining startup budget") {
			t.Fatalf("initRedisDependency() error = %v, want low-budget context", err)
		}
	})

	t.Run("mongo degraded", func(t *testing.T) {
		lowBudgetCtx, cancel := newLowBudgetCtx(t)
		defer cancel()

		runtime.cfg = config.Config{
			Mongo: config.MongoConfig{
				Enabled: true,
				URI:     "mongodb://127.0.0.1:27017/app",
			},
		}
		_, err := initMongoDependency(context.Background(), runtime, lowBudgetCtx)
		if err == nil {
			t.Fatal("initMongoDependency() error = nil, want non-nil")
		}
		if !errors.Is(err, config.ErrDependencyInit) {
			t.Fatalf("initMongoDependency() error = %v, want wrapped %v", err, config.ErrDependencyInit)
		}
		if !strings.Contains(err.Error(), "low remaining startup budget") {
			t.Fatalf("initMongoDependency() error = %v, want low-budget context", err)
		}
	})
}
