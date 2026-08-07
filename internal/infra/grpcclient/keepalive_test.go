// Does this client actually ping a connection with no active RPC?
//
// That is the behavior the documented shape depends on — one long-lived
// connection per dependency, built at startup — and without it a NAT or load
// balancer idle timeout discards that connection silently.
//
// The peer is a raw HTTP/2 speaker rather than a gRPC server, because a PING
// frame is the observable and no gRPC-level surface reports one. It opens no
// stream, so every ping it sees arrived on an idle connection. The raw peer in
// transparent_retry_test.go is the same shape for the same reason.
//
// It costs at least ten seconds of wall clock: grpc-go raises a client ping
// interval below ten seconds to ten, and the behavior lives inside its transport
// goroutines, so testing/synctest cannot reach it. The alternative is asserting
// that a dial option was passed, which proves nothing about pinging.

package grpcclient_test

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"golang.org/x/net/http2"
	"google.golang.org/grpc/credentials/insecure"
)

func TestClientPingsAConnectionWithNoActiveRPC(t *testing.T) {
	// grpc-go's floor. Configuring anything smaller does not ping sooner; it
	// pings at ten seconds while the configured value reads faster.
	const pingInterval = 10 * time.Second

	pinged := make(chan struct{}, 1)
	listener := listenLoopback(t)
	go serveIdlePingPeer(listener, pinged)

	cfg := grpcclient.DefaultConfig("passthrough:///" + listener.Addr().String())
	cfg.KeepalivePingInterval = pingInterval
	connection, err := grpcclient.New(cfg, grpcclient.Options{
		TransportCredentials: insecure.NewCredentials(),
	})
	if err != nil {
		t.Fatalf("grpcclient.New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := connection.Close(); err != nil {
			t.Errorf("ClientConn.Close() error = %v", err)
		}
	})

	// Bring the transport up without opening a stream: from here the connection
	// is idle, and only a keepalive ping can put a frame on it.
	connection.Connect()

	select {
	case <-pinged:
	case <-time.After(3 * pingInterval):
		t.Fatalf(
			"no ping reached an idle connection in %s: this client does not keep one alive",
			3*pingInterval,
		)
	}
}

// serveIdlePingPeer accepts one connection, completes the HTTP/2 handshake, and
// signals the first ping the client sends. It acks pings so the client has no
// reason to tear the connection down, and opens no stream, so what it observes
// is unambiguously a keepalive ping rather than RPC traffic.
func serveIdlePingPeer(listener net.Listener, pinged chan<- struct{}) {
	connection, err := listener.Accept()
	if err != nil {
		return
	}
	defer func() { _ = connection.Close() }()

	_ = exchangeIdlePings(connection, pinged)
}

func exchangeIdlePings(connection net.Conn, pinged chan<- struct{}) error {
	if _, err := io.ReadFull(connection, make([]byte, len(http2.ClientPreface))); err != nil {
		return fmt.Errorf("read HTTP/2 client preface: %w", err)
	}
	framer := http2.NewFramer(connection, connection)
	if err := framer.WriteSettings(); err != nil {
		return fmt.Errorf("write server settings: %w", err)
	}

	for {
		frame, err := framer.ReadFrame()
		if err != nil {
			return fmt.Errorf("read client frame: %w", err)
		}
		switch frame := frame.(type) {
		case *http2.SettingsFrame:
			if !frame.IsAck() {
				if err := framer.WriteSettingsAck(); err != nil {
					return fmt.Errorf("write settings ack: %w", err)
				}
			}
		case *http2.PingFrame:
			if frame.Flags.Has(http2.FlagPingAck) {
				continue
			}
			select {
			case pinged <- struct{}{}:
			default:
			}
			if err := framer.WritePing(true, frame.Data); err != nil {
				return fmt.Errorf("write ping ack: %w", err)
			}
		}
	}
}
