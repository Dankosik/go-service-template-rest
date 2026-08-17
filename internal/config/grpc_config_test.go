// The gRPC section as an operator supplies it: which values load, which are
// refused, and what an unconfigured deployment gets.
//
// Every case goes through the real loader rather than calling a validator, so it
// covers the env decoding and normalization a deployment actually depends on —
// " PlainText " arriving as plaintext is the rule, not an accident.
//
// This file owns whether a value is accepted. Whether the accepted set still
// matches what the transport adapter will build is a separate question with a
// second owner, and it is proven in internal/infra/grpc/config_parity_test.go —
// the import only works in that direction.

package config

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestDefaultGRPCServerConfigMatchesLoadedDefaults(t *testing.T) {
	t.Parallel()
	resetConfigEnv(t)

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	want := DefaultGRPCServerConfig()
	if !reflect.DeepEqual(cfg.GRPC.Server, want) {
		t.Fatalf("loaded gRPC defaults = %+v, want %+v", cfg.GRPC.Server, want)
	}

	mutated := DefaultGRPCServerConfig()
	mutated.MaxConnections++
	if got := DefaultGRPCServerConfig(); reflect.DeepEqual(got, mutated) {
		t.Fatal("DefaultGRPCServerConfig() returned state shared with a prior call")
	}
}

// TestDefaultUnaryTimeoutMatchesHTTPRequestTimeout holds the promise
// GRPCServerConfig.UnaryTimeout's own comment makes: that one service answers on
// one budget over both transports out of the box.
//
// The two are independent literals — 8s in grpc_config.go, "8s" in
// http_config.go — so nothing but this notices when one of them moves. It
// asserts the defaults path only, because that comment also allows a deployment
// to configure the two apart; a validation rule would forbid what it permits.
func TestDefaultUnaryTimeoutMatchesHTTPRequestTimeout(t *testing.T) {
	t.Parallel()
	resetConfigEnv(t)

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.GRPC.Server.UnaryTimeout != cfg.HTTP.RequestTimeout {
		t.Fatalf("default grpc.server.unary_timeout = %s, want http.request_timeout %s",
			cfg.GRPC.Server.UnaryTimeout, cfg.HTTP.RequestTimeout)
	}
}

func TestGRPCDefaultsAreDisabledAndBounded(t *testing.T) {
	t.Parallel()
	resetConfigEnv(t)

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.GRPC.Server.Enabled {
		t.Fatal("GRPC.Server.Enabled = true, want false")
	}
	if cfg.GRPC.Server.Addr != "" || cfg.GRPC.Server.TransportSecurity != "" {
		t.Fatalf(
			"disabled gRPC endpoint = (%q, %q), want empty",
			cfg.GRPC.Server.Addr,
			cfg.GRPC.Server.TransportSecurity,
		)
	}
	if cfg.GRPC.Server.MaxConnections <= 0 ||
		cfg.GRPC.Server.MaxConcurrentRPCs <= 0 ||
		cfg.GRPC.Server.MaxConcurrentStreams == 0 ||
		cfg.GRPC.Server.MaxHeaderListBytes == 0 ||
		cfg.GRPC.Server.MaxReceiveMessageBytes <= 0 ||
		cfg.GRPC.Server.MaxSendMessageBytes <= 0 {
		t.Fatalf("gRPC defaults are not fully bounded: %+v", cfg.GRPC.Server)
	}
	if cfg.GRPC.Server.AccessLogSuccessSampleRate != 1 ||
		cfg.GRPC.Server.AccessLogSlowThreshold != 0 ||
		cfg.GRPC.Server.TelemetryHealthChecks {
		t.Fatalf("gRPC observability defaults = %+v, want compatible logs and quiet health telemetry", cfg.GRPC.Server)
	}
}

func TestGRPCObservabilityPolicyValidation(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		key         string
		value       string
		wantErrPart string
	}{
		{
			name:        "negative success sample rate",
			key:         "APP__GRPC__SERVER__ACCESS_LOG_SUCCESS_SAMPLE_RATE",
			value:       "-0.1",
			wantErrPart: "access_log_success_sample_rate",
		},
		{
			name:        "large success sample rate",
			key:         "APP__GRPC__SERVER__ACCESS_LOG_SUCCESS_SAMPLE_RATE",
			value:       "1.1",
			wantErrPart: "access_log_success_sample_rate",
		},
		{
			name:        "negative slow threshold",
			key:         "APP__GRPC__SERVER__ACCESS_LOG_SLOW_THRESHOLD",
			value:       "-1ms",
			wantErrPart: "access_log_slow_threshold",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv(testCase.key, testCase.value)

			_, _, err := LoadDetailed(LoadOptions{})
			if !errors.Is(err, ErrValidate) {
				t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
			}
			if !strings.Contains(err.Error(), testCase.wantErrPart) {
				t.Fatalf("LoadDetailed() error = %v, want %q", err, testCase.wantErrPart)
			}
		})
	}
}

func TestGRPCEnabledTransportSecurity(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		env         map[string]string
		wantErrPart string
	}{
		{
			name: "explicit plaintext",
			env: map[string]string{
				"APP__GRPC__SERVER__ENABLED":            "true",
				"APP__GRPC__SERVER__ADDR":               "127.0.0.1:9091",
				"APP__GRPC__SERVER__TRANSPORT_SECURITY": " PlainText ",
				"APP__GRPC__SERVER__ALLOW_PLAINTEXT":    "true",
			},
		},
		{
			name: "tls certificate pair",
			env: map[string]string{
				"APP__GRPC__SERVER__ENABLED":            "true",
				"APP__GRPC__SERVER__ADDR":               ":9091",
				"APP__GRPC__SERVER__TRANSPORT_SECURITY": "tls",
				"APP__GRPC__SERVER__TLS__CERT_FILE":     "/run/secrets/service.crt",
				"APP__GRPC__SERVER__TLS__KEY_FILE":      "/run/secrets/service.key",
			},
		},
		{
			name: "implicit transport rejected",
			env: map[string]string{
				"APP__GRPC__SERVER__ENABLED": "true",
				"APP__GRPC__SERVER__ADDR":    ":9091",
			},
			wantErrPart: "transport_security is required",
		},
		{
			name: "plaintext requires second opt in",
			env: map[string]string{
				"APP__GRPC__SERVER__ENABLED":            "true",
				"APP__GRPC__SERVER__ADDR":               ":9091",
				"APP__GRPC__SERVER__TRANSPORT_SECURITY": "plaintext",
			},
			wantErrPart: "allow_plaintext must be true",
		},
		{
			name: "plaintext rejects stale tls paths",
			env: map[string]string{
				"APP__GRPC__SERVER__ENABLED":            "true",
				"APP__GRPC__SERVER__ADDR":               ":9091",
				"APP__GRPC__SERVER__TRANSPORT_SECURITY": "plaintext",
				"APP__GRPC__SERVER__ALLOW_PLAINTEXT":    "true",
				"APP__GRPC__SERVER__TLS__CERT_FILE":     "/run/secrets/service.crt",
			},
			wantErrPart: "must be empty",
		},
		{
			name: "tls requires complete pair",
			env: map[string]string{
				"APP__GRPC__SERVER__ENABLED":            "true",
				"APP__GRPC__SERVER__ADDR":               ":9091",
				"APP__GRPC__SERVER__TRANSPORT_SECURITY": "tls",
				"APP__GRPC__SERVER__TLS__CERT_FILE":     "/run/secrets/service.crt",
			},
			wantErrPart: "cert_file and key_file are required",
		},
		{
			name: "tls rejects plaintext opt in",
			env: map[string]string{
				"APP__GRPC__SERVER__ENABLED":            "true",
				"APP__GRPC__SERVER__ADDR":               ":9091",
				"APP__GRPC__SERVER__TRANSPORT_SECURITY": "tls",
				"APP__GRPC__SERVER__ALLOW_PLAINTEXT":    "true",
				"APP__GRPC__SERVER__TLS__CERT_FILE":     "/run/secrets/service.crt",
				"APP__GRPC__SERVER__TLS__KEY_FILE":      "/run/secrets/service.key",
			},
			wantErrPart: "allow_plaintext must be false",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			resetConfigEnv(t)
			for key, value := range testCase.env {
				t.Setenv(key, value)
			}

			cfg, _, err := LoadDetailed(LoadOptions{})
			wantErrPart := testCase.wantErrPart
			// profile:authn-oidc-jwt:start
			if testCase.name == "explicit plaintext" {
				wantErrPart = "authn OIDC profile requires"
			}
			// profile:authn-oidc-jwt:end
			//nolint:paralleltest // This test mutates process-global environment or working directory.

			// Every relation the gRPC lifetime bounds state, driven from the outside so a
			// rule that exists only in prose fails here.
			//
			// The conditional rules are the point: three of these configurations are refused
			// only because rotation is on, and the same values with no age are accepted.
			if wantErrPart == "" {
				if err != nil {
					t.Fatalf("LoadDetailed() error = %v", err)
				}
				if cfg.GRPC.Server.Enabled && cfg.GRPC.Server.TransportSecurity == "" {
					t.Fatal("enabled gRPC transport security was not normalized")
				}
				return
			}
			if !errors.Is(err, ErrValidate) {
				t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
			}
			if !strings.Contains(err.Error(), wantErrPart) {
				t.Fatalf("LoadDetailed() error = %v, want %q", err, wantErrPart)
			}
		})
	}
}

func TestGRPCAddressAndCapacityValidation(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		key         string
		value       string
		wantErrPart string
	}{
		{
			name:        "missing address",
			key:         "APP__GRPC__SERVER__ADDR",
			value:       "",
			wantErrPart: "addr cannot be empty",
		},
		{
			name:        "invalid port",
			key:         "APP__GRPC__SERVER__ADDR",
			value:       "127.0.0.1:0",
			wantErrPart: "port must be in range",
		},
		{
			name:        "zero connections",
			key:         "APP__GRPC__SERVER__MAX_CONNECTIONS",
			value:       "0",
			wantErrPart: "max_connections must be in range",
		},
		{
			name:        "zero process admission",
			key:         "APP__GRPC__SERVER__MAX_CONCURRENT_RPCS",
			value:       "0",
			wantErrPart: "max_concurrent_rpcs must be in range",
		},
		{
			name:        "zero streams",
			key:         "APP__GRPC__SERVER__MAX_CONCURRENT_STREAMS",
			value:       "0",
			wantErrPart: "max_concurrent_streams must be in range",
		},
		{
			name:        "zero receive message",
			key:         "APP__GRPC__SERVER__MAX_RECEIVE_MESSAGE_BYTES",
			value:       "0",
			wantErrPart: "max_receive_message_bytes must be in range",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv("APP__GRPC__SERVER__ENABLED", "true")
			t.Setenv("APP__GRPC__SERVER__ADDR", ":9091")
			t.Setenv("APP__GRPC__SERVER__TRANSPORT_SECURITY", "plaintext")
			t.Setenv("APP__GRPC__SERVER__ALLOW_PLAINTEXT", "true")
			t.Setenv(testCase.key, testCase.value)

			_, _, err := LoadDetailed(LoadOptions{})
			if !errors.Is(err, ErrValidate) {
				t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
			}
			if !strings.Contains(err.Error(), testCase.wantErrPart) {
				t.Fatalf("LoadDetailed() error = %v, want %q", err, testCase.wantErrPart)
			}
		})
	}
}

func TestGRPCProcessAdmissionCannotExceedConnectionCapacity(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP__GRPC__SERVER__MAX_CONNECTIONS", "1")
	t.Setenv("APP__GRPC__SERVER__MAX_CONCURRENT_STREAMS", "1")
	t.Setenv("APP__GRPC__SERVER__MAX_CONCURRENT_RPCS", "2")

	_, _, err := LoadDetailed(LoadOptions{})
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "max_connections * max_concurrent_streams") {
		t.Fatalf("LoadDetailed() error = %v, want connection capacity policy", err)
	}
}

func TestGRPCLifetimeBoundValidation(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		env         map[string]string
		wantErrPart string
	}{
		{
			name:        "negative unary timeout",
			env:         map[string]string{"APP__GRPC__SERVER__UNARY_TIMEOUT": "-1s"},
			wantErrPart: "unary_timeout",
		},
		{
			// grpc-go normalizes only zero to infinity, so a negative age arms
			// its timer immediately and rotates every connection at once.
			name:        "negative connection age",
			env:         map[string]string{"APP__GRPC__SERVER__MAX_CONNECTION_AGE": "-1s"},
			wantErrPart: "max_connection_age",
		},
		{
			name:        "zero idle bound",
			env:         map[string]string{"APP__GRPC__SERVER__MAX_CONNECTION_IDLE": "0s"},
			wantErrPart: "max_connection_idle",
		},
		{
			name:        "zero client ping minimum",
			env:         map[string]string{"APP__GRPC__SERVER__MIN_CLIENT_PING_INTERVAL": "0s"},
			wantErrPart: "min_client_ping_interval",
		},
		{
			// A zero grace is read as infinity, so the connection would drain
			// and never close.
			//nolint:paralleltest // This test mutates process-global environment or working directory.

			// The same values the conditional rules refuse are accepted with rotation off,
			// which is what makes them conditional rather than flat bounds.
			name: "rotation with a zero grace",
			env: map[string]string{
				"APP__GRPC__SERVER__MAX_CONNECTION_AGE":       "30s",
				"APP__GRPC__SERVER__MAX_CONNECTION_AGE_GRACE": "0s",
			},
			wantErrPart: "max_connection_age_grace",
		},
		{
			name: "grace below the unary timeout",
			env: map[string]string{
				"APP__GRPC__SERVER__MAX_CONNECTION_AGE":       "30s",
				"APP__GRPC__SERVER__MAX_CONNECTION_AGE_GRACE": "1s",
				"APP__GRPC__SERVER__UNARY_TIMEOUT":            "8s",
			},
			wantErrPart: "max_connection_age_grace",
		},
		{
			name: "stream timeout at or above the rotation age",
			env: map[string]string{
				"APP__GRPC__SERVER__MAX_CONNECTION_AGE": "30s",
				"APP__GRPC__SERVER__STREAM_TIMEOUT":     "30s",
			},
			wantErrPart: "stream_timeout",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			resetConfigEnv(t)
			for key, value := range testCase.env {
				t.Setenv(key, value)
			}

			_, _, err := LoadDetailed(LoadOptions{})
			if !errors.Is(err, ErrValidate) {
				t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
			}
			if !strings.Contains(err.Error(), testCase.wantErrPart) {
				t.Fatalf("LoadDetailed() error = %v, want %q", err, testCase.wantErrPart)
			}
		})
	}
}

func TestGRPCLifetimeBoundsAcceptTheShippedShapeWithoutRotation(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP__GRPC__SERVER__MAX_CONNECTION_AGE_GRACE", "1s")
	t.Setenv("APP__GRPC__SERVER__UNARY_TIMEOUT", "8s")
	t.Setenv("APP__GRPC__SERVER__STREAM_TIMEOUT", "30s")

	if _, _, err := LoadDetailed(LoadOptions{}); err != nil {
		t.Fatalf("LoadDetailed() with rotation off error = %v", err)
	}
}
