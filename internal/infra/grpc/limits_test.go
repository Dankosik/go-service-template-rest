// The finite transport bounds: does a server built by NewServer actually refuse
// an oversized message, oversized incoming metadata, and more concurrent streams
// than it allows?
//
// Every case drives a real connection, because these limits are enforced by
// grpc-go from the values Config hands it and not by any code in this package.

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
			registerUnaryTestService(registrar, testPayloadFullMethod,
				func(
					context.Context,
					*wrapperspb.BytesValue,
				) (*wrapperspb.BytesValue, error) {
					handlerCalled = true
					return &wrapperspb.BytesValue{}, nil
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
			registerUnaryTestService(registrar, testPayloadFullMethod,
				func(
					context.Context,
					*wrapperspb.BytesValue,
				) (*wrapperspb.BytesValue, error) {
					handlerCalled = true
					return &wrapperspb.BytesValue{Value: make([]byte, 1024)}, nil
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
		registerUnaryTestService(registrar, testPayloadFullMethod,
			func(
				context.Context,
				*wrapperspb.BytesValue,
			) (*wrapperspb.BytesValue, error) {
				handlerCalls.Add(1)
				return &wrapperspb.BytesValue{}, nil
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
