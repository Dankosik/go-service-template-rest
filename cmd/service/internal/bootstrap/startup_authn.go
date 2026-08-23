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

// authnBootstrapStage names one point in startup that authentication must
// already be established at. runWithRuntime reports each through
// runtimeWiring.authnStage, which production leaves empty and
// authn_bootstrap_test.go substitutes to prove the order.
type authnBootstrapStage string

const (
	authnStageTrustEstablished authnBootstrapStage = "trust_established"
	authnStageHTTPRouterBuilt  authnBootstrapStage = "http_router_built"
	authnStageHTTPServerBuilt  authnBootstrapStage = "http_server_built"
	authnStageGRPCServerBuilt  authnBootstrapStage = "grpc_server_built"
)
