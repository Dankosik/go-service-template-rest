package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel"
)

func TestMessagingCompositionRejectsEmptyHandlerBeforeConfig(t *testing.T) {
	if err := run(t.Context(), []string{"--config", "/does/not/exist"}, nil); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("run(nil handler builder) error = %v, want ErrRejected", err)
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

func TestWorkerDiagnosticsReadinessUsesImmediateMessagingState(t *testing.T) {
	healthSvc := health.New()
	if err := healthSvc.Refresh(t.Context(), time.Second, 3); err != nil {
		t.Fatalf("seed healthy readiness: %v", err)
	}
	server := newDiagnosticsServer("127.0.0.1:0", healthSvc, func() bool { return false }, telemetry.New())
	if server == nil {
		t.Fatal("newDiagnosticsServer() = nil")
	}
	recorder := httptest.NewRecorder()
	server.Handler.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/health/ready", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /health/ready status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
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
	err = run(t.Context(), nil, func(context.Context, config.Config, *slog.Logger) (natsjs.Handler, func(), error) {
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

func TestMessagingCompositionBuildsHandlerFromConfigAndCleansUpOnStartupFailure(t *testing.T) {
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve unavailable worker endpoint: %v", err)
	}
	url := "nats://" + listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close unavailable worker endpoint: %v", err)
	}
	setWorkerTestEnvironment(t, url, "127.0.0.1:19090")

	built := false
	cleaned := false
	previousMeter := otel.GetMeterProvider()
	previousTracer := otel.GetTracerProvider()
	err = run(t.Context(), nil, func(_ context.Context, cfg config.Config, log *slog.Logger) (natsjs.Handler, func(), error) {
		built = cfg.Messaging.Stream == "EVENTS" && log != nil &&
			otel.GetMeterProvider() != previousMeter && otel.GetTracerProvider() != previousTracer
		return func(context.Context, natsjs.Message) error { return nil }, func() { cleaned = true }, nil
	})
	if err == nil {
		t.Fatal("run(unavailable broker) error = nil")
	}
	if !built || !cleaned {
		t.Fatalf("handler construction state = built:%t cleaned:%t, want true/true; run error = %v", built, cleaned, err)
	}
}

func TestMessagingCompositionCleansUpInvalidHandlerResult(t *testing.T) {
	setWorkerTestEnvironment(t, "nats://127.0.0.1:1", "127.0.0.1:19090")
	cleaned := false
	err := run(t.Context(), nil, func(context.Context, config.Config, *slog.Logger) (natsjs.Handler, func(), error) {
		return nil, func() { cleaned = true }, nil
	})
	if !errors.Is(err, natsjs.ErrRejected) || !cleaned {
		t.Fatalf("invalid handler result = error:%v cleaned:%t, want ErrRejected and cleanup", err, cleaned)
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
