package grpcx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"buf.build/go/protovalidate"
	"github.com/example/go-service-template-rest/internal/failure"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const validationFailureDetail = "request validation failed"

func validationUnaryInterceptor(
	log *slog.Logger,
	validator protovalidate.Validator,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if !isHealthMethod(info.FullMethod) {
			if err := validateMessage(ctx, log, validator, info.FullMethod, request); err != nil {
				return nil, err
			}
		}
		return handler(ctx, request)
	}
}

func validationStreamInterceptor(
	log *slog.Logger,
	validator protovalidate.Validator,
) grpc.StreamServerInterceptor {
	return func(
		server any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if isHealthMethod(info.FullMethod) {
			return handler(server, stream)
		}
		return handler(server, &validatingServerStream{
			ServerStream: stream,
			validator:    validator,
			log:          log,
			method:       info.FullMethod,
		})
	}
}

type validatingServerStream struct {
	grpc.ServerStream

	validator protovalidate.Validator
	log       *slog.Logger
	method    string
}

func (s *validatingServerStream) RecvMsg(message any) error {
	if err := s.ServerStream.RecvMsg(message); err != nil {
		return err //nolint:wrapcheck // Preserve grpc-go receive status unchanged.
	}
	return validateMessage(s.Context(), s.log, s.validator, s.method, message)
}

func validateMessage(
	ctx context.Context,
	log *slog.Logger,
	validator protovalidate.Validator,
	method string,
	message any,
) error {
	protobuf, ok := message.(proto.Message)
	if !ok {
		err := fmt.Errorf("unsupported protobuf message type %T", message)
		recordUnhandledFailure(ctx, log, method, err)
		return ownedStatus(codes.Internal, failure.SanitizedDetail)
	}
	err := validator.Validate(protobuf)
	if err == nil {
		return nil
	}
	var validationErr *protovalidate.ValidationError
	if !errors.As(err, &validationErr) {
		recordUnhandledFailure(ctx, log, method, err)
		return ownedStatus(codes.Internal, failure.SanitizedDetail)
	}
	rendered := status.New(codes.InvalidArgument, validationFailureDetail)
	withDetails, detailErr := rendered.WithDetails(validationErr.ToProto())
	if detailErr != nil {
		recordUnhandledFailure(ctx, log, method, detailErr)
		return &ownedStatusError{status: rendered}
	}
	return &ownedStatusError{status: withDetails}
}
