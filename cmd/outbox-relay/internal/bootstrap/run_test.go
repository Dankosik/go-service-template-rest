package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

type fakeRelay struct {
	ready     atomic.Bool
	started   chan struct{}
	drain     chan struct{}
	drainOnce sync.Once
	run       func(context.Context, <-chan struct{}) postgresoutbox.RelayResult
}

func newFakeRelay(run func(context.Context, <-chan struct{}) postgresoutbox.RelayResult) *fakeRelay {
	return &fakeRelay{started: make(chan struct{}), drain: make(chan struct{}), run: run}
}

func (relay *fakeRelay) Ready() bool { return relay.ready.Load() }

func (relay *fakeRelay) StartDrain() {
	relay.ready.Store(false)
	relay.drainOnce.Do(func() { close(relay.drain) })
}

func (relay *fakeRelay) Run(ctx context.Context) postgresoutbox.RelayResult {
	relay.ready.Store(true)
	close(relay.started)
	return relay.run(ctx, relay.drain)
}

func TestOutboxRelayComposition(t *testing.T) {
	t.Run("nil builder wins before flags or config", func(t *testing.T) {
		err := run(t.Context(), []string{"--not-a-real-flag"}, nil)
		if !errors.Is(err, postgresoutbox.ErrConfig) {
			t.Fatalf("run() error = %v, want ErrConfig", err)
		}
	})

	t.Run("nil publisher rejects before postgres", func(t *testing.T) {
		setOutboxBootstrapEnvironment(t, true)
		built := false
		cleaned := false
		err := run(t.Context(), nil, func(context.Context, config.Config, *slog.Logger) (postgresoutbox.Publisher, func(context.Context), error) {
			built = true
			return nil, func(context.Context) { cleaned = true }, nil
		})
		if !errors.Is(err, postgresoutbox.ErrConfig) || !built || !cleaned {
			t.Fatalf("run() = %v built=%t cleaned=%t, want nil-publisher rejection with builder cleanup", err, built, cleaned)
		}
		if strings.Contains(err.Error(), "connect") {
			t.Fatalf("nil publisher reached postgres mutation: %v", err)
		}
	})

	t.Run("typed nil publisher rejects before postgres", func(t *testing.T) {
		setOutboxBootstrapEnvironment(t, true)
		cleaned := false
		var publisher *pointerTestPublisher
		err := run(t.Context(), nil, func(context.Context, config.Config, *slog.Logger) (postgresoutbox.Publisher, func(context.Context), error) {
			return publisher, func(context.Context) { cleaned = true }, nil
		})
		if !errors.Is(err, postgresoutbox.ErrConfig) || !cleaned {
			t.Fatalf("run() = %v cleaned=%t, want typed-nil rejection with builder cleanup", err, cleaned)
		}
		if strings.Contains(err.Error(), "connect") {
			t.Fatalf("typed nil publisher reached postgres mutation: %v", err)
		}
	})

	t.Run("invalid database combination rejects before builder", func(t *testing.T) {
		setOutboxBootstrapEnvironment(t, false)
		built := false
		err := run(t.Context(), nil, func(context.Context, config.Config, *slog.Logger) (postgresoutbox.Publisher, func(context.Context), error) {
			built = true
			return testPublisher{}, nil, nil
		})
		if !errors.Is(err, config.ErrValidate) || built {
			t.Fatalf("run() = %v built=%t, want config rejection before builder", err, built)
		}
	})

	t.Run("invalid drain budget rejects before builder", func(t *testing.T) {
		setOutboxBootstrapEnvironment(t, true)
		t.Setenv("APP__HTTP__GRACE_PERIOD", "33s")
		built := false
		err := run(t.Context(), nil, func(context.Context, config.Config, *slog.Logger) (postgresoutbox.Publisher, func(context.Context), error) {
			built = true
			return testPublisher{}, nil, nil
		})
		if !errors.Is(err, config.ErrValidate) || built || !strings.Contains(err.Error(), "post-drain cleanup") {
			t.Fatalf("run() = %v built=%t, want drain-budget rejection before builder", err, built)
		}
	})

	t.Run("complete drain budget reaches builder before postgres", func(t *testing.T) {
		setOutboxBootstrapEnvironment(t, true)
		t.Setenv("APP__HTTP__GRACE_PERIOD", "34s")
		builderErr := errors.New("builder admission canary")
		built := false
		err := run(t.Context(), nil, func(context.Context, config.Config, *slog.Logger) (postgresoutbox.Publisher, func(context.Context), error) {
			built = true
			return nil, nil, builderErr
		})
		if !built || !errors.Is(err, builderErr) {
			t.Fatalf("run() = %v built=%t, want accepted shutdown budget and builder canary", err, built)
		}
		if strings.Contains(err.Error(), "connect") {
			t.Fatalf("builder canary reached postgres mutation: %v", err)
		}
	})

	t.Run("overflowing drain budget rejects before builder", func(t *testing.T) {
		setOutboxBootstrapEnvironment(t, true)
		t.Setenv("APP__OUTBOX__DRAIN_TIMEOUT", time.Duration(1<<63-1).String())
		built := false
		err := run(t.Context(), nil, func(context.Context, config.Config, *slog.Logger) (postgresoutbox.Publisher, func(context.Context), error) {
			built = true
			return testPublisher{}, nil, nil
		})
		if !errors.Is(err, config.ErrValidate) || built {
			t.Fatalf("run() = %v built=%t, want overflow-safe drain rejection before builder", err, built)
		}
	})
}

func TestOutboxRelayGracefulDrain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		relay := newFakeRelay(func(_ context.Context, drain <-chan struct{}) postgresoutbox.RelayResult {
			<-drain
			return postgresoutbox.RelayResult{CleanupSafe: true}
		})
		runtimeCtx, runtimeCancel := context.WithCancel(context.Background())
		defer runtimeCancel()
		processCtx, processCancel := context.WithTimeout(context.Background(), time.Hour)
		defer processCancel()
		result := make(chan postgresoutbox.RelayResult, 1)
		go func() { result <- relay.Run(runtimeCtx) }()
		<-relay.started
		startedAt := time.Now()
		got, read := drainRelay(
			processCtx,
			20*time.Second,
			runtimeCancel,
			relay,
			result,
			postgresoutbox.RelayResult{CleanupSafe: true},
			false,
		)
		synctest.Wait()
		if !read || time.Since(startedAt) != 0 {
			t.Fatalf("graceful drain read=%t elapsed=%s, want true/0", read, time.Since(startedAt))
		}
		if !got.CleanupSafe || got.Err != nil {
			t.Fatalf("drainRelay() = %+v, want clean graceful drain", got)
		}
	})
}

func TestOutboxRelayForcedShutdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		relay := newFakeRelay(func(ctx context.Context, _ <-chan struct{}) postgresoutbox.RelayResult {
			<-ctx.Done()
			time.Sleep(time.Second)
			return postgresoutbox.RelayResult{CleanupSafe: false, Err: postgresoutbox.ErrPublisherStuck}
		})
		runtimeCtx, runtimeCancel := context.WithCancel(context.Background())
		processCtx, processCancel := context.WithTimeout(context.Background(), time.Hour)
		defer processCancel()
		result := make(chan postgresoutbox.RelayResult, 1)
		go func() { result <- relay.Run(runtimeCtx) }()
		<-relay.started
		startedAt := time.Now()
		got, read := drainRelay(
			processCtx,
			20*time.Second,
			runtimeCancel,
			relay,
			result,
			postgresoutbox.RelayResult{CleanupSafe: true},
			false,
		)
		synctest.Wait()
		if !read || time.Since(startedAt) != 21*time.Second {
			t.Fatalf("forced drain read=%t elapsed=%s, want true/21s", read, time.Since(startedAt))
		}
		if got.CleanupSafe || !errors.Is(got.Err, postgresoutbox.ErrPublisherStuck) {
			t.Fatalf("drainRelay() = %+v, want cleanup-unsafe stuck publisher", got)
		}
	})
}

func TestOutboxRelayOuterJoinTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		relay := newFakeRelay(func(context.Context, <-chan struct{}) postgresoutbox.RelayResult {
			<-release
			return postgresoutbox.RelayResult{CleanupSafe: false}
		})
		runtimeCtx, runtimeCancel := context.WithCancel(context.Background())
		processCtx, processCancel := context.WithTimeout(context.Background(), time.Hour)
		defer processCancel()
		result := make(chan postgresoutbox.RelayResult, 1)
		go func() { result <- relay.Run(runtimeCtx) }()
		<-relay.started
		startedAt := time.Now()
		got, read := drainRelay(
			processCtx,
			20*time.Second,
			runtimeCancel,
			relay,
			result,
			postgresoutbox.RelayResult{CleanupSafe: true},
			false,
		)
		if read || got.CleanupSafe || !strings.Contains(got.Err.Error(), "join outbox relay") ||
			time.Since(startedAt) != 20*time.Second+forcedJoin {
			t.Fatalf("outer join = %+v read=%t elapsed=%s", got, read, time.Since(startedAt))
		}
		close(release)
		synctest.Wait()
	})
}

func TestOutboxRelayForcedShutdownSkipsDependencyCleanup(t *testing.T) {
	var order []string
	if err := cleanupRelayDependencies(
		t.Context(),
		true,
		func(context.Context) { order = append(order, "publisher") },
		func() { order = append(order, "postgres") },
	); err != nil {
		t.Fatalf("safe cleanup error = %v", err)
	}
	if got := strings.Join(order, ","); got != "publisher,postgres" {
		t.Fatalf("safe cleanup order = %q, want publisher,postgres", got)
	}
	if err := cleanupRelayDependencies(
		t.Context(),
		false,
		func(context.Context) { order = append(order, "unsafe-publisher") },
		func() { order = append(order, "unsafe-postgres") },
	); err != nil {
		t.Fatalf("unsafe cleanup error = %v", err)
	}
	if got := strings.Join(order, ","); got != "publisher,postgres" {
		t.Fatalf("cleanup-unsafe dependencies were closed: %q", got)
	}
}

func TestOutboxRelayPublisherCleanupIsBounded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		started := make(chan struct{})
		release := make(chan struct{})
		result := make(chan error, 1)
		go func() {
			result <- closePublisher(context.Background(), func(context.Context) {
				close(started)
				<-release
			})
		}()
		<-started
		err := <-result
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("closePublisher(stuck) error = %v, want deadline", err)
		}
		close(release)
		synctest.Wait()
	})

	if err := closePublisher(t.Context(), func(context.Context) { panic("secret") }); err == nil ||
		!strings.Contains(err.Error(), "panicked") || strings.Contains(err.Error(), "secret") {
		t.Fatalf("closePublisher(panic) error = %v, want redacted panic", err)
	}
}

func TestOutboxRelayPublisherPanic(t *testing.T) {
	relay := newFakeRelay(func(context.Context, <-chan struct{}) postgresoutbox.RelayResult {
		return postgresoutbox.RelayResult{CleanupSafe: true, Err: postgresoutbox.ErrPublisherPanic}
	})
	got := runRelayLifecycle(t.Context(), t.Context(), lifecycleConfig(time.Second), telemetry.New(), relay)
	if !got.CleanupSafe || !errors.Is(got.Err, postgresoutbox.ErrPublisherPanic) || relay.Ready() {
		t.Fatalf("runRelayLifecycle() = %+v ready=%t, want cleanup-safe fatal panic", got, relay.Ready())
	}
	select {
	case <-relay.drain:
	default:
		t.Fatal("publisher panic did not start relay drain")
	}
}

func TestOutboxRelayReadinessAndLiveness(t *testing.T) {
	ready := atomic.Bool{}
	server := newDiagnosticsServer("127.0.0.1:1", ready.Load, telemetry.New())
	for _, test := range []struct {
		path string
		code int
	}{
		{path: "/health/live", code: http.StatusOK},
		{path: "/health/ready", code: http.StatusServiceUnavailable},
	} {
		recorder := httptest.NewRecorder()
		server.Handler.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, test.path, nil))
		if recorder.Code != test.code {
			t.Fatalf("GET %s status = %d, want %d", test.path, recorder.Code, test.code)
		}
	}
	ready.Store(true)
	recorder := httptest.NewRecorder()
	server.Handler.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/health/ready", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("ready status = %d, want 200", recorder.Code)
	}
}

func TestOutboxRelayFlagAndConfigMapping(t *testing.T) {
	t.Parallel()

	options, err := parseLoadOptions([]string{"--config", "base.yaml", "--config-overlay", "one.yaml", "--config-overlay=two.yaml"})
	if err != nil || options.ConfigPath != "base.yaml" || strings.Join(options.ConfigOverlays, ",") != "one.yaml,two.yaml" {
		t.Fatalf("parseLoadOptions() = %+v, %v", options, err)
	}
	for _, args := range [][]string{
		{"--unknown"},
		{"--config", " "},
		{"--config-overlay", " "},
		{"positional"},
	} {
		if _, err := parseLoadOptions(args); err == nil {
			t.Errorf("parseLoadOptions(%q) succeeded", args)
		}
	}

	postgresConfig := config.PostgresConfig{
		DSN: "dsn", ConnectTimeout: time.Second, HealthcheckTimeout: 2 * time.Second,
		MaxOpenConns: 3, MinIdleConns: 1, AcquireTimeout: 4 * time.Second,
		ConnMaxLifetime: 5 * time.Second, StatementTimeout: 6 * time.Second,
	}
	postgresMapped := postgresOptions(postgresConfig)
	if postgresMapped.DSN != postgresConfig.DSN || postgresMapped.MaxOpenConns != postgresConfig.MaxOpenConns ||
		postgresMapped.StatementTimeout != postgresConfig.StatementTimeout {
		t.Fatalf("postgresOptions() = %+v", postgresMapped)
	}
	outboxConfig := config.OutboxConfig{
		PollInterval: time.Second, PublishTimeout: 2 * time.Second, LeaseDuration: 4 * time.Second,
		MaxAttempts: 5, RetryBase: 6 * time.Second, RetryMax: 7 * time.Second,
		ObservationInterval: 8 * time.Second, CleanupInterval: 9 * time.Second,
		PublishedRetention: 10 * time.Second, CleanupBatchSize: 11,
	}
	relayMapped := relayConfig(outboxConfig)
	if relayMapped.LeaseDuration != outboxConfig.LeaseDuration || relayMapped.MaxAttempts != outboxConfig.MaxAttempts ||
		relayMapped.CleanupBatchSize != outboxConfig.CleanupBatchSize {
		t.Fatalf("relayConfig() = %+v", relayMapped)
	}
	if publisherMissing(testPublisher{}) {
		t.Fatal("publisherMissing rejected a concrete publisher")
	}
	if err := Run([]string{"--unknown"}, nil); !errors.Is(err, postgresoutbox.ErrConfig) {
		t.Fatalf("Run(nil builder) error = %v", err)
	}
	validRuntime := config.Config{
		Postgres:      config.PostgresConfig{Enabled: true},
		Outbox:        config.OutboxConfig{Enabled: true, DrainTimeout: time.Second},
		HTTP:          config.HTTPConfig{GracePeriod: time.Minute},
		Observability: config.ObservabilityConfig{Metrics: config.MetricsConfig{Addr: "127.0.0.1:0"}},
	}
	if err := validateRuntimeConfig(validRuntime); err != nil {
		t.Fatalf("validateRuntimeConfig(valid) error = %v", err)
	}
	for index, invalidRuntime := range []config.Config{
		{},
		func() config.Config { value := validRuntime; value.Postgres.Enabled = false; return value }(),
		func() config.Config { value := validRuntime; value.Observability.Metrics.Addr = ""; return value }(),
	} {
		if err := validateRuntimeConfig(invalidRuntime); err == nil {
			t.Errorf("validateRuntimeConfig(invalid %d) succeeded", index)
		}
	}
}

func TestOutboxRelayTelemetrySetupAndFailureClasses(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		App: config.AppConfig{Version: "test", Commit: "commit", Env: "test", InstanceID: "instance"},
		Observability: config.ObservabilityConfig{OTel: config.OTelConfig{
			ServiceName: "relay", TracesSampler: "always_off",
		}},
	}
	shutdown, err := setupTelemetry(t.Context(), cfg, telemetry.New(), slog.Default())
	if err != nil {
		t.Fatalf("setupTelemetry() error = %v", err)
	}
	shutdown(t.Context())
	if telemetryFailureReason(context.DeadlineExceeded) != "deadline_exceeded" ||
		telemetryFailureReason(context.Canceled) != "canceled" ||
		telemetryFailureReason(errors.New("setup")) != "setup_error" {
		t.Fatal("telemetryFailureReason classification mismatch")
	}
}

type testPublisher struct{}

func (testPublisher) Publish(context.Context, postgresoutbox.Event) error { return nil }

type pointerTestPublisher struct{}

func (*pointerTestPublisher) Publish(context.Context, postgresoutbox.Event) error { return nil }

func setOutboxBootstrapEnvironment(t *testing.T, postgresEnabled bool) {
	t.Helper()
	// profile:authn-oidc-jwt:start
	t.Setenv("APP__AUTHN__ISSUER", "https://issuer.example.com")
	t.Setenv("APP__AUTHN__AUDIENCE", "https://api.example.com")
	t.Setenv("APP__AUTHN__TRUSTED_PROXY_CIDRS", "127.0.0.0/8")
	// profile:authn-oidc-jwt:end
	t.Setenv("APP__POSTGRES__ENABLED", strconv.FormatBool(postgresEnabled))
	t.Setenv("APP__POSTGRES__DSN", "postgres://app:secret@127.0.0.1:1/app?sslmode=disable")
	t.Setenv("APP__OUTBOX__ENABLED", "true")
	t.Setenv("APP__OBSERVABILITY__METRICS__ADDR", "127.0.0.1:9099")
}

func lifecycleConfig(drain time.Duration) config.Config {
	return config.Config{
		HTTP:   config.HTTPConfig{GracePeriod: time.Second},
		Outbox: config.OutboxConfig{DrainTimeout: drain},
		Observability: config.ObservabilityConfig{
			Metrics: config.MetricsConfig{Addr: "127.0.0.1:0"},
		},
	}
}
