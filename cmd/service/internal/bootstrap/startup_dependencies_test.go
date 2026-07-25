package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func TestPostgresDependencyInitFailurePreservesWrappedCause(t *testing.T) {
	t.Parallel()

	rootCause := errors.New("dial tcp 127.0.0.1:5432: connect refused")
	err := postgresDependencyInitFailure(rootCause)
	if err == nil {
		t.Fatal("postgresDependencyInitFailure() error = nil, want non-nil")
	}
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("error = %v, want wrapped %v", err, errDependencyInit)
	}
	if !errors.Is(err, rootCause) {
		t.Fatalf("error = %v, want wrapped root cause", err)
	}
}

func TestPostgresDependencyInitFailureDoesNotDuplicateDependencyInitSentinel(t *testing.T) {
	t.Parallel()

	cause := fmt.Errorf("%w: dial failed", errDependencyInit)
	err := postgresDependencyInitFailure(cause)
	if err == nil {
		t.Fatal("postgresDependencyInitFailure() error = nil, want non-nil")
	}
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("postgresDependencyInitFailure() error = %v, want wrapped %v", err, errDependencyInit)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("postgresDependencyInitFailure() error = %v, want wrapped cause", err)
	}
	if count := strings.Count(err.Error(), errDependencyInit.Error()); count != 1 {
		t.Fatalf("postgresDependencyInitFailure() error = %v, dependency init count = %d, want 1", err, count)
	}
	if !strings.Contains(err.Error(), "postgres init failed") {
		t.Fatalf("postgresDependencyInitFailure() error = %v, want dependency context", err)
	}
}

func TestPostgresRuntimeReadinessProbeCapsContextDeadline(t *testing.T) {
	t.Parallel()

	const budget = 150 * time.Millisecond
	var probeDone <-chan struct{}
	probe := newPostgresReadinessProbe(testProbe{
		name: "postgres",
		check: func(ctx context.Context) error {
			probeDone = ctx.Done()
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
	case <-probeDone:
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

	synctest.Test(t, func(t *testing.T) {
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
	})
}

func TestInitPostgresDependencyRejectsDisabledProfile(t *testing.T) {
	t.Parallel()

	runtime := postgresStartupRuntime{
		tracer:        otel.Tracer("test"),
		bootstrapSpan: trace.SpanFromContext(context.Background()),
		cfg:           config.Config{},
		log:           slog.New(slog.DiscardHandler),
	}

	pg, err := initPostgresDependency(context.Background(), context.Background(), runtime)
	if err == nil {
		t.Fatal("initPostgresDependency() error = nil, want required-profile rejection")
	}
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("initPostgresDependency() error = %v, want wrapped %v", err, errDependencyInit)
	}
	if !strings.Contains(err.Error(), "required by the DATABASE=postgres profile") {
		t.Fatalf("initPostgresDependency() error = %v, want profile context", err)
	}
	if pg != nil {
		t.Fatal("initPostgresDependency() pool != nil, want nil")
	}
}

func TestInitRuntimeDependenciesRejectsUnavailablePostgres(t *testing.T) {
	t.Parallel()

	startupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dependencies, err := initRuntimeDependencies(context.Background(), startupCtx, startupBootstrap{
		tracer:        otel.Tracer("test"),
		bootstrapSpan: trace.SpanFromContext(context.Background()),
		cfg: config.Config{
			Postgres: config.PostgresConfig{
				Enabled:            true,
				DSN:                "postgres://app:app@127.0.0.1:1/app?sslmode=disable",
				ConnectTimeout:     10 * time.Millisecond,
				HealthcheckTimeout: 10 * time.Millisecond,
				MaxOpenConns:       1,
				ConnMaxLifetime:    time.Minute,
				StatementTimeout:   time.Second,
			},
		},
		log: slog.New(slog.DiscardHandler),
	})
	dependencies.Close(context.Background())

	if err == nil {
		t.Fatal("initRuntimeDependencies() error = nil, want unavailable PostgreSQL rejection")
	}
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("initRuntimeDependencies() error = %v, want wrapped %v", err, errDependencyInit)
	}
}

// A cancelled dependency context must not be reported as a healthy pool. The
// stage no longer pre-checks remaining budget: the context deadline enforces the
// bound, and the pre-check only changed the error message.
func TestInitPostgresDependencyRejectsCancelledDependencyContext(t *testing.T) {
	t.Parallel()

	probeCtx, cancel := context.WithCancel(context.Background())
	cancel()

	runtime := postgresStartupRuntime{
		tracer:        otel.Tracer("test"),
		bootstrapSpan: trace.SpanFromContext(context.Background()),
		cfg: config.Config{Postgres: config.PostgresConfig{
			Enabled:            true,
			DSN:                "postgres://user:pass@localhost:5432/app?sslmode=disable",
			ConnectTimeout:     time.Second,
			HealthcheckTimeout: time.Second,
			MaxOpenConns:       1,
			ConnMaxLifetime:    time.Minute,
			StatementTimeout:   time.Second,
		}},
		log: slog.New(slog.DiscardHandler),
	}

	pool, err := initPostgresDependency(context.Background(), probeCtx, runtime)
	if err == nil {
		t.Fatal("initPostgresDependency() error = nil, want cancellation rejection")
	}
	if pool != nil {
		t.Fatal("initPostgresDependency() pool != nil, want no pool handed back on failure")
	}
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("initPostgresDependency() error = %v, want wrapped %v", err, errDependencyInit)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("initPostgresDependency() error = %v, want wrapped context.Canceled", err)
	}
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

func TestValidateStartupBudgetCompatibilityRejectsDependencyTimeoutsAboveProbeBudgets(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		cfg     config.Config
		wantKey string
	}{
		{
			name: "postgres connect timeout",
			cfg: config.Config{
				Postgres: config.PostgresConfig{
					Enabled:        true,
					ConnectTimeout: postgresProbeBudget + time.Nanosecond,
				},
			},
			wantKey: "postgres.connect_timeout",
		},
		{
			name: "postgres healthcheck timeout",
			cfg: config.Config{
				Postgres: config.PostgresConfig{
					Enabled:            true,
					ConnectTimeout:     postgresProbeBudget,
					HealthcheckTimeout: postgresProbeBudget + time.Nanosecond,
				},
			},
			wantKey: "postgres.healthcheck_timeout",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateStartupBudgetCompatibility(tc.cfg)
			if err == nil {
				t.Fatal("validateStartupBudgetCompatibility() error = nil, want validation error")
			}
			if !errors.Is(err, config.ErrValidate) {
				t.Fatalf("error = %v, want ErrValidate", err)
			}
			if !strings.Contains(err.Error(), tc.wantKey) {
				t.Fatalf("error = %v, want key %q", err, tc.wantKey)
			}
		})
	}
}

func TestValidateStartupBudgetCompatibilityIgnoresDisabledDependencies(t *testing.T) {
	t.Parallel()

	err := validateStartupBudgetCompatibility(config.Config{
		HTTP: config.HTTPConfig{ReadinessTimeout: time.Second},
		Postgres: config.PostgresConfig{
			ConnectTimeout:     postgresProbeBudget + time.Second,
			HealthcheckTimeout: postgresProbeBudget + time.Second,
		},
	})
	if err != nil {
		t.Fatalf("validateStartupBudgetCompatibility() error = %v, want nil for disabled dependencies", err)
	}
}

func TestValidateStartupBudgetCompatibilityRequiresReadinessHeadroom(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		HTTP: config.HTTPConfig{
			ReadinessTimeout: time.Second,
		},
		Postgres: config.PostgresConfig{
			Enabled:            true,
			HealthcheckTimeout: time.Second,
		},
	}

	err := validateStartupBudgetCompatibility(cfg)
	if err == nil {
		t.Fatal("validateStartupBudgetCompatibility() error = nil, want readiness headroom validation error")
	}
	if !errors.Is(err, config.ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "startup headroom") {
		t.Fatalf("error = %v, want startup headroom context", err)
	}
	if !strings.Contains(err.Error(), "postgres.healthcheck_timeout") {
		t.Fatalf("error = %v, want readiness probe name", err)
	}

	cfg.HTTP.ReadinessTimeout = time.Second + startupReadinessHeadroom
	if err := validateStartupBudgetCompatibility(cfg); err != nil {
		t.Fatalf("validateStartupBudgetCompatibility() error = %v, want nil when headroom is included", err)
	}
}

func TestValidateStartupBudgetCompatibilityAllowsDefaultPostgresReadiness(t *testing.T) {
	resetBootstrapConfigEnv(t)
	t.Setenv("APP__POSTGRES__ENABLED", "true")
	t.Setenv("APP__POSTGRES__DSN", "postgres://user:pass@localhost:5432/app?sslmode=disable")

	cfg, _, err := config.LoadDetailed(config.LoadOptions{})
	if err != nil {
		t.Fatalf("config.LoadDetailed() error = %v", err)
	}
	if cfg.HTTP.ReadinessTimeout != 4*time.Second {
		t.Fatalf("HTTP.ReadinessTimeout = %s, want 4s default", cfg.HTTP.ReadinessTimeout)
	}
	if cfg.Postgres.HealthcheckTimeout != 3*time.Second {
		t.Fatalf("Postgres.HealthcheckTimeout = %s, want 3s default", cfg.Postgres.HealthcheckTimeout)
	}

	if err := validateStartupBudgetCompatibility(cfg); err != nil {
		t.Fatalf("validateStartupBudgetCompatibility() error = %v, want nil for default Postgres readiness headroom", err)
	}
}

func TestBootstrapConfigStageReturnsStartupCompatibilityFailure(t *testing.T) {
	resetBootstrapConfigEnv(t)
	t.Setenv("APP__POSTGRES__ENABLED", "true")
	t.Setenv("APP__POSTGRES__DSN", "postgres://user:pass@localhost:5432/app?sslmode=disable")
	t.Setenv("APP__POSTGRES__CONNECT_TIMEOUT", "6s")

	_, _, err := bootstrapConfigStage(context.Background(), config.LoadOptions{})
	if err == nil {
		t.Fatal("bootstrapConfigStage() error = nil, want startup compatibility validation error")
	}
	if !errors.Is(err, config.ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
}

func resetBootstrapConfigEnv(t *testing.T) {
	t.Helper()

	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if !strings.HasPrefix(key, "APP__") && key != "APP_CONFIG_ALLOWED_ROOTS" {
			continue
		}
		t.Setenv(key, value)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("os.Unsetenv(%q) error = %v", key, err)
		}
	}
}
