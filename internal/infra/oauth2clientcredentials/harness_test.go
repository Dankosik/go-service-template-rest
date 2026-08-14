package oauth2clientcredentials

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/httpclient"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/net/dns/dnsmessage"
)

const forbiddenCanary = "outbound-auth-forbidden-canary"

var fixedProviderTime = time.Date(2026, time.August, 12, 12, 0, 0, 0, time.UTC)

var suiteTestPKI *testPKI

func grpcTestPKI() *testPKI {
	if suiteTestPKI == nil || suiteTestPKI.pool == nil {
		panic("test PKI pool is not initialized")
	}
	return suiteTestPKI
}

type testPKI struct {
	root        *x509.Certificate
	rootKey     *ecdsa.PrivateKey
	pool        *x509.CertPool
	certificate map[string]tls.Certificate
}

func newTestPKI() (*testPKI, error) {
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate test root key: %w", err)
	}
	now := time.Now()
	rootTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "outbound-auth test root"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		return nil, fmt.Errorf("create test root certificate: %w", err)
	}
	root, err := x509.ParseCertificate(rootDER)
	if err != nil {
		return nil, fmt.Errorf("parse test root certificate: %w", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(root)
	return &testPKI{root: root, rootKey: rootKey, pool: pool, certificate: make(map[string]tls.Certificate)}, nil
}

func (p *testPKI) certificateFor(hostname string) (tls.Certificate, error) {
	if certificate, ok := p.certificate[hostname]; ok {
		return certificate, nil
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate %s test key: %w", hostname, err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 120))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate %s serial: %w", hostname, err)
	}
	now := time.Now()
	leafTemplate := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: hostname},
		DNSNames:     []string{hostname},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, p.root, &key.PublicKey, p.rootKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create %s certificate: %w", hostname, err)
	}
	certificate := tls.Certificate{Certificate: [][]byte{leafDER, p.root.Raw}, PrivateKey: key}
	p.certificate[hostname] = certificate
	return certificate, nil
}

func privateTestAddress(t *testing.T) netip.Addr {
	t.Helper()
	interfaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("net.Interfaces() error = %v", err)
	}
	for _, networkInterface := range interfaces {
		if networkInterface.Flags&net.FlagUp == 0 || networkInterface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, addressErr := networkInterface.Addrs()
		if addressErr != nil {
			continue
		}
		for _, candidate := range addresses {
			prefix, parseErr := netip.ParsePrefix(candidate.String())
			if parseErr == nil && prefix.Addr().Is4() && prefix.Addr().IsPrivate() {
				return prefix.Addr()
			}
		}
	}
	t.Fatal("no bindable private IPv4 address available for outbound-auth HTTP proof")
	return netip.Addr{}
}

func startPrivateHTTPSTestServer(
	t *testing.T,
	address netip.Addr,
	hostname string,
	handler http.Handler,
) string {
	t.Helper()
	certificate, err := grpcTestPKI().certificateFor(hostname)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", net.JoinHostPort(address.String(), "0"))
	if err != nil {
		t.Fatalf("listen on %s: %v", address, err)
	}
	tlsListener := tls.NewListener(listener, &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: time.Second}
	done := make(chan error, 1)
	go func() { done <- server.Serve(tlsListener) }()
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			t.Errorf("Server.Shutdown() error = %v", err)
		}
		if err := <-done; err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Server.Serve() error = %v", err)
		}
	})
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split TLS listener address: %v", err)
	}
	return (&url.URL{Scheme: "https", Host: net.JoinHostPort(hostname, port)}).String()
}

func installPrivateTestResolver(t *testing.T, hosts map[string]netip.Addr) {
	t.Helper()
	packetConn, err := (&net.ListenConfig{}).ListenPacket(t.Context(), "udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("start test DNS server: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		buffer := make([]byte, 1232)
		for {
			size, peer, readErr := packetConn.ReadFrom(buffer)
			if readErr != nil {
				done <- readErr
				return
			}
			response, responseErr := privateDNSResponse(buffer[:size], hosts)
			if responseErr != nil {
				done <- responseErr
				return
			}
			if _, writeErr := packetConn.WriteTo(response, peer); writeErr != nil {
				done <- writeErr
				return
			}
		}
	}()
	previousResolver := net.DefaultResolver
	net.DefaultResolver = &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "udp", packetConn.LocalAddr().String())
	}}
	t.Cleanup(func() {
		net.DefaultResolver = previousResolver
		if err := packetConn.Close(); err != nil {
			t.Errorf("close test DNS server: %v", err)
		}
		if err := <-done; err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("test DNS server error = %v", err)
		}
	})
}

func privateDNSResponse(query []byte, hosts map[string]netip.Addr) ([]byte, error) {
	var parser dnsmessage.Parser
	header, err := parser.Start(query)
	if err != nil {
		return nil, fmt.Errorf("parse DNS header: %w", err)
	}
	question, err := parser.Question()
	if err != nil {
		return nil, fmt.Errorf("parse DNS question: %w", err)
	}
	builder := dnsmessage.NewBuilder(nil, dnsmessage.Header{ID: header.ID, Response: true, RecursionDesired: header.RecursionDesired, RecursionAvailable: true})
	builder.EnableCompression()
	if err := builder.StartQuestions(); err != nil {
		return nil, fmt.Errorf("start DNS questions: %w", err)
	}
	if err := builder.Question(question); err != nil {
		return nil, fmt.Errorf("write DNS question: %w", err)
	}
	if err := builder.StartAnswers(); err != nil {
		return nil, fmt.Errorf("start DNS answers: %w", err)
	}
	hostname := strings.ToLower(strings.TrimSuffix(question.Name.String(), "."))
	if address, ok := hosts[hostname]; ok && question.Type == dnsmessage.TypeA {
		if err := builder.AResource(dnsmessage.ResourceHeader{Name: question.Name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET, TTL: 1}, dnsmessage.AResource{A: address.As4()}); err != nil {
			return nil, fmt.Errorf("write DNS A answer: %w", err)
		}
	}
	response, err := builder.Finish()
	if err != nil {
		return nil, fmt.Errorf("finish DNS response: %w", err)
	}
	return response, nil
}

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}

type closeTrackingBody struct {
	io.Reader

	closed *bool
}

func (b closeTrackingBody) Close() error {
	*b.closed = true
	return nil
}

func validTestConfig() Config {
	return Config{
		DependencyName:       "payments",
		ClientID:             " client:id+ ",
		ClientSecret:         " secret:p@ss+ ",
		ClientAuthentication: clientAuthenticationBasic,
		TokenEndpoint:        "https://auth.example.com/oauth/token",
		TokenTargetClass:     httpclient.ExternalHTTPS,
		Scopes:               []string{"payments.read", "payments.write"},
		Resource:             "https://payments.example.com",
		ResourceAuthority:    "https://payments.example.com",
		AcquisitionTimeout:   2 * time.Second,
	}
}

func providerResponse(status int, mediaType, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{mediaType}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func assertFailureClass(t *testing.T, err error, want FailureClass) {
	t.Helper()
	got, ok := FailureClassOf(err)
	if !ok || got != want {
		t.Fatalf("FailureClassOf(%v) = %q, %t; want %q, true", err, got, ok, want)
	}
	if strings.Contains(err.Error(), forbiddenCanary) {
		t.Fatalf("error disclosed canary: %v", err)
	}
}

func canceledContext(cause error) context.Context {
	if errors.Is(cause, context.DeadlineExceeded) {
		ctx, cancel := context.WithDeadline(context.Background(), fixedProviderTime.Add(-time.Second))
		cancel()
		return ctx
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

type movableClock struct {
	mu  sync.Mutex
	now time.Time
}

func newMovableClock(now time.Time) *movableClock {
	return &movableClock{now: now}
}

func (c *movableClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *movableClock) Advance(delta time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(delta)
	c.mu.Unlock()
}

type acquisitionStep struct {
	token              accessToken
	err                error
	entered            chan struct{}
	release            chan struct{}
	canceled           chan struct{}
	waitAfterCancel    chan struct{}
	ignoreCancellation bool
}

type scriptedAcquirer struct {
	mu    sync.Mutex
	steps []acquisitionStep
	calls int
}

func (s *scriptedAcquirer) acquire(ctx context.Context) (accessToken, error) {
	s.mu.Lock()
	index := s.calls
	s.calls++
	if index >= len(s.steps) {
		s.mu.Unlock()
		return accessToken{}, failure(FailureProviderUnavailable)
	}
	step := s.steps[index]
	s.mu.Unlock()

	if step.entered != nil {
		close(step.entered)
	}
	if step.release == nil {
		return step.token, step.err
	}
	if step.ignoreCancellation {
		<-step.release
		return step.token, step.err
	}
	select {
	case <-step.release:
		return step.token, step.err
	case <-ctx.Done():
		if step.canceled != nil {
			close(step.canceled)
		}
		if step.waitAfterCancel != nil {
			<-step.waitAfterCancel
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return accessToken{}, failure(FailureProviderTimeout)
		}
		return accessToken{}, failure(FailureProviderUnavailable)
	}
}

func (s *scriptedAcquirer) Calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

type testClientOptions struct {
	now           func() time.Time
	acquire       acquireToken
	jitter        func(time.Duration) time.Duration
	closeIdle     func()
	meterProvider metric.MeterProvider
	log           *slog.Logger
}

func requireTestClient(t *testing.T, cfg Config, options testClientOptions) *Client {
	t.Helper()
	if options.now == nil {
		options.now = time.Now
	}
	if options.jitter == nil {
		options.jitter = func(time.Duration) time.Duration { return 0 }
	}
	client, err := newClient(
		cfg,
		options.meterProvider,
		options.log,
		options.now,
		options.acquire,
		options.closeIdle,
	)
	if err != nil {
		t.Fatalf("newClient() error = %v", err)
	}
	client.jitter = options.jitter
	t.Cleanup(func() {
		if err := client.Close(context.Background()); err != nil {
			t.Errorf("Client.Close() cleanup error = %v", err)
		}
	})
	return client
}
