package natsjs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

func TestConnectOptionsApplyReconnectPolicyAndLifecycleHandlers(t *testing.T) {
	t.Parallel()
	client := unitClient(t, &recordingJetStream{}, RoleProducer)
	client.ready.Store(true)

	options := nats.GetDefaultOptions()
	for _, option := range client.connectOptions(context.Background(), Config{}) {
		if err := option(&options); err != nil {
			t.Fatalf("apply connection option: %v", err)
		}
	}
	if options.Name != "service-messaging" || options.Timeout != operationTimeout ||
		options.ReconnectWait != time.Second || options.ReconnectJitter != 50*time.Millisecond ||
		options.ReconnectJitterTLS != 50*time.Millisecond || options.MaxReconnect != 60 ||
		options.ReconnectBufSize != -1 {
		t.Fatalf("connection policy = %+v", options)
	}

	options.DisconnectedErrCB(nil, errors.New("disconnect"))
	if client.ready.Load() {
		t.Fatal("disconnect left the client ready")
	}
	client.ready.Store(true)
	options.ReconnectedCB(nil)
	if client.ready.Load() {
		t.Fatal("reconnect restored readiness before the probe")
	}
	select {
	case event := <-client.events:
		if event != eventReconnect {
			t.Fatalf("reconnect event = %v", event)
		}
	default:
		t.Fatal("reconnect did not request a readiness probe")
	}
	select {
	case <-client.reconnected:
	default:
		t.Fatal("reconnect did not notify waiters")
	}

	options.ClosedCB(nil)
	select {
	case <-client.closed:
	default:
		t.Fatal("unexpected close did not close the lifecycle channel")
	}
	if err := <-client.terminal; !errors.Is(err, ErrTerminal) {
		t.Fatalf("unexpected close terminal error = %v", err)
	}
}
