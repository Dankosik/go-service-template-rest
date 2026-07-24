package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func TestDependencyInitFailurePreservesWrappedCause(t *testing.T) {
	t.Parallel()

	rootCause := errors.New("dial tcp 127.0.0.1:5432: connect refused")
	err := dependencyInitFailure("postgres", rootCause)
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
	err := dependencyInitFailure("postgres", cause)
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
	if !strings.Contains(err.Error(), "postgres init failed") {
		t.Fatalf("dependencyInitFailure() error = %v, want dependency context", err)
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

func TestInitStartupDependenciesAllDisabled(t *testing.T) {
	t.Parallel()

	runtime := postgresStartupRuntime{
		tracer:        otel.Tracer("test"),
		bootstrapSpan: trace.SpanFromContext(context.Background()),
		cfg:           config.Config{},
		log:           slog.New(slog.DiscardHandler),
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
}

func TestInitPostgresDependencyRejectsLowRemainingStartupBudget(t *testing.T) {
	t.Parallel()

	probeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	runtime := postgresStartupRuntime{
		tracer:        otel.Tracer("test"),
		bootstrapSpan: trace.SpanFromContext(context.Background()),
		cfg: config.Config{Postgres: config.PostgresConfig{
			Enabled: true,
			DSN:     "postgres://user:pass@localhost:5432/app?sslmode=disable",
		}},
		log: slog.New(slog.DiscardHandler),
		networkPolicy: networkPolicy{
			egressAllowedSchemes: map[string]struct{}{"tcp": {}},
		},
	}

	_, err := initPostgresDependency(context.Background(), probeCtx, runtime)
	if err == nil {
		t.Fatal("initPostgresDependency() error = nil, want low-budget rejection")
	}
	if !errors.Is(err, errDependencyInit) {
		t.Fatalf("initPostgresDependency() error = %v, want wrapped %v", err, errDependencyInit)
	}
	if !strings.Contains(err.Error(), "postgres init skipped") {
		t.Fatalf("initPostgresDependency() error = %v, want skipped context", err)
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
