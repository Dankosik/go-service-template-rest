package runtimeopts

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	// profile:database-postgres:start
	"github.com/example/go-service-template-rest/internal/config/configtest"
	"github.com/example/go-service-template-rest/internal/infra/postgres"

	// profile:database-postgres:end
	"github.com/example/go-service-template-rest/internal/infra/telemetry"

	// profile:messaging-nats-jetstream:start
	"github.com/google/go-cmp/cmp"
	// profile:messaging-nats-jetstream:end
)

func TestAdapterOptionsPreserveConfiguredValues(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		App: config.AppConfig{Env: "production", Version: "v1.2.3", Commit: "abc123"},
		Log: config.LogConfig{Level: slog.LevelInfo},
		Observability: config.ObservabilityConfig{OTel: config.OTelConfig{
			ServiceName: "orders", TracesSampler: "parentbased_traceidratio", TracesSamplerArg: 0.25,
			Exporter: config.OTelExporterConfig{OTLPEndpoint: "https://otel.example/v1/traces", OTLPMetricsEndpoint: "https://otel.example/v1/metrics", OTLPHeaders: "authorization=Bearer token"},
		}},
		// profile:database-postgres:start
		Postgres: config.PostgresConfig{DSN: "postgres://db", MaxOpenConns: 5},
		// profile:database-postgres:end
	}

	resource := Resource(cfg, "pod-1")
	if resource.ServiceName != "orders" || resource.ServiceVersion != "v1.2.3" || resource.ServiceCommit != "abc123" || resource.ServiceInstanceID != "pod-1" || resource.DeploymentEnv != "production" {
		t.Fatalf("Resource() = %#v, want complete process identity", resource)
	}
	tracing := Tracing(cfg, "pod-1")
	if tracing.Resource != resource || tracing.TracesSampler != "parentbased_traceidratio" || tracing.TracesSamplerArg != 0.25 || tracing.Exporter.OTLPEndpoint != "https://otel.example/v1/traces" || tracing.Exporter.OTLPHeaders != "authorization=Bearer token" {
		t.Fatalf("Tracing() = %#v, want configured tracing adapter", tracing)
	}
	metrics := Metrics(cfg, "pod-1")
	if metrics.Resource != resource || metrics.Exporter.OTLPEndpoint != "https://otel.example/v1/metrics" || metrics.Exporter.SharedOTLPEndpoint != "https://otel.example/v1/traces" || metrics.Exporter.OTLPHeaders != "authorization=Bearer token" {
		t.Fatalf("Metrics() = %#v, want configured metrics adapter", metrics)
	}
	// profile:database-postgres:start
	if got := Postgres(cfg.Postgres); got.DSN != cfg.Postgres.DSN || got.MaxOpenConns != 5 {
		t.Fatalf("Postgres() = %#v, want DSN and pool ceiling", got)
	}
	// profile:database-postgres:end
}

// profile:database-postgres:start
func TestPostgresDefaultPoolCeilingMatchesAdapter(t *testing.T) {
	configtest.IsolateEnv(t)

	cfg, _, err := config.LoadDetailed(config.LoadOptions{})
	if err != nil {
		t.Fatalf("config.LoadDetailed() error = %v", err)
	}
	if got := Postgres(cfg.Postgres).MaxOpenConns; got != postgres.DefaultMaxOpenConns {
		t.Fatalf("Postgres default MaxOpenConns = %d, want adapter default %d", got, postgres.DefaultMaxOpenConns)
	}
}

// profile:database-postgres:end

func TestLoggerCarriesProcessIdentityAndExtraFields(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		App:           config.AppConfig{Env: "production", Version: "v1.2.3"},
		Log:           config.LogConfig{Level: slog.LevelInfo},
		Observability: config.ObservabilityConfig{OTel: config.OTelConfig{ServiceName: "orders"}},
	}
	if got, want := LoggerFields(cfg), []any{"service.name", "orders", "service.version", "v1.2.3", "deployment.environment.name", "production"}; !slices.Equal(got, want) {
		t.Fatalf("LoggerFields() = %#v, want %#v", got, want)
	}

	var output bytes.Buffer
	Logger(&output, cfg, "component", "worker").Info("started")
	for _, field := range []string{`"service.name":"orders"`, `"service.version":"v1.2.3"`, `"deployment.environment.name":"production"`, `"component":"worker"`} {
		if !bytes.Contains(output.Bytes(), []byte(field)) {
			t.Fatalf("logger output %q does not contain %q", output.String(), field)
		}
	}
}

func TestArmTeardownIgnoresCanceledParentAndSetsDeadline(t *testing.T) {
	t.Parallel()

	parent, cancelParent := context.WithCancel(context.Background())
	cancelParent()
	ctx, cancel, deadline := ArmTeardown(parent, time.Second)
	defer cancel()
	if ctx.Err() != nil {
		t.Fatalf("ArmTeardown context is canceled: %v", ctx.Err())
	}
	if got, ok := ctx.Deadline(); !ok || !got.Equal(deadline) {
		t.Fatalf("context deadline = %v, %t, want %v", got, ok, deadline)
	}
	if remaining := time.Until(deadline); remaining <= 0 || remaining > time.Second {
		t.Fatalf("deadline remaining = %s, want (0, 1s]", remaining)
	}
}

func TestStartRuntimeCancelsOnlyDuringStartup(t *testing.T) {
	t.Parallel()

	t.Run("startup cancellation", func(t *testing.T) {
		t.Parallel()

		startupCtx, cancelStartup := context.WithCancel(context.Background())
		runtimeCtx, cancelRuntime := context.WithCancel(context.Background())
		defer cancelRuntime()
		started, err := StartRuntime(startupCtx, runtimeCtx, cancelRuntime, func(ctx context.Context) error {
			cancelStartup()
			<-ctx.Done()
			return ctx.Err()
		})
		if started || !errors.Is(err, context.Canceled) {
			t.Fatalf("StartRuntime() = %t, %v, want false, context.Canceled", started, err)
		}
	})

	t.Run("runtime after admission", func(t *testing.T) {
		t.Parallel()

		startupCtx, cancelStartup := context.WithCancel(context.Background())
		runtimeCtx, cancelRuntime := context.WithCancel(context.Background())
		defer cancelRuntime()
		started, err := StartRuntime(startupCtx, runtimeCtx, cancelRuntime, func(context.Context) error { return nil })
		if !started || err != nil {
			t.Fatalf("StartRuntime() = %t, %v, want true, nil", started, err)
		}
		cancelStartup()
		if err := runtimeCtx.Err(); err != nil {
			t.Fatalf("runtime inherited cancellation after successful startup: %v", err)
		}
	})

	t.Run("cancellation wins admission race", func(t *testing.T) {
		t.Parallel()

		startupCtx, cancelStartup := context.WithCancel(context.Background())
		runtimeCtx, cancelRuntime := context.WithCancel(context.Background())
		defer cancelRuntime()
		started, err := StartRuntime(startupCtx, runtimeCtx, cancelRuntime, func(ctx context.Context) error {
			cancelStartup()
			<-ctx.Done()
			return nil
		})
		if !started || !errors.Is(err, context.Canceled) {
			t.Fatalf("StartRuntime() = %t, %v, want true, context.Canceled", started, err)
		}
	})
}

func TestInstallTelemetryReturnsUsableFlush(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	metrics := telemetry.New()
	flush, err := InstallTelemetry(context.Background(), config.Config{
		App:           config.AppConfig{Env: "test", Version: "v1"},
		Observability: config.ObservabilityConfig{OTel: config.OTelConfig{ServiceName: "runtimeopts-test"}},
	}, metrics, slog.New(slog.NewTextHandler(&output, nil)), "worker")
	if err != nil {
		t.Fatalf("InstallTelemetry() error = %v", err)
	}
	flush(context.Background())
}

func TestLogDegradedSignalIncludesSafeClassification(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logDegradedSignal(context.Background(), slog.New(slog.NewTextHandler(&output, nil)), "worker_tracing_degraded", errors.New("endpoint rejected"))
	for _, field := range []string{"worker_tracing_degraded", "operation=telemetry_init", "outcome=degraded", "reason=setup_error"} {
		if !bytes.Contains(output.Bytes(), []byte(field)) {
			t.Fatalf("degraded-signal output %q does not contain %q", output.String(), field)
		}
	}
}

// profile:messaging-nats-jetstream:start
func TestMessagingMappingRetainsBlankAddressesForAdapterValidation(t *testing.T) {
	t.Parallel()

	cfg := config.MessagingConfig{
		URLs: "nats://one, , nats://two,", CredentialsFile: "/run/secrets/nats.creds", RootCAFile: "/run/secrets/nats-ca.pem",
		AllowPlaintext: true, AllowUnauthenticated: true, Stream: "EVENTS", MaxPayloadBytes: 1024,
	}
	got := Messaging(cfg)
	want := Messaging(config.MessagingConfig{})
	want.URLs = []string{"nats://one", "", "nats://two", ""}
	want.CredentialsFile = "/run/secrets/nats.creds"
	want.RootCAFile = "/run/secrets/nats-ca.pem"
	want.AllowPlaintext = true
	want.AllowUnauthenticated = true
	want.Stream = "EVENTS"
	want.MaxPayloadBytes = 1024
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Messaging() mismatch (-want +got):\n%s", diff)
	}
}

// profile:messaging-nats-jetstream:end
