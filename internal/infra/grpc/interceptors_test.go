package grpcx

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/failure"
	"github.com/example/go-service-template-rest/internal/waittest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/emptypb"
)

// The validated method only this file drives: its request is the dynamic
// message from validationMessageType, which grpctest's typed descriptor cannot
// decode into.
const (
	validatedServiceName     = "grpcx.validation.test.Service"
	validatedUnaryFullMethod = "/" + validatedServiceName + "/Call"
)

// A caller that sends no deadline still hands a supplied policy a bounded
// context: the server's unary budget is installed before the policy runs.
func TestUnaryPolicySeesTheServerDeadline(t *testing.T) {
	cfg := testServerConfig()
	cfg.unaryTimeout = time.Hour

	remaining := make(chan time.Duration, 1)
	policy := func(
		ctx context.Context,
		request any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// No deadline reads as the zero time, which is far in the past.
		deadline, _ := ctx.Deadline()
		remaining <- time.Until(deadline)
		return handler(ctx, request)
	}
	register := func(registrar grpc.ServiceRegistrar) {
		registerUnaryTestService(registrar, testUnaryFullMethod,
			func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
				return &emptypb.Empty{}, nil
			})
	}
	_, connection := startTestServerWithOptions(t, cfg, register, Options{
		UnaryPolicy: []grpc.UnaryServerInterceptor{policy},
	})

	if err := connection.Invoke(t.Context(), testUnaryFullMethod, &emptypb.Empty{}, &emptypb.Empty{}); err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	got := waittest.Receive(t, remaining, 5*time.Second, "policy deadline observation")
	if got <= 0 || got > time.Hour {
		t.Fatalf("policy deadline remaining = %s, want within the 1h server budget", got)
	}
}

// Request validation runs after a supplied policy and before the handler: the
// policy still sees an invalid request, and the caller gets the validation
// status without the handler running.
func TestUnaryPolicyRunsBeforeValidation(t *testing.T) {
	messageType := validationMessageType(t)
	invalid := messageType.New().Interface()
	invalid.ProtoReflect().Set(
		invalid.ProtoReflect().Descriptor().Fields().ByName("value"),
		protoreflect.ValueOfString(rejectedValidationValue),
	)

	var policyCalled, handlerCalled atomic.Bool
	policy := func(
		ctx context.Context,
		request any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		policyCalled.Store(true)
		return handler(ctx, request)
	}
	register := func(registrar grpc.ServiceRegistrar) {
		registerValidatedUnaryService(registrar, messageType, func() {
			handlerCalled.Store(true)
		})
	}
	_, connection := startTestServerWithOptions(t, testServerConfig(), register, Options{
		UnaryPolicy: []grpc.UnaryServerInterceptor{policy},
	})

	err := connection.Invoke(t.Context(), validatedUnaryFullMethod, invalid, &emptypb.Empty{})
	assertValidationStatus(t, err)
	if !policyCalled.Load() || handlerCalled.Load() {
		t.Fatalf("policy/handler calls = %t/%t, want true/false", policyCalled.Load(), handlerCalled.Load())
	}
}

func TestPanickingUnaryPolicyIsSanitized(t *testing.T) {
	panicking := func(context.Context, any, *grpc.UnaryServerInfo, grpc.UnaryHandler) (any, error) {
		panic("policy panic")
	}
	register := func(registrar grpc.ServiceRegistrar) {
		registerUnaryTestService(registrar, testUnaryFullMethod,
			func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
				return &emptypb.Empty{}, nil
			})
	}
	_, connection := startTestServerWithOptions(t, testServerConfig(), register, Options{
		UnaryPolicy: []grpc.UnaryServerInterceptor{panicking},
	})

	err := connection.Invoke(t.Context(), testUnaryFullMethod, &emptypb.Empty{}, &emptypb.Empty{})
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

// registerValidatedUnaryService registers validatedUnaryFullMethod with a
// request of messageType, so the server's real validator sees the field rule.
// called runs when the handler is entered.
func registerValidatedUnaryService(
	registrar grpc.ServiceRegistrar,
	messageType protoreflect.MessageType,
	called func(),
) {
	handler := func(context.Context, any) (any, error) {
		called()
		return &emptypb.Empty{}, nil
	}
	registrar.RegisterService(&grpc.ServiceDesc{
		ServiceName: validatedServiceName,
		Methods: []grpc.MethodDesc{{
			MethodName: "Call",
			Handler: func(
				_ any,
				ctx context.Context,
				decode func(any) error,
				interceptor grpc.UnaryServerInterceptor,
			) (any, error) {
				request := messageType.New().Interface()
				if err := decode(request); err != nil {
					return nil, err
				}
				return interceptor(ctx, request, &grpc.UnaryServerInfo{FullMethod: validatedUnaryFullMethod}, handler)
			},
		}},
	}, nil)
}
