package bootstrap

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
)

func TestMessagingCompositionRejectsEmptyHandlerBeforeConfig(t *testing.T) {
	if err := run(t.Context(), []string{"--config", "/does/not/exist"}, nil); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("run(nil handler) error = %v, want ErrRejected", err)
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
	for key, value := range map[string]string{
		// profile:authn-oidc-jwt:start
		"APP__AUTHN__ISSUER":              "https://issuer.example.com",
		"APP__AUTHN__AUDIENCE":            "https://api.example.com",
		"APP__AUTHN__TRUSTED_PROXY_CIDRS": "127.0.0.0/8",
		// profile:authn-oidc-jwt:end
		"APP__MESSAGING__ENABLED":                         "true",
		"APP__MESSAGING__URLS":                            "nats://" + listener.Addr().String(),
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
		"APP__OBSERVABILITY__METRICS__ADDR":               "",
	} {
		t.Setenv(key, value)
	}
	err = run(t.Context(), nil, func(context.Context, natsjs.Message) error { return nil })
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

var _ natsjs.Handler = func(context.Context, natsjs.Message) error { return nil }
