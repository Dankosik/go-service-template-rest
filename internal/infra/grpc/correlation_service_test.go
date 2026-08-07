// One request ID across a real client and server, on unary and streaming RPCs:
// is the accepted identifier the one that reaches the handler, the response
// metadata, the logs, and the spans?
//
// This file owns the single streaming service in the package, because it is the
// only test that needs one, and it consumes Serve's result itself while
// asserting the shutdown sequence.

package grpcx

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/grpc/grpctest"
	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// The streaming method this file drives. It stays here rather than in
// harness_test.go because no other file registers it, and at the head rather
// than the tail for the reason harness_test.go gives for its own constants.
const correlationStreamFullMethod = "/grpcx.test.CorrelationService/Wait"

func TestServiceToServiceGRPCCorrelationAndCancellation(t *testing.T) {
	const requestID = "request_123"

	spans := newCorrelationSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(spans),
	)
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := tracerProvider.Shutdown(shutdownCtx); err != nil {
			t.Errorf("TracerProvider.Shutdown() error = %v", err)
		}
	})

	unaryObserved := make(chan correlationObservation, 1)
	streamObserved := make(chan correlationObservation, 1)
	streamEntered := make(chan struct{})
	streamDone := make(chan struct{})
	register := func(registrar grpc.ServiceRegistrar) {
		grpctest.Register(registrar, grpctest.ServerStream(
			correlationStreamFullMethod,
			func(stream grpc.ServerStream) error {
				var request emptypb.Empty
				if err := stream.RecvMsg(&request); err != nil {
					return fmt.Errorf("receive correlation stream request: %w", err)
				}
				streamObserved <- observeCorrelation(stream.Context())
				if err := stream.SendHeader(metadata.MD{}); err != nil {
					return fmt.Errorf("send correlation stream headers: %w", err)
				}
				close(streamEntered)
				<-stream.Context().Done()
				close(streamDone)
				return stream.Context().Err()
			},
		))
	}
	cfg := testServerConfig()
	cfg.TelemetryHealthChecks = true
	server, err := NewServer(cfg, Options{
		TracerProvider: tracerProvider,
		Propagators:    propagation.TraceContext{},
		Services:       []RegisterService{register},
		UnaryPolicy: []grpc.UnaryServerInterceptor{
			func(
				ctx context.Context,
				request any,
				info *grpc.UnaryServerInfo,
				handler grpc.UnaryHandler,
			) (any, error) {
				if info.FullMethod == healthgrpc.Health_Check_FullMethodName {
					unaryObserved <- observeCorrelation(ctx)
				}
				return handler(ctx, request)
			},
		},
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	server.MarkServing()

	connection, serveDone := serveOverTCP(t, server, grpcclient.Options{
		TracerProvider: tracerProvider,
		Propagation:    grpcclient.PropagationTrustedService,
	})

	// This test asserts the shutdown sequence itself at the end, consuming
	// serveDone and closing the connection there. Cleanup therefore only covers
	// the paths that return before reaching it, which is why it is guarded rather
	// than unconditional.
	serverStopped := false
	connectionClosed := false
	t.Cleanup(func() {
		if !connectionClosed {
			if err := connection.Close(); err != nil {
				t.Errorf("ClientConn.Close() error = %v", err)
			}
		}
		if !serverStopped {
			if err := server.Close(); err != nil {
				t.Errorf("Server.Close() error = %v", err)
			}
			select {
			case err := <-serveDone:
				if err != nil {
					t.Errorf("Server.Serve() error = %v", err)
				}
			case <-time.After(time.Second):
				t.Error("Server.Serve() did not return after Close")
			}
		}
	})

	callCtx, parent := tracerProvider.Tracer("grpcx-correlation-test").Start(
		reqctx.ContextWithRequestID(t.Context(), requestID),
		"parent",
	)
	parentSpanContext := parent.SpanContext()
	defer parent.End()

	var unaryHeader metadata.MD
	response, err := healthgrpc.NewHealthClient(connection).Check(
		callCtx,
		&healthgrpc.HealthCheckRequest{},
		grpc.Header(&unaryHeader),
	)
	if err != nil {
		t.Fatalf("Health.Check() error = %v", err)
	}
	if response.GetStatus() != healthgrpc.HealthCheckResponse_SERVING {
		t.Fatalf("Health.Check() status = %s, want SERVING", response.GetStatus())
	}
	assertResponseRequestID(t, unaryHeader, requestID)
	assertCorrelationObservation(t, <-unaryObserved, parentSpanContext.TraceID(), requestID)

	streamCtx, cancelStream := context.WithCancel(callCtx)
	stream, err := connection.NewStream(
		streamCtx,
		&grpc.StreamDesc{ServerStreams: true},
		correlationStreamFullMethod,
	)
	if err != nil {
		cancelStream()
		t.Fatalf("ClientConn.NewStream() error = %v", err)
	}
	if err := stream.SendMsg(&emptypb.Empty{}); err != nil {
		cancelStream()
		t.Fatalf("ClientStream.SendMsg() error = %v", err)
	}
	if err := stream.CloseSend(); err != nil {
		cancelStream()
		t.Fatalf("ClientStream.CloseSend() error = %v", err)
	}

	select {
	case <-streamEntered:
	case <-time.After(time.Second):
		cancelStream()
		t.Fatal("stream handler did not reach the cancellation boundary")
	}
	streamHeader, err := stream.Header()
	if err != nil {
		cancelStream()
		t.Fatalf("ClientStream.Header() error = %v", err)
	}
	assertResponseRequestID(t, streamHeader, requestID)
	assertCorrelationObservation(t, <-streamObserved, parentSpanContext.TraceID(), requestID)

	cancelStream()
	if err := stream.RecvMsg(&emptypb.Empty{}); status.Code(err) != codes.Canceled {
		t.Fatalf("ClientStream.RecvMsg() error = %v, want canonical Canceled", err)
	}
	select {
	case <-streamDone:
	case <-time.After(time.Second):
		t.Fatal("stream handler did not finish after caller cancellation")
	}

	spans.waitForEnded(t, 4)
	ended := spans.Ended()
	assertConcreteRPCParentage(
		t,
		ended,
		parentSpanContext,
		healthgrpc.Health_Check_FullMethodName[1:],
	)
	assertConcreteRPCParentage(
		t,
		ended,
		parentSpanContext,
		correlationStreamFullMethod[1:],
	)

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Server.Shutdown() error = %v", err)
	}
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("Server.Serve() error = %v", err)
		}
		serverStopped = true
	case <-shutdownCtx.Done():
		t.Fatalf("Server.Serve() did not return after Shutdown: %v", shutdownCtx.Err())
	}
	if err := connection.Close(); err != nil {
		t.Fatalf("ClientConn.Close() error = %v", err)
	}
	connectionClosed = true
}

type correlationObservation struct {
	requestID string
	traceID   trace.TraceID
}

func observeCorrelation(ctx context.Context) correlationObservation {
	return correlationObservation{
		requestID: reqctx.RequestID(ctx),
		traceID:   trace.SpanContextFromContext(ctx).TraceID(),
	}
}

func assertCorrelationObservation(
	t *testing.T,
	observed correlationObservation,
	wantTraceID trace.TraceID,
	wantRequestID string,
) {
	t.Helper()
	if observed.traceID != wantTraceID {
		t.Fatalf("server trace ID = %s, want %s", observed.traceID, wantTraceID)
	}
	if observed.requestID != wantRequestID {
		t.Fatalf("server request ID = %q, want %q", observed.requestID, wantRequestID)
	}
}

func assertResponseRequestID(t *testing.T, header metadata.MD, want string) {
	t.Helper()
	if got := header.Get(requestIDMetadataKey); len(got) != 1 || got[0] != want {
		t.Fatalf("response request IDs = %v, want [%s]", got, want)
	}
}

func assertConcreteRPCParentage(
	t *testing.T,
	spans []sdktrace.ReadOnlySpan,
	parent trace.SpanContext,
	method string,
) {
	t.Helper()

	var clientSpan sdktrace.ReadOnlySpan
	var serverSpan sdktrace.ReadOnlySpan
	for _, span := range spans {
		if span.Name() != method {
			continue
		}
		switch span.SpanKind() {
		case trace.SpanKindClient:
			clientSpan = span
		case trace.SpanKindServer:
			serverSpan = span
		case trace.SpanKindUnspecified,
			trace.SpanKindInternal,
			trace.SpanKindProducer,
			trace.SpanKindConsumer:
			continue
		}
	}
	if clientSpan == nil || serverSpan == nil {
		t.Fatalf(
			"spans for %s: client=%t server=%t; ended spans = %v",
			method,
			clientSpan != nil,
			serverSpan != nil,
			spanNames(spans),
		)
	}
	if clientSpan.Parent().SpanID() != parent.SpanID() {
		t.Fatalf(
			"%s client parent span ID = %s, want root %s",
			method,
			clientSpan.Parent().SpanID(),
			parent.SpanID(),
		)
	}
	if serverSpan.Parent().SpanID() != clientSpan.SpanContext().SpanID() {
		t.Fatalf(
			"%s server parent span ID = %s, want concrete client span %s",
			method,
			serverSpan.Parent().SpanID(),
			clientSpan.SpanContext().SpanID(),
		)
	}
	for _, span := range []sdktrace.ReadOnlySpan{clientSpan, serverSpan} {
		if span.SpanContext().TraceID() != parent.TraceID() {
			t.Fatalf(
				"%s %s trace ID = %s, want %s",
				span.SpanKind(),
				method,
				span.SpanContext().TraceID(),
				parent.TraceID(),
			)
		}
	}
}

type correlationSpanRecorder struct {
	*tracetest.SpanRecorder

	ended chan struct{}
}

var _ sdktrace.SpanProcessor = (*correlationSpanRecorder)(nil)

func newCorrelationSpanRecorder() *correlationSpanRecorder {
	return &correlationSpanRecorder{
		SpanRecorder: tracetest.NewSpanRecorder(),
		ended:        make(chan struct{}, 16),
	}
}

func (r *correlationSpanRecorder) OnEnd(span sdktrace.ReadOnlySpan) {
	r.SpanRecorder.OnEnd(span)
	r.ended <- struct{}{}
}

func (r *correlationSpanRecorder) waitForEnded(t *testing.T, count int) {
	t.Helper()
	for range count {
		select {
		case <-r.ended:
		case <-time.After(time.Second):
			t.Fatalf(
				"ended spans = %d, want at least %d; spans = %v",
				len(r.Ended()),
				count,
				spanNames(r.Ended()),
			)
		}
	}
}
