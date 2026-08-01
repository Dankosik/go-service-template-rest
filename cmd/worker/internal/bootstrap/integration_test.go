//go:build integration

package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const workerTestMaxDeliveryBytes = 1 << 20

func workerTestProducerConfig() natsjs.Config {
	return natsjs.Config{MaxPayloadBytes: 256 << 10, MaxPendingPublishes: 64}
}

func TestNATSWorkerComposition(t *testing.T) {
	url, js := workerNATSFixture(t)
	diagnosticsAddress := reserveWorkerAddress(t)

	setWorkerEnvironment(t, url, "composition-worker", diagnosticsAddress)

	entered := make(chan natsjs.Message, 1)
	release := make(chan struct{})
	runCtx, cancelRun := context.WithCancel(t.Context())
	runErr := make(chan error, 1)
	go func() {
		runErr <- run(runCtx, nil, func(context.Context, config.Config, *slog.Logger) (natsjs.Handler, func(), error) {
			return func(_ context.Context, msg natsjs.Message) error {
				entered <- msg
				<-release
				return nil
			}, nil, nil
		})
	}()
	waitWorker(t, 10*time.Second, func() bool {
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
}

func TestNATSWorkerHandlerPanicIsSupervised(t *testing.T) {
	url, js := workerNATSFixture(t)
	diagnosticsAddress := reserveWorkerAddress(t)
	setWorkerEnvironment(t, url, "panic-composition-worker", diagnosticsAddress)
	runCtx, cancelRun := context.WithCancel(t.Context())
	defer cancelRun()
	runErr := make(chan error, 1)
	go func() {
		runErr <- run(runCtx, nil, func(context.Context, config.Config, *slog.Logger) (natsjs.Handler, func(), error) {
			return func(context.Context, natsjs.Message) error {
				panic("worker panic canary")
			}, nil, nil
		})
	}()
	waitWorker(t, 10*time.Second, func() bool {
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
	err = receiveWorker(t, runErr, 10*time.Second, "supervised handler panic")
	if !errors.Is(err, natsjs.ErrTerminal) || strings.Contains(err.Error(), "worker panic canary") {
		t.Fatalf("worker panic run error = %v, want sanitized ErrTerminal", err)
	}
}

func workerNATSFixture(t *testing.T) (string, jetstream.JetStream) {
	t.Helper()
	container, err := testcontainers.GenericContainer(t.Context(), testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "nats:2.14.3-alpine", ExposedPorts: []string{"4222/tcp"}, Cmd: []string{"-js", "-sd", "/data"},
			WaitingFor: wait.ForAll(wait.ForListeningPort("4222/tcp"), wait.ForLog("Server is ready")).WithDeadline(time.Minute),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start NATS: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := container.Terminate(cleanupCtx); err != nil {
			t.Errorf("terminate NATS: %v", err)
		}
	})
	endpoint, err := container.Endpoint(t.Context(), "")
	if err != nil {
		t.Fatalf("resolve NATS endpoint: %v", err)
	}
	url := "nats://" + endpoint
	connection, err := nats.Connect(url, nats.Timeout(5*time.Second))
	if err != nil {
		t.Fatalf("connect fixture: %v", err)
	}
	t.Cleanup(connection.Close)
	js, err := jetstream.New(connection)
	if err != nil {
		t.Fatalf("create JetStream fixture: %v", err)
	}
	for _, stream := range []jetstream.StreamConfig{
		{Name: "EVENTS", Subjects: []string{"events.>"}, Storage: jetstream.FileStorage, MaxMsgSize: workerTestMaxDeliveryBytes},
		{Name: "EVENTS_DLQ", Subjects: []string{"dead.>"}, Storage: jetstream.FileStorage, MaxMsgSize: 2 * workerTestMaxDeliveryBytes},
	} {
		if _, err := js.CreateStream(t.Context(), stream); err != nil {
			t.Fatalf("create stream %s: %v", stream.Name, err)
		}
	}
	return url, js
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

func reserveWorkerAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve worker diagnostics address: %v", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("release worker diagnostics address: %v", err)
	}
	return address
}

func waitWorkerHTTPStatus(t *testing.T, address, path string, want int) {
	t.Helper()
	client := &http.Client{Timeout: 500 * time.Millisecond}
	waitWorker(t, 5*time.Second, func() bool {
		response, err := client.Get("http://" + address + path)
		if err != nil {
			return false
		}
		_ = response.Body.Close()
		return response.StatusCode == want
	}, path)
}

func receiveWorker[T any](t *testing.T, values <-chan T, timeout time.Duration, description string) T {
	t.Helper()
	select {
	case value := <-values:
		return value
	case <-time.After(timeout):
		var zero T
		t.Fatalf("timed out waiting for %s", description)
		return zero
	}
}

func waitWorker(t *testing.T, timeout time.Duration, predicate func() bool, description string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), timeout)
	defer cancel()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		if predicate() {
			return
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatalf("timed out waiting for %s", description)
		}
	}
}
