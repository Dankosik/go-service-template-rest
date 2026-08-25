package grpcx

import (
	"context"
	"fmt"
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

const (
	expectedValidationViolationLimit = 10
	rejectedValidationValue          = "secret-invalid-value"
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
		protoreflect.ValueOfString(rejectedValidationValue),
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
				return handlerErrorBoundary(log, nil)(
					stream.Context(),
					testStreamFullMethod,
					func(context.Context) error {
						if err := stream.RecvMsg(messageType.New().Interface()); err != nil {
							return fmt.Errorf("receive validated message: %w", err)
						}
						return nil
					},
				)
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
	if strings.Contains(err.Error(), rejectedValidationValue) {
		t.Fatalf("validation status disclosed rejected value: %v", err)
	}
	details := converted.Details()
	if len(details) != 1 {
		t.Fatalf("validation details = %v, want one structured violation", details)
	}
	violations, ok := details[0].(*validate.Violations)
	if !ok {
		t.Fatalf("validation detail type = %T", details[0])
	}
	if got := len(violations.GetViolations()); got != 1 {
		t.Fatalf("validation violations = %d, want 1", got)
	}
	assertPublicValidationViolation(t, violations.GetViolations()[0], "value", "value.private")
}

func TestPublicValidationViolationsBoundsAndRemovesPeerValues(t *testing.T) {
	raw := &protovalidate.Violation{Proto: validate.Violation_builder{
		Field: validate.FieldPath_builder{Elements: []*validate.FieldPathElement{
			validate.FieldPathElement_builder{
				FieldNumber: new(int32(1)),
				FieldName:   new("values"),
				StringKey:   new(rejectedValidationValue),
			}.Build(),
		}}.Build(),
		RuleId:  new("values.private"),
		Message: new(rejectedValidationValue),
	}.Build()}
	source := make([]*protovalidate.Violation, expectedValidationViolationLimit+1)
	for index := range source {
		source[index] = raw
	}
	validationErr := &protovalidate.ValidationError{Violations: source}

	got := publicValidationViolations(validationErr).GetViolations()
	if len(got) != expectedValidationViolationLimit {
		t.Fatalf("validation violations = %d, want %d", len(got), expectedValidationViolationLimit)
	}
	for _, violation := range got {
		assertPublicValidationViolation(t, violation, "values", "values.private")
	}
	if raw.Proto.GetMessage() != rejectedValidationValue ||
		raw.Proto.GetField().GetElements()[0].GetStringKey() != rejectedValidationValue {
		t.Fatal("public validation projection mutated the validator-owned error")
	}
}

func assertPublicValidationViolation(
	t *testing.T,
	violation *validate.Violation,
	wantField string,
	wantRule string,
) {
	t.Helper()
	if violation.HasMessage() || strings.Contains(violation.String(), rejectedValidationValue) {
		t.Fatalf("public validation violation disclosed rejected value: %v", violation)
	}
	if violation.GetRuleId() != wantRule {
		t.Fatalf("validation rule ID = %q, want %q", violation.GetRuleId(), wantRule)
	}
	field := violation.GetField().GetElements()
	if len(field) != 1 || field[0].GetFieldName() != wantField || field[0].HasSubscript() {
		t.Fatalf("public validation field path = %v, want %s without a subscript", violation.GetField(), wantField)
	}
}

func validationMessageType(t *testing.T) protoreflect.MessageType { //nolint:ireturn // Dynamic test descriptor API.
	t.Helper()
	options := new(descriptorpb.FieldOptions)
	proto.SetExtension(options, validate.E_Field, validate.FieldRules_builder{
		Cel: []*validate.Rule{validate.Rule_builder{
			Id:         new("value.private"),
			Expression: new("this == 'allowed' ? '' : 'rejected: ' + this"),
		}.Build()},
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
				Number:  new(int32(1)),
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
