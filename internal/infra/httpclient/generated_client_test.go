package httpclient

import (
	// profile:outbound-auth-http:start
	"context"
	// profile:outbound-auth-http:end
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"

	// profile:outbound-auth-http:start
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"

	// profile:outbound-auth-http:end
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	// profile:outbound-auth-http:start
	"sync/atomic"
	// profile:outbound-auth-http:end
	"testing"
	"time"

	// profile:outbound-auth-http:start
	"golang.org/x/net/dns/dnsmessage"
	// profile:outbound-auth-http:end
)

func TestGeneratedClientComposition(t *testing.T) {
	packageDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	repositoryRoot := filepath.Clean(filepath.Join(packageDirectory, "..", "..", ".."))
	// The child package must stay below the module root so Go internal-package
	// visibility and the generated import path match a real consumer.
	generatedDirectory, err := os.MkdirTemp( //nolint:usetesting // t.TempDir cannot select an in-module parent.
		packageDirectory,
		"generatedclientcheck-",
	)
	if err != nil {
		t.Fatalf("os.MkdirTemp() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(generatedDirectory); err != nil {
			t.Errorf("remove generated client fixture: %v", err)
		}
	})

	generatedPath := filepath.Join(generatedDirectory, "client.gen.go")
	generate := exec.CommandContext(
		t.Context(),
		"bash",
		filepath.Join(repositoryRoot, "scripts", "run-go-tool.sh"),
		"oapi-codegen",
		"-generate",
		"types,client",
		"-package",
		"generatedclient",
		"-o",
		generatedPath,
		filepath.Join(packageDirectory, "testdata", "generated-client.yaml"),
	)
	generate.Dir = repositoryRoot
	if output, err := generate.CombinedOutput(); err != nil {
		t.Fatalf("generate pinned oapi-codegen client: %v\n%s", err, output)
	}

	const consumerTest = `package generatedclient

import (
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/httpclient"
)

func TestGeneratedClientUsesBoundedHTTPClient(t *testing.T) {
	cfg := httpclient.Config{
		DependencyName:         "generated-fixture",
		BaseURL:                "https://localhost:443",
		TargetClass:            httpclient.ExternalHTTPS,
		RequestTimeout:         time.Second,
		ResponseHeaderTimeout:  time.Second,
		MaxResponseHeaderBytes: 16 << 10,
		MaxResponseBodyBytes:   1 << 20,
		MaxConnsPerHost:        2,
		Propagation:            httpclient.PropagationTrustedService,
	}
	bounded, err := httpclient.New(cfg, nil)
	if err != nil {
		t.Fatalf("httpclient.New() error = %v", err)
	}
	t.Cleanup(bounded.CloseIdleConnections)

	var _ HttpRequestDoer = bounded
	client, err := NewClient(bounded.BaseURL(), WithHTTPClient(bounded))
	if err != nil {
		t.Fatalf("generated NewClient() error = %v", err)
	}
	_, err = client.GetHealth(t.Context())
	if !errors.Is(err, httpclient.ErrTargetDenied) {
		t.Fatalf("generated GetHealth() error = %v, want shared client address denial", err)
	}
}
`
	consumerPath := filepath.Join(generatedDirectory, "composition_test.go")
	if err := os.WriteFile(consumerPath, []byte(consumerTest), 0o600); err != nil {
		t.Fatalf("write generated consumer test: %v", err)
	}

	relativePackage, err := filepath.Rel(repositoryRoot, generatedDirectory)
	if err != nil {
		t.Fatalf("relative generated package path: %v", err)
	}
	testCommand := exec.CommandContext(
		t.Context(),
		"go",
		"test",
		"-vet=off",
		"./"+filepath.ToSlash(relativePackage),
		"-run",
		"^TestGeneratedClientUsesBoundedHTTPClient$",
		"-count=1",
	)
	testCommand.Dir = repositoryRoot
	if output, err := testCommand.CombinedOutput(); err != nil {
		t.Fatalf("compile and run generated client composition: %v\n%s", err, output)
	}

	entries, err := os.ReadDir(generatedDirectory)
	if err != nil {
		t.Fatalf("read generated directory: %v", err)
	}
	if len(entries) != 2 {
		var names []string
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("generated fixture files = %s, want generated code and one consumer test", strings.Join(names, ", "))
	}
}

// profile:outbound-auth-http:start

func TestGeneratedClientUsesAuthenticatedDoer(t *testing.T) {
	const (
		tokenHost    = "token.generated.internal"
		resourceHost = "resource.generated.internal"
	)
	address := generatedClientPrivateAddress(t)
	pki := newGeneratedClientPKI(t)
	dnsAddress := startGeneratedClientDNS(t, map[string]netip.Addr{tokenHost: address, resourceHost: address})
	var tokenCalls atomic.Int32
	tokenURL := startGeneratedClientTLSServer(t, address, tokenHost, pki, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		tokenCalls.Add(1)
		if username, password, ok := request.BasicAuth(); !ok || username != "generated-client" || password != "generated-secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"generated-operation-token","token_type":"Bearer","expires_in":60}`)
	})) + "/oauth/token"
	var resourceCalls atomic.Int32
	resourceURL := startGeneratedClientTLSServer(t, address, resourceHost, pki, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		resourceCalls.Add(1)
		if request.Header.Get("Authorization") != "Bearer generated-operation-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	caPath := filepath.Join(t.TempDir(), "outbound-auth-test-ca.pem")
	if err := os.WriteFile(caPath, pki.rootPEM, 0o600); err != nil {
		t.Fatalf("write generated child CA: %v", err)
	}

	packageDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	repositoryRoot := filepath.Clean(filepath.Join(packageDirectory, "..", "..", ".."))
	generatedDirectory, err := os.MkdirTemp(packageDirectory, "generatedoauthclientcheck-") //nolint:usetesting // child must remain inside the module.
	if err != nil {
		t.Fatalf("os.MkdirTemp() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(generatedDirectory); err != nil {
			t.Errorf("remove generated OAuth client fixture: %v", err)
		}
	})
	generatedPath := filepath.Join(generatedDirectory, "client.gen.go")
	generate := exec.CommandContext(
		t.Context(),
		"bash",
		filepath.Join(repositoryRoot, "scripts", "run-go-tool.sh"),
		"oapi-codegen",
		"-generate",
		"types,client",
		"-package",
		"generatedclient",
		"-o",
		generatedPath,
		filepath.Join(packageDirectory, "testdata", "generated-client.yaml"),
	)
	generate.Dir = repositoryRoot
	if output, err := generate.CombinedOutput(); err != nil {
		t.Fatalf("generate pinned authenticated client: %v\n%s", err, output)
	}

	//nolint:dupword // Generated source must replace and then restore net.DefaultResolver.
	const consumerTest = `package generatedclient

import (
	"context"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/httpclient"
	"github.com/example/go-service-template-rest/internal/infra/oauth2clientcredentials"
)

func TestMain(m *testing.M) {
	rootPEM, err := os.ReadFile(os.Getenv("OUTBOUND_AUTH_TEST_CA"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "read fallback root: %v\n", err)
		os.Exit(1)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(rootPEM) {
		fmt.Fprintln(os.Stderr, "parse fallback root: no certificate")
		os.Exit(1)
	}
	x509.SetFallbackRoots(pool)
	previousResolver := net.DefaultResolver
	net.DefaultResolver = &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "udp", os.Getenv("OUTBOUND_AUTH_TEST_DNS"))
	}}
	code := m.Run()
	net.DefaultResolver = previousResolver
	os.Exit(code)
}

func TestGeneratedClientUsesAuthenticatedDoer(t *testing.T) {
	cfg := oauth2clientcredentials.Config{
		ClientID:               "generated-client",
		ClientSecret:           "generated-secret",
		TokenURL:               os.Getenv("OUTBOUND_AUTH_TEST_TOKEN_URL"),
		TokenTargetClass:       httpclient.PrivateHTTPS,
		TokenPrivateHostSuffix: "generated.internal",
	}
	owner, err := oauth2clientcredentials.New(cfg)
	if err != nil {
		t.Fatalf("oauth2clientcredentials.New() error = %v", err)
	}
	var resource *httpclient.Client
	t.Cleanup(func() {
		if err := owner.Close(context.Background()); err != nil {
			t.Errorf("oauth owner close: %v", err)
		}
		if resource != nil {
			resource.CloseIdleConnections()
		}
	})
	resource, err = httpclient.New(httpclient.Config{
		DependencyName: "generated-fixture",
		BaseURL: os.Getenv("OUTBOUND_AUTH_TEST_RESOURCE_URL"),
		TargetClass: httpclient.PrivateHTTPS,
		PrivateHostSuffix: "generated.internal",
		RequestTimeout: time.Second,
		ResponseHeaderTimeout: time.Second,
		MaxResponseHeaderBytes: 16 << 10,
		MaxResponseBodyBytes: 1 << 20,
		MaxConnsPerHost: 1,
	}, nil)
	if err != nil {
		t.Fatalf("httpclient.New() error = %v", err)
	}
	authenticated, err := owner.HTTP(resource)
	if err != nil {
		t.Fatalf("HTTP() error = %v", err)
	}
	var _ HttpRequestDoer = authenticated
	client, err := NewClient(resource.BaseURL(), WithHTTPClient(authenticated))
	if err != nil {
		t.Fatalf("generated NewClient() error = %v", err)
	}
	response, err := client.GetHealth(t.Context())
	if err != nil {
		t.Fatalf("generated GetHealth() error = %v", err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	if err := response.Body.Close(); err != nil {
		t.Fatalf("close generated response: %v", err)
	}
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("generated status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}
}
`
	consumerPath := filepath.Join(generatedDirectory, "composition_test.go")
	if err := os.WriteFile(consumerPath, []byte(consumerTest), 0o600); err != nil {
		t.Fatalf("write authenticated generated consumer: %v", err)
	}
	relativePackage, err := filepath.Rel(repositoryRoot, generatedDirectory)
	if err != nil {
		t.Fatalf("relative generated OAuth package path: %v", err)
	}
	testCommand := exec.CommandContext(
		t.Context(),
		"go",
		"test",
		"-vet=off",
		"./"+filepath.ToSlash(relativePackage),
		"-run",
		"^TestGeneratedClientUsesAuthenticatedDoer$",
		"-count=1",
	)
	testCommand.Dir = repositoryRoot
	testCommand.Env = append(os.Environ(),
		"GODEBUG=x509usefallbackroots=1",
		"OUTBOUND_AUTH_TEST_CA="+caPath,
		"OUTBOUND_AUTH_TEST_DNS="+dnsAddress,
		"OUTBOUND_AUTH_TEST_TOKEN_URL="+tokenURL,
		"OUTBOUND_AUTH_TEST_RESOURCE_URL="+resourceURL,
	)
	if output, err := testCommand.CombinedOutput(); err != nil {
		t.Fatalf("compile and run authenticated generated client: %v\n%s", err, output)
	}
	if got := tokenCalls.Load(); got != 1 {
		t.Fatalf("generated provider calls = %d, want 1", got)
	}
	if got := resourceCalls.Load(); got != 1 {
		t.Fatalf("generated resource calls = %d, want 1", got)
	}
}

// profile:outbound-auth-http:end

type generatedClientPKI struct {
	root    *x509.Certificate
	rootKey *ecdsa.PrivateKey
	rootPEM []byte
}

func newGeneratedClientPKI(t *testing.T) *generatedClientPKI {
	t.Helper()
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate root key: %v", err)
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "generated client test root"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, template, template, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatalf("create root certificate: %v", err)
	}
	root, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatalf("parse root certificate: %v", err)
	}
	return &generatedClientPKI{root: root, rootKey: rootKey, rootPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootDER})}
}

func (p *generatedClientPKI) certificate(t *testing.T, hostname string) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate leaf key: %v", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 120))
	if err != nil {
		t.Fatalf("generate leaf serial: %v", err)
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: hostname},
		DNSNames:     []string{hostname},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, template, p.root, &key.PublicKey, p.rootKey)
	if err != nil {
		t.Fatalf("create leaf certificate: %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{leafDER, p.root.Raw}, PrivateKey: key}
}

// profile:outbound-auth-http:start

func generatedClientPrivateAddress(t *testing.T) netip.Addr {
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
	t.Fatal("no bindable private IPv4 address available for generated OAuth client proof")
	return netip.Addr{}
}

func startGeneratedClientTLSServer(
	t *testing.T,
	address netip.Addr,
	hostname string,
	pki *generatedClientPKI,
	handler http.Handler,
) string {
	t.Helper()
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", net.JoinHostPort(address.String(), "0"))
	if err != nil {
		t.Fatalf("listen for generated TLS server: %v", err)
	}
	tlsListener := tls.NewListener(listener, &tls.Config{Certificates: []tls.Certificate{pki.certificate(t, hostname)}, MinVersion: tls.VersionTLS12})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: time.Second}
	done := make(chan error, 1)
	go func() { done <- server.Serve(tlsListener) }()
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			t.Errorf("generated TLS Server.Shutdown() error = %v", err)
		}
		if err := <-done; err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("generated TLS Server.Serve() error = %v", err)
		}
	})
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split generated TLS listener: %v", err)
	}
	return "https://" + net.JoinHostPort(hostname, port)
}

func startGeneratedClientDNS(t *testing.T, hosts map[string]netip.Addr) string {
	t.Helper()
	packetConn, err := (&net.ListenConfig{}).ListenPacket(t.Context(), "udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("start generated client DNS: %v", err)
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
			response, responseErr := generatedClientDNSResponse(buffer[:size], hosts)
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
	t.Cleanup(func() {
		if err := packetConn.Close(); err != nil {
			t.Errorf("close generated client DNS: %v", err)
		}
		if err := <-done; err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("generated client DNS error = %v", err)
		}
	})
	return packetConn.LocalAddr().String()
}

func generatedClientDNSResponse(query []byte, hosts map[string]netip.Addr) ([]byte, error) {
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
			return nil, fmt.Errorf("write DNS answer: %w", err)
		}
	}
	response, err := builder.Finish()
	if err != nil {
		return nil, fmt.Errorf("finish DNS response: %w", err)
	}
	return response, nil
}

// profile:outbound-auth-http:end
