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

func TestRunDependencyProbe(t *testing.T) {
	t.Parallel()
	tracer := otel.Tracer("test")

	t.Run("budget blocked", func(t *testing.T) {
		t.Parallel()

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
	})

	t.Run("dependency local timeout keeps parent valid", func(t *testing.T) {
		t.Parallel()

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
	})

	t.Run("expired child deadline after nil probe result fails probe", func(t *testing.T) {
		t.Parallel()

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
	})

	t.Run("parent cancellation during probe records parent error", func(t *testing.T) {
		t.Parallel()

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
	})

	t.Run("probe success", func(t *testing.T) {
		t.Parallel()

		res := runDependencyProbe(context.Background(), tracer, dependencyProbeSpec{
			stage:        "stage",
			dep:          "dep",
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

func TestDependencyInitAbortFailureDoesNotDuplicateDependencyInitSentinel(t *testing.T) {
	t.Parallel()

	cause := fmt.Errorf("%w: startup.probe.postgres aborted", errDependencyInit)
	err := dependencyInitAbortFailure("postgres", probeExecutionResult{budgetBlocked: true, err: cause})
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
	if !strings.Contains(err.Error(), "postgres init skipped") {
		t.Fatalf("dependencyInitAbortFailure() error = %v, want skipped context", err)
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

	runtime := dependencyProbeRuntime{
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
