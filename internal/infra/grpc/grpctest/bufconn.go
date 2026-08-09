package grpctest

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// bufconnBuffer is the in-memory listener's buffer. It is large enough that no
// test in this repository blocks a writer on it, which is what keeps a stalled
// assertion a fault in the code under test rather than in the harness.
const bufconnBuffer = 1 << 20

// BufconnServer is the part of a gRPC server [ServeBufconn] drives.
//
// It is an interface rather than the concrete server type because
// internal/infra/grpc's own tests import this package; naming that type here
// would make the pair an import cycle.
type BufconnServer interface {
	Serve(listener net.Listener) error
	Close() error
}

// ServeBufconn serves server over an in-memory listener and returns a client
// connected to it. Both are released when the test finishes.
//
// The teardown order is why this is shared rather than retyped. Close only
// starts the transport stop, and that stop is what closes the listener, so the
// client goes first, then the server, and only then is Serve joined. Closing the
// listener before that join races the stop and surfaces as an Accept failure
// rather than a clean one — a shutdown bug that belongs to the test.
//
// The connection is insecure because the listener is in memory and has no
// transport to secure. A test proving transport trust needs a real socket.
func ServeBufconn(tb testing.TB, server BufconnServer) *grpc.ClientConn {
	tb.Helper()

	listener := bufconn.Listen(bufconnBuffer)
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()

	connection, err := grpc.NewClient(
		"passthrough:///bufconn",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		// Same order as the cleanup below, because the serving goroutine is
		// already running and this path still owns joining it.
		if closeErr := server.Close(); closeErr != nil {
			tb.Errorf("Server.Close() after client setup failure = %v", closeErr)
		}
		<-serveDone
		_ = listener.Close()
		tb.Fatalf("grpc.NewClient() error = %v", err)
	}

	tb.Cleanup(func() {
		if err := connection.Close(); err != nil {
			tb.Errorf("ClientConn.Close() error = %v", err)
		}
		if err := server.Close(); err != nil {
			tb.Errorf("Server.Close() error = %v", err)
		}
		if err := <-serveDone; err != nil {
			tb.Errorf("Server.Serve() error = %v", err)
		}
		if err := listener.Close(); err != nil {
			tb.Errorf("bufconn.Listener.Close() error = %v", err)
		}
	})
	return connection
}
