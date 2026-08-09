package config

// profile:grpc:start

import (
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"

	"github.com/example/go-service-template-rest/internal/grpclimits"
)

func validateGRPCConfig(cfg *GRPCConfig) error {
	server := &cfg.Server
	server.Addr = strings.TrimSpace(server.Addr)
	server.TransportSecurity = strings.ToLower(strings.TrimSpace(server.TransportSecurity))
	server.TLS.CertFile = strings.TrimSpace(server.TLS.CertFile)
	server.TLS.KeyFile = strings.TrimSpace(server.TLS.KeyFile)
	server.TLS.ClientCAFile = strings.TrimSpace(server.TLS.ClientCAFile)

	if err := validateGRPCCapacityBounds(*server); err != nil {
		return err
	}
	// The rate and the threshold are proven again by grpcx.validateConfig when the
	// server is built, so a direct package user cannot recover an unbounded
	// default. internal/grpclimits owns the rule both sides apply.
	if violation := grpclimits.ValidateAccessLog(grpclimits.AccessLog{
		SuccessSampleRate: server.AccessLogSuccessSampleRate,
		SlowThreshold:     server.AccessLogSlowThreshold,
	}); violation != nil {
		return rejectGRPCBound(violation)
	}
	switch server.TransportSecurity {
	case "", "plaintext", secureTransportTLS:
	default:
		return fmt.Errorf(
			"%w: grpc.server.transport_security must be one of plaintext or tls",
			ErrValidate,
		)
	}
	if !server.Enabled {
		return nil
	}
	if err := validateGRPCAddress(server.Addr); err != nil {
		return err
	}
	if err := validateEnabledGRPCTransport(server); err != nil {
		return err
	}
	// profile:authn-oidc-jwt:start
	if err := validateGRPCAuthnTransport(server); err != nil {
		return err
	}
	// profile:authn-oidc-jwt:end
	return nil
}

func validateEnabledGRPCTransport(server *GRPCServerConfig) error {
	switch server.TransportSecurity {
	case "plaintext":
		if !server.AllowPlaintext {
			return fmt.Errorf(
				"%w: grpc.server.allow_plaintext must be true when transport_security is plaintext",
				ErrValidate,
			)
		}
		if server.TLS.CertFile != "" || server.TLS.KeyFile != "" || server.TLS.ClientCAFile != "" {
			return fmt.Errorf(
				"%w: grpc.server.tls cert_file, key_file, and client_ca_file must be empty when transport_security is plaintext",
				ErrValidate,
			)
		}
	case secureTransportTLS:
		if server.AllowPlaintext {
			return fmt.Errorf(
				"%w: grpc.server.allow_plaintext must be false when transport_security is tls",
				ErrValidate,
			)
		}
		if server.TLS.CertFile == "" || server.TLS.KeyFile == "" {
			return fmt.Errorf(
				"%w: grpc.server.tls cert_file and key_file are required when transport_security is tls",
				ErrValidate,
			)
		}
	default:
		return fmt.Errorf(
			"%w: grpc.server.transport_security is required when grpc.server.enabled is true",
			ErrValidate,
		)
	}
	return nil
}

// profile:authn-oidc-jwt:start

func validateGRPCAuthnTransport(server *GRPCServerConfig) error {
	if server.TransportSecurity != secureTransportTLS {
		return fmt.Errorf(
			"%w: authn OIDC profile requires grpc.server.transport_security=tls",
			ErrValidate,
		)
	}
	return nil
}

// profile:authn-oidc-jwt:end

func validateGRPCCapacityBounds(cfg GRPCServerConfig) error {
	// Checked in order rather than joined, so a configuration breaking more than
	// one bound is refused with the same message every restart.
	if err := validateIntRange("grpc.server.max_connections", cfg.MaxConnections, 1, 1_000_000); err != nil {
		return err
	}
	if err := validateIntRange("grpc.server.max_concurrent_rpcs", cfg.MaxConcurrentRPCs, 1, 100_000); err != nil {
		return err
	}
	if err := validateIntRange("grpc.server.max_concurrent_streams", cfg.MaxConcurrentStreams, 1, 100_000); err != nil {
		return err
	}
	if err := validateIntRange("grpc.server.max_header_list_bytes", cfg.MaxHeaderListBytes, 1, math.MaxInt32); err != nil {
		return err
	}
	if err := validateIntRange(
		"grpc.server.max_receive_message_bytes", cfg.MaxReceiveMessageBytes, 1, math.MaxInt32,
	); err != nil {
		return err
	}
	if err := validateIntRange(
		"grpc.server.max_send_message_bytes", cfg.MaxSendMessageBytes, 1, math.MaxInt32,
	); err != nil {
		return err
	}

	connectionCapacity := uint64(cfg.MaxConnections) * uint64(cfg.MaxConcurrentStreams)
	if uint64(cfg.MaxConcurrentRPCs) > connectionCapacity {
		return fmt.Errorf(
			"%w: grpc.server.max_concurrent_rpcs must be <= max_connections * max_concurrent_streams (%d)",
			ErrValidate,
			connectionCapacity,
		)
	}
	return validateGRPCLifetimeBounds(cfg)
}

// validateGRPCLifetimeBounds applies the shared time bounds to the loaded
// configuration. internal/grpclimits owns the rules and their order; this
// function owns only naming the configuration leaf an operator edits.
func validateGRPCLifetimeBounds(cfg GRPCServerConfig) error {
	if violation := grpclimits.ValidateLifetime(grpclimits.Lifetime{
		UnaryTimeout:          cfg.UnaryTimeout,
		StreamTimeout:         cfg.StreamTimeout,
		MaxConnectionIdle:     cfg.MaxConnectionIdle,
		MaxConnectionAge:      cfg.MaxConnectionAge,
		MaxConnectionAgeGrace: cfg.MaxConnectionAgeGrace,
		ServerPingInterval:    cfg.ServerPingInterval,
		ServerPingTimeout:     cfg.ServerPingTimeout,
		MinClientPingInterval: cfg.MinClientPingInterval,
	}); violation != nil {
		return rejectGRPCBound(violation)
	}
	return nil
}

// rejectGRPCBound renders a shared bound violation as the configuration leaf an
// operator has to edit, which is the only spelling that helps whoever reads a
// startup rejection.
func rejectGRPCBound(violation *grpclimits.Violation) error {
	return fmt.Errorf("%w: grpc.server.%s %s", ErrValidate, violation.Field, violation.Rule)
}

func validateGRPCAddress(addr string) error {
	if addr == "" {
		return fmt.Errorf("%w: grpc.server.addr cannot be empty when grpc.server.enabled is true", ErrValidate)
	}
	_, rawPort, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("%w: grpc.server.addr must be host:port", ErrValidate)
	}
	port, err := strconv.ParseUint(rawPort, 10, 16)
	if err != nil || port == 0 {
		return fmt.Errorf("%w: grpc.server.addr port must be in range [1,65535]", ErrValidate)
	}
	return nil
}

// profile:grpc:end
