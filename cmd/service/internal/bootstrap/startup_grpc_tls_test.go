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
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

// TestGRPCServerServesARotatedCertificateWithoutARestart is the claim the
// reloader exists for, driven through the composition root and a real handshake:
// the process keeps running, the files change, and the next caller is served the
// new certificate.
//
// A test that only asserted the reloader returned the new pair would pass with
// the credentials never wired to it.
func TestGRPCServerServesARotatedCertificateWithoutARestart(t *testing.T) {
	t.Parallel()

	authority := newTestAuthority(t)
	directory := t.TempDir()
	settings := writeTestPair(t, directory, authority, 1)
	address := startTLSGRPCRuntime(t, settings)

	var served atomic.Int64
	if err := checkHealthOver(t, address, capturingClientTLS(&served)); err != nil {
		t.Fatalf("health check over the original certificate: %v", err)
	}
	if got := served.Load(); got != 1 {
		t.Fatalf("served certificate serial = %d, want the original 1", got)
	}

	writeTestPair(t, directory, authority, 2)

	if err := checkHealthOver(t, address, capturingClientTLS(&served)); err != nil {
		t.Fatalf("health check over the rotated certificate: %v", err)
	}
	if got := served.Load(); got != 2 {
		t.Fatalf("served certificate serial = %d, want the rotated 2; the pair on disk was not picked up", got)
	}
}

// TestGRPCServerRefusesTLS12 pins the version floor at the boundary a caller
// actually meets.
func TestGRPCServerRefusesTLS12(t *testing.T) {
	t.Parallel()

	authority := newTestAuthority(t)
	settings := writeTestPair(t, t.TempDir(), authority, 1)
	address := startTLSGRPCRuntime(t, settings)

	var served atomic.Int64
	client := capturingClientTLS(&served)
	client.MaxVersion = tls.VersionTLS12
	if err := checkHealthOver(t, address, client); err == nil {
		t.Fatal("a TLS 1.2 caller completed a health check, but this listener floors at 1.3")
	}
}

// TestGRPCServerMutualTLS covers both halves of the client-certificate decision,
// because a listener that accepted an unauthenticated caller would still pass a
// test that only drove the authenticated one.
func TestGRPCServerMutualTLS(t *testing.T) {
	t.Parallel()

	authority := newTestAuthority(t)
	directory := t.TempDir()
	settings := writeTestPair(t, directory, authority, 1)
	settings.ClientCAFile = writeTestFile(t, directory, "clients.pem", authority.pem)
	address := startTLSGRPCRuntime(t, settings)

	var served atomic.Int64
	t.Run("without a client certificate", func(t *testing.T) {
		t.Parallel()

		if err := checkHealthOver(t, address, capturingClientTLS(&served)); err == nil {
			t.Fatal("an unauthenticated caller completed a health check against a mutual-TLS listener")
		}
	})

	t.Run("with a client certificate", func(t *testing.T) {
		t.Parallel()

		certificate, key := authority.issue(t, 2, x509.ExtKeyUsageClientAuth)
		pair, err := tls.X509KeyPair(certificate, key)
		if err != nil {
			t.Fatalf("tls.X509KeyPair() error = %v", err)
		}
		client := capturingClientTLS(&served)
		client.Certificates = []tls.Certificate{pair}
		if err := checkHealthOver(t, address, client); err != nil {
			t.Fatalf("health check with a trusted client certificate: %v", err)
		}
	})
}

// TestGRPCServerTLSRejectsAnEmptyClientCAFile keeps a misconfigured trust root a
// startup failure. Accepting it as an empty pool would reject every caller at
// handshake time instead, which reads as a client problem.
func TestGRPCServerTLSRejectsAnEmptyClientCAFile(t *testing.T) {
	t.Parallel()

	authority := newTestAuthority(t)
	directory := t.TempDir()
	settings := writeTestPair(t, directory, authority, 1)
	settings.ClientCAFile = writeTestFile(t, directory, "empty.pem", []byte("not a certificate\n"))

	built, err := grpcServerTLS(settings, slog.New(slog.DiscardHandler))
	if built != nil {
		t.Fatal("grpcServerTLS() returned settings for a CA file holding no certificate")
	}
	if err == nil || !strings.Contains(err.Error(), "contains no certificate") {
		t.Fatalf("grpcServerTLS() error = %v, want a refusal naming the empty CA file", err)
	}
}

// TestCertificateReloaderKeepsTheLastGoodPair drives the two ways a rotation is
// observed mid-write. Both must serve rather than fail: the alternative is an
// outage caused by the order a tool happened to write two files in.
func TestCertificateReloaderKeepsTheLastGoodPair(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name    string
		corrupt func(t *testing.T, certFile, keyFile string, authority testAuthority)
	}{
		{
			name: "half-written rotation",
			corrupt: func(t *testing.T, certFile, _ string, authority testAuthority) {
				t.Helper()
				// The certificate half of a new pair, beside the old key.
				rotated, _ := authority.issue(t, 2, x509.ExtKeyUsageServerAuth)
				overwriteTestFile(t, certFile, rotated)
			},
		},
		{
			name: "deleted key",
			corrupt: func(t *testing.T, _, keyFile string, _ testAuthority) {
				t.Helper()
				if err := os.Remove(keyFile); err != nil {
					t.Fatalf("os.Remove(%q) error = %v", keyFile, err)
				}
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			authority := newTestAuthority(t)
			directory := t.TempDir()
			settings := writeTestPair(t, directory, authority, 1)

			reloader, err := newCertificateReloader(
				settings.CertFile,
				settings.KeyFile,
				slog.New(slog.DiscardHandler),
			)
			if err != nil {
				t.Fatalf("newCertificateReloader() error = %v", err)
			}
			original, err := reloader.certificate(nil)
			if err != nil {
				t.Fatalf("certificate() error = %v", err)
			}

			testCase.corrupt(t, settings.CertFile, settings.KeyFile, authority)

			served, err := reloader.certificate(nil)
			if err != nil {
				t.Fatalf("certificate() after %s error = %v, want the last good pair", testCase.name, err)
			}
			if served != original {
				t.Fatalf("certificate() after %s returned a different pair, want the last good one", testCase.name)
			}
		})
	}
}

// TestCertificateReloaderRetriesAFinishedRotation is the other half of the case
// above: keeping the last good pair must not mean keeping it forever. The
// rejected stamp is deliberately not recorded, so the completed write is picked
// up on the next handshake.
func TestCertificateReloaderRetriesAFinishedRotation(t *testing.T) {
	t.Parallel()

	authority := newTestAuthority(t)
	directory := t.TempDir()
	settings := writeTestPair(t, directory, authority, 1)

	reloader, err := newCertificateReloader(settings.CertFile, settings.KeyFile, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("newCertificateReloader() error = %v", err)
	}

	rotatedCert, rotatedKey := authority.issue(t, 2, x509.ExtKeyUsageServerAuth)
	overwriteTestFile(t, settings.CertFile, rotatedCert)
	if _, err := reloader.certificate(nil); err != nil {
		t.Fatalf("certificate() error = %v", err)
	}
	overwriteTestFile(t, settings.KeyFile, rotatedKey)

	served, err := reloader.certificate(nil)
	if err != nil {
		t.Fatalf("certificate() error = %v", err)
	}
	if serial := leafSerial(t, served.Certificate[0]); serial != 2 {
		t.Fatalf("certificate() serial = %d, want the completed rotation 2", serial)
	}
}

// startTLSGRPCRuntime composes the runtime the way cmd/service does and serves it
// on loopback, so every assertion above crosses the credentials newGRPCRuntime
// actually built.
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
	server.MarkServing()

	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		_ = server.Close()
		t.Fatalf("net.Listen() error = %v", err)
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	t.Cleanup(func() {
		_ = server.Close()
		if err := <-serveDone; err != nil {
			t.Errorf("Server.Serve() error = %v", err)
		}
	})
	return listener.Addr().String()
}

// checkHealthOver performs one standard health Check, which is what forces the
// handshake these tests are about. Its error is returned rather than fataled,
// because a refused handshake is the expected outcome in half of them.
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
	return err //nolint:wrapcheck // The handshake outcome is this helper's whole result.
}

// capturingClientTLS records the serial of the certificate the listener
// presented. Server verification is skipped because identity is not what these
// tests assert — which certificate was served is.
func capturingClientTLS(served *atomic.Int64) *tls.Config {
	return &tls.Config{
		//nolint:gosec // Test client; the peer certificate is inspected below rather than trusted.
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			leaf, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				return err //nolint:wrapcheck // Reported verbatim into the handshake failure.
			}
			served.Store(leaf.SerialNumber.Int64())
			return nil
		},
	}
}

// testAuthority signs the leaves these tests present. One authority issues both
// halves of the mutual-TLS case, so the trust root under test is the one the
// listener was configured with.
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

// issue returns one PEM leaf and its key. The serial is the identity the
// rotation assertions read back off the wire.
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

// writeTestPair issues one server pair and puts it at fixed names inside
// directory, which is what lets a rotation overwrite it in place the way a
// renewal does.
func writeTestPair(t *testing.T, directory string, authority testAuthority, serial int64) config.GRPCTLSConfig {
	t.Helper()

	certificate, key := authority.issue(t, serial, x509.ExtKeyUsageServerAuth)
	return config.GRPCTLSConfig{
		CertFile:     writeTestFile(t, directory, "service.crt", certificate),
		KeyFile:      writeTestFile(t, directory, "service.key", key),
		ClientCAFile: "",
	}
}

func writeTestFile(t *testing.T, directory, name string, contents []byte) string {
	t.Helper()

	path := filepath.Join(directory, name)
	overwriteTestFile(t, path, contents)
	return path
}

// overwriteTestFile advances the modification time as well as the bytes. A
// rotation inside one filesystem timestamp tick would otherwise be invisible to
// the reloader's stamp, and the test would be asserting the clock's resolution
// rather than the reload.
func overwriteTestFile(t *testing.T, path string, contents []byte) {
	t.Helper()

	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
	advanced := time.Now().Add(time.Second)
	if err := os.Chtimes(path, advanced, advanced); err != nil {
		t.Fatalf("os.Chtimes(%q) error = %v", path, err)
	}
}

func leafSerial(t *testing.T, der []byte) int64 {
	t.Helper()

	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("x509.ParseCertificate() error = %v", err)
	}
	return leaf.SerialNumber.Int64()
}
