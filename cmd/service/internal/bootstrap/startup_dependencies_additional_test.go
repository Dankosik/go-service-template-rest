package bootstrap

import (
	"context"
	"io"
	"log/slog"
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

func TestInitStartupDependenciesAllDisabled(t *testing.T) {
	t.Parallel()

	metrics := telemetry.New()
	runtime := dependencyProbeRuntime{
		tracer:                    otel.Tracer("test"),
		bootstrapSpan:             trace.SpanFromContext(context.Background()),
		cfg:                       config.Config{},
		metrics:                   metrics,
		log:                       slog.New(slog.NewJSONHandler(io.Discard, nil)),
		deployTelemetry:           newDeployTelemetryRecorder(slog.New(slog.NewJSONHandler(io.Discard, nil)), metrics, "test"),
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
