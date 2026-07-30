// Package grpcreference is an isolated example of all four gRPC method
// cardinalities. Production services copy the ownership shape, not this API.
package grpcreference

import (
	"context"
	"errors"
	"fmt"
	"io"

	referencev1 "github.com/example/go-service-template-rest/examples/grpc-reference-service/internal/gen/proto/reference/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxReferenceStreamMessages = 1024
	maxClientStreamValueBytes  = 1 << 20
)

// Service implements the generated reference contract without hiding it behind
// a template-specific interface.
type Service struct {
	referencev1.UnimplementedEchoServiceServer
}

func (Service) Unary(_ context.Context, request *referencev1.UnaryRequest) (*referencev1.UnaryResponse, error) {
	return referencev1.UnaryResponse_builder{Value: new(request.GetValue())}.Build(), nil
}

func (Service) ServerStream(
	request *referencev1.ServerStreamRequest,
	stream referencev1.EchoService_ServerStreamServer,
) error {
	count := request.GetCount()
	if count > maxReferenceStreamMessages {
		//nolint:wrapcheck // This canonical status and safe detail are the public RPC contract.
		return status.Error(codes.ResourceExhausted, "server stream exceeds the reference message limit")
	}
	for index := range count {
		sequence := index + 1
		if err := stream.Send(referencev1.ServerStreamResponse_builder{
			Value:    new(request.GetValue()),
			Sequence: new(sequence),
		}.Build()); err != nil {
			if ctxErr := stream.Context().Err(); ctxErr != nil {
				return fmt.Errorf("send server-stream response: %w", ctxErr)
			}
			return fmt.Errorf("send server-stream response: %w", err)
		}
	}
	return nil
}

func (Service) ClientStream(stream referencev1.EchoService_ClientStreamServer) error {
	values := make([]string, 0)
	totalValueBytes := 0
	for {
		request, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			if err := stream.SendAndClose(referencev1.ClientStreamResponse_builder{Values: values}.Build()); err != nil {
				if ctxErr := stream.Context().Err(); ctxErr != nil {
					return fmt.Errorf("close canceled client stream: %w", ctxErr)
				}
				return fmt.Errorf("close client stream: %w", err)
			}
			return nil
		}
		if err != nil {
			if ctxErr := stream.Context().Err(); ctxErr != nil {
				return fmt.Errorf("receive canceled client stream: %w", ctxErr)
			}
			return fmt.Errorf("receive client-stream request: %w", err)
		}
		value := request.GetValue()
		if len(values) >= maxReferenceStreamMessages ||
			len(value) > maxClientStreamValueBytes-totalValueBytes {
			//nolint:wrapcheck // This canonical status and safe detail are the public RPC contract.
			return status.Error(codes.ResourceExhausted, "client stream exceeds the reference aggregation limit")
		}
		values = append(values, value)
		totalValueBytes += len(value)
	}
}

func (Service) BidiStream(stream referencev1.EchoService_BidiStreamServer) error {
	for {
		request, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			if ctxErr := stream.Context().Err(); ctxErr != nil {
				return fmt.Errorf("receive canceled bidi stream: %w", ctxErr)
			}
			return fmt.Errorf("receive bidi-stream request: %w", err)
		}
		if err := stream.Send(referencev1.BidiStreamResponse_builder{
			Value: new(request.GetValue()),
		}.Build()); err != nil {
			if ctxErr := stream.Context().Err(); ctxErr != nil {
				return fmt.Errorf("send canceled bidi stream: %w", ctxErr)
			}
			return fmt.Errorf("send bidi-stream response: %w", err)
		}
	}
}
