package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/background"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/postgresidempotency"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/example/go-service-template-rest/internal/waittest"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestHTTPIdempotencyZeroRegistrationIsInert(t *testing.T) {
	t.Parallel()
	reader := telemetrytest.InstallManualReader(t)
	for _, delay := range []time.Duration{0, -time.Second} {
		cfg := config.Config{HTTPIdempotency: config.HTTPIdempotencyConfig{OwnerRecoveryDelay: delay}}
		runtime, err := initHTTPIdempotencyRuntime(t.Context(), cfg, nil, nil)
		if err != nil {
			t.Fatalf("empty registration with delay %s returned %v", delay, err)
		}
		if runtime.store != nil || runtime.maintain != nil || runtime.interval != 0 {
			t.Fatalf("empty registration constructed runtime %+v", runtime)
		}
		if probes := runtime.ReadinessProbes(); probes != nil {
			t.Fatalf("empty registration probes = %v, want nil", probes)
		}
		if err := runtime.Run(t.Context()); err != nil {
			t.Fatalf("empty registration Run() error = %v", err)
		}
	}
	telemetrytest.ForEachMetric(t, reader, func(measured metricdata.Metrics) {
		if strings.HasPrefix(measured.Name, "http.idempotency.") {
			t.Errorf("empty registration created metric %s", measured.Name)
		}
	})
}

func TestHTTPIdempotencyActiveConfigMapping(t *testing.T) {
	t.Parallel()
	base := config.HTTPIdempotencyConfig{
		OwnerRecoveryDelay:     30 * time.Second,
		MaintenanceInterval:    5 * time.Second,
		CleanupBatchSize:       101,
		MaxMaintenanceLag:      time.Minute,
		MaxRelationBytes:       1 << 30,
		AdmissionHeadroomBytes: 1 << 20,
	}
	for _, delay := range []time.Duration{30 * time.Second, 50 * time.Second} {
		cfg := base
		cfg.OwnerRecoveryDelay = delay
		want := postgresidempotency.StoreOptions{
			OwnerRecoveryDelay:     delay,
			CleanupBatchSize:       cfg.CleanupBatchSize,
			MaxMaintenanceLag:      cfg.MaxMaintenanceLag,
			MaxRelationBytes:       cfg.MaxRelationBytes,
			AdmissionHeadroomBytes: cfg.AdmissionHeadroomBytes,
		}
		if got := idempotencyStoreOptions(cfg); got != want {
			t.Fatalf("idempotencyStoreOptions(%s) = %+v, want %+v", delay, got, want)
		}
	}

	for _, delay := range []time.Duration{0, -time.Second} {
		cfg := config.Config{
			Postgres:        config.PostgresConfig{Enabled: true},
			HTTPIdempotency: base,
		}
		cfg.HTTPIdempotency.OwnerRecoveryDelay = delay
		_, err := initHTTPIdempotencyRuntime(t.Context(), cfg, nil, []httpx.IdempotencyOperation{{}})
		if err == nil || !strings.Contains(err.Error(), "owner_recovery_delay") {
			t.Fatalf("active delay %s error = %v, want owner_recovery_delay rejection", delay, err)
		}
	}
}

func TestHTTPIdempotencyEpochLossDrains(t *testing.T) {
	t.Parallel()
	supervisor := newSupervisedBackground(t.Context(), slog.New(slog.DiscardHandler))
	terminalErrors := make(chan error, 1)
	terminalErrors <- postgresidempotency.ErrEpochLost
	runtime := httpIdempotencyRuntime{
		interval:       time.Hour,
		terminalErrors: terminalErrors,
		maintain: func(context.Context) error {
			return errors.New("maintenance waited for cadence")
		},
	}
	supervisor.Go(background.Task{Name: "http_idempotency_maintenance", Run: runtime.Run})

	failure := waittest.Receive(t, supervisor.Failures(), time.Second, "maintenance epoch loss to reach the supervisor")
	if !errors.Is(failure, postgresidempotency.ErrEpochLost) {
		t.Fatalf("supervisor failure = %v, want epoch loss", failure)
	}
	healthSvc := health.New(supervisor)
	if err := healthSvc.Refresh(t.Context(), time.Second, 1); !errors.Is(err, postgresidempotency.ErrEpochLost) {
		t.Fatalf("readiness refresh = %v, want cached epoch loss", err)
	}
	if err := healthSvc.Cached(); !errors.Is(err, postgresidempotency.ErrEpochLost) {
		t.Fatalf("cached readiness = %v, want epoch loss", err)
	}

	shutdownCalled := false
	server := newFakeRuntimeServer()
	server.onShutdown = func(context.Context) error {
		shutdownCalled = true
		return nil
	}
	failures := make(chan error, 1)
	failures <- failure
	err := serveRuntime(t.Context(), t.Context(), serveRuntimeArgs{
		cfg: config.Config{
			App:  config.AppConfig{Env: "test"},
			HTTP: config.HTTPConfig{Addr: "127.0.0.1:0", ShutdownTimeout: time.Second},
		},
		log:                slog.New(slog.DiscardHandler),
		healthSvc:          healthSvc,
		httpSrv:            server,
		readinessCheck:     func(context.Context) error { return nil },
		backgroundFailures: failures,
		admission:          new(startupAdmissionController),
		shutdown:           testShutdownBudget(),
	})
	if !errors.Is(err, postgresidempotency.ErrEpochLost) {
		t.Fatalf("serveRuntime() error = %v, want epoch loss", err)
	}
	if !shutdownCalled || !errors.Is(healthSvc.Cached(), health.ErrDraining) {
		t.Fatalf("normal drain = (shutdown %t, readiness %v)", shutdownCalled, healthSvc.Cached())
	}
	if err := supervisor.Shutdown(testShutdownBudget().stage(t.Context(), backgroundShutdownTimeout)); err == nil {
		t.Fatal("supervisor shutdown lost the terminal maintenance failure")
	}
}

func TestHTTPIdempotencyMaintenanceReadiness(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		calls := 0
		runtime := httpIdempotencyRuntime{
			interval: time.Second,
			maintain: func(context.Context) error {
				calls++
				if calls == 1 {
					return postgresidempotency.ErrUnavailable
				}
				cancel()
				return nil
			},
		}
		err := runtime.Run(ctx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Run() error = %v, want context canceled", err)
		}
		if calls != 2 {
			t.Fatalf("maintenance calls = %d, want one retry at the next cadence", calls)
		}
	})
}

func TestHTTPIdempotencyReadinessLifecycle(t *testing.T) {
	t.Parallel()
	zero := httpIdempotencyRuntime{}
	if zero.ReadinessProbes() != nil {
		t.Fatal("zero runtime registered readiness")
	}
	active := httpIdempotencyRuntime{store: &postgresidempotency.Store{}}
	probes := active.ReadinessProbes()
	if len(probes) != 1 || probes[0].Name() != "postgres_http_idempotency" {
		t.Fatalf("active readiness probes = %v, want named Store probe", probes)
	}
	if err := probes[0].Check(t.Context()); !errors.Is(err, postgresidempotency.ErrUnavailable) {
		t.Fatalf("zero Store cached Check() = %v, want unavailable", err)
	}
	testHTTPIdempotencyTerminalRuntime(t, postgresidempotency.ErrIntegrityConflict)
}

func testHTTPIdempotencyTerminalRuntime(t *testing.T, terminal error) {
	t.Helper()
	synctest.Test(t, func(t *testing.T) {
		calls := 0
		runtime := httpIdempotencyRuntime{
			interval: time.Second,
			maintain: func(context.Context) error {
				calls++
				return terminal
			},
		}
		err := runtime.Run(t.Context())
		if !errors.Is(err, terminal) {
			t.Fatalf("Run() error = %v, want %v", err, terminal)
		}
		if calls != 1 {
			t.Fatalf("maintenance calls after terminal error = %d, want 1", calls)
		}
	})
}
