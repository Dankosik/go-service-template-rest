//go:build !race

package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/config/configtest"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/infra/natsjs/natsjstest"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/nats-io/nats.go/jetstream"
)

func TestWorkerRunDrainsGracefully(t *testing.T) {
	server := natsjstest.Start(t, natsjstest.WithStreams(
		jetstream.StreamConfig{Name: "EVENTS", Subjects: []string{"events.>"}, Storage: jetstream.FileStorage, MaxMsgSize: 1 << 20},
		jetstream.StreamConfig{Name: "EVENTS_DLQ", Subjects: []string{"dead.>"}, Storage: jetstream.FileStorage, MaxMsgSize: 2 << 20},
	))
	waittest.Until(t, 5*time.Second, func() bool {
		name, err := server.JS.StreamNameBySubject(t.Context(), "events.test")
		return err == nil && name == "EVENTS"
	}, "source stream availability")
	waittest.Until(t, 5*time.Second, func() bool {
		name, err := server.JS.StreamNameBySubject(t.Context(), "dead.events.test")
		return err == nil && name == "EVENTS_DLQ"
	}, "dead-letter stream availability")
	diagnosticsAddr := waittest.FreeTCPAddr(t, "worker diagnostics")
	setDefaultWorkerTestEnvironment(t, server.URL, diagnosticsAddr)

	runCtx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	cleaned := make(chan struct{}, 1)
	result := make(chan error, 1)
	go func() {
		result <- run(runCtx, nil, func(context.Context, config.Config, *slog.Logger) (natsjs.Handler, func(context.Context), error) {
			return func(context.Context, natsjs.Message) error { return nil }, func(context.Context) { cleaned <- struct{}{} }, nil
		})
	}()

	// Readiness rather than the durable consumer, which the broker publishes
	// midway through admission: cancelling on it races the rest of NewWorker,
	// whose remaining probes then fail on the cancelled context and are reported
	// as a rejected admission.
	probe := &http.Client{Timeout: time.Second}
	waittest.Until(t, startupTimeout, func() bool {
		select {
		case err := <-result:
			t.Fatalf("run() before worker readiness: %v", err)
		default:
		}
		request, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet, "http://"+diagnosticsAddr+"/health/ready", http.NoBody,
		)
		if err != nil {
			return false
		}
		response, err := probe.Do(request)
		if err != nil {
			return false
		}
		defer func() { _ = response.Body.Close() }()
		return response.StatusCode == http.StatusOK
	}, "worker readiness")
	cancel()
	if err := waittest.Receive(t, result, 5*time.Second, "graceful worker shutdown"); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	waittest.ReceiveSignal(t, cleaned, time.Second, "feature cleanup")
}

func setDefaultWorkerTestEnvironment(t *testing.T, messagingURL, diagnosticsAddr string) {
	t.Helper()
	configtest.IsolateEnv(t)
	for key, value := range map[string]string{
		"APP__HTTP__READINESS_PROPAGATION_DELAY":            "0s",
		"APP__MESSAGING__ENABLED":                           "true",
		"APP__MESSAGING__URLS":                              messagingURL,
		"APP__MESSAGING__ALLOW_PLAINTEXT":                   "true",
		"APP__MESSAGING__ALLOW_UNAUTHENTICATED":             "true",
		"APP__MESSAGING__STREAM":                            "EVENTS",
		"APP__MESSAGING__MIN_STREAM_REPLICAS":               "1",
		"APP__MESSAGING__MIN_STREAM_RETENTION":              "24h",
		"APP__MESSAGING__WORKER__CONSUMER":                  "coverage-worker",
		"APP__MESSAGING__WORKER__FILTER_SUBJECT":            "events.test",
		"APP__MESSAGING__WORKER__DEAD_LETTER_SUBJECT":       "dead.events.test",
		"APP__MESSAGING__WORKER__HANDLER_TIMEOUT":           "1s",
		"APP__MESSAGING__WORKER__RETRY_DELAYS":              "50ms",
		"APP__MESSAGING__WORKER__DEAD_LETTER_RETRY_DELAY":   "50ms",
		"APP__MESSAGING__WORKER__DRAIN_TIMEOUT":             "2s",
		"APP__OBSERVABILITY__METRICS__ADDR":                 diagnosticsAddr,
		"APP__OBSERVABILITY__OTEL__EXPORTER__OTLP_ENDPOINT": "",
	} {
		t.Setenv(key, value)
	}
}
