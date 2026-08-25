package grpcx

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

// What a built [Server] does for the rest of its life: publish health, serve,
// drain, and stop. Health and lifecycle are one owner rather than two, because
// draining is a health transition before it is a stop — StartDrain publishes
// NOT_SERVING and only then does Shutdown wait on RPCs. Construction lives in
// options.go.

// Server adapts grpc.Server to the process runtime's bounded lifecycle.
//
// Every Server comes from [NewServer]; a nil pointer and the zero value are
// programming errors rather than a disabled server, and the methods below do
// not defend against them. A composition root that builds this transport
// conditionally holds the absence in its own variable — cmd/service keeps a nil
// grpcRuntimeServer interface and checks it once per use — because a method
// that quietly did nothing on a nil receiver would turn an unchecked [NewServer]
// error into a process that starts and never serves.
type Server struct {
	server *grpc.Server
	health *health.Server

	healthMu sync.Mutex
	draining bool

	healthDrain healthDrain
	drain       *rpcDrain
	stopOnce    sync.Once
	stopDone    chan struct{}
}

// SetServing publishes the same cached readiness state as the HTTP probe.
func (s *Server) SetServing(ready bool) {
	s.healthMu.Lock()
	defer s.healthMu.Unlock()
	if s.draining {
		return
	}
	status := healthgrpc.HealthCheckResponse_NOT_SERVING
	if ready {
		status = healthgrpc.HealthCheckResponse_SERVING
	}
	s.health.SetServingStatus("", status)
}

// StartDrain makes every registered health service NOT_SERVING and prevents a
// later startup result from making it serving again.
func (s *Server) StartDrain() {
	s.healthMu.Lock()
	defer s.healthMu.Unlock()
	if s.draining {
		return
	}
	s.draining = true
	s.health.Shutdown()
	s.healthDrain.stop()
}

// Serve runs the native server and normalizes the expected stop result.
func (s *Server) Serve(listener net.Listener) error {
	err := s.server.Serve(listener)
	if err == nil || errors.Is(err, grpc.ErrServerStopped) {
		return nil
	}
	return fmt.Errorf("serve native gRPC server: %w", err)
}

// Shutdown rejects new business RPCs and waits for pre-drain RPCs to publish
// their terminal status before GracefulStop. If ctx expires, Stop cancels every
// RPC context but does not wait for a handler that ignores that cancellation.
func (s *Server) Shutdown(ctx context.Context) error {
	s.StartDrain()
	drained := s.drain.start()
	select {
	case <-drained:
		s.gracefulStop()
		return nil
	default:
	}

	select {
	case <-drained:
		s.gracefulStop()
		return nil
	case <-s.stopDone:
		return nil
	case <-ctx.Done():
		s.forceStop()
		return fmt.Errorf("shutdown gRPC server: %w", ctx.Err())
	}
}

// Close immediately stops the transport and abandons active RPCs. Repeated calls
// are safe.
func (s *Server) Close() error {
	s.forceStop()
	return nil
}

func (s *Server) forceStop() {
	s.healthDrain.stop()
	s.stopOnce.Do(func() {
		s.server.Stop()
		close(s.stopDone)
	})
}

func (s *Server) gracefulStop() {
	s.stopOnce.Do(func() {
		s.server.GracefulStop()
		close(s.stopDone)
	})
}
