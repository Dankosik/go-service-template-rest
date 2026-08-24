package bootstrap

import (
	"context"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/getkin/kin-openapi/openapi3filter"

	// profile:grpc:start
	"google.golang.org/grpc"
	// profile:grpc:end
)

// authnRuntime is the exact authentication surface bootstrap consumes. The
// interface lives here, at the consumer that needs substitution for ordered
// startup proof; the concrete runtime remains the only production
// implementation.
type authnRuntime interface {
	Close()
	ResolveHTTP(ctx context.Context, input *openapi3filter.AuthenticationInput) (reqctx.Principal, error)
	// profile:grpc:start
	UnaryInterceptor() grpc.UnaryServerInterceptor
	StreamInterceptor() grpc.StreamServerInterceptor
	// profile:grpc:end
}
