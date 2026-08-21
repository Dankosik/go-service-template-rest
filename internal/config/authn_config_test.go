package config

import (
	"errors"
	"strings"
	"testing"
)

func TestAuthnConfigRequiresCompleteSafePolicy(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		key   string
		value string
		want  string
	}{
		{name: "missing issuer", key: "APP__AUTHN__ISSUER", value: "", want: "authn.issuer"},
		{name: "plaintext issuer", key: "APP__AUTHN__ISSUER", value: "http://issuer.example.com", want: "authn.issuer"},
		{name: "issuer query", key: "APP__AUTHN__ISSUER", value: "https://issuer.example.com?tenant=a", want: "authn.issuer"},
		{name: "missing audience", key: "APP__AUTHN__AUDIENCE", value: "", want: "authn.audience"},
		{name: "unknown token profile", key: "APP__AUTHN__TOKEN_PROFILE", value: "strict", want: "authn.token_profile"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv(testCase.key, testCase.value)
			_, _, err := LoadDetailed(LoadOptions{})
			if !errors.Is(err, ErrValidate) {
				t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
			}
			if !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("LoadDetailed() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func TestAuthnConfigDefaultsAndCanonicalizesTokenProfile(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP__AUTHN__TOKEN_PROFILE", " RFC9068 ")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.Authn.TokenProfile != "rfc9068" {
		t.Fatalf("TokenProfile = %q, want rfc9068", cfg.Authn.TokenProfile)
	}
}

// profile:grpc:start

//nolint:paralleltest // This test mutates process-global environment.
func TestAuthnRequiresGRPCTLS(t *testing.T) {
	resetConfigEnv(t)
	for name, value := range map[string]string{
		"APP__GRPC__SERVER__ENABLED":            "true",
		"APP__GRPC__SERVER__ADDR":               "127.0.0.1:9443",
		"APP__GRPC__SERVER__TRANSPORT_SECURITY": "plaintext",
	} {
		t.Setenv(name, value)
	}

	_, _, err := LoadDetailed(LoadOptions{})
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "authn OIDC profile requires grpc.server.transport_security=tls") {
		t.Fatalf("LoadDetailed() error = %v, want OIDC gRPC TLS requirement", err)
	}
}

// profile:grpc:end
