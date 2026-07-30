package config

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestDefaultGRPCServerConfigMatchesLoadedDefaults(t *testing.T) {
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

func TestGRPCDefaultsAreDisabledAndBounded(t *testing.T) {
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
	for _, tc := range []struct {
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
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			for key, value := range tc.env {
				t.Setenv(key, value)
			}

			cfg, _, err := LoadDetailed(LoadOptions{})
			if tc.wantErrPart == "" {
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
			if !strings.Contains(err.Error(), tc.wantErrPart) {
				t.Fatalf("LoadDetailed() error = %v, want %q", err, tc.wantErrPart)
			}
		})
	}
}

func TestGRPCAddressAndCapacityValidation(t *testing.T) {
	for _, tc := range []struct {
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
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv("APP__GRPC__SERVER__ENABLED", "true")
			t.Setenv("APP__GRPC__SERVER__ADDR", ":9091")
			t.Setenv("APP__GRPC__SERVER__TRANSPORT_SECURITY", "plaintext")
			t.Setenv("APP__GRPC__SERVER__ALLOW_PLAINTEXT", "true")
			t.Setenv(tc.key, tc.value)

			_, _, err := LoadDetailed(LoadOptions{})
			if !errors.Is(err, ErrValidate) {
				t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
			}
			if !strings.Contains(err.Error(), tc.wantErrPart) {
				t.Fatalf("LoadDetailed() error = %v, want %q", err, tc.wantErrPart)
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
