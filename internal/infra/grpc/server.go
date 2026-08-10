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

	// Health inputs, all guarded by healthMu and combined by publishHealthLocked.
	healthMu sync.Mutex
	draining bool
	admitted bool
	// profile:authn-oidc-jwt:start
	authnReady bool
	// profile:authn-oidc-jwt:end

	gracefulOnce sync.Once
	stopOnce     sync.Once
	gracefulDone chan struct{}
	forceStarted chan struct{}
}

// publishHealthLocked republishes standard health from the current inputs. It
// owns the whole rule so each caller below only records its own input:
//
//	SERVING iff startup admitted the process and authentication trust is
//	current. Drain is terminal and is published by StartDrain instead, so a
//	later input can never make a draining server serving again.
//
// Adding a readiness input means adding a field and one term here, not another
// copy of the composite in a third method.
func (s *Server) publishHealthLocked() {
	serving := s.admitted
	// profile:authn-oidc-jwt:start
	serving = serving && s.authnReady
	// profile:authn-oidc-jwt:end
	status := healthgrpc.HealthCheckResponse_NOT_SERVING
	if serving {
		status = healthgrpc.HealthCheckResponse_SERVING
	}
	s.health.SetServingStatus("", status)
}

// MarkServing publishes the shared startup admission result to gRPC health.
func (s *Server) MarkServing() {
	s.healthMu.Lock()
	defer s.healthMu.Unlock()
	if s.draining {
		return
	}
	s.admitted = true
	s.publishHealthLocked()
}

// profile:authn-oidc-jwt:start

// SetAuthnReady composes current authentication trust into standard health.
//
// Its proof lives in authn_health_test.go rather than server_test.go, because
// this method and that file are both removed when a service is generated with
// AUTHN=none.
func (s *Server) SetAuthnReady(ready bool) {
	s.healthMu.Lock()
	defer s.healthMu.Unlock()
	if s.draining {
		return
	}
	s.authnReady = ready
	s.publishHealthLocked()
}

// profile:authn-oidc-jwt:end

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
}

// Serve runs the native server and normalizes the expected stop result.
func (s *Server) Serve(listener net.Listener) error {
	err := s.server.Serve(listener)
	if err == nil || errors.Is(err, grpc.ErrServerStopped) {
		return nil
	}
	return fmt.Errorf("serve native gRPC server: %w", err)
}

// Shutdown waits for active RPCs until ctx expires, then forces every remaining
// transport stream to stop. grpc-go's stop calls may remain parked while a
// handler ignores its canceled RPC context, so this adapter deliberately does
// not join those process-lifetime goroutines past the caller's shutdown budget.
func (s *Server) Shutdown(ctx context.Context) error {
	s.gracefulOnce.Do(func() {
		go func() {
			s.server.GracefulStop()
			close(s.gracefulDone)
		}()
	})

	select {
	case <-s.gracefulDone:
		return nil
	case <-s.forceStarted:
		return nil
	case <-ctx.Done():
		s.forceStop()
		return fmt.Errorf("shutdown gRPC server: %w", ctx.Err())
	}
}

// Close immediately abandons active RPCs. Repeated calls are safe.
func (s *Server) Close() error {
	s.forceStop()
	return nil
}

func (s *Server) forceStop() {
	s.stopOnce.Do(func() {
		go s.server.Stop()
		close(s.forceStarted)
	})
}
