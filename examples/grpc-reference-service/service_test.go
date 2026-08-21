// All four method cardinalities, and the bound each streaming one refuses to
// cross.
//
// Every case dials a server built by grpcx.NewServer rather than a bare
// grpc.NewServer, which is what makes the status assertions mean anything: a
// handler here returns a domain error and the transport renders the code, so
// against a bare server the ResourceExhausted expectations below would read as
// satisfied while a composed service answered INTERNAL.
//
// This is the example a service author copies, so it doubles as the worked proof
// that a handler never picks its own codes.Code — see DomainErrors in service.go
// and docs/grpc/runtime-and-streaming.md.

package grpcreference_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"slices"
	"testing"
	"time"

	grpcreference "github.com/example/go-service-template-rest/examples/grpc-reference-service"
	referencev1 "github.com/example/go-service-template-rest/examples/grpc-reference-service/internal/gen/proto/reference/v1"
	grpcx "github.com/example/go-service-template-rest/internal/infra/grpc"
	"github.com/example/go-service-template-rest/internal/infra/grpc/grpctest"
	"github.com/example/go-service-template-rest/internal/waittest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestReferenceServiceSupportsAllCardinalities(t *testing.T) {
	client := referencev1.NewEchoServiceClient(newConnection(t))
	ctx := t.Context()

	unary, err := client.Unary(ctx, referencev1.UnaryRequest_builder{Value: new("unary")}.Build())
	if err != nil {
		t.Fatalf("Unary() error = %v", err)
	}
	if got := unary.GetValue(); got != "unary" {
		t.Fatalf("Unary() value = %q, want unary", got)
	}

	serverStream, err := client.ServerStream(ctx, referencev1.ServerStreamRequest_builder{
		Value: new("server"),
		Count: new(int32(3)),
	}.Build())
	if err != nil {
		t.Fatalf("ServerStream() error = %v", err)
	}
	for index := range int32(3) {
		wantSequence := index + 1
		response, receiveErr := serverStream.Recv()
		if receiveErr != nil {
			t.Fatalf("ServerStream().Recv() error = %v", receiveErr)
		}
		if response.GetValue() != "server" || response.GetSequence() != wantSequence {
			t.Fatalf(
				"ServerStream().Recv() = (%q, %d), want (server, %d)",
				response.GetValue(),
				response.GetSequence(),
				wantSequence,
			)
		}
	}
	if _, receiveErr := serverStream.Recv(); !errors.Is(receiveErr, io.EOF) {
		t.Fatalf("ServerStream() terminal error = %v, want EOF", receiveErr)
	}

	clientStream, err := client.ClientStream(ctx)
	if err != nil {
		t.Fatalf("ClientStream() error = %v", err)
	}
	for _, value := range []string{"one", "two", "three"} {
		if sendErr := clientStream.Send(referencev1.ClientStreamRequest_builder{
			Value: new(value),
		}.Build()); sendErr != nil {
			t.Fatalf("ClientStream().Send() error = %v", sendErr)
		}
	}
	clientStreamResponse, err := clientStream.CloseAndRecv()
	if err != nil {
		t.Fatalf("ClientStream().CloseAndRecv() error = %v", err)
	}
	if got, want := clientStreamResponse.GetValues(), []string{"one", "two", "three"}; !slices.Equal(got, want) {
		t.Fatalf("ClientStream() values = %v, want %v", got, want)
	}

	bidiStream, err := client.BidiStream(ctx)
	if err != nil {
		t.Fatalf("BidiStream() error = %v", err)
	}
	bidiValues := []string{"left", "right"}
	sendDone := make(chan error, 1)
	go func() {
		for _, value := range bidiValues {
			if sendErr := bidiStream.Send(referencev1.BidiStreamRequest_builder{
				Value: new(value),
			}.Build()); sendErr != nil {
				sendDone <- sendErr
				return
			}
		}
		sendDone <- bidiStream.CloseSend()
	}()
	for _, value := range bidiValues {
		response, receiveErr := bidiStream.Recv()
		if receiveErr != nil {
			t.Fatalf("BidiStream().Recv() error = %v", receiveErr)
		}
		if got := response.GetValue(); got != value {
			t.Fatalf("BidiStream().Recv() value = %q, want %q", got, value)
		}
	}
	if err := <-sendDone; err != nil {
		t.Fatalf("concurrent BidiStream().Send()/CloseSend() error = %v", err)
	}
	if _, receiveErr := bidiStream.Recv(); !errors.Is(receiveErr, io.EOF) {
		t.Fatalf("BidiStream() terminal error = %v, want EOF", receiveErr)
	}
}

func TestServerStreamBounds(t *testing.T) {
	client := referencev1.NewEchoServiceClient(newConnection(t))

	for _, count := range []int32{-1, 0, 1, 1024} {
		t.Run(fmt.Sprintf("count_%d", count), func(t *testing.T) {
			stream, err := client.ServerStream(t.Context(), referencev1.ServerStreamRequest_builder{
				Value: new("bounded"),
				Count: new(count),
			}.Build())
			if err != nil {
				t.Fatalf("ServerStream(count=%d) error = %v", count, err)
			}

			for index := range count {
				wantSequence := index + 1
				response, err := stream.Recv()
				if err != nil {
					t.Fatalf("ServerStream(count=%d).Recv(sequence=%d) error = %v", count, wantSequence, err)
				}
				if response.GetValue() != "bounded" || response.GetSequence() != wantSequence {
					t.Fatalf(
						"ServerStream(count=%d).Recv() = (%q, %d), want (bounded, %d)",
						count,
						response.GetValue(),
						response.GetSequence(),
						wantSequence,
					)
				}
			}
			if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
				t.Fatalf("ServerStream(count=%d) terminal error = %v, want EOF", count, err)
			}
		})
	}

	for _, count := range []int32{1025, math.MaxInt32} {
		t.Run(fmt.Sprintf("reject_%d", count), func(t *testing.T) {
			stream, err := client.ServerStream(t.Context(), referencev1.ServerStreamRequest_builder{
				Value: new("rejected"),
				Count: new(count),
			}.Build())
			if err != nil {
				if code := status.Code(err); code != codes.ResourceExhausted {
					t.Fatalf("ServerStream(count=%d) code = %s, want %s", count, code, codes.ResourceExhausted)
				}
				return
			}
			if _, err := stream.Recv(); status.Code(err) != codes.ResourceExhausted {
				t.Fatalf(
					"ServerStream(count=%d) first result = %v, want %s before any response",
					count,
					err,
					codes.ResourceExhausted,
				)
			}
		})
	}
}

func TestReferenceClientStreamRejectsUnboundedAggregation(t *testing.T) {
	client := referencev1.NewEchoServiceClient(newConnection(t))

	t.Run("aggregate bytes", func(t *testing.T) {
		stream, err := client.ClientStream(t.Context())
		if err != nil {
			t.Fatalf("ClientStream() error = %v", err)
		}

		oversized := string(make([]byte, (1<<20)+1))
		if err := stream.Send(referencev1.ClientStreamRequest_builder{Value: new(oversized)}.Build()); err != nil {
			t.Fatalf("ClientStream().Send() error = %v", err)
		}
		_, err = stream.CloseAndRecv()
		if code := status.Code(err); code != codes.ResourceExhausted {
			t.Fatalf("ClientStream().CloseAndRecv() code = %s, want %s", code, codes.ResourceExhausted)
		}
	})

	t.Run("message count", func(t *testing.T) {
		stream, err := client.ClientStream(t.Context())
		if err != nil {
			t.Fatalf("ClientStream() error = %v", err)
		}

		empty := referencev1.ClientStreamRequest_builder{Value: new("")}.Build()
		for range 1025 {
			if err := stream.Send(empty); err != nil {
				break
			}
		}
		_, err = stream.CloseAndRecv()
		if code := status.Code(err); code != codes.ResourceExhausted {
			t.Fatalf("ClientStream().CloseAndRecv() code = %s, want %s", code, codes.ResourceExhausted)
		}
	})
}

func TestReferenceBidiStreamReleasesServerWorkOnCancellation(t *testing.T) {
	handlerDone := make(chan struct{})
	client := referencev1.NewEchoServiceClient(newConnectionWithService(t, cancellationService{
		Service: grpcreference.Service{},
		done:    handlerDone,
	}))
	ctx, cancel := context.WithCancel(t.Context())
	stream, err := client.BidiStream(ctx)
	if err != nil {
		t.Fatalf("BidiStream() error = %v", err)
	}
	if err := stream.Send(referencev1.BidiStreamRequest_builder{Value: new("active")}.Build()); err != nil {
		t.Fatalf("BidiStream().Send() error = %v", err)
	}
	if _, err := stream.Recv(); err != nil {
		t.Fatalf("BidiStream().Recv() before cancellation = %v", err)
	}

	cancel()
	if _, err := stream.Recv(); status.Code(err) != codes.Canceled {
		t.Fatalf("BidiStream().Recv() after cancellation = %v, want Canceled", err)
	}
	waittest.ReceiveSignal(t, handlerDone, time.Second, "BidiStream handler to release server work after cancellation")
}

func newConnection(t *testing.T) *grpc.ClientConn {
	t.Helper()

	return newConnectionWithService(t, grpcreference.Service{})
}

func newConnectionWithService(t *testing.T, service referencev1.EchoServiceServer) *grpc.ClientConn {
	t.Helper()

	// Compose through the production adapter rather than a bare grpc.NewServer.
	// Every client-visible status asserted below is produced by that interceptor
	// chain, not by the handler alone: against a bare server the assertions would
	// still pass while the composed service returned INTERNAL.
	server, err := grpcx.NewServer(
		grpcx.Options{
			Logger:       slog.New(slog.DiscardHandler),
			DomainErrors: grpcreference.DomainErrors(),
			Services: []grpcx.RegisterService{
				func(registrar grpc.ServiceRegistrar) {
					referencev1.RegisterEchoServiceServer(registrar, service)
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("grpcx.NewServer() error = %v", err)
	}

	return grpctest.ServeBufconn(t, server)
}

type cancellationService struct {
	grpcreference.Service

	done chan struct{}
}

func (s cancellationService) BidiStream(stream referencev1.EchoService_BidiStreamServer) error {
	defer close(s.done)
	if err := s.Service.BidiStream(stream); err != nil {
		return fmt.Errorf("run cancellation test service: %w", err)
	}
	return nil
}
