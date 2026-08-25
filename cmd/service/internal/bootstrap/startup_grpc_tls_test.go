package bootstrap

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

func TestGRPCServerTLS13AndMutualTLS(t *testing.T) {
	authority := newTestAuthority(t)
	directory := t.TempDir()
	settings := writeTestPair(t, directory, authority, 1)
	settings.ClientCAFile = writeTestFile(t, directory, "clients.pem", authority.pem)
	address := startTLSGRPCRuntime(t, settings)

	client := &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // Test verifies the presented pair separately from hostname trust.
		MaxVersion:         tls.VersionTLS12,
	}
	if err := checkHealthOver(t, address, client); err == nil {
		t.Fatal("TLS 1.2 caller reached the listener")
	}

	client = &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // The issuing CA is tested by the server side.
		MinVersion:         tls.VersionTLS13,
	}
	if err := checkHealthOver(t, address, client); err == nil {
		t.Fatal("caller without a client certificate reached the mutual-TLS listener")
	}

	certificate, key := authority.issue(t, 2, x509.ExtKeyUsageClientAuth)
	pair, err := tls.X509KeyPair(certificate, key)
	if err != nil {
		t.Fatalf("tls.X509KeyPair() error = %v", err)
	}
	client.Certificates = []tls.Certificate{pair}
	if err := checkHealthOver(t, address, client); err != nil {
		t.Fatalf("trusted mutual-TLS health check: %v", err)
	}
}

func TestGRPCServerTLSRejectsAnEmptyClientCAFile(t *testing.T) {
	authority := newTestAuthority(t)
	directory := t.TempDir()
	settings := writeTestPair(t, directory, authority, 1)
	settings.ClientCAFile = writeTestFile(t, directory, "empty.pem", []byte("not a certificate\n"))
	built, err := grpcServerTLS(settings)
	if built != nil || err == nil || !strings.Contains(err.Error(), "contains no certificate") {
		t.Fatalf("grpcServerTLS() = (%v, %v), want empty CA refusal", built, err)
	}
}

func startTLSGRPCRuntime(t *testing.T, settings config.GRPCTLSConfig) string {
	t.Helper()
	cfg := grpcRuntimeTestConfig()
	cfg.GRPC.Server.TransportSecurity = "tls"
	cfg.GRPC.Server.TLS = settings
	server, err := newGRPCRuntime(
		cfg,
		slog.New(slog.DiscardHandler),
		telemetry.New(),
		nil,
		grpcRuntimeBindings{},
	)
	if err != nil {
		t.Fatalf("newGRPCRuntime() error = %v", err)
	}
	server.SetServing(true)
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() {
		_ = server.Close()
		if err := <-done; err != nil {
			t.Errorf("Serve() error = %v", err)
		}
	})
	return listener.Addr().String()
}

func checkHealthOver(t *testing.T, address string, client *tls.Config) error {
	t.Helper()
	connection, err := grpc.NewClient(address, grpc.WithTransportCredentials(credentials.NewTLS(client)))
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	defer func() { _ = connection.Close() }()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	_, err = healthgrpc.NewHealthClient(connection).Check(ctx, &healthgrpc.HealthCheckRequest{})
	return err //nolint:wrapcheck // Handshake outcome is the assertion.
}

type testAuthority struct {
	certificate *x509.Certificate
	key         ed25519.PrivateKey
	pem         []byte
}

func newTestAuthority(t *testing.T) testAuthority {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey() error = %v", err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1000),
		NotBefore:             time.Unix(0, 0),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, public, private)
	if err != nil {
		t.Fatalf("x509.CreateCertificate() error = %v", err)
	}
	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("x509.ParseCertificate() error = %v", err)
	}
	return testAuthority{
		certificate: parsed,
		key:         private,
		pem:         pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
	}
}

func (a testAuthority) issue(t *testing.T, serial int64, usage x509.ExtKeyUsage) ([]byte, []byte) {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey() error = %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		NotBefore:    time.Unix(0, 0),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{usage},
		DNSNames:     []string{"example.com"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, a.certificate, public, a.key)
	if err != nil {
		t.Fatalf("x509.CreateCertificate() error = %v", err)
	}
	encodedKey, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		t.Fatalf("x509.MarshalPKCS8PrivateKey() error = %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encodedKey})
}

func writeTestPair(t *testing.T, directory string, authority testAuthority, serial int64) config.GRPCTLSConfig {
	t.Helper()
	certificate, key := authority.issue(t, serial, x509.ExtKeyUsageServerAuth)
	return config.GRPCTLSConfig{
		CertFile: writeTestFile(t, directory, "service.crt", certificate),
		KeyFile:  writeTestFile(t, directory, "service.key", key),
	}
}

func writeTestFile(t *testing.T, directory, name string, contents []byte) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
	return path
}
