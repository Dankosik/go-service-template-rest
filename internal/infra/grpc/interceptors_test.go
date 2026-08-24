package grpcx

import (
	"context"
	"log/slog"
	"slices"
	"testing"
	"time"

	"buf.build/go/protovalidate"
	"github.com/example/go-service-template-rest/internal/failure"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestUnaryInterceptorOrder(t *testing.T) {
	validator, err := protovalidate.New()
	if err != nil {
		t.Fatalf("protovalidate.New() error = %v", err)
	}
	messageType := validationMessageType(t)
	invalid := messageType.New().Interface()
	invalid.ProtoReflect().Set(
		invalid.ProtoReflect().Descriptor().Fields().ByName("value"),
		protoreflect.ValueOfString("invalid"),
	)

	policyCalled := false
	handlerCalled := false
	policy := func(
		ctx context.Context,
		request any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		policyCalled = true
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("supplied policy ran before the unary deadline")
		}
		return handler(ctx, request)
	}
	chain := unaryChain(
		slog.New(slog.DiscardHandler),
		newAdmissionPolicy(1, 1, serverLoad{}),
		time.Minute,
		[]grpc.UnaryServerInterceptor{policy},
		validator,
		handlerErrorBoundary(slog.New(slog.DiscardHandler), nil),
	)
	_, err = invokeUnaryInterceptors(t.Context(), invalid, chain, func(context.Context, any) (any, error) {
		handlerCalled = true
		return &emptypb.Empty{}, nil
	})
	assertValidationStatus(t, err)
	if !policyCalled || handlerCalled {
		t.Fatalf("policy/handler calls = %t/%t, want true/false", policyCalled, handlerCalled)
	}

	panickingPolicy := func(context.Context, any, *grpc.UnaryServerInfo, grpc.UnaryHandler) (any, error) {
		panic("policy panic")
	}
	chain = unaryChain(
		slog.New(slog.DiscardHandler),
		newAdmissionPolicy(1, 1, serverLoad{}),
		time.Minute,
		[]grpc.UnaryServerInterceptor{panickingPolicy},
		validator,
		handlerErrorBoundary(slog.New(slog.DiscardHandler), nil),
	)
	_, err = invokeUnaryInterceptors(t.Context(), &emptypb.Empty{}, chain, func(context.Context, any) (any, error) {
		return &emptypb.Empty{}, nil
	})
	assertSanitizedInternal(t, err)
}

func TestServerRecoversStreamingHandlerPanic(t *testing.T) {
	register := func(registrar grpc.ServiceRegistrar) {
		registerStreamTestService(registrar, testStreamFullMethod, func(grpc.ServerStream) error {
			panic("stream handler panic")
		})
	}
	_, connection := startTestServer(t, testServerConfig(), register)
	stream, err := connection.NewStream(
		t.Context(),
		&grpc.StreamDesc{ServerStreams: true},
		testStreamFullMethod,
	)
	if err != nil {
		t.Fatalf("ClientConn.NewStream() error = %v", err)
	}
	if err := stream.SendMsg(&emptypb.Empty{}); err != nil {
		assertSanitizedInternal(t, err)
		return
	}
	if err := stream.CloseSend(); err != nil {
		t.Fatalf("ClientStream.CloseSend() error = %v", err)
	}
	err = stream.RecvMsg(&emptypb.Empty{})
	assertSanitizedInternal(t, err)
}

func assertSanitizedInternal(t *testing.T, err error) {
	t.Helper()
	assertStatusCode(t, err, codes.Internal)
	if got := status.Convert(err).Message(); got != failure.SanitizedDetail {
		t.Fatalf("status detail = %q, want %q", got, failure.SanitizedDetail)
	}
}

func invokeUnaryInterceptors(
	ctx context.Context,
	request any,
	interceptors []grpc.UnaryServerInterceptor,
	handler grpc.UnaryHandler,
) (any, error) {
	info := &grpc.UnaryServerInfo{FullMethod: testUnaryFullMethod}
	for _, interceptor := range slices.Backward(interceptors) {
		next := handler
		handler = func(ctx context.Context, request any) (any, error) {
			return interceptor(ctx, request, info, next)
		}
	}
	return handler(ctx, request)
}
