package bootstrap

import (
	"context"
	"errors"
	"fmt"
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
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		res := runDependencyProbe(ctx, tracer, dependencyProbeSpec{
			stage:        "stage",
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
		if res.err == nil {
			t.Fatal("err = nil, want budget error")
		}
		if res.parentErr != nil {
			t.Fatalf("parentErr = %v, want nil for low remaining startup budget", res.parentErr)
		}
		if !shouldAbortDegradedDependencyStartup(res) {
			t.Fatal("shouldAbortDegradedDependencyStartup() = false, want true")
		}
	})

	t.Run("dependency local timeout keeps parent valid", func(t *testing.T) {
		res := runDependencyProbe(context.Background(), tracer, dependencyProbeSpec{
			stage:        "stage",
			dep:          "dep",
			budget:       time.Second,
			minRemaining: 0,
			probe: func(context.Context) error {
				return fmt.Errorf("dependency-local timeout: %w", context.DeadlineExceeded)
			},
		})
		if res.budgetBlocked {
			t.Fatal("budgetBlocked = true, want false")
		}
		if res.parentErr != nil {
			t.Fatalf("parentErr = %v, want nil", res.parentErr)
		}
		if !errors.Is(res.err, context.DeadlineExceeded) {
			t.Fatalf("err = %v, want wrapped %v", res.err, context.DeadlineExceeded)
		}
		if shouldAbortDegradedDependencyStartup(res) {
			t.Fatal("shouldAbortDegradedDependencyStartup() = true, want false for dependency-local timeout")
		}
	})

	t.Run("expired child deadline after nil probe result fails probe", func(t *testing.T) {
		res := runDependencyProbe(context.Background(), tracer, dependencyProbeSpec{
			stage:        "stage",
			dep:          "dep",
			budget:       time.Millisecond,
			minRemaining: 0,
			probe: func(probeCtx context.Context) error {
				<-probeCtx.Done()
				return nil
			},
		})
		if res.budgetBlocked {
			t.Fatal("budgetBlocked = true, want false")
		}
		if res.parentErr != nil {
			t.Fatalf("parentErr = %v, want nil", res.parentErr)
		}
		if !errors.Is(res.err, context.DeadlineExceeded) {
			t.Fatalf("err = %v, want wrapped %v", res.err, context.DeadlineExceeded)
		}
		if shouldAbortDegradedDependencyStartup(res) {
			t.Fatal("shouldAbortDegradedDependencyStartup() = true, want false for dependency-local timeout")
		}
	})

	t.Run("parent cancellation during probe aborts degraded startup", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		res := runDependencyProbe(ctx, tracer, dependencyProbeSpec{
			stage:        "stage",
			dep:          "dep",
			budget:       time.Second,
			minRemaining: 0,
			probe: func(probeCtx context.Context) error {
				cancel()
				<-probeCtx.Done()
				return fmt.Errorf("parent canceled: %w", probeCtx.Err())
			},
		})
		if res.budgetBlocked {
			t.Fatal("budgetBlocked = true, want false")
		}
		if !errors.Is(res.parentErr, context.Canceled) {
			t.Fatalf("parentErr = %v, want wrapped %v", res.parentErr, context.Canceled)
		}
		if !errors.Is(res.err, context.Canceled) {
			t.Fatalf("err = %v, want wrapped %v", res.err, context.Canceled)
		}
		if !shouldAbortDegradedDependencyStartup(res) {
			t.Fatal("shouldAbortDegradedDependencyStartup() = false, want true")
		}
	})

	t.Run("probe success", func(t *testing.T) {
		res := runDependencyProbe(context.Background(), tracer, dependencyProbeSpec{
			stage:        "stage",
			dep:          "dep",
			mode:         "cache",
			budget:       time.Second,
			minRemaining: 0,
			probe: func(context.Context) error {
				return nil
			},
		})
		if res.budgetBlocked || res.err != nil {
			t.Fatalf("unexpected result: %+v", res)
		}
	})
}

func TestStartupDependencyProbeLabelsUseCanonicalProbeStage(t *testing.T) {
	t.Parallel()

	labels := newStartupDependencyProbeLabels("redis")
	if labels.resolveStage != "startup.resolve.redis" {
		t.Fatalf("resolveStage = %q, want %q", labels.resolveStage, "startup.resolve.redis")
	}
	if labels.probeStage != "startup.probe.redis" {
		t.Fatalf("probeStage = %q, want %q", labels.probeStage, "startup.probe.redis")
	}
	if labels.operation != "redis_probe" {
		t.Fatalf("operation = %q, want %q", labels.operation, "redis_probe")
	}
}

func TestDependencyInitFailurePreservesWrappedCause(t *testing.T) {
	t.Parallel()

	rootCause := errors.New("dial tcp 127.0.0.1:6379: connect refused")
	err := dependencyInitFailure("redis", rootCause)
	if err == nil {
		t.Fatal("dependencyInitFailure() error = nil, want non-nil")
	}
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("error = %v, want wrapped %v", err, errDependencyInit)
	}
	if !errors.Is(err, rootCause) {
		t.Fatalf("error = %v, want wrapped root cause", err)
	}
}

func TestDependencyInitFailureDoesNotDuplicateDependencyInitSentinel(t *testing.T) {
	t.Parallel()

	cause := fmt.Errorf("%w: dial failed", errDependencyInit)
	err := dependencyInitFailure("redis", cause)
	if err == nil {
		t.Fatal("dependencyInitFailure() error = nil, want non-nil")
	}
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("dependencyInitFailure() error = %v, want wrapped %v", err, errDependencyInit)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("dependencyInitFailure() error = %v, want wrapped cause", err)
	}
	if count := strings.Count(err.Error(), errDependencyInit.Error()); count != 1 {
		t.Fatalf("dependencyInitFailure() error = %v, dependency init count = %d, want 1", err, count)
	}
	if !strings.Contains(err.Error(), "redis init failed") {
		t.Fatalf("dependencyInitFailure() error = %v, want dependency context", err)
	}
}

func TestDependencyInitAbortFailureDoesNotDuplicateDependencyInitSentinel(t *testing.T) {
	t.Parallel()

	cause := fmt.Errorf("%w: startup.probe.redis aborted", errDependencyInit)
	err := dependencyInitAbortFailure("redis", probeExecutionResult{budgetBlocked: true, err: cause})
	if err == nil {
		t.Fatal("dependencyInitAbortFailure() error = nil, want non-nil")
	}
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("dependencyInitAbortFailure() error = %v, want wrapped %v", err, errDependencyInit)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("dependencyInitAbortFailure() error = %v, want wrapped cause", err)
	}
	if count := strings.Count(err.Error(), errDependencyInit.Error()); count != 1 {
		t.Fatalf("dependencyInitAbortFailure() error = %v, dependency init count = %d, want 1", err, count)
	}
	if !strings.Contains(err.Error(), "redis init skipped") {
		t.Fatalf("dependencyInitAbortFailure() error = %v, want skipped context", err)
	}
}

func TestInitRedisDependencyAddressErrorClassifiedAsDependencyInit(t *testing.T) {
	t.Parallel()

	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	runtime := dependencyProbeRuntime{
		tracer:        otel.Tracer("test"),
		bootstrapSpan: trace.SpanFromContext(context.Background()),
		metrics:       metrics,
		log:           logger,
		networkPolicy: networkPolicy{egressAllowedSchemes: map[string]struct{}{"tcp": {}}},
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
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("initRedisDependency() error = %v, want wrapped %v", err, errDependencyInit)
	}

	metricsText := collectServiceMetricsText(t, metrics)
	assertStartupRejectionMetric(t, metricsText, telemetry.StartupRejectionReasonDependencyInit)
	assertConfigFailureMetricAbsent(t, metricsText, telemetry.StartupRejectionReasonDependencyInit)
	assertConfigFailureMetricAbsent(t, metricsText, telemetry.StartupRejectionReasonPolicyViolation)
}

func TestInitRedisDependencyPolicyDenialRemainsPolicyViolation(t *testing.T) {
	t.Parallel()

	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	runtime := dependencyProbeRuntime{
		tracer:        otel.Tracer("test"),
		bootstrapSpan: trace.SpanFromContext(context.Background()),
		metrics:       metrics,
		log:           logger,
		networkPolicy: networkPolicy{egressAllowedSchemes: map[string]struct{}{"tcp": {}}},
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
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("initRedisDependency() error = %v, want wrapped %v", err, errDependencyInit)
	}

	metricsText := collectServiceMetricsText(t, metrics)
	assertStartupRejectionMetric(t, metricsText, telemetry.StartupRejectionReasonPolicyViolation)
	assertConfigFailureMetricAbsent(t, metricsText, telemetry.StartupRejectionReasonPolicyViolation)
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
		tracer:        otel.Tracer("test"),
		bootstrapSpan: trace.SpanFromContext(context.Background()),
		metrics:       metrics,
		log:           logger,
		networkPolicy: networkPolicy{egressAllowedSchemes: map[string]struct{}{"tcp": {}}},
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

func TestPostgresRuntimeReadinessProbeCapsContextDeadline(t *testing.T) {
	t.Parallel()

	const budget = 150 * time.Millisecond
	var captured context.Context
	probe := newPostgresReadinessProbe(testProbe{
		name: "postgres",
		check: func(ctx context.Context) error {
			captured = ctx
			deadline, ok := ctx.Deadline()
			if !ok {
				t.Fatal("probe context has no deadline, want healthcheck budget deadline")
			}
			remaining := time.Until(deadline)
			if remaining <= 0 {
				t.Fatalf("probe context remaining deadline = %s, want positive", remaining)
			}
			if remaining > budget+25*time.Millisecond {
				t.Fatalf("probe context remaining deadline = %s, want <= %s", remaining, budget)
			}
			if remaining < budget/2 {
				t.Fatalf("probe context remaining deadline = %s, want near %s", remaining, budget)
			}
			return nil
		},
	}, budget)

	parent, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if got := probe.Name(); got != "postgres" {
		t.Fatalf("probe.Name() = %q, want postgres", got)
	}
	if err := probe.Check(parent); err != nil {
		t.Fatalf("probe.Check() error = %v, want nil", err)
	}
	select {
	case <-captured.Done():
	default:
		t.Fatal("probe context was not canceled after Check returned")
	}
}

func TestPostgresRuntimeReadinessProbeDoesNotExtendShorterParentDeadline(t *testing.T) {
	t.Parallel()

	parent, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	parentDeadline, ok := parent.Deadline()
	if !ok {
		t.Fatal("parent context has no deadline")
	}

	probe := newPostgresReadinessProbe(testProbe{
		name: "postgres",
		check: func(ctx context.Context) error {
			childDeadline, ok := ctx.Deadline()
			if !ok {
				t.Fatal("probe context has no deadline, want parent deadline")
			}
			if childDeadline.After(parentDeadline.Add(time.Millisecond)) {
				t.Fatalf("probe deadline = %s, want no later than parent deadline %s", childDeadline, parentDeadline)
			}
			if remaining := time.Until(childDeadline); remaining <= 0 {
				t.Fatalf("probe context remaining deadline = %s, want positive", remaining)
			}
			return nil
		},
	}, time.Second)

	if err := probe.Check(parent); err != nil {
		t.Fatalf("probe.Check() error = %v, want nil", err)
	}
}

func TestPostgresRuntimeReadinessProbeFailsAfterChildDeadlineWithNilProbeResult(t *testing.T) {
	t.Parallel()

	probe := newPostgresReadinessProbe(testProbe{
		name: "postgres",
		check: func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		},
	}, time.Millisecond)

	if err := probe.Check(context.Background()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("probe.Check() error = %v, want wrapped %v", err, context.DeadlineExceeded)
	}
}

func TestInitStartupDependenciesAllDisabled(t *testing.T) {
	t.Parallel()

	metrics := telemetry.New()
	runtime := dependencyProbeRuntime{
		tracer:        otel.Tracer("test"),
		bootstrapSpan: trace.SpanFromContext(context.Background()),
		cfg:           config.Config{},
		metrics:       metrics,
		log:           slog.New(slog.NewJSONHandler(io.Discard, nil)),
		networkPolicy: networkPolicy{},
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

func TestInitMongoDependencyRecordsDegradedStatusMetric(t *testing.T) {
	t.Parallel()

	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	runtime := dependencyProbeRuntime{
		tracer:        otel.Tracer("test"),
		bootstrapSpan: trace.SpanFromContext(context.Background()),
		metrics:       metrics,
		log:           logger,
		networkPolicy: networkPolicy{egressAllowedSchemes: map[string]struct{}{"tcp": {}}},
		cfg: config.Config{
			Mongo: config.MongoConfig{
				Enabled:        true,
				URI:            "mongodb://127.0.0.1:1/app",
				ConnectTimeout: 10 * time.Millisecond,
			},
		},
	}

	probe, err := initMongoDependency(context.Background(), runtime, context.Background())
	if err != nil {
		t.Fatalf("initMongoDependency() error = %v, want nil degraded startup", err)
	}
	if probe != nil {
		t.Fatal("initMongoDependency() probe != nil, want nil without readiness flag")
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `startup_dependency_status{dep="mongo",mode="degraded_read_only_or_stale"} 1`) {
		t.Fatalf("missing mongo degraded status:\n%s", metricsText)
	}
}

func TestInitRedisCacheDependencyRecordsFeatureOffWhenReadinessNotRequired(t *testing.T) {
	t.Parallel()

	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	runtime := dependencyProbeRuntime{
		tracer:        otel.Tracer("test"),
		bootstrapSpan: trace.SpanFromContext(context.Background()),
		metrics:       metrics,
		log:           logger,
		networkPolicy: networkPolicy{egressAllowedSchemes: map[string]struct{}{"tcp": {}}},
		cfg: config.Config{
			Redis: config.RedisConfig{
				Enabled:     true,
				Mode:        config.RedisModeCache,
				Addr:        "127.0.0.1:1",
				DialTimeout: 10 * time.Millisecond,
			},
		},
	}

	probe, err := initRedisDependency(context.Background(), runtime, context.Background())
	if err != nil {
		t.Fatalf("initRedisDependency() error = %v, want nil degraded startup", err)
	}
	if probe != nil {
		t.Fatal("initRedisDependency() probe != nil, want nil without readiness flag")
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `startup_dependency_status{dep="redis",mode="feature_off"} 1`) {
		t.Fatalf("missing redis feature-off degraded status:\n%s", metricsText)
	}
}

func TestReadinessRequiredDegradedDependenciesRejectStartup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  config.Config
		run  func(dependencyProbeRuntime) error
	}{
		{
			name: "redis cache readiness required",
			cfg: config.Config{
				Redis: config.RedisConfig{
					Enabled:     true,
					Mode:        config.RedisModeCache,
					Addr:        "127.0.0.1:1",
					DialTimeout: 10 * time.Millisecond,
				},
				FeatureFlags: config.FeatureFlagsConfig{
					RedisReadinessProbe: true,
				},
			},
			run: func(runtime dependencyProbeRuntime) error {
				_, err := initRedisDependency(context.Background(), runtime, context.Background())
				return err
			},
		},
		{
			name: "redis store mode readiness required",
			cfg: config.Config{
				Redis: config.RedisConfig{
					Enabled:     true,
					Mode:        config.RedisModeStore,
					Addr:        "127.0.0.1:1",
					DialTimeout: 10 * time.Millisecond,
				},
			},
			run: func(runtime dependencyProbeRuntime) error {
				_, err := initRedisDependency(context.Background(), runtime, context.Background())
				return err
			},
		},
		{
			name: "mongo readiness required",
			cfg: config.Config{
				Mongo: config.MongoConfig{
					Enabled:        true,
					URI:            "mongodb://127.0.0.1:1/app",
					ConnectTimeout: 10 * time.Millisecond,
				},
				FeatureFlags: config.FeatureFlagsConfig{
					MongoReadinessProbe: true,
				},
			},
			run: func(runtime dependencyProbeRuntime) error {
				_, err := initMongoDependency(context.Background(), runtime, context.Background())
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := telemetry.New()
			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			runtime := dependencyProbeRuntime{
				tracer:        otel.Tracer("test"),
				bootstrapSpan: trace.SpanFromContext(context.Background()),
				metrics:       metrics,
				log:           logger,
				networkPolicy: networkPolicy{egressAllowedSchemes: map[string]struct{}{"tcp": {}}},
				cfg:           tt.cfg,
			}

			err := tt.run(runtime)
			if err == nil {
				t.Fatal("dependency init error = nil, want readiness-required startup rejection")
			}
			if !errors.Is(err, errDependencyInit) {
				t.Fatalf("dependency init error = %v, want wrapped %v", err, errDependencyInit)
			}

			metricsText := collectServiceMetricsText(t, metrics)
			assertStartupRejectionMetric(t, metricsText, telemetry.StartupRejectionReasonDependencyInit)
		})
	}
}

func TestDependencyCleanupStackRunsInReverseOrder(t *testing.T) {
	t.Parallel()

	var closed []string
	stack := dependencyCleanupStack{}
	stack.add(func() { closed = append(closed, "postgres") })
	stack.add(func() { closed = append(closed, "redis") })

	stack.run()

	if got := strings.Join(closed, ","); got != "redis,postgres" {
		t.Fatalf("cleanup order = %q, want %q", got, "redis,postgres")
	}
	stack.run()
	if got := strings.Join(closed, ","); got != "redis,postgres" {
		t.Fatalf("cleanup rerun changed order = %q", got)
	}
}

func TestDegradedDependenciesAbortOnCanceledStartup(t *testing.T) {
	t.Parallel()

	metrics := telemetry.New()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	runtime := dependencyProbeRuntime{
		tracer:        otel.Tracer("test"),
		bootstrapSpan: trace.SpanFromContext(context.Background()),
		metrics:       metrics,
		log:           logger,
		networkPolicy: networkPolicy{egressAllowedSchemes: map[string]struct{}{"tcp": {}}},
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
		tracer:        otel.Tracer("test"),
		bootstrapSpan: trace.SpanFromContext(context.Background()),
		metrics:       metrics,
		log:           logger,
		networkPolicy: networkPolicy{egressAllowedSchemes: map[string]struct{}{"tcp": {}}},
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
		tracer:        otel.Tracer("test"),
		bootstrapSpan: trace.SpanFromContext(context.Background()),
		metrics:       metrics,
		log:           logger,
		networkPolicy: networkPolicy{egressAllowedSchemes: map[string]struct{}{"tcp": {}}},
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
		if !errors.Is(err, errDependencyInit) {
			t.Fatalf("initRedisDependency() error = %v, want wrapped %v", err, errDependencyInit)
		}
		if !strings.Contains(err.Error(), "low remaining startup budget") {
			t.Fatalf("initRedisDependency() error = %v, want low-budget context", err)
		}
		if !strings.Contains(err.Error(), "startup.probe.redis") {
			t.Fatalf("initRedisDependency() error = %v, want canonical probe stage", err)
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
		if !errors.Is(err, errDependencyInit) {
			t.Fatalf("initMongoDependency() error = %v, want wrapped %v", err, errDependencyInit)
		}
		if !strings.Contains(err.Error(), "low remaining startup budget") {
			t.Fatalf("initMongoDependency() error = %v, want low-budget context", err)
		}
		if !strings.Contains(err.Error(), "startup.probe.mongo") {
			t.Fatalf("initMongoDependency() error = %v, want canonical probe stage", err)
		}
	})
}

type testProbe struct {
	name  string
	check func(context.Context) error
}

func (p testProbe) Name() string {
	return p.name
}

func (p testProbe) Check(ctx context.Context) error {
	return p.check(ctx)
}
