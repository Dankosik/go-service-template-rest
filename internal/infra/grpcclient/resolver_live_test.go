// Resolver guard, live: over a real TLS connection, do New's three closures
// hold together — no resolver-supplied service config, no proxy, and no
// resolver-supplied metadata on the wire?
//
// It runs in a child process because the proxy route is only reachable through
// process-global environment variables, which the parent sets for the child
// alone. The parent owns the TLS server and the recording proxy and asserts what
// each observed; the child owns the client and asserts what it could reach.
//
// resolver_internal_test.go covers the wrapper's own behavior, and
// resolver_selection_test.go covers which builder grpc-go selects.

package grpcclient_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/status"
)

const resolverLiveChildEnv = "GRPCCLIENT_RESOLVER_LIVE_CHILD"

func TestResolverServiceConfigProxyTLSAreClosedByClientPolicy(t *testing.T) {
	serverCertificate, leafCertificate := testTLSMaterial(t)
	var handlerCalls atomic.Int64
	serverAddress := startLiveTLSHealthServer(
		t,
		credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{serverCertificate},
			MinVersion:   tls.VersionTLS12,
		}),
		&handlerCalls,
	)
	proxy := startRecordingProxy(t)

	runTestBinaryChild(
		t,
		"TestResolverServiceConfigProxyTLSChild",
		resolverLiveChildEnv+"=1",
		"GRPCCLIENT_RESOLVER_LIVE_SERVER="+serverAddress,
		"GRPCCLIENT_RESOLVER_LIVE_ROOT="+base64.StdEncoding.EncodeToString(leafCertificate.Raw),
		"HTTPS_PROXY=http://"+proxy.address,
		"https_proxy=http://"+proxy.address,
		"HTTP_PROXY=http://"+proxy.address,
		"http_proxy=http://"+proxy.address,
		"NO_PROXY=",
		"no_proxy=",
	)
	if accepts := proxy.accepts.Load(); accepts != 1 {
		t.Fatalf("fake proxy accepts = %d, want exactly the native control connection", accepts)
	}
	if calls := handlerCalls.Load(); calls != 1 {
		t.Fatalf("TLS server handler calls = %d, want exactly one bounded-client call", calls)
	}
}

func TestResolverServiceConfigProxyTLSChild(t *testing.T) {
	if os.Getenv(resolverLiveChildEnv) == "" {
		t.Skip("subprocess-only live resolver fixture")
	}
	proxyURL, err := http.ProxyFromEnvironment(&http.Request{URL: &url.URL{
		Scheme: "https",
		Host:   "service.example.test:443",
	}})
	if err != nil || proxyURL == nil {
		t.Fatalf("proxy fixture is ineffective: URL = %v, error = %v", proxyURL, err)
	}

	serverAddress := os.Getenv("GRPCCLIENT_RESOLVER_LIVE_SERVER")
	if serverAddress == "" {
		t.Fatal("live server address is missing")
	}
	rawCertificate, err := base64.StdEncoding.DecodeString(
		os.Getenv("GRPCCLIENT_RESOLVER_LIVE_ROOT"),
	)
	if err != nil {
		t.Fatalf("decode live root certificate: %v", err)
	}
	leafCertificate, err := x509.ParseCertificate(rawCertificate)
	if err != nil {
		t.Fatalf("parse live root certificate: %v", err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(leafCertificate)
	clientCredentials := credentials.NewTLS(&tls.Config{
		RootCAs:    roots,
		MinVersion: tls.VersionTLS12,
	})

	const (
		scheme       = "resolver-live-proof"
		balancerName = "resolver_live_proof_balancer"
	)
	var balancerBuilds atomic.Int64
	var balancerPicks atomic.Int64
	balancer.Register(&proofBalancerBuilder{
		name:   balancerName,
		builds: &balancerBuilds,
		picks:  &balancerPicks,
	})
	builder := &liveResolverBuilder{
		scheme:     scheme,
		address:    serverAddress,
		serverName: "example.com",
		serviceConfig: fmt.Sprintf(
			`{"loadBalancingConfig":[{%q:{}}],"methodConfig":[{"name":[{"service":"grpc.health.v1.Health","method":"Check"}],"retryPolicy":{"maxAttempts":4,"initialBackoff":"0.01s","maxBackoff":"0.01s","backoffMultiplier":1,"retryableStatusCodes":["UNAVAILABLE"]}}]}`,
			balancerName,
		),
	}
	resolver.Register(builder)
	target := scheme + ":///service.example.test:443"

	// First prove that the configured environment is an effective proxy control:
	// a native client without WithNoProxy reaches the fake proxy.
	controlBuilder := &liveResolverBuilder{
		scheme:     scheme,
		address:    serverAddress,
		serverName: "example.com",
	}
	controlConnection, err := grpc.NewClient(
		target,
		grpc.WithResolvers(controlBuilder),
		grpc.WithTransportCredentials(clientCredentials),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient() proxy control error = %v", err)
	}
	controlCtx, cancelControl := context.WithTimeout(t.Context(), 300*time.Millisecond)
	_, controlErr := healthgrpc.NewHealthClient(controlConnection).Check(
		controlCtx,
		&healthgrpc.HealthCheckRequest{},
	)
	cancelControl()
	if status.Code(controlErr) != codes.DeadlineExceeded {
		t.Fatalf("native proxy control error = %v, want deadline while fake proxy holds connection", controlErr)
	}
	if err := controlConnection.Close(); err != nil {
		t.Fatalf("proxy control ClientConn.Close() error = %v", err)
	}

	connection, err := grpcclient.New(
		grpcclient.DefaultConfig(target),
		grpcclient.Options{TransportCredentials: clientCredentials},
	)
	if err != nil {
		t.Fatalf("grpcclient.New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := connection.Close(); err != nil {
			t.Errorf("ClientConn.Close() error = %v", err)
		}
	})
	callCtx, cancelCall := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancelCall()
	_, err = healthgrpc.NewHealthClient(connection).Check(
		callCtx,
		&healthgrpc.HealthCheckRequest{},
	)
	if got := status.Code(err); got != codes.Unavailable {
		t.Fatalf("direct committed call status = %s, want %s (error %v)", got, codes.Unavailable, err)
	}
	if builds := balancerBuilds.Load(); builds != 0 {
		t.Fatalf("resolver-selected balancer builds = %d, want 0", builds)
	}
	if picks := balancerPicks.Load(); picks != 0 {
		t.Fatalf("resolver-selected balancer picks = %d, want 0", picks)
	}
	if !builder.disableServiceConfig.Load() {
		t.Fatal("resolver BuildOptions.DisableServiceConfig = false, want true")
	}

	wrongBuilder := &liveResolverBuilder{
		scheme:     "resolver-live-wrong-authority",
		address:    serverAddress,
		serverName: "wrong.invalid",
	}
	resolver.Register(wrongBuilder)
	wrongConnection, err := grpcclient.New(
		grpcclient.DefaultConfig(wrongBuilder.scheme+":///service"),
		grpcclient.Options{TransportCredentials: clientCredentials},
	)
	if err != nil {
		t.Fatalf("grpcclient.New() wrong-authority construction error = %v", err)
	}
	t.Cleanup(func() { _ = wrongConnection.Close() })
	wrongCtx, cancelWrong := context.WithTimeout(t.Context(), time.Second)
	defer cancelWrong()
	_, err = healthgrpc.NewHealthClient(wrongConnection).Check(
		wrongCtx,
		&healthgrpc.HealthCheckRequest{},
	)
	if err == nil {
		t.Fatal("wrong resolver ServerName unexpectedly passed TLS verification")
	}
}

func startLiveTLSHealthServer(
	t *testing.T,
	serverCredentials credentials.TransportCredentials,
	handlerCalls *atomic.Int64,
) string {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(
		t.Context(),
		"tcp",
		net.JoinHostPort(nonLoopbackIPv4(t), "0"),
	)
	if err != nil {
		t.Fatalf("live TLS server Listen() error = %v", err)
	}
	// The bare address, not a passthrough target: this proof dials through the
	// resolver under test.
	return serveTestServer(
		t,
		listener,
		func(server *grpc.Server) {
			healthgrpc.RegisterHealthServer(server, committedUnavailableHealthServer{
				handlerCalls: handlerCalls,
			})
		},
		grpc.Creds(serverCredentials),
	)
}

func nonLoopbackIPv4(t *testing.T) string {
	t.Helper()

	addresses, err := net.InterfaceAddrs()
	if err != nil {
		t.Fatalf("net.InterfaceAddrs() error = %v", err)
	}
	for _, address := range addresses {
		ip, _, err := net.ParseCIDR(address.String())
		if err == nil && ip.To4() != nil && !ip.IsLoopback() {
			return ip.String()
		}
	}
	t.Fatal("no non-loopback IPv4 address is available for live proxy proof")
	return ""
}

type committedUnavailableHealthServer struct {
	healthgrpc.UnimplementedHealthServer

	handlerCalls *atomic.Int64
}

func (s committedUnavailableHealthServer) Check(
	ctx context.Context,
	_ *healthgrpc.HealthCheckRequest,
) (*healthgrpc.HealthCheckResponse, error) {
	s.handlerCalls.Add(1)
	if err := grpc.SendHeader(ctx, nil); err != nil {
		return nil, fmt.Errorf("commit response headers: %w", err)
	}
	return nil, fmt.Errorf(
		"return committed test failure: %w",
		status.Error(codes.Unavailable, "committed test failure"),
	)
}

type liveResolverBuilder struct {
	scheme               string
	address              string
	serverName           string
	serviceConfig        string
	disableServiceConfig atomic.Bool
}

func (b *liveResolverBuilder) Build( //nolint:ireturn // Implements resolver.Builder for the live selection proof.
	_ resolver.Target,
	connection resolver.ClientConn,
	options resolver.BuildOptions,
) (resolver.Resolver, error) {
	b.disableServiceConfig.Store(options.DisableServiceConfig)
	state := resolver.State{
		Addresses: []resolver.Address{{
			Addr:       b.address,
			ServerName: b.serverName,
		}},
	}
	if b.serviceConfig != "" {
		state.ServiceConfig = connection.ParseServiceConfig(b.serviceConfig)
		if state.ServiceConfig.Err != nil {
			return nil, fmt.Errorf("parse live resolver service config: %w", state.ServiceConfig.Err)
		}
	}
	if err := connection.UpdateState(state); err != nil {
		return nil, fmt.Errorf("publish live resolver state: %w", err)
	}
	return nopResolver{}, nil
}

func (b *liveResolverBuilder) Scheme() string {
	return b.scheme
}

type recordingProxy struct {
	address string
	accepts atomic.Int64
}

func startRecordingProxy(t *testing.T) *recordingProxy {
	t.Helper()

	listener := listenLoopback(t)
	proxy := &recordingProxy{
		address: listener.Addr().String(),
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			proxy.accepts.Add(1)
			go func() {
				_, _ = io.Copy(io.Discard, connection)
				_ = connection.Close()
			}()
		}
	}()
	t.Cleanup(func() {
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("proxy listener Close() error = %v", err)
		}
		<-done
	})
	return proxy
}

type proofBalancerBuilder struct {
	name   string
	builds *atomic.Int64
	picks  *atomic.Int64
}

func (b *proofBalancerBuilder) Build( //nolint:ireturn // Implements balancer.Builder for suppression proof.
	connection balancer.ClientConn,
	_ balancer.BuildOptions,
) balancer.Balancer {
	b.builds.Add(1)
	connection.UpdateState(balancer.State{
		ConnectivityState: connectivity.Ready,
		Picker:            proofPicker{picks: b.picks},
	})
	return proofBalancer{}
}

func (b *proofBalancerBuilder) Name() string {
	return b.name
}

type proofBalancer struct{}

func (proofBalancer) UpdateClientConnState(balancer.ClientConnState) error { return nil }
func (proofBalancer) ResolverError(error)                                  {}
func (proofBalancer) UpdateSubConnState(balancer.SubConn, balancer.SubConnState) {
}
func (proofBalancer) Close()    {}
func (proofBalancer) ExitIdle() {}

type proofPicker struct {
	picks *atomic.Int64
}

func (p proofPicker) Pick(balancer.PickInfo) (balancer.PickResult, error) {
	p.picks.Add(1)
	return balancer.PickResult{}, fmt.Errorf(
		"proof balancer picked: %w",
		status.Error(codes.Unavailable, "proof balancer picked"),
	)
}
