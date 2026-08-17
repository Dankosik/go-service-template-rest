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
		{name: "missing proxies", key: "APP__AUTHN__TRUSTED_PROXY_CIDRS", value: "", want: "trusted_proxy_cidrs"},
		{name: "invalid proxy", key: "APP__AUTHN__TRUSTED_PROXY_CIDRS", value: "not-a-cidr", want: "trusted_proxy_cidrs"},
		{name: "IPv4 wildcard proxy", key: "APP__AUTHN__TRUSTED_PROXY_CIDRS", value: "0.0.0.0/0", want: "wildcard CIDR"},
		{name: "IPv6 wildcard proxy", key: "APP__AUTHN__TRUSTED_PROXY_CIDRS", value: "::/0", want: "wildcard CIDR"},
		{name: "duplicate proxy", key: "APP__AUTHN__TRUSTED_PROXY_CIDRS", value: "127.0.0.0/8,127.0.0.1/8", want: "duplicate CIDR"},
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

func TestAuthnConfigCanonicalizesTrustedProxyCIDRs(t *testing.T) {

	resetConfigEnv(t)
	t.Setenv("APP__AUTHN__TRUSTED_PROXY_CIDRS", " 127.0.0.1/8,2001:db8:1::1/32 ")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	const want = "127.0.0.0/8,2001:db8::/32"
	if cfg.Authn.TrustedProxyCIDRs != want {
		t.Fatalf("TrustedProxyCIDRs = %q, want %q", cfg.Authn.TrustedProxyCIDRs, want)
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
		"APP__GRPC__SERVER__ALLOW_PLAINTEXT":    "true",
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
