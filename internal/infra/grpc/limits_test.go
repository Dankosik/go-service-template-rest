package grpcx

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestServerEnforcesReceiveAndSendMessageLimits(t *testing.T) {
	t.Run("receive", func(t *testing.T) {
		cfg := testServerConfig()
		cfg.MaxReceiveMessageBytes = 64
		handlerCalled := false
		register := func(registrar grpc.ServiceRegistrar) {
			registerPayloadService(registrar, payloadService{
				call: func(
					context.Context,
					*wrapperspb.BytesValue,
				) (*wrapperspb.BytesValue, error) {
					handlerCalled = true
					return &wrapperspb.BytesValue{}, nil
				},
			})
		}
		_, connection := startTestServer(t, cfg, register)

		err := connection.Invoke(
			t.Context(),
			testPayloadFullMethod,
			&wrapperspb.BytesValue{Value: make([]byte, 1024)},
			&wrapperspb.BytesValue{},
		)
		assertStatusCode(t, err, codes.ResourceExhausted)
		if handlerCalled {
			t.Fatal("oversized received message entered the handler")
		}
	})

	t.Run("send", func(t *testing.T) {
		cfg := testServerConfig()
		cfg.MaxSendMessageBytes = 64
		handlerCalled := false
		register := func(registrar grpc.ServiceRegistrar) {
			registerPayloadService(registrar, payloadService{
				call: func(
					context.Context,
					*wrapperspb.BytesValue,
				) (*wrapperspb.BytesValue, error) {
					handlerCalled = true
					return &wrapperspb.BytesValue{Value: make([]byte, 1024)}, nil
				},
			})
		}
		_, connection := startTestServer(t, cfg, register)

		err := connection.Invoke(
			t.Context(),
			testPayloadFullMethod,
			&wrapperspb.BytesValue{},
			&wrapperspb.BytesValue{},
		)
		assertStatusCode(t, err, codes.ResourceExhausted)
		if !handlerCalled {
			t.Fatal("send-limit test did not enter the handler")
		}
	})
}

func TestServerRejectsOversizedIncomingMetadataBeforeHandler(t *testing.T) {
	cfg := testServerConfig()
	cfg.MaxHeaderListBytes = 512
	var handlerCalls atomic.Int64
	register := func(registrar grpc.ServiceRegistrar) {
		registerPayloadService(registrar, payloadService{
			call: func(
				context.Context,
				*wrapperspb.BytesValue,
			) (*wrapperspb.BytesValue, error) {
				handlerCalls.Add(1)
				return &wrapperspb.BytesValue{}, nil
			},
		})
	}
	_, connection := startTestServer(t, cfg, register)

	if err := connection.Invoke(
		t.Context(),
		testPayloadFullMethod,
		&wrapperspb.BytesValue{},
		&wrapperspb.BytesValue{},
	); err != nil {
		t.Fatalf("baseline Invoke() error = %v", err)
	}
	ctx := metadata.NewOutgoingContext(
		t.Context(),
		metadata.Pairs("oversized", strings.Repeat("x", 4<<10)),
	)
	err := connection.Invoke(
		ctx,
		testPayloadFullMethod,
		&wrapperspb.BytesValue{},
		&wrapperspb.BytesValue{},
	)
	if err == nil {
		t.Fatal("Invoke() with oversized metadata succeeded")
	}
	if calls := handlerCalls.Load(); calls != 1 {
		t.Fatalf("handler calls = %d, want only the baseline call", calls)
	}
}

func TestServerMaxConcurrentStreamsQueuesAdditionalStreams(t *testing.T) {
	cfg := testServerConfig()
	cfg.MaxConcurrentStreams = 1
	server, connection := startTestServer(t, cfg, nil)
	server.MarkServing()
	healthClient := healthgrpc.NewHealthClient(connection)

	firstCtx, cancelFirst := context.WithCancel(t.Context())
	first, err := healthClient.Watch(firstCtx, &healthgrpc.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("first Health.Watch() error = %v", err)
	}
	if _, err := first.Recv(); err != nil {
		t.Fatalf("first Health.Watch().Recv() error = %v", err)
	}

	secondCtx, cancelSecond := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancelSecond()
	second, err := healthClient.Watch(secondCtx, &healthgrpc.HealthCheckRequest{})
	if err == nil {
		_, err = second.Recv()
	}
	if !errors.Is(err, context.DeadlineExceeded) && status.Code(err) != codes.DeadlineExceeded {
		t.Fatalf("second Health.Watch() error = %v, want stream-cap deadline", err)
	}

	cancelFirst()
	if _, err := first.Recv(); status.Code(err) != codes.Canceled {
		t.Fatalf("first Health.Watch().Recv() after cancel = %v, want Canceled", err)
	}

	thirdCtx, cancelThird := context.WithTimeout(t.Context(), time.Second)
	defer cancelThird()
	third, err := healthClient.Watch(thirdCtx, &healthgrpc.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("third Health.Watch() after release error = %v", err)
	}
	if _, err := third.Recv(); err != nil {
		t.Fatalf("third Health.Watch().Recv() after release = %v", err)
	}
}

const testPayloadFullMethod = "/grpcx.test.PayloadService/Call"

type payloadServiceServer interface {
	Call(ctx context.Context, request *wrapperspb.BytesValue) (*wrapperspb.BytesValue, error)
}

type payloadService struct {
	call func(context.Context, *wrapperspb.BytesValue) (*wrapperspb.BytesValue, error)
}

func (s payloadService) Call(
	ctx context.Context,
	request *wrapperspb.BytesValue,
) (*wrapperspb.BytesValue, error) {
	return s.call(ctx, request)
}

func registerPayloadService( //nolint:dupl // Manual descriptors intentionally mirror generated unary handlers.
	registrar grpc.ServiceRegistrar,
	service payloadServiceServer,
) {
	registrar.RegisterService(&grpc.ServiceDesc{
		ServiceName: "grpcx.test.PayloadService",
		HandlerType: (*payloadServiceServer)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "Call",
				Handler: func(
					implementation any,
					ctx context.Context,
					decode func(any) error,
					interceptor grpc.UnaryServerInterceptor,
				) (any, error) {
					request := new(wrapperspb.BytesValue)
					if err := decode(request); err != nil {
						return nil, err
					}
					service, ok := implementation.(payloadServiceServer)
					if !ok {
						return nil, errors.New("test payload service implementation has unexpected type")
					}
					handler := func(ctx context.Context, decoded any) (any, error) {
						typedRequest, ok := decoded.(*wrapperspb.BytesValue)
						if !ok {
							return nil, errors.New("test payload request has unexpected type")
						}
						return service.Call(ctx, typedRequest)
					}
					if interceptor == nil {
						return handler(ctx, request)
					}
					return interceptor(ctx, request, &grpc.UnaryServerInfo{
						Server:     implementation,
						FullMethod: testPayloadFullMethod,
					}, handler)
				},
			},
		},
	}, service)
}
