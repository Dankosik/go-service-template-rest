package grpcx

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"buf.build/go/protovalidate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestProtovalidatePrivacyAndUnaryStreamParity(t *testing.T) {
	validator, err := protovalidate.New()
	if err != nil {
		t.Fatalf("protovalidate.New() error = %v", err)
	}
	messageType := validationMessageType(t)
	invalid := messageType.New().Interface()
	invalid.ProtoReflect().Set(
		invalid.ProtoReflect().Descriptor().Fields().ByName("value"),
		protoreflect.ValueOfString("secret-invalid-value"),
	)
	log := slog.New(slog.DiscardHandler)

	t.Run("unary", func(t *testing.T) {
		called := false
		interceptor := validationUnaryInterceptor(log, validator)
		_, err := interceptor(
			t.Context(),
			invalid,
			&grpc.UnaryServerInfo{FullMethod: testUnaryFullMethod},
			func(_ context.Context, request any) (any, error) {
				called = true
				return request, nil
			},
		)
		assertValidationStatus(t, err)
		if called {
			t.Fatal("invalid unary request reached handler")
		}
	})

	t.Run("stream receive", func(t *testing.T) {
		interceptor := validationStreamInterceptor(log, validator)
		err := interceptor(
			nil,
			&validationTestStream{ctx: t.Context(), source: invalid},
			&grpc.StreamServerInfo{FullMethod: testStreamFullMethod},
			func(_ any, stream grpc.ServerStream) error {
				return stream.RecvMsg(messageType.New().Interface())
			},
		)
		assertValidationStatus(t, err)
	})
}

func assertValidationStatus(t *testing.T, err error) {
	t.Helper()
	converted := status.Convert(err)
	if converted.Code() != codes.InvalidArgument || converted.Message() != validationFailureDetail {
		t.Fatalf("validation status = %s %q", converted.Code(), converted.Message())
	}
	if strings.Contains(err.Error(), "secret-invalid-value") {
		t.Fatalf("validation status disclosed rejected value: %v", err)
	}
	if details := converted.Details(); len(details) != 1 {
		t.Fatalf("validation details = %v, want one structured violation", details)
	} else if _, ok := details[0].(*validate.Violations); !ok {
		t.Fatalf("validation detail type = %T", details[0])
	}
}

func validationMessageType(t *testing.T) protoreflect.MessageType { //nolint:ireturn // Dynamic test descriptor API.
	t.Helper()
	maxLen := uint64(1)
	options := new(descriptorpb.FieldOptions)
	proto.SetExtension(options, validate.E_Field, validate.FieldRules_builder{
		String: validate.StringRules_builder{MaxLen: &maxLen}.Build(),
	}.Build())
	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:       new("grpcx_validation_test.proto"),
		Package:    new("grpcx.validation.test"),
		Syntax:     new("proto3"),
		Dependency: []string{"buf/validate/validate.proto"},
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: new("Request"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:    new("value"),
				Number:  proto.Int32(1),
				Label:   descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:    descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Options: options,
			}},
		}},
	}, protoregistry.GlobalFiles)
	if err != nil {
		t.Fatalf("protodesc.NewFile() error = %v", err)
	}
	return dynamicpb.NewMessageType(file.Messages().ByName("Request"))
}

type validationTestStream struct {
	grpc.ServerStream

	ctx    context.Context //nolint:containedctx // Fake ServerStream owns its Context result.
	source proto.Message
}

func (s *validationTestStream) Context() context.Context { return s.ctx }

func (s *validationTestStream) RecvMsg(target any) error {
	proto.Merge(target.(proto.Message), s.source) //nolint:forcetypeassert // Test controls the message type.
	return nil
}
