// Package grpctest builds hand-written gRPC service descriptors for tests.
//
// A test that drives a real RPC through a real server, from a package owning no
// generated proto of its own, needs a hand-written grpc.ServiceDesc. Written once
// per package, those descriptors drift the way these did before this package
// existed: one copy guarded the nil interceptor grpc-go passes when the server
// chained none and another did not, so the same test service was safe in one
// package and a nil-func panic in the next.
//
// A method is named the way a caller invokes it, so one constant names both the
// registration and the grpc.ClientConn.Invoke or NewStream that drives it:
//
//	const fullMethod = "/example.test.Service/Call"
//	grpctest.Register(registrar, grpctest.Unary(fullMethod, call))
//
// This package is test support that lives outside _test.go because three test
// binaries import it; it has no production caller and is removed with the rest
// of the gRPC profile.
package grpctest

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// Method is one method on a test service. Build it with [Unary] or
// [ServerStream] and register it with [Register].
type Method struct {
	service string
	unary   *grpc.MethodDesc
	stream  *grpc.StreamDesc
}

// Unary describes one unary method named by fullMethod
// ("/package.Service/Method") and backed by call.
//
// The request type travels as a type parameter, so one descriptor serves every
// message type instead of one near-identical copy per test package.
func Unary[Request any, RequestPtr interface {
	*Request
	proto.Message
}](
	fullMethod string,
	call func(context.Context, RequestPtr) (RequestPtr, error),
) Method {
	service, method := splitFullMethod(fullMethod)
	return Method{
		service: service,
		unary: &grpc.MethodDesc{
			MethodName: method,
			Handler: func(
				implementation any,
				ctx context.Context,
				decode func(any) error,
				interceptor grpc.UnaryServerInterceptor,
			) (any, error) {
				request := RequestPtr(new(Request))
				if err := decode(request); err != nil {
					return nil, err
				}
				handler := func(ctx context.Context, decoded any) (any, error) {
					typed, ok := decoded.(RequestPtr)
					if !ok {
						return nil, fmt.Errorf("test request is %T, want %T", decoded, request)
					}
					return call(ctx, typed)
				}
				// grpc-go passes a nil interceptor when the server chained none.
				// This guard is what makes the descriptor safe to serve from a
				// bare grpc.NewServer, which calling a nil func would not be.
				if interceptor == nil {
					return handler(ctx, request)
				}
				return interceptor(ctx, request, &grpc.UnaryServerInfo{
					Server:     implementation,
					FullMethod: fullMethod,
				}, handler)
			},
		},
	}
}

// ServerStream describes one server-streaming method named by fullMethod and
// backed by call.
//
// call receives the stream itself rather than a decoded request: whether the
// client opens with a message at all differs per caller, so the one that cares
// calls stream.RecvMsg and the one that does not is not made to.
func ServerStream(fullMethod string, call func(grpc.ServerStream) error) Method {
	service, method := splitFullMethod(fullMethod)
	return Method{
		service: service,
		stream: &grpc.StreamDesc{
			StreamName:    method,
			ServerStreams: true,
			Handler: func(_ any, stream grpc.ServerStream) error {
				return call(stream)
			},
		},
	}
}

// Register registers first and rest as one service on registrar. Every method
// must name that same service, because grpc-go refuses a second registration of
// one service name.
//
// The implementation registers as nil, which is grpc-go's documented way to skip
// the HandlerType check: the typed call already lives in each method's closure,
// so there is no implementation value for that check to prove anything about.
func Register(registrar grpc.ServiceRegistrar, first Method, rest ...Method) {
	descriptor := grpc.ServiceDesc{ServiceName: first.service}
	for _, method := range append([]Method{first}, rest...) {
		if method.service != descriptor.ServiceName {
			panic(fmt.Sprintf(
				"grpctest.Register got methods on %q and %q; one call registers one service",
				descriptor.ServiceName,
				method.service,
			))
		}
		if method.unary != nil {
			descriptor.Methods = append(descriptor.Methods, *method.unary)
		}
		if method.stream != nil {
			descriptor.Streams = append(descriptor.Streams, *method.stream)
		}
	}
	registrar.RegisterService(&descriptor, nil)
}

func splitFullMethod(fullMethod string) (string, string) {
	trimmed, prefixed := strings.CutPrefix(fullMethod, "/")
	service, method, separated := strings.Cut(trimmed, "/")
	if !prefixed || !separated || service == "" || method == "" || strings.Contains(method, "/") {
		panic(fmt.Sprintf("test method %q is not /package.Service/Method", fullMethod))
	}
	return service, method
}
