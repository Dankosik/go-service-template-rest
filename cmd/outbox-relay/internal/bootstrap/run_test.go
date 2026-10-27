package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/riverqueue/river"
)

type lifecycleMessaging struct {
	ready            atomic.Bool
	stopPublishCalls int
	shutdownCalls    int
}

func (*lifecycleMessaging) Name() string                { return "messaging" }
func (*lifecycleMessaging) Check(context.Context) error { return nil }
func (m *lifecycleMessaging) Ready() bool               { return m.ready.Load() }
func (*lifecycleMessaging) Run(ctx context.Context) error {
	<-ctx.Done()
	return fmt.Errorf("run messaging fixture: %w", ctx.Err())
}
func (m *lifecycleMessaging) StopPublish() { m.stopPublishCalls++ }
func (m *lifecycleMessaging) Shutdown(context.Context) error {
	m.shutdownCalls++
	return nil
}

type lifecycleRiver struct {
	started chan struct{}
	stopped chan struct{}
	stopErr error
}

func (r *lifecycleRiver) Start(context.Context) error {
	close(r.started)
	return nil
}
func (r *lifecycleRiver) Stop(context.Context) error          { return r.stopErr }
func (r *lifecycleRiver) StopAndCancel(context.Context) error { return r.stopErr }
func (r *lifecycleRiver) Stopped() <-chan struct{}            { return r.stopped }

type healthyPostgres struct{}

func (healthyPostgres) Ping(context.Context) error { return nil }

func TestRiverClientConfigUsesFixedBoundedCapacity(t *testing.T) {
	t.Parallel()

	got := riverClientConfig(river.NewWorkers(), slog.Default())
	if got.Queues[postgresoutbox.Queue].MaxWorkers != defaultOutboxWorkers {
		t.Fatalf("MaxWorkers = %d, want %d", got.Queues[postgresoutbox.Queue].MaxWorkers, defaultOutboxWorkers)
	}
	if got.CancelledJobRetentionPeriod != -1 || got.DiscardedJobRetentionPeriod != -1 {
		t.Fatal("unfinished outbox jobs can be deleted")
	}
	if len(got.Plugins) != 1 {
		t.Fatalf("Plugins = %d, want 1", len(got.Plugins))
	}
	if !got.PollOnly {
		t.Fatal("River LISTEN must stay disabled on the statement-bounded pool")
	}
	if got.SoftStopTimeout != outboxDrain {
		t.Fatalf("SoftStopTimeout = %s, want relay-owned %s", got.SoftStopTimeout, outboxDrain)
	}
}

func TestValidateRuntimeConfig(t *testing.T) {
	t.Parallel()

	valid := config.Config{}
	valid.Postgres.Enabled = true
	valid.Messaging.URLs = "tls://nats.example:4222"
	valid.Observability.Metrics.Addr = "127.0.0.1:9090"
	valid.HTTP.GracePeriod = 45 * time.Second
	if err := validateRuntimeConfig(valid); err != nil {
		t.Fatalf("validateRuntimeConfig() error = %v", err)
	}
	for name, mutate := range map[string]func(*config.Config){
		"postgres":    func(cfg *config.Config) { cfg.Postgres.Enabled = false },
		"messaging":   func(cfg *config.Config) { cfg.Messaging.URLs = "" },
		"diagnostics": func(cfg *config.Config) { cfg.Observability.Metrics.Addr = "" },
		"grace":       func(cfg *config.Config) { cfg.HTTP.GracePeriod = 36 * time.Second },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			invalid := valid
			mutate(&invalid)
			if err := validateRuntimeConfig(invalid); !errors.Is(err, config.ErrValidate) {
				t.Fatalf("validateRuntimeConfig() error = %v, want ErrValidate", err)
			}
		})
	}
}

// TestRelayReadinessRequiresImmediateMessagingReadiness proves the readiness
// endpoint reads the live messaging connection on every request: a NATS
// disconnect turns the next probe 503 without waiting for a dependency refresh,
// which here is an hour away.
func TestRelayReadinessRequiresImmediateMessagingReadiness(t *testing.T) {
	riverClient := &lifecycleRiver{started: make(chan struct{}), stopped: make(chan struct{})}
	messaging := &lifecycleMessaging{}
	messaging.ready.Store(true)
	address := waittest.FreeTCPAddr(t, "outbox diagnostics")
	cfg := config.Config{}
	cfg.HTTP.GracePeriod = time.Second
	cfg.HTTP.ReadinessTimeout = 100 * time.Millisecond
	cfg.Health.RefreshInterval = time.Hour
	cfg.Health.FailureThreshold = 1
	cfg.Observability.Metrics.Addr = address

	signalCtx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	result := make(chan error, 1)
	go func() {
		_, _, err := runLifecycle(
			signalCtx, t.Context(), cfg, slog.New(slog.DiscardHandler), telemetry.NewMetrics(),
			healthyPostgres{}, messaging, riverClient,
		)
		result <- err
	}()
	waittest.ReceiveSignal(t, riverClient.started, 5*time.Second, "River start")
	waittest.Until(t, 5*time.Second, func(ctx context.Context) bool {
		status, err := readinessStatus(ctx, address)
		return err == nil && status == http.StatusOK
	}, "outbox /health/ready to report 200")

	messaging.ready.Store(false)
	status, err := readinessStatus(t.Context(), address)
	if err != nil {
		t.Fatalf("GET /health/ready after messaging disconnected: %v", err)
	}
	if status != http.StatusServiceUnavailable {
		t.Fatalf("GET /health/ready after messaging disconnected = %d, want 503", status)
	}

	cancel()
	if err := waittest.Receive(t, result, 5*time.Second, "runLifecycle to return"); err != nil {
		t.Fatalf("runLifecycle() error = %v", err)
	}
}

func readinessStatus(ctx context.Context, address string) (int, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+address+"/health/ready", http.NoBody)
	if err != nil {
		return 0, fmt.Errorf("build readiness request: %w", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0, fmt.Errorf("get readiness: %w", err)
	}
	_ = response.Body.Close()
	return response.StatusCode, nil
}

func TestRunLifecycleDoesNotCloseMessagingBeforeRiverStops(t *testing.T) {
	stopErr := errors.New("River jobs still running")
	riverClient := &lifecycleRiver{
		started: make(chan struct{}), stopped: make(chan struct{}), stopErr: stopErr,
	}
	messaging := &lifecycleMessaging{}
	messaging.ready.Store(true)
	cfg := config.Config{}
	cfg.HTTP.GracePeriod = time.Second
	cfg.HTTP.ReadinessTimeout = 100 * time.Millisecond
	cfg.Health.RefreshInterval = time.Second
	cfg.Health.FailureThreshold = 1
	cfg.Observability.Metrics.Addr = "127.0.0.1:0"

	signalCtx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	type outcome struct {
		cleanupSafe bool
		err         error
	}
	result := make(chan outcome, 1)
	go func() {
		cleanupSafe, _, err := runLifecycle(
			signalCtx, t.Context(), cfg, slog.New(slog.DiscardHandler), telemetry.NewMetrics(),
			healthyPostgres{}, messaging, riverClient,
		)
		result <- outcome{cleanupSafe: cleanupSafe, err: err}
	}()
	<-riverClient.started
	cancel()

	select {
	case got := <-result:
		if got.cleanupSafe || !errors.Is(got.err, stopErr) {
			t.Fatalf("runLifecycle() = cleanup safe %t, error %v", got.cleanupSafe, got.err)
		}
		if messaging.stopPublishCalls != 0 || messaging.shutdownCalls != 0 {
			t.Fatalf(
				"messaging cleanup before River join = stop publish %d, shutdown %d",
				messaging.stopPublishCalls, messaging.shutdownCalls,
			)
		}
	case <-time.After(time.Second):
		t.Fatal("runLifecycle() did not return after River stop failed")
	}
}
