package natsjs

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestClientLifecycleWithoutBroker(t *testing.T) {
	client := unitClient(t, &recordingJetStream{})
	client.ready.Store(true)
	if client.Name() != "messaging" || client.Producer() == nil || !client.Ready() {
		t.Fatal("client accessors did not expose ready producer state")
	}
	client.StopPublish()
	if client.Ready() {
		t.Fatal("draining client remained ready")
	}
	if err := client.Check(t.Context()); !errors.Is(err, ErrRejected) {
		t.Fatalf("Check() error = %v, want ErrRejected", err)
	}

	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if err := client.Run(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run(canceled) error = %v", err)
	}
	want := errors.New("terminal")
	client.signalTerminal(want)
	client.signalTerminal(errors.New("ignored duplicate"))
	if err := client.Run(t.Context()); !errors.Is(err, want) {
		t.Fatalf("Run(terminal) error = %v", err)
	}
	if err := client.Shutdown(t.Context()); err != nil {
		t.Fatalf("Shutdown(without connection) error = %v", err)
	}
	client.Close()
	client.Close()

	var nilClient *Client
	if nilClient.Ready() {
		t.Fatal("nil client reported ready")
	}
	if err := nilClient.Check(t.Context()); !errors.Is(err, ErrRejected) {
		t.Fatalf("nil Check() error = %v", err)
	}
	if err := nilClient.Shutdown(t.Context()); err != nil {
		t.Fatalf("nil Shutdown() error = %v", err)
	}
	nilClient.StopPublish()
	nilClient.Close()
}

func TestClientConnectionAdmissionAndTimeoutOptions(t *testing.T) {
	valid := Config{
		URLs: []string{"nats://127.0.0.1:4222"}, Stream: "EVENTS", MaxPayloadBytes: testMaxPayloadBytes,
		AllowPlaintext: true, AllowUnauthenticated: true,
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := Connect(canceled, valid, Observability{}); !errors.Is(err, ErrRejected) {
		t.Fatalf("Connect(canceled) error = %v, want ErrRejected", err)
	}
	if _, err := Connect(t.Context(), Config{}, Observability{}); !errors.Is(err, ErrRejected) {
		t.Fatalf("Connect(invalid) error = %v, want ErrRejected", err)
	}

	client := unitClient(t, &recordingJetStream{})
	withCredentials := valid
	withCredentials.CredentialsFile = "/run/secrets/nats.creds"
	withCredentials.RootCAFile = "/run/secrets/nats-ca.pem"
	if options := client.connectOptions(t.Context(), withCredentials); len(options) < 10 {
		t.Fatalf("connectOptions() count = %d", len(options))
	}

	expired, expiredCancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer expiredCancel()
	if got := boundedTimeout(expired); got != time.Nanosecond {
		t.Fatalf("boundedTimeout(expired) = %v", got)
	}
	short, shortCancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer shortCancel()
	if got := boundedTimeout(short); got <= 0 || got > 10*time.Millisecond {
		t.Fatalf("boundedTimeout(short) = %v", got)
	}
	if got := boundedTimeout(t.Context()); got != operationTimeout {
		t.Fatalf("boundedTimeout(no deadline) = %v", got)
	}
}
