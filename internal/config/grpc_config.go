package config

// profile:grpc:start

import (
	"fmt"
	"strings"
)

type GRPCConfig struct {
	Server GRPCServerConfig `koanf:"server"`
}

type GRPCServerConfig struct {
	Enabled           bool          `koanf:"enabled"`
	Addr              string        `koanf:"addr"`
	TransportSecurity string        `koanf:"transport_security"`
	TLS               GRPCTLSConfig `koanf:"tls"`
}

type GRPCTLSConfig struct {
	CertFile     string `koanf:"cert_file"`
	KeyFile      string `koanf:"key_file"`
	ClientCAFile string `koanf:"client_ca_file"`
}

func DefaultGRPCServerConfig() GRPCServerConfig {
	return GRPCServerConfig{}
}

func grpcDefaults() map[string]any {
	return map[string]any{
		"grpc.server.enabled":            false,
		"grpc.server.addr":               "",
		"grpc.server.transport_security": "",
		"grpc.server.tls.cert_file":      "",
		"grpc.server.tls.key_file":       "",
		"grpc.server.tls.client_ca_file": "",
	}
}

func validateGRPCConfig(cfg *GRPCConfig) error {
	server := &cfg.Server
	server.Addr = strings.TrimSpace(server.Addr)
	server.TransportSecurity = strings.ToLower(strings.TrimSpace(server.TransportSecurity))
	server.TLS.CertFile = strings.TrimSpace(server.TLS.CertFile)
	server.TLS.KeyFile = strings.TrimSpace(server.TLS.KeyFile)
	server.TLS.ClientCAFile = strings.TrimSpace(server.TLS.ClientCAFile)

	if !server.Enabled {
		return nil
	}
	if err := validateGRPCAddress(server.Addr); err != nil {
		return err
	}
	switch server.TransportSecurity {
	case "plaintext":
		if server.TLS.CertFile != "" || server.TLS.KeyFile != "" || server.TLS.ClientCAFile != "" {
			return fmt.Errorf(
				"%w: grpc.server.tls fields must be empty when transport_security is plaintext",
				ErrValidate,
			)
		}
	case secureTransportTLS:
		if server.TLS.CertFile == "" || server.TLS.KeyFile == "" {
			return fmt.Errorf(
				"%w: grpc.server.tls cert_file and key_file are required when transport_security is tls",
				ErrValidate,
			)
		}
	default:
		return fmt.Errorf(
			"%w: grpc.server.transport_security must be one of plaintext or tls",
			ErrValidate,
		)
	}
	// profile:authn-bearer:start
	if server.TransportSecurity != secureTransportTLS {
		return fmt.Errorf(
			"%w: authn OIDC profile requires grpc.server.transport_security=tls",
			ErrValidate,
		)
	}
	// profile:authn-bearer:end
	return nil
}

func validateGRPCAddress(addr string) error {
	if addr == "" {
		return fmt.Errorf("%w: grpc.server.addr cannot be empty when grpc.server.enabled is true", ErrValidate)
	}
	return validateHostPortAddr("grpc.server.addr", addr)
}

// profile:grpc:end
