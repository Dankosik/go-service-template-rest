package natsjs

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestClientStateTransitions(t *testing.T) {
	client := unitClient(t, &recordingJetStream{}, RoleProducer)
	client.ready.Store(true)
	if client.Name() != "messaging" || client.Producer() == nil || !client.Ready() {
		t.Fatal("client accessors did not expose admitted producer state")
	}
	client.draining.Store(true)
	if client.Ready() {
		t.Fatal("draining client remained ready")
	}
	client.draining.Store(false)
	if err := client.Check(t.Context()); err == nil || client.Ready() {
		t.Fatalf("Check(without connection) error = %v, ready = %t", err, client.Ready())
	}
	var nilClient *Client
	if err := nilClient.Check(t.Context()); err == nil {
		t.Fatal("nil Client.Check() error = nil")
	}
	if err := nilClient.Shutdown(t.Context()); err != nil {
		t.Fatalf("nil Client.Shutdown() error = %v", err)
	}
	nilClient.StopPublish()
	nilClient.Close()

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
	if client.waitForReconnect(t.Context(), errors.New("failure")) {
		t.Fatal("client without connection reported reconnect")
	}
	if err := client.Shutdown(t.Context()); err != nil {
		t.Fatalf("Shutdown(without connection) error = %v", err)
	}
	client.Close()

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
