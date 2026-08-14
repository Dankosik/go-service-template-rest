package oauth2clientcredentials

// profile:outbound-auth-grpc:start

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"

	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type grpcOperationKey struct{}

type grpcOperation struct {
	token operationToken

	mu  sync.Mutex
	err error
}

func (o *grpcOperation) fail(err error) {
	o.mu.Lock()
	if o.err == nil {
		o.err = err
	}
	o.mu.Unlock()
}

func (o *grpcOperation) failure() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.err
}

type grpcCredential struct {
	client *Client
}

func newGRPCCredential(client *Client) (*grpcCredential, error) {
	if client == nil {
		return nil, failure(FailureInvalidConfiguration)
	}
	return &grpcCredential{client: client}, nil
}

// GetRequestMetadata supplies one bearer credential to an admitted TLS attempt.
func (c *grpcCredential) GetRequestMetadata(ctx context.Context, requestURI ...string) (map[string]string, error) {
	if c == nil || c.client == nil || ctx == nil || !c.matchesResourceAuthority(requestURI) {
		return nil, failure(FailureInvalidConfiguration)
	}

	if operation, ok := ctx.Value(grpcOperationKey{}).(*grpcOperation); ok {
		value, err := operation.token.authorization()
		if err != nil {
			operation.fail(err)
			return nil, err
		}
		return map[string]string{"authorization": "Bearer " + value}, nil
	}

	token, err := c.client.resolve(ctx)
	if err != nil {
		if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
			//nolint:wrapcheck // Preserve the exact caller-context gRPC status.
			return nil, status.FromContextError(ctx.Err()).Err()
		}
		return nil, err
	}
	value, err := token.authorization()
	if err != nil {
		return nil, err
	}
	return map[string]string{"authorization": "Bearer " + value}, nil
}

// RequireTransportSecurity prevents grpc-go from sending the credential on plaintext connections.
func (*grpcCredential) RequireTransportSecurity() bool {
	return true
}

func (c *grpcCredential) wrap( //nolint:ireturn // Generated gRPC clients consume this standard interface.
	base grpc.ClientConnInterface,
) (grpc.ClientConnInterface, error) {
	if c == nil || c.client == nil || base == nil {
		return nil, failure(FailureInvalidConfiguration)
	}
	return grpcApplicationConn{credential: c, base: base}, nil
}

func (c *grpcCredential) matchesResourceAuthority(requestURI []string) bool {
	if len(requestURI) != 1 {
		return false
	}
	parsed, err := url.Parse(requestURI[0])
	if err != nil || parsed.User != nil || !strings.EqualFold(parsed.Scheme, "https") {
		return false
	}
	resource, err := url.Parse(c.client.config.ResourceAuthority)
	if err != nil {
		return false
	}
	return grpcAuthority(parsed) == grpcAuthority(resource)
}

func (c *grpcCredential) observeRPC(ctx context.Context, err error) {
	var result string
	//nolint:exhaustive // Only downstream auth rejections emit this metric.
	switch status.Code(err) {
	case codes.Unauthenticated:
		result = resultUnauthenticated
	case codes.PermissionDenied:
		result = resultForbidden
	default:
		return
	}
	c.client.telemetry.recordResourceRejection(ctx, transportGRPC, result)
}

// GRPCClient is one correctly credentialed connection for generated clients.
type GRPCClient struct {
	connection  *grpc.ClientConn
	application grpc.ClientConnInterface
}

// NewGRPCClient binds one credential owner to application and grpc-go control RPCs.
func NewGRPCClient(client *Client, cfg grpcclient.Config, options grpcclient.Options) (*GRPCClient, error) {
	if options.PerRPCCredentials != nil || options.ObserveRPC != nil {
		return nil, failure(FailureInvalidConfiguration)
	}
	credential, err := newGRPCCredential(client)
	if err != nil {
		return nil, err
	}
	options.PerRPCCredentials = credential
	options.ObserveRPC = credential.observeRPC
	connection, err := grpcclient.New(cfg, options)
	if err != nil {
		return nil, failure(FailureInvalidConfiguration)
	}
	application, err := credential.wrap(connection)
	if err != nil {
		_ = connection.Close()
		return nil, err
	}
	return &GRPCClient{connection: connection, application: application}, nil
}

// Invoke starts one authenticated logical unary operation.
func (c *GRPCClient) Invoke(ctx context.Context, method string, args, reply any, options ...grpc.CallOption) error {
	if c == nil || c.application == nil {
		return failure(FailureInvalidConfiguration)
	}
	return c.application.Invoke(ctx, method, args, reply, options...) //nolint:wrapcheck // Preserve the downstream status.
}

// NewStream starts one authenticated logical streaming operation.
func (c *GRPCClient) NewStream( //nolint:ireturn // Required by grpc.ClientConnInterface.
	ctx context.Context,
	description *grpc.StreamDesc,
	method string,
	options ...grpc.CallOption,
) (grpc.ClientStream, error) {
	if c == nil || c.application == nil {
		return nil, failure(FailureInvalidConfiguration)
	}
	return c.application.NewStream(ctx, description, method, options...) //nolint:wrapcheck // Preserve the downstream status.
}

// Close releases the underlying shared connection.
func (c *GRPCClient) Close() error {
	if c == nil || c.connection == nil {
		return failure(FailureInvalidConfiguration)
	}
	return c.connection.Close() //nolint:wrapcheck // grpc-go owns the connection close result.
}

func grpcAuthority(parsed *url.URL) string {
	port := parsed.Port()
	if port == "443" {
		port = ""
	}
	return strings.ToLower(parsed.Hostname()) + "\x00" + port
}

type grpcApplicationConn struct {
	credential *grpcCredential
	base       grpc.ClientConnInterface
}

func (c grpcApplicationConn) Invoke(
	ctx context.Context,
	method string,
	args any,
	reply any,
	options ...grpc.CallOption,
) error {
	operation, callCtx, err := c.startOperation(ctx, options)
	if err != nil {
		return err
	}
	err = c.base.Invoke(callCtx, method, args, reply, options...)
	if operationErr := operation.failure(); operationErr != nil {
		return operationErr
	}
	//nolint:wrapcheck // Preserve the downstream gRPC status unchanged.
	return err
}

func (c grpcApplicationConn) NewStream( //nolint:ireturn // Required by grpc.ClientConnInterface.
	ctx context.Context,
	description *grpc.StreamDesc,
	method string,
	options ...grpc.CallOption,
) (grpc.ClientStream, error) {
	operation, callCtx, err := c.startOperation(ctx, options)
	if err != nil {
		return nil, err
	}
	stream, err := c.base.NewStream(callCtx, description, method, options...)
	if operationErr := operation.failure(); operationErr != nil {
		return nil, operationErr
	}
	//nolint:wrapcheck // Preserve the downstream gRPC status unchanged.
	if err != nil {
		return stream, err
	}
	return &grpcApplicationStream{ClientStream: stream, operation: operation}, nil
}

type grpcApplicationStream struct {
	grpc.ClientStream

	operation *grpcOperation
}

func (s *grpcApplicationStream) Header() (metadata.MD, error) {
	values, err := s.ClientStream.Header()
	if operationErr := s.operation.failure(); operationErr != nil {
		return nil, operationErr
	}
	//nolint:wrapcheck // Preserve the downstream gRPC status unchanged.
	return values, err
}

func (s *grpcApplicationStream) CloseSend() error {
	return s.finish(s.ClientStream.CloseSend())
}

func (s *grpcApplicationStream) SendMsg(message any) error {
	return s.finish(s.ClientStream.SendMsg(message))
}

func (s *grpcApplicationStream) RecvMsg(message any) error {
	return s.finish(s.ClientStream.RecvMsg(message))
}

func (s *grpcApplicationStream) finish(err error) error {
	if operationErr := s.operation.failure(); operationErr != nil {
		return operationErr
	}
	return err
}

func (c grpcApplicationConn) startOperation(
	ctx context.Context,
	options []grpc.CallOption,
) (*grpcOperation, context.Context, error) {
	if ctx == nil || hasOutgoingAuthorization(ctx) || hasPerRPCCredentials(options) {
		return nil, nil, failure(FailureInvalidConfiguration)
	}
	token, err := c.credential.client.resolve(ctx)
	if err != nil {
		return nil, nil, err
	}
	operation := &grpcOperation{token: token}
	return operation, context.WithValue(ctx, grpcOperationKey{}, operation), nil
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
	_ credentials.PerRPCCredentials = (*grpcCredential)(nil)
	_ grpc.ClientConnInterface      = (*GRPCClient)(nil)
	_ grpc.ClientConnInterface      = grpcApplicationConn{}
)

// profile:outbound-auth-grpc:end
