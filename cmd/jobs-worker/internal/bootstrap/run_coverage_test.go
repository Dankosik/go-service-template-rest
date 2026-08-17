package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/config/configtest"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/infra/postgresjobs"
	"github.com/example/go-service-template-rest/internal/infra/postgresmigrate"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/jobs"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestJobsWorkerRunBuildFailureStopsBeforePostgres(t *testing.T) {
	setJobsWorkerRunEnvironment(t)

	built := false
	want := errors.New("registry failure")
	err := run(t.Context(), nil, func(context.Context, config.Config, *slog.Logger) (*jobs.Registry, func(context.Context), error) {
		built = true
		return nil, nil, want
	})
	if !errors.Is(err, want) || !strings.Contains(err.Error(), "build jobs registry") {
		t.Fatalf("run() error = %v, want wrapped registry failure", err)
	}
	if !built {
		t.Fatal("run() did not build the registry after valid startup configuration")
	}
}

func TestJobsWorkerRunCleansUpRejectedRegistry(t *testing.T) {
	setJobsWorkerRunEnvironment(t)
	cleaned := make(chan struct{}, 1)
	err := run(t.Context(), nil, func(context.Context, config.Config, *slog.Logger) (*jobs.Registry, func(context.Context), error) {
		return nil, func(context.Context) { cleaned <- struct{}{} }, nil
	})
	if !errors.Is(err, postgresjobs.ErrConfig) || !strings.Contains(err.Error(), "registry is not registered") {
		t.Fatalf("run() error = %v, want missing registry", err)
	}
	waittest.ReceiveSignal(t, cleaned, time.Second, "rejected registry cleanup")
}

//nolint:paralleltest // This test mutates process-global environment or working directory.
func TestJobsWorkerRunRejectsEmptyRegistryBeforePostgres(t *testing.T) {
	setJobsWorkerRunEnvironment(t)
	t.Setenv("APP__POSTGRES__DSN", "postgres://jobs:password@127.0.0.1:1/jobs?sslmode=disable")
	err := run(t.Context(), nil, func(context.Context, config.Config, *slog.Logger) (*jobs.Registry, func(context.Context), error) {
		return new(jobs.Registry), nil, nil
	})
	if !errors.Is(err, postgresjobs.ErrConfig) || !strings.Contains(err.Error(), "no definitions") {
		t.Fatalf("run() error = %v, want empty registry rejection before PostgreSQL", err)
	}
}

//nolint:paralleltest // This test mutates process-global environment or working directory.
func TestJobsWorkerRunReportsPostgresStartupFailure(t *testing.T) {
	setJobsWorkerRunEnvironment(t)
	t.Setenv("APP__POSTGRES__DSN", "postgres://jobs:password@127.0.0.1:1/jobs?sslmode=disable")
	err := run(t.Context(), nil, func(context.Context, config.Config, *slog.Logger) (*jobs.Registry, func(context.Context), error) {
		return jobsWorkerRegistry(t, time.Second), nil, nil
	})
	if err == nil || !strings.Contains(err.Error(), "initialize jobs worker postgres") {
		t.Fatalf("run() error = %v, want postgres startup failure", err)
	}
}

//nolint:paralleltest // This test mutates process-global environment or working directory.
func TestJobsWorkerRunRejectsDefinitionEnvelopeBeforePostgres(t *testing.T) {
	setJobsWorkerRunEnvironment(t)
	t.Setenv("APP__POSTGRES__DSN", "postgres://jobs:password@127.0.0.1:1/jobs?sslmode=disable")
	registry := jobsWorkerRegistry(t, 2*time.Minute)
	err := run(t.Context(), nil, func(context.Context, config.Config, *slog.Logger) (*jobs.Registry, func(context.Context), error) {
		return registry, nil, nil
	})
	if !errors.Is(err, postgresjobs.ErrConfig) || !strings.Contains(err.Error(), "termination envelope") {
		t.Fatalf("run() error = %v, want definition envelope rejection before PostgreSQL", err)
	}
}

//nolint:paralleltest // This test mutates process-global environment or working directory.
func TestJobsWorkerRunStartsAndDrains(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, pgtest.DefaultImage,
		tcpostgres.WithDatabase("app"), tcpostgres.WithUsername("app"), tcpostgres.WithPassword("app"), tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Skipf("start PostgreSQL: %v", err)
	}
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(container) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("PostgreSQL connection string: %v", err)
	}
	if _, err := postgresmigrate.MigrateUp(ctx, postgresmigrate.MigrationOptions{
		DSN: dsn, SourceFS: os.DirFS("../../../../"), SourcePath: "migrations", ConnectTimeout: 3 * time.Second,
		StatementTimeout: time.Minute, LockTimeout: 15 * time.Second, CleanupTimeout: 15 * time.Second,
	}); err != nil {
		t.Fatalf("migrate PostgreSQL: %v", err)
	}

	setJobsWorkerRunEnvironment(t)
	t.Setenv("APP__POSTGRES__DSN", dsn)
	diagnosticsAddr := os.Getenv("APP__OBSERVABILITY__METRICS__ADDR")
	runCtx, cancelRun := context.WithCancel(t.Context())
	defer cancelRun()
	result := make(chan error, 1)
	go func() {
		result <- run(runCtx, nil, func(context.Context, config.Config, *slog.Logger) (*jobs.Registry, func(context.Context), error) {
			return jobsWorkerRegistry(t, time.Second), nil, nil
		})
	}()

	client := &http.Client{Timeout: time.Second}
	waittest.Until(t, 10*time.Second, func() bool {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+diagnosticsAddr+"/health/ready", http.NoBody)
		if err != nil {
			return false
		}
		response, err := client.Do(request)
		if err != nil {
			return false
		}
		defer func() { _ = response.Body.Close() }()
		return response.StatusCode == http.StatusOK
	}, "jobs worker readiness")
	cancelRun()
	if err := waittest.Receive(t, result, 10*time.Second, "jobs worker drain"); err != nil {
		t.Fatalf("run() error = %v", err)
	}
}

func TestJobsWorkerRunWrapperRejectsNilBuilder(t *testing.T) {
	if err := Run(nil, nil); !errors.Is(err, postgresjobs.ErrConfig) {
		t.Fatalf("Run(nil builder) error = %v, want ErrConfig", err)
	}
}

func TestJobsWorkerRunRejectsInvalidArgumentsBeforeConfig(t *testing.T) {
	err := run(t.Context(), []string{"unexpected"}, func(context.Context, config.Config, *slog.Logger) (*jobs.Registry, func(context.Context), error) {
		return nil, nil, nil
	})
	if err == nil {
		t.Fatal("run() accepted invalid arguments")
	}
}

func TestJobsWorkerLifecycleRejectsUnavailableDiagnostics(t *testing.T) {
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen diagnostics sentinel: %v", err)
	}
	defer func() { _ = listener.Close() }()

	cfg := jobsWorkerLifecycleConfig(t)
	cfg.Observability.Metrics.Addr = listener.Addr().String()
	got := runLifecycle(t.Context(), t.Context(), cfg, telemetry.New(), &lifecycleEngineStub{})
	if got.Err == nil || !got.CleanupSafe {
		t.Fatalf("runLifecycle() = %+v, want safe diagnostics startup failure", got)
	}
}

//nolint:paralleltest // This test mutates process-global environment or working directory.
func setJobsWorkerRunEnvironment(t *testing.T) {
	t.Helper()
	configtest.IsolateEnv(t)
	for key, value := range map[string]string{
		"APP__APP__ENV":                      "local",
		"APP__POSTGRES__ENABLED":             "true",
		"APP__POSTGRES__DSN":                 "postgres://jobs:password@localhost:5432/jobs?sslmode=disable",
		"APP__POSTGRES__MAX_OPEN_CONNS":      "2",
		"APP__JOBS__ENABLED":                 "true",
		"APP__JOBS__POLL_INTERVAL":           "1s",
		"APP__JOBS__MAX_CONCURRENCY":         "1",
		"APP__JOBS__LEASE_DURATION":          "6s",
		"APP__JOBS__STORE_OPERATION_TIMEOUT": "1s",
		"APP__JOBS__OBSERVATION_INTERVAL":    "1s",
		"APP__JOBS__DRAIN_TIMEOUT":           "1s",
		"APP__OBSERVABILITY__METRICS__ADDR":  waittest.FreeTCPAddr(t, "jobs worker diagnostics"),
	} {
		t.Setenv(key, value)
	}
}

func jobsWorkerRegistry(t *testing.T, envelope time.Duration) *jobs.Registry {
	t.Helper()
	definition, err := jobs.NewDefinition(jobs.DefinitionInput[map[string]string]{
		Revision:        jobs.Revision{Kind: "test", ArgsVersion: "v1", PolicyVersion: "p1"},
		MaxPayloadBytes: 1024,
		Validate:        func(map[string]string) error { return nil },
		Policy: jobs.Policy{
			Effect: jobs.EffectPolicy{AmbiguousAction: jobs.AmbiguousEffectOutcomeUnknown},
			Retry: jobs.RetryPolicy{
				MaxAttempts: 1, MaxElapsed: time.Hour, InitialBackoff: time.Second, MaxBackoff: time.Second,
				HintPolicy: jobs.RetryHintIgnore, Jitter: jobs.JitterNone, MaxRecoveryWave: 1,
			},
			Recovery:           jobs.RecoveryPolicy{Mode: jobs.RecoveryUnavailable, Attempts: jobs.BudgetPreserved, Elapsed: jobs.BudgetPreserved},
			MaxAttemptDuration: envelope, TerminationEnvelope: envelope,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	registry := new(jobs.Registry)
	if err := jobs.Register(registry, definition, func(context.Context, jobs.HandlerInput[map[string]string]) jobs.HandlerResult {
		return jobs.HandlerResult{Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted}
	}); err != nil {
		t.Fatal(err)
	}
	return registry
}
