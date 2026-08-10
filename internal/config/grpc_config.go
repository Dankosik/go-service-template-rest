package config

// profile:grpc:start

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/grpclimits"
)

type GRPCConfig struct {
	Server GRPCServerConfig `koanf:"server"`
}

type GRPCServerConfig struct {
	Enabled bool   `koanf:"enabled"`
	Addr    string `koanf:"addr"`

	// TransportSecurity is explicit whenever the server is enabled. "tls"
	// loads the configured certificate and key during bootstrap; "plaintext"
	// additionally requires AllowPlaintext so an operator cannot disable
	// transport security by changing one value.
	TransportSecurity string        `koanf:"transport_security"`
	AllowPlaintext    bool          `koanf:"allow_plaintext"`
	TLS               GRPCTLSConfig `koanf:"tls"`

	// MaxConnections bounds accepted HTTP/2 connections. MaxConcurrentRPCs is
	// the process-wide handler admission limit shared by unary and streaming
	// RPCs; MaxConcurrentStreams is the per-connection HTTP/2 stream limit.
	MaxConnections       int `koanf:"max_connections"`
	MaxConcurrentRPCs    int `koanf:"max_concurrent_rpcs"`
	MaxConcurrentStreams int `koanf:"max_concurrent_streams"`

	MaxHeaderListBytes     int  `koanf:"max_header_list_bytes"`
	MaxReceiveMessageBytes int  `koanf:"max_receive_message_bytes"`
	MaxSendMessageBytes    int  `koanf:"max_send_message_bytes"`
	AccessLogHealthChecks  bool `koanf:"access_log_health_checks"`
	// AccessLogSuccessSampleRate applies only to successful RPCs below
	// AccessLogSlowThreshold. Errors always remain visible while INFO logging is
	// enabled. The default of 1 preserves one completion record per business RPC.
	AccessLogSuccessSampleRate float64       `koanf:"access_log_success_sample_rate"`
	AccessLogSlowThreshold     time.Duration `koanf:"access_log_slow_threshold"`
	// TelemetryHealthChecks re-enables server-side OTel spans and duration
	// metrics for routine standard health polling.
	TelemetryHealthChecks bool `koanf:"telemetry_health_checks"`

	// UnaryTimeout caps how long one unary RPC may occupy a handler, and
	// StreamTimeout does the same for a stream. Both derive the RPC context's
	// deadline, so a caller deadline that is already earlier wins. Non-positive
	// disables the cap for that kind.
	//
	// UnaryTimeout defaults to http.request_timeout's value so one service
	// answers on one budget over both transports out of the box; it is a
	// separate key so a deployment can still move the two apart. StreamTimeout
	// is off by default because a long-lived stream's duration policy belongs to
	// the feature that owns the stream.
	UnaryTimeout  time.Duration `koanf:"unary_timeout"`
	StreamTimeout time.Duration `koanf:"stream_timeout"`

	// The liveness bounds, always on. None can end an RPC in progress: the idle
	// clock runs only while nothing is outstanding, and the ping bound closes
	// only when a ping goes unanswered, which means the peer is gone. Without
	// them a dead peer holds a listener slot for grpc-go's two-hour default.
	//
	// MinClientPingInterval and PermitPingWithoutStream bound what a client may
	// do rather than what this server does. grpc-go's defaults reject a ping
	// more often than every five minutes and any ping with no active stream,
	// which would disconnect this repository's own client half.
	MaxConnectionIdle       time.Duration `koanf:"max_connection_idle"`
	ServerPingInterval      time.Duration `koanf:"server_ping_interval"`
	ServerPingTimeout       time.Duration `koanf:"server_ping_timeout"`
	MinClientPingInterval   time.Duration `koanf:"min_client_ping_interval"`
	PermitPingWithoutStream bool          `koanf:"permit_ping_without_stream"`

	// MaxConnectionAge rotates connections and is off by default, because it is
	// the only bound here that ends work in progress: at this age the connection
	// is drained with GOAWAY and force-closed once the grace expires, cutting
	// every RPC and stream still running. Enable it behind an L4 balancer or any
	// hop that pins a caller to one replica for a connection's lifetime, and
	// accept that a stream outliving the age ends with UNAVAILABLE.
	//
	// The grace must be positive whenever an age is set: grpc-go reads a zero
	// grace as infinity, which would drain the connection and never close it.
	MaxConnectionAge      time.Duration `koanf:"max_connection_age"`
	MaxConnectionAgeGrace time.Duration `koanf:"max_connection_age_grace"`
}

type GRPCTLSConfig struct {
	CertFile string `koanf:"cert_file"`
	KeyFile  string `koanf:"key_file"`

	// ClientCAFile turns the listener into a mutual-TLS boundary: the
	// authorities in this file are the only ones a client certificate may chain
	// to, and a caller presenting none is refused during the handshake. Empty
	// leaves client certificates unrequested, which is the shipped default —
	// mutual TLS is a deployment's trust decision, not a transport default.
	//
	// Unlike the pair above it is read once, at startup. A leaf certificate
	// rotates on its issuer's schedule and is reloaded per handshake; a trust
	// root changes on a deployment's, which already restarts the process.
	ClientCAFile string `koanf:"client_ca_file"`
}

// DefaultGRPCServerConfig returns the canonical production defaults without
// reading files or environment variables. Runtime loading and benchmark-only
// composition consume the same value so their transport bounds cannot drift.
func DefaultGRPCServerConfig() GRPCServerConfig {
	return GRPCServerConfig{
		Enabled:           false,
		Addr:              "",
		TransportSecurity: "",
		AllowPlaintext:    false,
		TLS: GRPCTLSConfig{
			CertFile:     "",
			KeyFile:      "",
			ClientCAFile: "",
		},
		MaxConnections:             4096,
		MaxConcurrentRPCs:          256,
		MaxConcurrentStreams:       100,
		MaxHeaderListBytes:         16 << 10,
		MaxReceiveMessageBytes:     4 << 20,
		MaxSendMessageBytes:        4 << 20,
		AccessLogHealthChecks:      false,
		AccessLogSuccessSampleRate: 1,
		AccessLogSlowThreshold:     0,
		TelemetryHealthChecks:      false,
		// unary_timeout matches http.request_timeout so one service answers on
		// one budget over both transports; stream_timeout is off because a
		// long-lived stream's duration policy belongs to its feature.
		UnaryTimeout:  8 * time.Second,
		StreamTimeout: 0,
		// The liveness bounds detect a vanished peer within roughly the ping
		// interval plus its timeout, well under grpc-go's 2h default, and
		// reclaim a connection nobody is using. min_client_ping_interval is
		// below the client half's 30s ping interval, and permitting a ping with
		// no active stream is what lets that client keep an idle connection
		// through a NAT or balancer idle timeout.
		MaxConnectionIdle:       15 * time.Minute,
		ServerPingInterval:      time.Minute,
		ServerPingTimeout:       20 * time.Second,
		MinClientPingInterval:   10 * time.Second,
		PermitPingWithoutStream: true,
		// Rotation is off: it is the only bound here that ends work in progress.
		// The grace is still 10s rather than zero, so enabling rotation by
		// setting an age alone is a working configuration rather than a startup
		// refusal — and 10s exceeds unary_timeout, which is the relation
		// validateGRPCLifetimeBounds enforces.
		MaxConnectionAge:      0,
		MaxConnectionAgeGrace: 10 * time.Second,
	}
}

func grpcDefaults() map[string]any {
	grpcServer := DefaultGRPCServerConfig()

	// gRPC is compiled into the enabled template profile but remains disabled
	// until a service chooses an address and an explicit transport security mode.
	return map[string]any{
		"grpc.server.enabled":                        grpcServer.Enabled,
		"grpc.server.addr":                           grpcServer.Addr,
		"grpc.server.transport_security":             grpcServer.TransportSecurity,
		"grpc.server.allow_plaintext":                grpcServer.AllowPlaintext,
		"grpc.server.tls.cert_file":                  grpcServer.TLS.CertFile,
		"grpc.server.tls.key_file":                   grpcServer.TLS.KeyFile,
		"grpc.server.tls.client_ca_file":             grpcServer.TLS.ClientCAFile,
		"grpc.server.max_connections":                grpcServer.MaxConnections,
		"grpc.server.max_concurrent_rpcs":            grpcServer.MaxConcurrentRPCs,
		"grpc.server.max_concurrent_streams":         grpcServer.MaxConcurrentStreams,
		"grpc.server.max_header_list_bytes":          grpcServer.MaxHeaderListBytes,
		"grpc.server.max_receive_message_bytes":      grpcServer.MaxReceiveMessageBytes,
		"grpc.server.max_send_message_bytes":         grpcServer.MaxSendMessageBytes,
		"grpc.server.access_log_health_checks":       grpcServer.AccessLogHealthChecks,
		"grpc.server.access_log_success_sample_rate": grpcServer.AccessLogSuccessSampleRate,
		"grpc.server.access_log_slow_threshold":      grpcServer.AccessLogSlowThreshold.String(),
		"grpc.server.telemetry_health_checks":        grpcServer.TelemetryHealthChecks,
		"grpc.server.unary_timeout":                  grpcServer.UnaryTimeout.String(),
		"grpc.server.stream_timeout":                 grpcServer.StreamTimeout.String(),
		"grpc.server.max_connection_idle":            grpcServer.MaxConnectionIdle.String(),
		"grpc.server.server_ping_interval":           grpcServer.ServerPingInterval.String(),
		"grpc.server.server_ping_timeout":            grpcServer.ServerPingTimeout.String(),
		"grpc.server.min_client_ping_interval":       grpcServer.MinClientPingInterval.String(),
		"grpc.server.permit_ping_without_stream":     grpcServer.PermitPingWithoutStream,
		"grpc.server.max_connection_age":             grpcServer.MaxConnectionAge.String(),
		"grpc.server.max_connection_age_grace":       grpcServer.MaxConnectionAgeGrace.String(),
	}
}

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

	if cfg.MaxConnections <= math.MaxInt/cfg.MaxConcurrentStreams {
		connectionCapacity := cfg.MaxConnections * cfg.MaxConcurrentStreams
		if cfg.MaxConcurrentRPCs > connectionCapacity {
			return fmt.Errorf(
				"%w: grpc.server.max_concurrent_rpcs must be <= max_connections * max_concurrent_streams (%d)",
				ErrValidate,
				connectionCapacity,
			)
		}
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
	return validateHostPortAddr("grpc.server.addr", addr)
}

// profile:grpc:end
