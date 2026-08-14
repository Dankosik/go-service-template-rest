// Idle keepalive is observed at the raw HTTP/2 boundary because no gRPC-level
// surface reports PING frames. Health is disabled and HEADERS are forbidden, so
// a PING cannot be confused with an RPC or standard health Watch stream.

package grpcclient_test

import (
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"github.com/example/go-service-template-rest/internal/waittest"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestClientIdleKeepaliveIsOptIn(t *testing.T) {
	const observationWindow = 35 * time.Second

	defaultPeer := startIdlePingPeer(t)
	explicitPeer := startIdlePingPeer(t)

	defaultConfig := grpcclient.DefaultConfig("passthrough:///" + defaultPeer.address)
	defaultConfig.HealthCheck = false
	defaultConnection := newIdleClient(t, defaultConfig)

	explicitConfig := grpcclient.DefaultConfig("passthrough:///" + explicitPeer.address)
	explicitConfig.HealthCheck = false
	explicitConfig.KeepalivePingInterval = 10 * time.Second
	explicitConfig.KeepalivePingTimeout = time.Second
	explicitConnection := newIdleClient(t, explicitConfig)

	defaultConnection.Connect()
	explicitConnection.Connect()
	waitForIdleHandshake(t, "default", defaultPeer.handshake)
	waitForIdleHandshake(t, "explicit", explicitPeer.handshake)

	timer := time.NewTimer(observationWindow)
	defer timer.Stop()
	explicitPings := explicitPeer.pings
	explicitPingObserved := false
	for {
		select {
		case <-defaultPeer.pings:
			t.Fatal("default client sent an idle keepalive PING")
		case <-defaultPeer.headers:
			t.Fatal("default client opened an RPC stream during idle observation")
		case <-explicitPeer.headers:
			t.Fatal("keepalive-enabled client opened an RPC stream during idle observation")
		case <-explicitPings:
			explicitPingObserved = true
			explicitPings = nil
		case <-timer.C:
			if !explicitPingObserved {
				t.Fatal("keepalive-enabled client sent no PING during the observation window")
			}
			return
		}
	}
}

func newIdleClient(t *testing.T, cfg grpcclient.Config) *grpc.ClientConn {
	t.Helper()

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
	return connection
}

type idlePingPeer struct {
	address   string
	handshake <-chan struct{}
	pings     <-chan struct{}
	headers   <-chan struct{}
}

func startIdlePingPeer(t *testing.T) idlePingPeer {
	t.Helper()

	listener := listenLoopback(t)
	handshake := make(chan struct{})
	pings := make(chan struct{}, 1)
	headers := make(chan struct{}, 1)
	done := make(chan error, 1)
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			done <- err
			return
		}
		defer func() { _ = connection.Close() }()
		done <- exchangeIdleFrames(connection, handshake, pings, headers)
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		select {
		case err := <-done:
			if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				t.Errorf("idle HTTP/2 peer error = %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("idle HTTP/2 peer did not stop")
		}
	})

	return idlePingPeer{
		address:   listener.Addr().String(),
		handshake: handshake,
		pings:     pings,
		headers:   headers,
	}
}

func waitForIdleHandshake(t *testing.T, label string, handshake <-chan struct{}) {
	t.Helper()
	waittest.ReceiveSignal(t, handshake, 5*time.Second, label+" idle peer HTTP/2 handshake")
}

func exchangeIdleFrames(
	connection net.Conn,
	handshake chan<- struct{},
	pings chan<- struct{},
	headers chan<- struct{},
) error {
	if _, err := io.ReadFull(connection, make([]byte, len(http2.ClientPreface))); err != nil {
		return fmt.Errorf("read HTTP/2 client preface: %w", err)
	}
	framer := http2.NewFramer(connection, connection)
	if err := framer.WriteSettings(); err != nil {
		return fmt.Errorf("write server settings: %w", err)
	}

	handshakeComplete := false
	for {
		frame, err := framer.ReadFrame()
		if err != nil {
			return fmt.Errorf("read client frame: %w", err)
		}
		switch frame := frame.(type) {
		case *http2.SettingsFrame:
			if frame.IsAck() {
				continue
			}
			if err := framer.WriteSettingsAck(); err != nil {
				return fmt.Errorf("write settings ack: %w", err)
			}
			if !handshakeComplete {
				close(handshake)
				handshakeComplete = true
			}
		case *http2.HeadersFrame:
			select {
			case headers <- struct{}{}:
			default:
			}
		case *http2.PingFrame:
			if frame.Flags.Has(http2.FlagPingAck) {
				continue
			}
			select {
			case pings <- struct{}{}:
			default:
			}
			if err := framer.WritePing(true, frame.Data); err != nil {
				return fmt.Errorf("write ping ack: %w", err)
			}
		}
	}
}
