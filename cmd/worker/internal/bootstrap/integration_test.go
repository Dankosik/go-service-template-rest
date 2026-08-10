//go:build integration

package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/infra/natsjs/natsjstest"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/nats-io/nats.go/jetstream"
)

const workerTestMaxDeliveryBytes = 1 << 20

func workerTestProducerConfig() natsjs.Config {
	return natsjs.Config{
		MinStreamReplicas: 1, MinStreamRetention: 24 * time.Hour,
		MaxPayloadBytes: 256 << 10, MaxPendingPublishes: 64,
	}
}

func TestNATSWorkerComposition(t *testing.T) {
	url, js := workerNATSFixture(t)
	diagnosticsAddress := waittest.FreeTCPAddr(t, "worker diagnostics")

	setWorkerEnvironment(t, url, "composition-worker", diagnosticsAddress)

	entered := make(chan natsjs.Message, 1)
	release := make(chan struct{})
	cleaned := make(chan struct{})
	runCtx, cancelRun := context.WithCancel(t.Context())
	runErr := make(chan error, 1)
	go func() {
		runErr <- run(runCtx, nil, func(_ context.Context, _ config.Config, _ *slog.Logger) (natsjs.Handler, func(context.Context), error) {
			return func(_ context.Context, msg natsjs.Message) error {
				entered <- msg
				<-release
				return nil
			}, func(context.Context) { close(cleaned) }, nil
		})
	}()
	waittest.Until(t, 10*time.Second, func() bool {
		_, err := js.Consumer(t.Context(), "EVENTS", "composition-worker")
		return err == nil
	}, "worker consumer admission")

	producerCfg := workerTestProducerConfig()
	producerCfg.URLs = []string{url}
	producerCfg.AllowPlaintext = true
	producerCfg.AllowUnauthenticated = true
	producerCfg.Stream = "EVENTS"
	producer, err := natsjs.Connect(t.Context(), producerCfg, natsjs.RoleProducer, natsjs.Observability{})
	if err != nil {
		t.Fatalf("connect fixture producer: %v", err)
	}
	t.Cleanup(producer.Close)
	if _, err := producer.Producer().Publish(t.Context(), natsjs.Event{
		Subject: "events.test", MessageID: natsjs.NewID(), PublicationID: natsjs.NewID(),
		Type: "composition.test", Schema: "v1", CreatedAt: time.Now().UTC(), Payload: []byte("worker composition"),
	}); err != nil {
		t.Fatalf("publish worker fixture: %v", err)
	}
	select {
	case got := <-entered:
		if string(got.Payload()) != "worker composition" {
			t.Fatalf("handler payload = %q", got.Payload())
		}
	case err := <-runErr:
		t.Fatalf("worker stopped before delivery: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for worker delivery")
	}
	cancelRun()
	waitWorkerHTTPStatus(t, diagnosticsAddress, "/health/ready", http.StatusServiceUnavailable)
	waitWorkerHTTPStatus(t, diagnosticsAddress, "/metrics", http.StatusOK)
	close(release)
	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("worker run error = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("worker composition did not drain")
	}
	select {
	case <-cleaned:
	default:
		t.Fatal("graceful worker returned before feature cleanup")
	}
}

func TestNATSWorkerForcedShutdownDoesNotRaceHandlerCleanup(t *testing.T) {
	url, js := workerNATSFixture(t)
	diagnosticsAddress := waittest.FreeTCPAddr(t, "worker diagnostics")
	setWorkerEnvironment(t, url, "forced-cleanup-worker", diagnosticsAddress)
	t.Setenv("APP__MESSAGING__WORKER__DRAIN_TIMEOUT", "50ms")

	entered := make(chan struct{})
	release := make(chan struct{})
	exited := make(chan struct{})
	cleaned := make(chan struct{}, 1)
	runCtx, cancelRun := context.WithCancel(t.Context())
	runErr := make(chan error, 1)
	go func() {
		runErr <- run(runCtx, nil, func(context.Context, config.Config, *slog.Logger) (natsjs.Handler, func(context.Context), error) {
			return func(context.Context, natsjs.Message) error {
				close(entered)
				<-release
				close(exited)
				return nil
			}, func(context.Context) { cleaned <- struct{}{} }, nil
		})
	}()
	waittest.Until(t, 10*time.Second, func() bool {
		_, err := js.Consumer(t.Context(), "EVENTS", "forced-cleanup-worker")
		return err == nil
	}, "forced-cleanup worker admission")

	producerCfg := workerTestProducerConfig()
	producerCfg.URLs = []string{url}
	producerCfg.AllowPlaintext = true
	producerCfg.AllowUnauthenticated = true
	producerCfg.Stream = "EVENTS"
	producer, err := natsjs.Connect(t.Context(), producerCfg, natsjs.RoleProducer, natsjs.Observability{})
	if err != nil {
		t.Fatalf("connect forced-cleanup producer: %v", err)
	}
	t.Cleanup(producer.Close)
	if _, err := producer.Producer().Publish(t.Context(), natsjs.Event{
		Subject: "events.test", MessageID: natsjs.NewID(), PublicationID: natsjs.NewID(),
		Type: "composition.forced-cleanup", Schema: "v1", CreatedAt: time.Now().UTC(), Payload: []byte("forced cleanup"),
	}); err != nil {
		t.Fatalf("publish forced-cleanup fixture: %v", err)
	}
	waittest.Receive(t, entered, 10*time.Second, "forced-cleanup handler entry")
	cancelRun()
	if err := waittest.Receive(t, runErr, 5*time.Second, "forced worker shutdown"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("forced worker shutdown error = %v, want deadline exceeded", err)
	}
	select {
	case <-cleaned:
		t.Fatal("feature cleanup raced an uncooperative handler")
	default:
	}
	close(release)
	waittest.Receive(t, exited, 5*time.Second, "forced handler exit")
}

func TestNATSWorkerHandlerPanicIsSupervised(t *testing.T) {
	url, js := workerNATSFixture(t)
	diagnosticsAddress := waittest.FreeTCPAddr(t, "worker diagnostics")
	setWorkerEnvironment(t, url, "panic-composition-worker", diagnosticsAddress)
	runCtx, cancelRun := context.WithCancel(t.Context())
	defer cancelRun()
	runErr := make(chan error, 1)
	go func() {
		runErr <- run(runCtx, nil, func(context.Context, config.Config, *slog.Logger) (natsjs.Handler, func(context.Context), error) {
			return func(context.Context, natsjs.Message) error {
				panic("worker panic canary")
			}, nil, nil
		})
	}()
	waittest.Until(t, 10*time.Second, func() bool {
		_, err := js.Consumer(t.Context(), "EVENTS", "panic-composition-worker")
		return err == nil
	}, "panic worker consumer admission")
	producerCfg := workerTestProducerConfig()
	producerCfg.URLs = []string{url}
	producerCfg.AllowPlaintext = true
	producerCfg.AllowUnauthenticated = true
	producerCfg.Stream = "EVENTS"
	producer, err := natsjs.Connect(t.Context(), producerCfg, natsjs.RoleProducer, natsjs.Observability{})
	if err != nil {
		t.Fatalf("connect panic fixture producer: %v", err)
	}
	t.Cleanup(producer.Close)
	if _, err := producer.Producer().Publish(t.Context(), natsjs.Event{
		Subject: "events.test", MessageID: natsjs.NewID(), PublicationID: natsjs.NewID(),
		Type: "composition.panic", Schema: "v1", CreatedAt: time.Now().UTC(), Payload: []byte("panic"),
	}); err != nil {
		t.Fatalf("publish panic fixture: %v", err)
	}
	err = waittest.Receive(t, runErr, 10*time.Second, "supervised handler panic")
	if !errors.Is(err, natsjs.ErrTerminal) || strings.Contains(err.Error(), "worker panic canary") {
		t.Fatalf("worker panic run error = %v, want sanitized ErrTerminal", err)
	}
}

func workerNATSFixture(t *testing.T) (string, jetstream.JetStream) {
	t.Helper()
	server := natsjstest.Start(t, natsjstest.WithStreams(
		jetstream.StreamConfig{
			Name: "EVENTS", Subjects: []string{"events.>"},
			Storage: jetstream.FileStorage, MaxMsgSize: workerTestMaxDeliveryBytes,
		},
		jetstream.StreamConfig{
			Name: "EVENTS_DLQ", Subjects: []string{"dead.>"},
			Storage: jetstream.FileStorage, MaxMsgSize: 2 * workerTestMaxDeliveryBytes,
		},
	))
	return server.URL, server.JS
}

func setWorkerEnvironment(t *testing.T, url, consumer, diagnosticsAddress string) {
	t.Helper()
	for key, value := range map[string]string{
		// profile:authn-oidc-jwt:start
		"APP__AUTHN__ISSUER":              "https://issuer.example.com",
		"APP__AUTHN__AUDIENCE":            "https://api.example.com",
		"APP__AUTHN__TRUSTED_PROXY_CIDRS": "127.0.0.0/8",
		// profile:authn-oidc-jwt:end
		"APP__HTTP__READINESS_PROPAGATION_DELAY":            "0s",
		"APP__MESSAGING__ENABLED":                           "true",
		"APP__MESSAGING__URLS":                              url,
		"APP__MESSAGING__ALLOW_PLAINTEXT":                   "true",
		"APP__MESSAGING__ALLOW_UNAUTHENTICATED":             "true",
		"APP__MESSAGING__STREAM":                            "EVENTS",
		"APP__MESSAGING__MIN_STREAM_REPLICAS":               "1",
		"APP__MESSAGING__MIN_STREAM_RETENTION":              "24h",
		"APP__MESSAGING__WORKER__CONSUMER":                  consumer,
		"APP__MESSAGING__WORKER__FILTER_SUBJECT":            "events.test",
		"APP__MESSAGING__WORKER__DEAD_LETTER_SUBJECT":       "dead.events.test",
		"APP__MESSAGING__WORKER__HANDLER_TIMEOUT":           "1s",
		"APP__MESSAGING__WORKER__RETRY_DELAYS":              "50ms",
		"APP__MESSAGING__WORKER__DEAD_LETTER_RETRY_DELAY":   "50ms",
		"APP__MESSAGING__WORKER__DRAIN_TIMEOUT":             "2s",
		"APP__OBSERVABILITY__METRICS__ADDR":                 diagnosticsAddress,
		"APP__OBSERVABILITY__OTEL__EXPORTER__OTLP_ENDPOINT": "",
	} {
		t.Setenv(key, value)
	}
}

func waitWorkerHTTPStatus(t *testing.T, address, path string, want int) {
	t.Helper()
	client := &http.Client{Timeout: 500 * time.Millisecond}
	waittest.Until(t, 5*time.Second, func() bool {
		response, err := client.Get("http://" + address + path)
		if err != nil {
			return false
		}
		_ = response.Body.Close()
		return response.StatusCode == want
	}, path)
}
