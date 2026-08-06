package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
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

// This package is the only one that wires internal/config and postgresoutbox
// together, so it is the only one that can hold their restated ceilings to each
// other. The postgresoutbox ceiling block owns why the values are restated at
// all; the three tests here own which part each of them pins.
//
// This one pins the four numeric ceilings: each must be accepted and one past
// it rejected. A ceiling raised in one place and not the other would otherwise
// surface only as an operator rejected at load time by a bound the relay no
// longer enforces.
func TestOutboxConfigBoundsMatchRelayCeilings(t *testing.T) {
	for _, bound := range []struct {
		key     string
		ceiling int
	}{
		{key: "APP__OUTBOX__BATCH_SIZE", ceiling: postgresoutbox.MaxBatchSize},
		{key: "APP__OUTBOX__PUBLISH_CONCURRENCY", ceiling: postgresoutbox.MaxPublishConcurrency},
		{key: "APP__OUTBOX__MAX_ATTEMPTS", ceiling: postgresoutbox.MaxAttempts},
		{key: "APP__OUTBOX__CLEANUP_BATCH_SIZE", ceiling: postgresoutbox.MaxCleanupBatchSize},
	} {
		t.Run(bound.key, func(t *testing.T) {
			setOutboxBootstrapEnvironment(t, true)
			t.Setenv(bound.key, strconv.Itoa(bound.ceiling))
			if _, _, err := config.LoadDetailed(config.LoadOptions{}); err != nil {
				t.Errorf("LoadDetailed(%s=%d) error = %v, want the ceiling accepted", bound.key, bound.ceiling, err)
			}
			t.Setenv(bound.key, strconv.Itoa(bound.ceiling+1))
			if _, _, err := config.LoadDetailed(config.LoadOptions{}); !errors.Is(err, config.ErrValidate) {
				t.Errorf("LoadDetailed(%s=%d) error = %v, want ErrValidate", bound.key, bound.ceiling+1, err)
			}
		})
	}
}

// The fifth restated constant. The lease budget is the one rule that spends
// internal/config's copy of postgresoutbox.PublisherJoinTimeout, so it is also
// the only place that can pin it from outside the package. A lease exactly
// equal to the budget it must exceed is rejected and one millisecond more is
// accepted; a copy that drifted either way moves one of those two verdicts.
func TestOutboxLeaseBudgetSpendsTheRelayJoinTimeout(t *testing.T) {
	const (
		publishTimeout   = 4 * time.Second
		acquireTimeout   = 2 * time.Second
		statementTimeout = 3 * time.Second
	)
	budget := publishTimeout + postgresoutbox.PublisherJoinTimeout + acquireTimeout + statementTimeout

	for _, lease := range []struct {
		name         string
		value        time.Duration
		wantAccepted bool
	}{
		{name: "at the budget", value: budget},
		{name: "past the budget", value: budget + time.Millisecond, wantAccepted: true},
	} {
		t.Run(lease.name, func(t *testing.T) {
			setOutboxBootstrapEnvironment(t, true)
			t.Setenv("APP__OUTBOX__PUBLISH_TIMEOUT", publishTimeout.String())
			t.Setenv("APP__POSTGRES__ACQUIRE_TIMEOUT", acquireTimeout.String())
			t.Setenv("APP__POSTGRES__STATEMENT_TIMEOUT", statementTimeout.String())
			t.Setenv("APP__OUTBOX__LEASE_DURATION", lease.value.String())
			_, _, err := config.LoadDetailed(config.LoadOptions{})
			if accepted := err == nil; accepted != lease.wantAccepted {
				t.Errorf("LoadDetailed(lease_duration=%s) error = %v, want accepted=%t",
					lease.value, err, lease.wantAccepted)
			}
		})
	}
}

// The two tests above pin constants. This one pins the field set, from the
// relay side: every relay budget an operator can set must be a RelayConfig
// field that ValidateRelayConfig rejects when it is out of range. Forget it
// there and a direct NewRelay caller runs on a zero value.
//
// The mirror claim — that internal/config rejects the same field — belongs to
// TestOutboxConfigValidatesEveryField in that package, which walks the same
// struct. Asserting it here too would mean two reflective walks, two
// type-to-zero mappers, and two skip lists keyed differently, all describing
// one rule. Together the two tests still cover both sides of every field.
//
// It walks config.OutboxConfig rather than listing fields, so a setting added
// later is covered without editing this test. The shipped defaults are loaded
// first, which is what makes the per-field mutations meaningful: each one
// starts from a configuration both sides accept.
func TestRelayRejectsEveryOutboxBudget(t *testing.T) {
	// Enabled is a switch, and DrainTimeout belongs to the process lifecycle
	// that validateRuntimeConfig owns rather than to the relay's own budget —
	// RelayConfig carries neither. Every other field is a relay budget, and zero
	// is out of range for all of them.
	processOwned := map[string]bool{"Enabled": true, "DrainTimeout": true}

	setOutboxBootstrapEnvironment(t, true)
	defaults, _, err := config.LoadDetailed(config.LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed(shipped defaults) error = %v", err)
	}
	if err := postgresoutbox.ValidateRelayConfig(relayConfig(defaults.Outbox)); err != nil {
		t.Fatalf("the shipped outbox defaults do not satisfy the relay: %v", err)
	}

	for field := range reflect.TypeFor[config.OutboxConfig]().Fields() {
		if processOwned[field.Name] {
			continue
		}
		t.Run(field.Name, func(t *testing.T) {
			relay := relayConfig(defaults.Outbox)
			budget := reflect.ValueOf(&relay).Elem().FieldByName(field.Name)
			if !budget.IsValid() {
				t.Fatalf("RelayConfig has no %s; the two sides no longer describe the same budget", field.Name)
			}
			budget.SetZero()
			if err := postgresoutbox.ValidateRelayConfig(relay); !errors.Is(err, postgresoutbox.ErrConfig) {
				t.Errorf("postgresoutbox accepted a zero %s: %v", field.Name, err)
			}
		})
	}
}

// relayConfig is a hand-written field-for-field copy. A field added to both
// OutboxConfig and RelayConfig but forgotten there still compiles and reaches
// the relay as a zero value, which ValidateRelayConfig only catches for the
// fields it happens to range over. Mapping an OutboxConfig whose every field is
// set must therefore leave no RelayConfig field zero.
func TestRelayConfigMapsEveryOutboxField(t *testing.T) {
	t.Parallel()

	var source config.OutboxConfig
	settable := reflect.ValueOf(&source).Elem()
	for index := range settable.NumField() {
		field := settable.Field(index)
		//nolint:exhaustive // The kinds OutboxConfig uses; the default rejects a new one.
		switch field.Kind() {
		case reflect.Bool:
			field.SetBool(true)
		case reflect.Int, reflect.Int64:
			// Distinct values, so a mapper that copies the wrong source field
			// into two targets is still visible when this test is read.
			field.SetInt(int64(index) + 1)
		default:
			t.Fatalf("OutboxConfig.%s is a %s; teach this test how to fill it",
				settable.Type().Field(index).Name, field.Kind())
		}
	}

	mapped := reflect.ValueOf(relayConfig(source))
	for index := range mapped.NumField() {
		if mapped.Field(index).IsZero() {
			t.Errorf("relayConfig() left RelayConfig.%s zero; add it to the mapper in run.go",
				mapped.Type().Field(index).Name)
		}
	}
}

const (
	// forcedDrainTimeout is the drain budget the drainRelay tests grant, long
	// enough that only a deliberately unresponsive relay reaches cancellation.
	forcedDrainTimeout = 20 * time.Second
	// fakeRelayCleanup is how long the forced-shutdown fake keeps working after
	// cancellation, standing in for a real relay finalizing its batch.
	fakeRelayCleanup = time.Second
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
			return postgresoutbox.RelayResult{}
		})
		runtimeCtx, runtimeCancel := context.WithCancel(context.Background())
		defer runtimeCancel()
		processCtx, processCancel := context.WithTimeout(context.Background(), time.Hour)
		defer processCancel()
		result := make(chan postgresoutbox.RelayResult, 1)
		go func() { result <- relay.Run(runtimeCtx) }()
		<-relay.started
		startedAt := time.Now()
		relay.StartDrain()
		got := drainRelay(processCtx, 20*time.Second, runtimeCancel, result)
		synctest.Wait()
		if time.Since(startedAt) != 0 {
			t.Fatalf("graceful drain elapsed = %s, want 0", time.Since(startedAt))
		}
		if got.CleanupUnsafe || got.Err != nil {
			t.Fatalf("drainRelay() = %+v, want the relay's own clean drain result", got)
		}
	})
}

func TestOutboxRelayForcedShutdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		relay := newFakeRelay(func(ctx context.Context, _ <-chan struct{}) postgresoutbox.RelayResult {
			<-ctx.Done()
			time.Sleep(fakeRelayCleanup)
			return postgresoutbox.RelayResult{CleanupUnsafe: true, Err: postgresoutbox.ErrPublisherStuck}
		})
		runtimeCtx, runtimeCancel := context.WithCancel(context.Background())
		processCtx, processCancel := context.WithTimeout(context.Background(), time.Hour)
		defer processCancel()
		result := make(chan postgresoutbox.RelayResult, 1)
		go func() { result <- relay.Run(runtimeCtx) }()
		<-relay.started
		startedAt := time.Now()
		relay.StartDrain()
		got := drainRelay(processCtx, forcedDrainTimeout, runtimeCancel, result)
		synctest.Wait()
		// The drain budget elapses first, then cancellation lets the relay run
		// its own cleanup before reporting.
		if want := forcedDrainTimeout + fakeRelayCleanup; time.Since(startedAt) != want {
			t.Fatalf("forced drain elapsed = %s, want %s", time.Since(startedAt), want)
		}
		if !got.CleanupUnsafe || !errors.Is(got.Err, postgresoutbox.ErrPublisherStuck) {
			t.Fatalf("drainRelay() = %+v, want cleanup-unsafe stuck publisher", got)
		}
	})
}

func TestOutboxRelayOuterJoinTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		relay := newFakeRelay(func(context.Context, <-chan struct{}) postgresoutbox.RelayResult {
			<-release
			return postgresoutbox.RelayResult{CleanupUnsafe: true}
		})
		runtimeCtx, runtimeCancel := context.WithCancel(context.Background())
		processCtx, processCancel := context.WithTimeout(context.Background(), time.Hour)
		defer processCancel()
		result := make(chan postgresoutbox.RelayResult, 1)
		go func() { result <- relay.Run(runtimeCtx) }()
		<-relay.started
		startedAt := time.Now()
		relay.StartDrain()
		got := drainRelay(processCtx, forcedDrainTimeout, runtimeCancel, result)
		if !got.CleanupUnsafe || !strings.Contains(got.Err.Error(), "join outbox relay") ||
			time.Since(startedAt) != forcedDrainTimeout+forcedJoin {
			t.Fatalf("outer join = %+v elapsed=%s", got, time.Since(startedAt))
		}
		close(release)
		synctest.Wait()
	})
}

func TestOutboxRelayTeardownReleasesOnceAndSkipsWhenUnsafe(t *testing.T) {
	var order []string
	teardown := relayTeardown{
		publisher: func(context.Context) { order = append(order, "publisher") },
		pool:      func() { order = append(order, "postgres") },
	}
	if err := teardown.release(t.Context()); err != nil {
		t.Fatalf("safe release error = %v", err)
	}
	if got := strings.Join(order, ","); got != "publisher,postgres" {
		t.Fatalf("safe release order = %q, want publisher,postgres", got)
	}
	// run defers release twice; the second call must find nothing owed.
	if err := teardown.release(t.Context()); err != nil {
		t.Fatalf("repeat release error = %v", err)
	}
	if got := strings.Join(order, ","); got != "publisher,postgres" {
		t.Fatalf("release ran twice: %q", got)
	}

	unsafe := relayTeardown{
		publisher: func(context.Context) { order = append(order, "unsafe-publisher") },
		pool:      func() { order = append(order, "unsafe-postgres") },
		unsafe:    true,
	}
	if err := unsafe.release(t.Context()); err != nil {
		t.Fatalf("unsafe release error = %v", err)
	}
	if got := strings.Join(order, ","); got != "publisher,postgres" {
		t.Fatalf("cleanup-unsafe dependencies were closed: %q", got)
	}
}

// A startup failure before the pool exists releases only the publisher.
func TestOutboxRelayTeardownWithoutPoolReleasesPublisher(t *testing.T) {
	closed := false
	teardown := relayTeardown{publisher: func(context.Context) { closed = true }}
	if err := teardown.release(t.Context()); err != nil {
		t.Fatalf("release error = %v", err)
	}
	if !closed {
		t.Fatal("publisher was not closed when no pool had been built")
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
		return postgresoutbox.RelayResult{Err: postgresoutbox.ErrPublisherPanic}
	})
	got := runRelayLifecycle(t.Context(), t.Context(), lifecycleConfig(), telemetry.New(), relay)
	if got.CleanupUnsafe || !errors.Is(got.Err, postgresoutbox.ErrPublisherPanic) || relay.Ready() {
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
	server := newDiagnosticsServer(ready.Load, telemetry.New())
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
	if err := postgresoutbox.ValidatePublisher(testPublisher{}); err != nil {
		t.Fatalf("ValidatePublisher(concrete publisher) error = %v", err)
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
	for _, invalidRuntime := range []struct {
		name   string
		config config.Config
	}{
		{name: "outbox disabled", config: config.Config{}},
		{
			name:   "postgres disabled",
			config: func() config.Config { value := validRuntime; value.Postgres.Enabled = false; return value }(),
		},
		{
			name:   "missing diagnostics address",
			config: func() config.Config { value := validRuntime; value.Observability.Metrics.Addr = ""; return value }(),
		},
	} {
		if err := validateRuntimeConfig(invalidRuntime.config); err == nil {
			t.Errorf("validateRuntimeConfig(%s) succeeded", invalidRuntime.name)
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
	setupTelemetry(t.Context(), cfg, telemetry.New(), slog.Default())(t.Context())

	for _, class := range []struct {
		name string
		err  error
		want string
	}{
		{name: "deadline", err: context.DeadlineExceeded, want: "deadline_exceeded"},
		{name: "canceled", err: context.Canceled, want: "canceled"},
		{name: "other", err: errors.New("setup"), want: "setup_error"},
	} {
		if got := telemetryFailureReason(class.err); got != class.want {
			t.Errorf("telemetryFailureReason(%s) = %q, want %q", class.name, got, class.want)
		}
	}
}

// A relay that cannot install telemetry still publishes, so setup degrades
// instead of aborting startup. A nil registry is the one failure SetupMetrics
// reports without touching an endpoint, which makes it the cheap stand-in for
// any hard metrics failure.
func TestOutboxRelayTelemetrySetupDegradesInsteadOfFailing(t *testing.T) {
	t.Parallel()

	metrics := &telemetry.Metrics{}
	cleanup := setupTelemetry(t.Context(), config.Config{}, metrics, slog.Default())
	if cleanup == nil {
		t.Fatal("setupTelemetry() returned no cleanup after a metrics failure")
	}
	cleanup(t.Context())
	if provider := metrics.MeterProvider(); provider == nil {
		t.Fatal("MeterProvider() = nil after a metrics failure, want the no-op provider")
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

func lifecycleConfig() config.Config {
	return config.Config{
		HTTP:   config.HTTPConfig{GracePeriod: time.Second},
		Outbox: config.OutboxConfig{DrainTimeout: time.Second},
		Observability: config.ObservabilityConfig{
			Metrics: config.MetricsConfig{Addr: "127.0.0.1:0"},
		},
	}
}
