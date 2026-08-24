package oauth2clientcredentials

// profile:outbound-auth-grpc:start

import (
	"context"
	"fmt"
	"strings"

	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

type grpcCredential struct {
	source oauth2.TokenSource
}

func (c grpcCredential) GetRequestMetadata(ctx context.Context, _ ...string) (map[string]string, error) {
	if ctx == nil || c.source == nil {
		return nil, ErrInvalidConfiguration
	}
	requestInfo, _ := credentials.RequestInfoFromContext(ctx)
	if err := credentials.CheckSecurityLevel(requestInfo.AuthInfo, credentials.PrivacyAndIntegrity); err != nil {
		return nil, ErrUnavailable
	}
	token, err := c.source.Token()
	if err != nil {
		return nil, ErrUnavailable
	}
	return map[string]string{"authorization": token.Type() + " " + token.AccessToken}, nil
}

func (grpcCredential) RequireTransportSecurity() bool { return true }

type grpcConnection interface {
	grpc.ClientConnInterface
	Close() error
}

// GRPCClient is an authenticated connection that rejects a second caller-owned
// authorization source before an RPC reaches grpc-go.
type GRPCClient struct {
	connection grpcConnection
}

// GRPC builds one authenticated connection without provider or resource I/O.
func (c *Client) GRPC(cfg grpcclient.Config, options grpcclient.Options) (*GRPCClient, error) {
	if !c.available() || options.PerRPCCredentials != nil {
		return nil, ErrInvalidConfiguration
	}
	options.PerRPCCredentials = grpcCredential{source: clientTokenSource{client: c}}
	connection, err := grpcclient.New(cfg, options)
	if err != nil {
		return nil, fmt.Errorf("build authenticated gRPC client: %w", err)
	}
	return &GRPCClient{connection: connection}, nil
}

func (c *GRPCClient) Invoke(
	ctx context.Context,
	method string,
	args any,
	reply any,
	options ...grpc.CallOption,
) error {
	if c == nil || c.connection == nil || ctx == nil || hasOutgoingAuthorization(ctx) || hasPerRPCCredentials(options) {
		return ErrInvalidConfiguration
	}
	return c.connection.Invoke(ctx, method, args, reply, options...) //nolint:wrapcheck // Preserve downstream gRPC status.
}

func (c *GRPCClient) NewStream( //nolint:ireturn // grpc.ClientConnInterface requires grpc.ClientStream.
	ctx context.Context,
	description *grpc.StreamDesc,
	method string,
	options ...grpc.CallOption,
) (grpc.ClientStream, error) {
	if c == nil || c.connection == nil || ctx == nil || hasOutgoingAuthorization(ctx) || hasPerRPCCredentials(options) {
		return nil, ErrInvalidConfiguration
	}
	return c.connection.NewStream(ctx, description, method, options...) //nolint:wrapcheck // Preserve downstream gRPC status.
}

func (c *GRPCClient) Close() error {
	if c == nil || c.connection == nil {
		return nil
	}
	return c.connection.Close() //nolint:wrapcheck // The owned connection defines its close contract.
}

func hasOutgoingAuthorization(ctx context.Context) bool {
	values, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return false
	}
	for key := range values {
		if strings.EqualFold(key, "authorization") {
			return true
		}
	}
	return false
}

func hasPerRPCCredentials(options []grpc.CallOption) bool {
	for _, option := range options {
		switch option.(type) {
		case grpc.PerRPCCredsCallOption, *grpc.PerRPCCredsCallOption:
			return true
		}
	}
	return false
}

var (
	_ credentials.PerRPCCredentials = grpcCredential{}
	_ grpc.ClientConnInterface      = (*GRPCClient)(nil)
)

// profile:outbound-auth-grpc:end
