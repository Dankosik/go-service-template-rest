package oauth2clientcredentials

import (
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/httpclient"
)

func TestOutboundAuthConfigContract(t *testing.T) {
	t.Parallel()
	joinedScopes := make([]string, 17)
	for index := range joinedScopes {
		joinedScopes[index] = strings.Repeat(string(rune('a'+index)), maxScopeBytes)
	}

	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "missing dependency", mutate: func(cfg *Config) { cfg.DependencyName = "" }},
		{name: "invalid dependency", mutate: func(cfg *Config) { cfg.DependencyName = "Payments" }},
		{name: "long dependency", mutate: func(cfg *Config) { cfg.DependencyName = "a" + strings.Repeat("b", maxDependencyNameBytes) }},
		{name: "missing client id", mutate: func(cfg *Config) { cfg.ClientID = "" }},
		{name: "long client id", mutate: func(cfg *Config) { cfg.ClientID = strings.Repeat("i", maxClientIDBytes+1) }},
		{name: "client id control", mutate: func(cfg *Config) { cfg.ClientID = "client\nidentifier" }},
		{name: "client id Unicode control", mutate: func(cfg *Config) { cfg.ClientID = "client\u0085identifier" }},
		{name: "missing client secret", mutate: func(cfg *Config) { cfg.ClientSecret = "" }},
		{name: "blank client secret", mutate: func(cfg *Config) { cfg.ClientSecret = "   " }},
		{name: "long client secret", mutate: func(cfg *Config) { cfg.ClientSecret = strings.Repeat("s", maxClientSecretBytes+1) }},
		{name: "client secret control", mutate: func(cfg *Config) { cfg.ClientSecret = "secret\x7f" }},
		{name: "unsupported client authentication", mutate: func(cfg *Config) { cfg.ClientAuthentication = "client_secret_post" }},
		{name: "non-exact client authentication", mutate: func(cfg *Config) { cfg.ClientAuthentication = " client_secret_basic " }},
		{name: "relative token endpoint", mutate: func(cfg *Config) { cfg.TokenEndpoint = "/oauth/token" }},
		{name: "plaintext token endpoint", mutate: func(cfg *Config) { cfg.TokenEndpoint = "http://auth.example.com/oauth/token" }},
		{name: "token endpoint user info", mutate: func(cfg *Config) { cfg.TokenEndpoint = "https://user@auth.example.com/oauth/token" }},
		{name: "token endpoint query", mutate: func(cfg *Config) { cfg.TokenEndpoint += "?tenant=one" }},
		{name: "token endpoint fragment", mutate: func(cfg *Config) { cfg.TokenEndpoint += "#secret" }},
		{name: "long token endpoint", mutate: func(cfg *Config) {
			cfg.TokenEndpoint = "https://auth.example.com/" + strings.Repeat("a", maxEndpointBytes)
		}},
		{name: "unknown target class", mutate: func(cfg *Config) { cfg.TokenTargetClass = httpclient.TargetClass(255) }},
		{name: "external private suffix", mutate: func(cfg *Config) { cfg.TokenPrivateHostSuffix = "internal" }},
		{name: "private suffix missing", mutate: func(cfg *Config) {
			cfg.TokenEndpoint = "https://auth.service.internal/oauth/token"
			cfg.TokenTargetClass = httpclient.PrivateHTTPS
			cfg.TokenPrivateHostSuffix = ""
		}},
		{name: "private suffix invalid", mutate: func(cfg *Config) {
			cfg.TokenEndpoint = "https://auth.service.internal/oauth/token"
			cfg.TokenTargetClass = httpclient.PrivateHTTPS
			cfg.TokenPrivateHostSuffix = "-internal"
		}},
		{name: "too many scopes", mutate: func(cfg *Config) { cfg.Scopes = make([]string, maxScopes+1) }},
		{name: "long scope", mutate: func(cfg *Config) { cfg.Scopes = []string{strings.Repeat("s", maxScopeBytes+1)} }},
		{name: "invalid scope", mutate: func(cfg *Config) { cfg.Scopes = []string{"payments read"} }},
		{name: "duplicate scope", mutate: func(cfg *Config) { cfg.Scopes = []string{"payments.read", "payments.read"} }},
		{name: "joined scopes too long", mutate: func(cfg *Config) { cfg.Scopes = joinedScopes }},
		{name: "resource and audience", mutate: func(cfg *Config) { cfg.Audience = "payments-api" }},
		{name: "relative resource", mutate: func(cfg *Config) { cfg.Resource = "/payments" }},
		{name: "resource fragment", mutate: func(cfg *Config) { cfg.Resource += "#secret" }},
		{name: "long audience", mutate: func(cfg *Config) { cfg.Resource = ""; cfg.Audience = strings.Repeat("a", maxAudienceBytes+1) }},
		{name: "audience control", mutate: func(cfg *Config) { cfg.Resource = ""; cfg.Audience = "payments\napi" }},
		{name: "audience Unicode control", mutate: func(cfg *Config) { cfg.Resource = ""; cfg.Audience = "payments\u0085api" }},
		{name: "plaintext resource authority", mutate: func(cfg *Config) { cfg.ResourceAuthority = "http://payments.example.com" }},
		{name: "resource authority path", mutate: func(cfg *Config) { cfg.ResourceAuthority += "/api" }},
		{name: "short timeout", mutate: func(cfg *Config) { cfg.AcquisitionTimeout = minAcquisitionTimeout - time.Nanosecond }},
		{name: "long timeout", mutate: func(cfg *Config) { cfg.AcquisitionTimeout = maxAcquisitionTimeout + time.Nanosecond }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cfg := validTestConfig()
			test.mutate(&cfg)
			_, err := validateConfig(cfg)
			assertFailureClass(t, err, FailureInvalidConfiguration)
			if cfg.ClientID != "" && strings.Contains(err.Error(), cfg.ClientID) ||
				cfg.ClientSecret != "" && strings.Contains(err.Error(), cfg.ClientSecret) {
				t.Fatalf("validation error disclosed configuration: %v", err)
			}
		})
	}

	t.Run("canonical valid private binding", func(t *testing.T) {
		t.Parallel()
		cfg := validTestConfig()
		cfg.TokenEndpoint = "HTTPS://AUTH.SERVICE.INTERNAL/oauth/token"
		cfg.TokenTargetClass = httpclient.PrivateHTTPS
		cfg.TokenPrivateHostSuffix = ".SERVICE.INTERNAL."
		cfg.Scopes = []string{"payments.write", "payments.read"}
		originalID, originalSecret := cfg.ClientID, cfg.ClientSecret

		got, err := validateConfig(cfg)
		if err != nil {
			t.Fatalf("validateConfig() error = %v", err)
		}
		if got.TokenEndpoint != "https://auth.service.internal/oauth/token" || got.TokenPrivateHostSuffix != "service.internal" {
			t.Fatalf("canonical token target = %q, %q", got.TokenEndpoint, got.TokenPrivateHostSuffix)
		}
		if got.ClientID != originalID || got.ClientSecret != originalSecret {
			t.Fatal("validateConfig() changed exact client credential bytes")
		}
		if strings.Join(got.Scopes, " ") != "payments.read payments.write" {
			t.Fatalf("Scopes = %q", got.Scopes)
		}
		if cfg.Scopes[0] != "payments.write" {
			t.Fatal("validateConfig() mutated caller-owned scopes")
		}
	})
}

func TestTokenEndpointHTTPPolicy(t *testing.T) {
	t.Parallel()
	cfg := validTestConfig()
	httpConfig := tokenHTTPConfig(cfg)
	if httpConfig.DependencyName != cfg.DependencyName || httpConfig.BaseURL != cfg.TokenEndpoint ||
		httpConfig.TargetClass != cfg.TokenTargetClass || httpConfig.PrivateHostSuffix != cfg.TokenPrivateHostSuffix {
		t.Fatalf("token target policy = %#v", httpConfig)
	}
	if !httpConfig.OneAttempt || !httpConfig.DisableInstrumentation || httpConfig.Retry != (httpclient.RetryPolicy{}) ||
		httpConfig.Propagation != httpclient.PropagationNone {
		t.Fatalf("token retry/instrumentation policy = %#v", httpConfig)
	}
	if httpConfig.RequestTimeout != cfg.AcquisitionTimeout || httpConfig.ResponseHeaderTimeout != cfg.AcquisitionTimeout ||
		httpConfig.MaxResponseHeaderBytes != maxProviderHeaderBytes || httpConfig.MaxResponseBodyBytes != maxProviderBodyBytes ||
		httpConfig.MaxConnsPerHost != 1 || httpConfig.MaxIdleConnsPerHost != 1 {
		t.Fatalf("token bounds = %#v", httpConfig)
	}

	client, err := newTokenHTTPClient(cfg, nil)
	if err != nil {
		t.Fatalf("newTokenHTTPClient() error = %v", err)
	}
	t.Cleanup(client.CloseIdleConnections)
	if client.BaseURL() != cfg.TokenEndpoint {
		t.Fatalf("BaseURL() = %q, want %q", client.BaseURL(), cfg.TokenEndpoint)
	}

	mismatched := cfg
	mismatched.TokenEndpoint = "https://other.example.com/oauth/token"
	if _, err := newProvider(mismatched, client, time.Now); err == nil {
		t.Fatal("newProvider() admitted a token client bound to another authority")
	} else {
		assertFailureClass(t, err, FailureInvalidConfiguration)
	}
}
