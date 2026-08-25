package config

import (
	"errors"
	// profile:authn-oidc-introspection:start
	"path/filepath"
	// profile:authn-oidc-introspection:end
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

// profile:authn-oidc-jwt:start

func TestAuthnConfigRejectsUnknownTokenProfile(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP__AUTHN__TOKEN_PROFILE", "strict")
	_, _, err := LoadDetailed(LoadOptions{})
	if !errors.Is(err, ErrValidate) || !strings.Contains(err.Error(), "authn.token_profile") {
		t.Fatalf("LoadDetailed() error = %v, want token profile", err)
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

// profile:authn-oidc-jwt:end

// profile:authn-oidc-introspection:start

func TestIntrospectionConfigRequiresCompleteTuple(t *testing.T) {
	for _, testCase := range []struct {
		name string
		key  string
		want string
	}{
		{name: "missing endpoint", key: "APP__AUTHN__INTROSPECTION_ENDPOINT", want: "authn.introspection_endpoint"},
		{name: "missing target class", key: "APP__AUTHN__INTROSPECTION_TARGET_CLASS", want: "authn.introspection_target_class"},
		{name: "missing client id", key: "APP__AUTHN__INTROSPECTION_CLIENT_ID", want: "authn.introspection_client_id"},
		{name: "missing client secret", key: "APP__AUTHN__INTROSPECTION_CLIENT_SECRET", want: "authn.introspection_client_secret"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			resetConfigEnv(t)
			setIntrospectionTestEnv(t)
			t.Setenv(testCase.key, "")
			_, _, err := LoadDetailed(LoadOptions{})
			if !errors.Is(err, ErrValidate) || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("LoadDetailed() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func TestIntrospectionConfigAdmitsCompleteEnvironmentTuple(t *testing.T) {
	resetConfigEnv(t)
	setIntrospectionTestEnv(t)
	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.Authn.IntrospectionEndpoint != "https://idp.example.com/oauth/introspect" ||
		cfg.Authn.IntrospectionTargetClass != "external-https" ||
		cfg.Authn.IntrospectionClientID != "rs-client" ||
		cfg.Authn.IntrospectionClientSecret != "rs-secret" {
		t.Fatalf("introspection snapshot = %+v", cfg.Authn)
	}
}

func TestIntrospectionConfigRejectsUnknownKeyAndSecretFile(t *testing.T) {
	resetConfigEnv(t)
	setIntrospectionTestEnv(t)
	_, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("complete tuple error = %v", err)
	}

	resetConfigEnv(t)
	setIntrospectionTestEnv(t)
	t.Setenv("APP__AUTHN__UNKNOWN", "x")
	_, _, err = LoadDetailed(LoadOptions{})
	if err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("unknown key error = %v", err)
	}

	resetConfigEnv(t)
	setIntrospectionTestEnv(t)
	path := writeTempConfig(t, "authn:\n  introspection_client_secret: file-secret-canary\n")
	_, _, err = LoadDetailed(LoadOptions{ConfigPath: path})
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("YAML secret error = %v", err)
	}
	if strings.Contains(err.Error(), "file-secret-canary") {
		t.Fatalf("secret disclosed: %v", err)
	}
}

func TestIntrospectionConfigPrivateSuffix(t *testing.T) {
	resetConfigEnv(t)
	setIntrospectionTestEnv(t)
	t.Setenv("APP__AUTHN__INTROSPECTION_TARGET_CLASS", "private-https")
	t.Setenv("APP__AUTHN__INTROSPECTION_ENDPOINT", "https://idp.service.internal/introspect")
	t.Setenv("APP__AUTHN__INTROSPECTION_PRIVATE_HOST_SUFFIX", "")
	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil || !strings.Contains(err.Error(), "introspection_private_host_suffix") {
		t.Fatalf("missing suffix error = %v", err)
	}

	resetConfigEnv(t)
	setIntrospectionTestEnv(t)
	t.Setenv("APP__AUTHN__INTROSPECTION_PRIVATE_HOST_SUFFIX", "service.internal")
	_, _, err = LoadDetailed(LoadOptions{})
	if err == nil || !strings.Contains(err.Error(), "introspection_private_host_suffix") {
		t.Fatalf("forbidden suffix error = %v", err)
	}
}

func TestIntrospectionDisclosureBoundary(t *testing.T) {
	resetConfigEnv(t)
	const canary = "introspection-secret-canary"
	t.Setenv("APP__AUTHN__INTROSPECTION_ENDPOINT", "https://idp.example.com/oauth/introspect")
	t.Setenv("APP__AUTHN__INTROSPECTION_TARGET_CLASS", "external-https")
	t.Setenv("APP__AUTHN__INTROSPECTION_CLIENT_ID", "rs-client")
	t.Setenv("APP__AUTHN__INTROSPECTION_CLIENT_SECRET", canary)
	t.Setenv("APP__AUTHN__INTROSPECTION_ENDPOINT", "http://idp.example.com/oauth/introspect")
	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if strings.Contains(err.Error(), canary) {
		t.Fatalf("disclosed secret: %v", err)
	}
}

func TestIntrospectionEnvExampleIncludesPrivateHostSuffix(t *testing.T) {
	values := readEnvExample(t, filepath.Join("..", "..", "env", ".env.example"))
	if _, ok := values["APP__AUTHN__INTROSPECTION_PRIVATE_HOST_SUFFIX"]; !ok {
		t.Fatal("env/.env.example is missing APP__AUTHN__INTROSPECTION_PRIVATE_HOST_SUFFIX")
	}
}

func setIntrospectionTestEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP__AUTHN__INTROSPECTION_ENDPOINT", "https://idp.example.com/oauth/introspect")
	t.Setenv("APP__AUTHN__INTROSPECTION_TARGET_CLASS", "external-https")
	t.Setenv("APP__AUTHN__INTROSPECTION_CLIENT_ID", "rs-client")
	t.Setenv("APP__AUTHN__INTROSPECTION_CLIENT_SECRET", "rs-secret")
}

// profile:authn-oidc-introspection:end

//nolint:paralleltest // This test mutates process-global environment.
func TestAuthnRequiresHTTPAdmission(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP__HTTP__MAX_IN_FLIGHT", "0")

	_, _, err := LoadDetailed(LoadOptions{})
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "authn OIDC profile requires http.max_in_flight > 0") {
		t.Fatalf("LoadDetailed() error = %v, want OIDC HTTP admission requirement", err)
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
