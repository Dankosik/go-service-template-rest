// Package grpcreference is an isolated example of all four gRPC method
// cardinalities. Production services copy the ownership shape, not this API.
package grpcreference

import (
	"context"
	"errors"
	"fmt"
	"io"

	referencev1 "github.com/example/go-service-template-rest/examples/grpc-reference-service/internal/gen/proto/reference/v1"
	"github.com/example/go-service-template-rest/internal/problem"
)

const (
	maxReferenceStreamMessages = 1024
	maxClientStreamValueBytes  = 1 << 20
)

// ErrStreamLimit reports that a stream exceeded this example's bounded
// aggregation.
//
// It is a domain identity, not a transport status: returning
// status.Error(codes.ResourceExhausted, …) here would reach the caller as
// INTERNAL instead, because the shared transport sanitizes any handler error it
// did not build itself. Classification belongs to DomainErrors below, which is
// the same seam a production service uses.
var ErrStreamLimit = errors.New("stream exceeds the reference aggregation limit")

// DomainErrors classifies this example's domain identities for the shared gRPC
// transport. A composition root passes the result as grpcx.Options.DomainErrors;
// see docs/grpc.md for the equivalent step in a production service.
func DomainErrors() []problem.Mapper {
	return []problem.Mapper{
		func(err error) (problem.Mapped, bool) {
			if !errors.Is(err, ErrStreamLimit) {
				return problem.Mapped{}, false
			}
			// Detail is the service's own wording; the wrapped error text never
			// reaches the caller.
			return problem.Mapped{
				Code:   problem.CodeRequestEntityTooLarge,
				Detail: "stream exceeds the reference aggregation limit",
			}, true
		},
	}
}

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
		return fmt.Errorf("server stream requested %d messages: %w", count, ErrStreamLimit)
	}
	for index := range count {
		sequence := index + 1
		if err := stream.Send(referencev1.ServerStreamResponse_builder{
			Value:    new(request.GetValue()),
			Sequence: new(sequence),
		}.Build()); err != nil {
			return streamFailure(stream.Context(), "send server-stream response", err)
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
				return streamFailure(stream.Context(), "close client stream", err)
			}
			return nil
		}
		if err != nil {
			return streamFailure(stream.Context(), "receive client-stream request", err)
		}
		value := request.GetValue()
		if len(values) >= maxReferenceStreamMessages ||
			len(value) > maxClientStreamValueBytes-totalValueBytes {
			return fmt.Errorf(
				"client stream reached %d messages and %d bytes: %w",
				len(values),
				totalValueBytes,
				ErrStreamLimit,
			)
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
			return streamFailure(stream.Context(), "receive bidi-stream request", err)
		}
		if err := stream.Send(referencev1.BidiStreamResponse_builder{
			Value: new(request.GetValue()),
		}.Build()); err != nil {
			return streamFailure(stream.Context(), "send bidi-stream response", err)
		}
	}
}

// streamFailure reports why a stream operation failed, preferring the stream's
// context error: a Send or Recv that failed only because the caller went away
// should name that rather than the transport symptom it produced.
//
// One action string covers both, because the wrapped error already says which
// happened — "receive bidi-stream request: context canceled" needs no second
// word for "canceled". None of this text reaches the client; the shared
// transport sanitizes any error it did not build itself, so this wording is for
// the server's own logs.
func streamFailure(ctx context.Context, action string, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("%s: %w", action, ctxErr)
	}
	return fmt.Errorf("%s: %w", action, err)
}
