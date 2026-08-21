package oauth2clientcredentials

import (
	"context"
	"crypto/tls"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

type testAuthInfo struct {
	credentials.CommonAuthInfo
}

func (testAuthInfo) AuthType() string { return "test" }

type staticCredential struct{}

func (staticCredential) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer competing"}, nil
}

func (staticCredential) RequireTransportSecurity() bool { return true }

func TestGRPCCredentialUsesCallerContextAndRequiresSecureTransport(t *testing.T) {
	owner := newClient(func(context.Context) (*oauth2.Token, error) {
		return validTestToken("opaque"), nil
	}, nil)
	credential := grpcCredential{client: owner}
	if !credential.RequireTransportSecurity() {
		t.Fatal("credential does not require transport security")
	}
	secure := credentials.NewContextWithRequestInfo(context.Background(), credentials.RequestInfo{
		AuthInfo: testAuthInfo{CommonAuthInfo: credentials.CommonAuthInfo{SecurityLevel: credentials.PrivacyAndIntegrity}},
	})
	values, err := credential.GetRequestMetadata(secure)
	if err != nil || values["authorization"] != "Bearer opaque" {
		t.Fatalf("GetRequestMetadata() = %v, %v", values, err)
	}
	insecure := credentials.NewContextWithRequestInfo(context.Background(), credentials.RequestInfo{
		AuthInfo: testAuthInfo{CommonAuthInfo: credentials.CommonAuthInfo{SecurityLevel: credentials.NoSecurity}},
	})
	if _, err := credential.GetRequestMetadata(insecure); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("insecure GetRequestMetadata() error = %v", err)
	}
}

func TestGRPCBuildsAuthenticatedGuardedConnection(t *testing.T) {
	owner := newClient(func(context.Context) (*oauth2.Token, error) {
		return validTestToken("opaque"), nil
	}, nil)
	config := grpcclient.DefaultConfig("dns:///resource.example.com:443")
	options := grpcclient.Options{TransportCredentials: credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})}
	client, err := owner.GRPC(config, options)
	if err != nil {
		t.Fatalf("GRPC() error = %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	options.PerRPCCredentials = staticCredential{}
	if _, err := owner.GRPC(config, options); !errors.Is(err, ErrInvalidConfiguration) {
		t.Fatalf("GRPC() competing credential error = %v", err)
	}
}

type recordingGRPCConnection struct {
	invokes atomic.Int32
	streams atomic.Int32
}

var errUnexpectedResourceCall = errors.New("unexpected resource RPC")

func (c *recordingGRPCConnection) Invoke(
	context.Context,
	string,
	any,
	any,
	...grpc.CallOption,
) error {
	c.invokes.Add(1)
	return errUnexpectedResourceCall
}

func (c *recordingGRPCConnection) NewStream( //nolint:ireturn // grpc.ClientConnInterface requires grpc.ClientStream.
	context.Context,
	*grpc.StreamDesc,
	string,
	...grpc.CallOption,
) (grpc.ClientStream, error) {
	c.streams.Add(1)
	return nil, errUnexpectedResourceCall
}

func (*recordingGRPCConnection) Close() error { return nil }

func TestGRPCClientRejectsCompetingAuthorizationBeforeResourceRPC(t *testing.T) {
	base := new(recordingGRPCConnection)
	client := &GRPCClient{connection: base}
	outgoing := metadata.NewOutgoingContext(t.Context(), metadata.Pairs("authorization", "Bearer competing"))
	if err := client.Invoke(outgoing, "/provider.Service/Get", nil, nil); !errors.Is(err, ErrInvalidConfiguration) {
		t.Fatalf("Invoke() error = %v, want ErrInvalidConfiguration", err)
	}
	if _, err := client.NewStream(
		t.Context(),
		new(grpc.StreamDesc),
		"/provider.Service/Watch",
		grpc.PerRPCCredentials(staticCredential{}),
	); !errors.Is(err, ErrInvalidConfiguration) {
		t.Fatalf("NewStream() error = %v, want ErrInvalidConfiguration", err)
	}
	if base.invokes.Load() != 0 || base.streams.Load() != 0 {
		t.Fatalf("resource calls = invokes:%d streams:%d, want zero", base.invokes.Load(), base.streams.Load())
	}
	if err := client.Invoke(t.Context(), "/provider.Service/Get", nil, nil); !errors.Is(err, errUnexpectedResourceCall) {
		t.Fatalf("Invoke() error = %v", err)
	}
	if _, err := client.NewStream(t.Context(), new(grpc.StreamDesc), "/provider.Service/Watch"); !errors.Is(err, errUnexpectedResourceCall) {
		t.Fatalf("NewStream() error = %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := new(GRPCClient).Close(); err != nil {
		t.Fatalf("empty Close() error = %v", err)
	}
}
