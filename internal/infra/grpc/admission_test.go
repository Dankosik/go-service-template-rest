// The admission budgets through a server built by NewServer: does filling one
// from one RPC kind shed what the other owns?
//
// NewServer builds the unary and stream policy lists separately and sizes health
// from a separate budget. Only a real server can show that both RPC kinds share
// one business limit and health remains independently callable.

package grpcx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/example/go-service-template-rest/internal/waittest"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

const admissionStreamFullMethod = "/grpcx.test.AdmissionService/Hold"

type failingMeterProvider struct{ metric.MeterProvider }

func (failingMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter { //nolint:ireturn // Test provider seam.
	return failingMeter{}
}

type failingMeter struct{ metric.Meter }

func (failingMeter) Int64UpDownCounter(string, ...metric.Int64UpDownCounterOption) (metric.Int64UpDownCounter, error) { //nolint:ireturn // Test failure seam.
	return nil, errors.New("instrument failed")
}

func (failingMeter) Int64Counter(string, ...metric.Int64CounterOption) (metric.Int64Counter, error) { //nolint:ireturn // Test failure seam.
	return nil, errors.New("instrument failed")
}

// An operator reads saturation from the admission instruments: a held business
// RPC is active load, and each refusal from either budget is counted where the
// operator looks for it.
func TestServerPublishesAdmissionMetrics(t *testing.T) {
	cfg := testServerConfig()
	cfg.maxConcurrentRPCs = 1
	cfg.maxConcurrentHealthRPCs = 1

	occupied := make(chan struct{})
	release := make(chan struct{})
	register := func(registrar grpc.ServiceRegistrar) {
		registerStreamTestService(registrar, admissionStreamFullMethod,
			func(stream grpc.ServerStream) error {
				var request emptypb.Empty
				if err := stream.RecvMsg(&request); err != nil {
					return fmt.Errorf("receive admission stream request: %w", err)
				}
				close(occupied)
				<-release
				return nil
			})
		registerUnaryTestService(registrar, testUnaryFullMethod,
			func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
				return &emptypb.Empty{}, nil
			})
	}
	reader, provider := telemetrytest.NewManualMeterProvider(t)
	_, connection := startTestServerWithOptions(t, cfg, register, Options{MeterProvider: provider})
	releaseHandler := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseHandler)

	stream, err := connection.NewStream(
		t.Context(),
		&grpc.StreamDesc{ServerStreams: true},
		admissionStreamFullMethod,
	)
	if err != nil {
		t.Fatalf("ClientConn.NewStream() error = %v", err)
	}
	if err := stream.SendMsg(&emptypb.Empty{}); err != nil {
		t.Fatalf("ClientStream.SendMsg() error = %v", err)
	}
	waittest.ReceiveSignal(t, occupied, 5*time.Second, "streaming handler admission slot")
	if got := telemetrytest.Int64SumValue(t.Context(), t, reader, "rpc.server.active_requests"); got != 1 {
		t.Fatalf("rpc.server.active_requests while held = %d, want 1", got)
	}

	err = connection.Invoke(t.Context(), testUnaryFullMethod, &emptypb.Empty{}, &emptypb.Empty{})
	assertStatusCode(t, err, codes.ResourceExhausted)

	// One standing watch fills the health budget; the second is refused.
	watchCtx, cancelWatch := context.WithCancel(t.Context())
	t.Cleanup(cancelWatch)
	health := healthgrpc.NewHealthClient(connection)
	held, err := health.Watch(watchCtx, &healthgrpc.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("Health.Watch() error = %v", err)
	}
	if _, err := held.Recv(); err != nil {
		t.Fatalf("Health.Watch() first status = %v", err)
	}
	refused, err := health.Watch(watchCtx, &healthgrpc.HealthCheckRequest{})
	if err == nil {
		_, err = refused.Recv()
	}
	assertStatusCode(t, err, codes.ResourceExhausted)

	releaseHandler()
	if err := stream.RecvMsg(&emptypb.Empty{}); !errors.Is(err, io.EOF) {
		t.Fatalf("held stream end = %v, want io.EOF", err)
	}
	if got := telemetrytest.Int64SumValue(t.Context(), t, reader, "rpc.server.active_requests"); got != 0 {
		t.Fatalf("rpc.server.active_requests after release = %d, want 0", got)
	}
	if got := telemetrytest.Int64SumValue(t.Context(), t, reader, "rpc.server.shed_requests"); got != 1 {
		t.Fatalf("rpc.server.shed_requests = %d, want 1", got)
	}
	if got := telemetrytest.Int64SumValue(t.Context(), t, reader, "rpc.server.health.shed_requests"); got != 1 {
		t.Fatalf("rpc.server.health.shed_requests = %d, want 1", got)
	}
}

func TestAdmissionPolicyReportsInstrumentFailures(t *testing.T) {
	previous := otel.GetErrorHandler()
	t.Cleanup(func() { otel.SetErrorHandler(previous) })
	var reported atomic.Int32
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(error) { reported.Add(1) }))

	policy := newAdmissionPolicy(1, 0, newServerLoad(failingMeterProvider{}), newRPCDrain())
	if got := reported.Load(); got != 3 {
		t.Fatalf("reported instrument failures = %d, want 3", got)
	}
	// Missing instruments leave admission and refusal working.
	noop := func(context.Context) error { return nil }
	if err := policy.business.around(t.Context(), noop); err != nil {
		t.Fatalf("business around without instruments: %v", err)
	}
	assertStatusCode(t, policy.health.around(t.Context(), noop), codes.ResourceExhausted)
}

func TestAdmissionBudgetIsProcessWide(t *testing.T) {
	cfg := testServerConfig()
	// One slot for the whole process, so a second budget is the only way both
	// RPCs below can be admitted.
	cfg.maxConcurrentRPCs = 1

	occupied := make(chan struct{})
	release := make(chan struct{})
	register := func(registrar grpc.ServiceRegistrar) {
		registerStreamTestService(registrar, admissionStreamFullMethod,
			func(stream grpc.ServerStream) error {
				var request emptypb.Empty
				if err := stream.RecvMsg(&request); err != nil {
					return fmt.Errorf("receive admission stream request: %w", err)
				}
				close(occupied)
				<-release
				return nil
			})
		registerUnaryTestService(registrar, testUnaryFullMethod,
			func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
				return &emptypb.Empty{}, nil
			})
	}
	_, connection := startTestServer(t, cfg, register)
	// Registered after the harness's teardown so it runs before it: the handler
	// must leave its slot before the server is closed, or the test leaks the
	// goroutine it is holding.
	t.Cleanup(func() { close(release) })

	stream, err := connection.NewStream(
		t.Context(),
		&grpc.StreamDesc{ServerStreams: true},
		admissionStreamFullMethod,
	)
	if err != nil {
		t.Fatalf("ClientConn.NewStream() error = %v", err)
	}
	if err := stream.SendMsg(&emptypb.Empty{}); err != nil {
		t.Fatalf("ClientStream.SendMsg() error = %v", err)
	}
	waittest.ReceiveSignal(t, occupied, 5*time.Second, "streaming handler admission slot")

	// The unary chain answers from the same budget, or it has one of its own.
	err = connection.Invoke(t.Context(), testUnaryFullMethod, &emptypb.Empty{}, &emptypb.Empty{})
	assertStatusCode(t, err, codes.ResourceExhausted)
}

// TestHealthServiceSurvivesBusinessSaturation drives the consequence the separate
// health budget exists for.
//
// A grpc-go health-aware client holds one Health/Watch open per subchannel for
// the connection's whole life. Shedding that watch does not cost the caller one
// RPC: its balancer stops selecting the backend entirely, so business saturation
// on one replica would remove it from every health-checking caller at once. Check
// answering is the second half — a platform that cannot probe a saturated
// instance restarts a healthy one.
func TestHealthServiceSurvivesBusinessSaturation(t *testing.T) {
	cfg := testServerConfig()
	// One business slot, held for the whole test by the stream below.
	cfg.maxConcurrentRPCs = 1

	occupied := make(chan struct{})
	release := make(chan struct{})
	register := func(registrar grpc.ServiceRegistrar) {
		registerStreamTestService(registrar, admissionStreamFullMethod,
			func(stream grpc.ServerStream) error {
				var request emptypb.Empty
				if err := stream.RecvMsg(&request); err != nil {
					return fmt.Errorf("receive admission stream request: %w", err)
				}
				close(occupied)
				<-release
				return nil
			})
	}
	_, connection := startTestServer(t, cfg, register)
	t.Cleanup(func() { close(release) })

	stream, err := connection.NewStream(
		t.Context(),
		&grpc.StreamDesc{ServerStreams: true},
		admissionStreamFullMethod,
	)
	if err != nil {
		t.Fatalf("ClientConn.NewStream() error = %v", err)
	}
	if err := stream.SendMsg(&emptypb.Empty{}); err != nil {
		t.Fatalf("ClientStream.SendMsg() error = %v", err)
	}
	waittest.ReceiveSignal(t, occupied, 5*time.Second, "streaming handler admission slot")

	health := healthgrpc.NewHealthClient(connection)
	if _, err := health.Check(t.Context(), &healthgrpc.HealthCheckRequest{}); err != nil {
		t.Fatalf("Health.Check() against a full business budget = %v", err)
	}

	// Canceled with the test, which is what ends the watch this opens.
	watchCtx, cancelWatch := context.WithCancel(t.Context())
	t.Cleanup(cancelWatch)
	watch, err := health.Watch(watchCtx, &healthgrpc.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("Health.Watch() against a full business budget = %v", err)
	}
	// A shed watch fails here rather than at the call above: grpc-go returns the
	// stream before the server has answered.
	if _, err := watch.Recv(); err != nil {
		t.Fatalf("Health.Watch() first status against a full business budget = %v", err)
	}
}
