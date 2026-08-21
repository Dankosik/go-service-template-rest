package bootstrap

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"

	"github.com/example/go-service-template-rest/internal/config"
)

func grpcServerTLS(settings config.GRPCTLSConfig) (*tls.Config, error) {
	pair, err := tls.LoadX509KeyPair(settings.CertFile, settings.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load gRPC server TLS certificate: %w", err)
	}
	built := &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{pair},
	}
	if settings.ClientCAFile == "" {
		return built, nil
	}
	authorities, err := clientCertificateAuthorities(settings.ClientCAFile)
	if err != nil {
		return nil, err
	}
	built.ClientCAs = authorities
	built.ClientAuth = tls.RequireAndVerifyClientCert
	return built, nil
}

func clientCertificateAuthorities(name string) (*x509.CertPool, error) {
	root, err := os.OpenRoot(filepath.Dir(name))
	if err != nil {
		return nil, fmt.Errorf("open gRPC client CA directory: %w", err)
	}
	defer func() { _ = root.Close() }()
	encoded, err := root.ReadFile(filepath.Base(name))
	if err != nil {
		return nil, fmt.Errorf("read gRPC client CA file: %w", err)
	}
	authorities := x509.NewCertPool()
	if !authorities.AppendCertsFromPEM(encoded) {
		return nil, fmt.Errorf("gRPC client CA file %s contains no certificate", name)
	}
	return authorities, nil
}
