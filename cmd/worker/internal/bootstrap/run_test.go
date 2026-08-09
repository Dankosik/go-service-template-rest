package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"go.opentelemetry.io/otel"
)

func TestMessagingCompositionRejectsEmptyHandlerBeforeConfig(t *testing.T) {
	if err := run(t.Context(), []string{"--config", "/does/not/exist"}, nil); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("run(nil handler builder) error = %v, want ErrRejected", err)
	}
}

func TestMessagingCompositionRejectsDisabledTransportWithRegisteredHandler(t *testing.T) {
	// profile:authn-oidc-jwt:start
	t.Setenv("APP__AUTHN__ISSUER", "https://issuer.example.com")
	t.Setenv("APP__AUTHN__AUDIENCE", "https://api.example.com")
	t.Setenv("APP__AUTHN__TRUSTED_PROXY_CIDRS", "127.0.0.0/8")
	// profile:authn-oidc-jwt:end
	t.Setenv("APP__MESSAGING__ENABLED", "false")
	built := false
	err := run(t.Context(), nil, func(context.Context, config.Config, *slog.Logger, *natsjs.Producer) (natsjs.Handler, func(context.Context), error) {
		built = true
		return func(context.Context, natsjs.Message) error { return nil }, nil, nil
	})
	if !errors.Is(err, natsjs.ErrRejected) || !strings.Contains(err.Error(), "messaging must be enabled for worker") {
		t.Fatalf("run(disabled messaging) error = %v, want disabled ErrRejected", err)
	}
	if built {
		t.Fatal("worker built the feature handler while messaging was disabled")
	}
}

func TestMessagingCompositionParsesWorkerBounds(t *testing.T) {
	cfg := config.MessagingConfig{
		MaxPayloadBytes: 256 << 10,
		Worker: config.MessagingWorkerConfig{
			Consumer: "events-worker", FilterSubject: "events.>", DeadLetterSubject: "dead.events",
			MaxConcurrency: 8, MaxDeliveryBytes: 1 << 20, HandlerTimeout: time.Second,
			RetryDelays: "10ms, 20ms", DeadLetterRetryDelay: time.Second, DrainTimeout: 2 * time.Second,
		},
	}
	got, err := messagingWorkerConfig(cfg)
	if err != nil {
		t.Fatalf("messagingWorkerConfig() error = %v", err)
	}
	if len(got.RetryDelays) != 2 || got.RetryDelays[0] != 10*time.Millisecond || got.RetryDelays[1] != 20*time.Millisecond {
		t.Fatalf("retry delays = %v", got.RetryDelays)
	}
}

func TestMessagingCompositionRejectsInvalidWorkerBeforeConnection(t *testing.T) {
	cfg := config.MessagingConfig{MaxPayloadBytes: 256 << 10, Worker: config.MessagingWorkerConfig{DrainTimeout: time.Second}}
	if _, err := messagingWorkerConfig(cfg); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("messagingWorkerConfig(invalid) error = %v, want ErrRejected", err)
	}
}

func TestWorkerShutdownBudgetFitsProcessGrace(t *testing.T) {
	t.Parallel()

	if err := validateWorkerShutdownBudget(45*time.Second, 20*time.Second); err != nil {
		t.Fatalf("shipped worker shutdown budget does not fit: %v", err)
	}
	if err := validateWorkerShutdownBudget(36*time.Second, 20*time.Second); !errors.Is(err, config.ErrValidate) {
		t.Fatalf("validateWorkerShutdownBudget(overrun) error = %v, want ErrValidate", err)
	}
}

//nolint:paralleltest // Installs a process-wide tracer provider for span capture.
func TestWorkerLoggerCorrelatesRecords(t *testing.T) {
	telemetrytest.InstallSpanRecorder(t)

	var out bytes.Buffer
	log := newWorkerLogger(&out, config.Config{
		App:           config.AppConfig{Env: "test", Version: "v1"},
		Observability: config.ObservabilityConfig{OTel: config.OTelConfig{ServiceName: "worker"}},
	})
	ctx, span := otel.Tracer("test").Start(context.Background(), "worker-log-test")
	log.InfoContext(ctx, "worker_message")
	span.End()

	var record map[string]any
	if err := json.Unmarshal(out.Bytes(), &record); err != nil {
		t.Fatalf("unmarshal worker log %q: %v", out.String(), err)
	}
	for _, key := range []string{"trace_id", "span_id"} {
		if value, ok := record[key].(string); !ok || value == "" {
			t.Fatalf("worker log %v is missing %s", record, key)
		}
	}
}

//nolint:paralleltest // Telemetry setup installs process-wide providers.
func TestWorkerTelemetrySetupCanBeCleanedWithinCallerBudget(t *testing.T) {
	telemetrytest.RestoreGlobals(t)
	telemetrytest.ClearAmbientExporterEnv(t)

	cleanup, err := runtimeopts.InstallTelemetry(t.Context(), config.Config{
		App: config.AppConfig{
			Env: "test", Version: "v1", Commit: "test-commit", InstanceID: "worker-test",
		},
		Observability: config.ObservabilityConfig{OTel: config.OTelConfig{
			ServiceName: "worker", TracesSampler: "always_off",
		}},
	}, telemetry.New(), slog.New(slog.DiscardHandler), "worker")
	if err != nil {
		t.Fatalf("runtimeopts.InstallTelemetry() error = %v", err)
	}
	cleanupCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cleanup(cleanupCtx)
}

func TestWorkerCompositionHelpers(t *testing.T) {
	t.Parallel()

	options, err := parseLoadOptions([]string{
		"--config", "config.yaml",
		"--config-overlay", "first.yaml",
		"--config-overlay", "second.yaml",
	})
	if err != nil {
		t.Fatalf("parseLoadOptions() error = %v", err)
	}
	if options.ConfigPath != "config.yaml" || len(options.ConfigOverlays) != 2 || options.ConfigOverlays[1] != "second.yaml" {
		t.Fatalf("parseLoadOptions() = %+v", options)
	}
	for _, args := range [][]string{
		{"--config", ""},
		{"--config-overlay", ""},
		{"unexpected"},
	} {
		if _, err := parseLoadOptions(args); err == nil {
			t.Fatalf("parseLoadOptions(%q) error = nil", args)
		}
	}

	clientCfg := runtimeopts.Messaging(config.MessagingConfig{
		URLs: " nats://one:4222, nats://two:4222 ", Stream: "EVENTS", MaxPayloadBytes: 1024,
	})
	if len(clientCfg.URLs) != 2 || clientCfg.URLs[0] != "nats://one:4222" || clientCfg.URLs[1] != "nats://two:4222" {
		t.Fatalf("runtimeopts.Messaging() URLs = %q", clientCfg.URLs)
	}
	if clientCfg.Stream != "EVENTS" || clientCfg.MaxPayloadBytes != 1024 {
		t.Fatalf("runtimeopts.Messaging() = %+v", clientCfg)
	}
}

func TestHandlerCleanupSafetyTracksWorkerExit(t *testing.T) {
	t.Parallel()

	stopped := make(chan struct{})
	close(stopped)
	if !handlerStoppedBeforeReturn(nil, stopped) {
		t.Fatal("completed graceful worker was not safe to clean up")
	}
	if !handlerStoppedBeforeReturn(errors.New("forced"), stopped) {
		t.Fatal("completed forced worker was not safe to clean up")
	}
	if handlerStoppedBeforeReturn(errors.New("forced"), make(chan struct{})) {
		t.Fatal("running forced handler was marked safe to clean up")
	}
}

//nolint:paralleltest // Worker setup installs process-wide telemetry providers.
func TestMessagingCompositionDoesNotBuildHandlerBeforeBrokerAdmission(t *testing.T) {
	telemetrytest.RestoreGlobals(t)
	telemetrytest.ClearAmbientExporterEnv(t)

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve unavailable worker endpoint: %v", err)
	}
	url := "nats://" + listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close unavailable worker endpoint: %v", err)
	}
	setWorkerTestEnvironment(t, url, "127.0.0.1:0")
	built := false
	err = run(t.Context(), nil, func(context.Context, config.Config, *slog.Logger, *natsjs.Producer) (natsjs.Handler, func(context.Context), error) {
		built = true
		return func(context.Context, natsjs.Message) error { return nil }, nil, nil
	})
	if err == nil {
		t.Fatal("run(unavailable broker) error = nil")
	}
	if built {
		t.Fatal("worker built the feature handler before broker topology admission")
	}
}

func TestWorkerDiagnosticsReadinessUsesImmediateMessagingState(t *testing.T) {
	healthSvc := health.New()
	if err := healthSvc.Refresh(t.Context(), time.Second, 3); err != nil {
		t.Fatalf("seed healthy readiness: %v", err)
	}
	server := runtimeopts.DiagnosticsServer(workerReady(func() bool { return false }, healthSvc), telemetry.New())
	recorder := httptest.NewRecorder()
	server.Handler.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/health/ready", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /health/ready status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}

	server = runtimeopts.DiagnosticsServer(workerReady(func() bool { return true }, healthSvc), telemetry.New())
	for path := range map[string]struct{}{"/health/live": {}, "/health/ready": {}, "/metrics": {}} {
		recorder = httptest.NewRecorder()
		server.Handler.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want %d", path, recorder.Code, http.StatusOK)
		}
	}
}

func TestMessagingCompositionRejectsMissingDiagnosticsBeforeConnection(t *testing.T) {
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for worker connection sentinel: %v", err)
	}
	accepted := make(chan struct{}, 1)
	acceptDone := make(chan struct{})
	go func() {
		defer close(acceptDone)
		connection, acceptErr := listener.Accept()
		if acceptErr == nil {
			_ = connection.Close()
			accepted <- struct{}{}
		}
	}()
	setWorkerTestEnvironment(t, "nats://"+listener.Addr().String(), "")
	err = run(t.Context(), nil, func(context.Context, config.Config, *slog.Logger, *natsjs.Producer) (natsjs.Handler, func(context.Context), error) {
		return func(context.Context, natsjs.Message) error { return nil }, nil, nil
	})
	if !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("run(missing diagnostics) error = %v, want ErrRejected", err)
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("close worker connection sentinel: %v", err)
	}
	<-acceptDone
	select {
	case <-accepted:
		t.Fatal("worker connected before rejecting missing diagnostics address")
	default:
	}
}

func setWorkerTestEnvironment(t *testing.T, messagingURL, diagnosticsAddr string) {
	t.Helper()
	for key, value := range map[string]string{
		// profile:authn-oidc-jwt:start
		"APP__AUTHN__ISSUER":              "https://issuer.example.com",
		"APP__AUTHN__AUDIENCE":            "https://api.example.com",
		"APP__AUTHN__TRUSTED_PROXY_CIDRS": "127.0.0.0/8",
		// profile:authn-oidc-jwt:end
		"APP__MESSAGING__ENABLED":                         "true",
		"APP__MESSAGING__URLS":                            messagingURL,
		"APP__MESSAGING__ALLOW_PLAINTEXT":                 "true",
		"APP__MESSAGING__ALLOW_UNAUTHENTICATED":           "true",
		"APP__MESSAGING__STREAM":                          "EVENTS",
		"APP__MESSAGING__WORKER__CONSUMER":                "missing-diagnostics-worker",
		"APP__MESSAGING__WORKER__FILTER_SUBJECT":          "events.test",
		"APP__MESSAGING__WORKER__DEAD_LETTER_SUBJECT":     "dead.events.test",
		"APP__MESSAGING__WORKER__HANDLER_TIMEOUT":         "1s",
		"APP__MESSAGING__WORKER__RETRY_DELAYS":            "50ms",
		"APP__MESSAGING__WORKER__DEAD_LETTER_RETRY_DELAY": "50ms",
		"APP__MESSAGING__WORKER__DRAIN_TIMEOUT":           "2s",
		"APP__OBSERVABILITY__METRICS__ADDR":               diagnosticsAddr,
	} {
		t.Setenv(key, value)
	}
}

var _ natsjs.Handler = func(context.Context, natsjs.Message) error { return nil }
