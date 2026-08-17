package grpcclient_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/grpc/grpctest"
	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"github.com/example/go-service-template-rest/internal/waittest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/resolver/manual"
	"google.golang.org/protobuf/types/known/emptypb"
)

const healthProbeMethod = "/grpcclient.healthtest.Backend/Probe"

func TestRoundRobinHealthControlsEligibility(t *testing.T) {
	first := startHealthBackend(t, healthgrpc.HealthCheckResponse_SERVING, nil)
	second := startHealthBackend(t, healthgrpc.HealthCheckResponse_SERVING, nil)
	connection := newHealthConnection(t, "grpcclient-health-transition", first, second)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	for first.calls.Load() == 0 || second.calls.Load() == 0 {
		if err := invokeHealthProbe(ctx, connection); err != nil {
			t.Fatalf("initial probe error = %v", err)
		}
	}

	first.health.SetServingStatus("", healthgrpc.HealthCheckResponse_NOT_SERVING)
	requireConsecutiveBackendCalls(ctx, t, connection, second, first, 8)
	first.calls.Store(0)
	second.calls.Store(0)
	requireExactBackendCalls(ctx, t, connection, second, first, 8)

	first.health.SetServingStatus("", healthgrpc.HealthCheckResponse_SERVING)
	requireBackendReached(ctx, t, connection, first)
}

func TestRoundRobinHealthTransitionDoesNotCancelInflightRPC(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var blocked atomic.Bool
	first := startHealthBackend(t, healthgrpc.HealthCheckResponse_SERVING, func(ctx context.Context) error {
		if !blocked.CompareAndSwap(false, true) {
			return nil
		}
		close(entered)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	second := startHealthBackend(t, healthgrpc.HealthCheckResponse_SERVING, nil)
	connection := newHealthConnection(t, "grpcclient-health-inflight", first, second)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	inflight := startUntilBackendEntered(ctx, t, connection, entered)

	first.health.SetServingStatus("", healthgrpc.HealthCheckResponse_NOT_SERVING)
	requireConsecutiveBackendCalls(ctx, t, connection, second, first, 8)
	select {
	case err := <-inflight:
		t.Fatalf("in-flight RPC ended after health transition: %v", err)
	default:
	}

	close(release)
	if err := waittest.Receive(t, inflight, 2*time.Second, "released in-flight RPC"); err != nil {
		t.Fatalf("released in-flight RPC error = %v", err)
	}

	first.calls.Store(0)
	second.calls.Store(0)
	requireExactBackendCalls(ctx, t, connection, second, first, 8)
}

func TestHealthUnimplementedFallsBackToConnectivity(t *testing.T) {
	backend := startBackend(t, nil, nil)
	connection := newHealthConnection(t, "grpcclient-health-unimplemented", backend)

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	if err := invokeHealthProbe(ctx, connection); err != nil {
		t.Fatalf("application RPC error = %v", err)
	}
	if got := backend.calls.Load(); got != 1 {
		t.Fatalf("application handler calls = %d, want 1", got)
	}
}

func TestClientConnCloseCancelsHealthWatch(t *testing.T) {
	observer := newHealthWatchObserver()
	address := serveTestServer(t, listenLoopback(t), func(server *grpc.Server) {
		healthgrpc.RegisterHealthServer(server, observer)
	})
	backend := &healthBackend{address: address}
	connection := newHealthConnection(t, "grpcclient-health-close", backend)
	connection.Connect()

	waittest.ReceiveSignal(t, observer.started, 2*time.Second, "standard health Watch start")
	if err := connection.Close(); err != nil {
		t.Fatalf("ClientConn.Close() error = %v", err)
	}
	waittest.ReceiveSignal(t, observer.stopped, 2*time.Second, "standard health Watch stop after ClientConn.Close")
}

type healthBackend struct {
	address string
	health  *health.Server
	calls   atomic.Int64
}

func startHealthBackend(
	t *testing.T,
	status healthgrpc.HealthCheckResponse_ServingStatus,
	call func(context.Context) error,
) *healthBackend {
	t.Helper()

	healthServer := health.NewServer()
	healthServer.SetServingStatus("", status)
	return startBackend(t, healthServer, call)
}

func startBackend(
	t *testing.T,
	healthServer *health.Server,
	call func(context.Context) error,
) *healthBackend {
	t.Helper()

	backend := &healthBackend{health: healthServer}
	backend.address = serveTestServer(t, listenLoopback(t), func(server *grpc.Server) {
		grpctest.Register(server, grpctest.Unary(
			healthProbeMethod,
			func(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
				backend.calls.Add(1)
				if call != nil {
					if err := call(ctx); err != nil {
						return nil, err
					}
				}
				return &emptypb.Empty{}, nil
			},
		))
		if healthServer != nil {
			healthgrpc.RegisterHealthServer(server, healthServer)
		}
	})
	return backend
}

func newHealthConnection(t *testing.T, scheme string, backends ...*healthBackend) *grpc.ClientConn {
	t.Helper()

	addresses := make([]resolver.Address, 0, len(backends))
	for _, backend := range backends {
		addresses = append(addresses, resolver.Address{Addr: backend.address})
	}
	builder := manual.NewBuilderWithScheme(scheme)
	resolver.Register(builder)
	builder.InitialState(resolver.State{Addresses: addresses})

	connection, err := grpcclient.New(
		grpcclient.DefaultConfig(scheme+":///backends"),
		grpcclient.Options{TransportCredentials: insecure.NewCredentials()},
	)
	if err != nil {
		t.Fatalf("grpcclient.New() error = %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	return connection
}

func invokeHealthProbe(ctx context.Context, connection *grpc.ClientConn) error {
	//nolint:wrapcheck // The test helpers inspect grpc-go's exact call result.
	return connection.Invoke(ctx, healthProbeMethod, &emptypb.Empty{}, &emptypb.Empty{})
}

func requireConsecutiveBackendCalls(
	ctx context.Context,
	t *testing.T,
	connection *grpc.ClientConn,
	want, other *healthBackend,
	count int,
) {
	t.Helper()

	consecutive := 0
	for consecutive < count {
		wantBefore, otherBefore := want.calls.Load(), other.calls.Load()
		if err := invokeHealthProbe(ctx, connection); err != nil {
			if ctx.Err() != nil {
				t.Fatalf("health convergence: %v", err)
			}
			consecutive = 0
			continue
		}
		if want.calls.Load() == wantBefore+1 && other.calls.Load() == otherBefore {
			consecutive++
		} else {
			consecutive = 0
		}
	}
}

func requireExactBackendCalls(
	ctx context.Context,
	t *testing.T,
	connection *grpc.ClientConn,
	want, other *healthBackend,
	count int,
) {
	t.Helper()

	for range count {
		if err := invokeHealthProbe(ctx, connection); err != nil {
			t.Fatalf("validation probe error = %v", err)
		}
	}
	if got := want.calls.Load(); got != int64(count) {
		t.Fatalf("eligible backend calls = %d, want %d", got, count)
	}
	if got := other.calls.Load(); got != 0 {
		t.Fatalf("unhealthy backend calls = %d, want 0", got)
	}
}

func requireBackendReached(
	ctx context.Context,
	t *testing.T,
	connection *grpc.ClientConn,
	backend *healthBackend,
) {
	t.Helper()

	before := backend.calls.Load()
	for backend.calls.Load() == before {
		if err := invokeHealthProbe(ctx, connection); err != nil && ctx.Err() != nil {
			t.Fatalf("restored backend was not reached: %v", err)
		}
	}
}

func startUntilBackendEntered(
	ctx context.Context,
	t *testing.T,
	connection *grpc.ClientConn,
	entered <-chan struct{},
) <-chan error {
	t.Helper()

	for {
		result := make(chan error, 1)
		go func() { result <- invokeHealthProbe(ctx, connection) }()
		select {
		case <-entered:
			return result
		case err := <-result:
			if err != nil {
				t.Fatalf("probe before in-flight entry error = %v", err)
			}
		case <-ctx.Done():
			t.Fatalf("blocking backend was not selected: %v", ctx.Err())
		}
	}
}

type healthWatchObserver struct {
	healthgrpc.UnimplementedHealthServer

	server      *health.Server
	started     chan struct{}
	stopped     chan struct{}
	startedOnce sync.Once
	stoppedOnce sync.Once
}

func newHealthWatchObserver() *healthWatchObserver {
	return &healthWatchObserver{
		server:  health.NewServer(),
		started: make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

func (s *healthWatchObserver) Watch(
	request *healthgrpc.HealthCheckRequest,
	stream healthgrpc.Health_WatchServer,
) error {
	s.startedOnce.Do(func() { close(s.started) })
	err := s.server.Watch(request, stream)
	s.stoppedOnce.Do(func() { close(s.stopped) })
	//nolint:wrapcheck // The test server preserves grpc-go's health result.
	return err
}
