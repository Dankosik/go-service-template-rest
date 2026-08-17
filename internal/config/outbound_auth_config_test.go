package config

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestOutboundAuthConfigContract(t *testing.T) {

	invalid := []struct {
		name  string
		key   string
		value string
	}{
		{"missing dependency", "APP__OUTBOUND_AUTH__DEPENDENCY", ""},
		{"invalid dependency", "APP__OUTBOUND_AUTH__DEPENDENCY", "Payments"},
		{"missing client id", "APP__OUTBOUND_AUTH__CLIENT_ID", ""},
		{"missing client secret", "APP__OUTBOUND_AUTH__CLIENT_SECRET", ""},
		{"unsupported client authentication", "APP__OUTBOUND_AUTH__CLIENT_AUTHENTICATION", "client_secret_post"},
		{"non-exact client authentication", "APP__OUTBOUND_AUTH__CLIENT_AUTHENTICATION", " client_secret_basic "},
		{"plaintext token endpoint", "APP__OUTBOUND_AUTH__TOKEN_ENDPOINT", "http://auth.example.com/token"},
		{"token endpoint query", "APP__OUTBOUND_AUTH__TOKEN_ENDPOINT", "https://auth.example.com/token?tenant=one"},
		{"unknown target class", "APP__OUTBOUND_AUTH__TOKEN_TARGET_CLASS", "private_http"},
		{"external suffix", "APP__OUTBOUND_AUTH__TOKEN_PRIVATE_HOST_SUFFIX", "internal"},
		//nolint:dupword // The duplicated word is the rejected input under test.
		{"duplicate scope", "APP__OUTBOUND_AUTH__SCOPES", "payments.read payments.read"},
		{"invalid scope", "APP__OUTBOUND_AUTH__SCOPES", "payments\\read"},
		{"Unicode scope separator", "APP__OUTBOUND_AUTH__SCOPES", "payments.read\u00a0payments.write"},
		{"resource authority path", "APP__OUTBOUND_AUTH__RESOURCE_AUTHORITY", "https://payments.example.com/api"},
		{"short acquisition timeout", "APP__OUTBOUND_AUTH__ACQUISITION_TIMEOUT", "99ms"},
		{"long acquisition timeout", "APP__OUTBOUND_AUTH__ACQUISITION_TIMEOUT", "31s"},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {

			resetConfigEnv(t)
			t.Setenv(test.key, test.value)
			_, _, err := LoadDetailed(LoadOptions{})
			if !errors.Is(err, ErrValidate) {
				t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
			}
			if strings.Contains(err.Error(), test.value) && test.value != "" {
				t.Fatalf("LoadDetailed() error disclosed rejected value %q: %v", test.value, err)
			}
		})
	}

	t.Run("resource and audience conflict", func(t *testing.T) {

		resetConfigEnv(t)
		t.Setenv("APP__OUTBOUND_AUTH__AUDIENCE", "payments-api")
		_, _, err := LoadDetailed(LoadOptions{})
		if !errors.Is(err, ErrValidate) {
			t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
		}
	})

	t.Run("second binding is unrepresentable", func(t *testing.T) {

		resetConfigEnv(t)
		t.Setenv("APP__OUTBOUND_AUTH__SECOND__CLIENT_ID", "other")
		_, _, err := LoadDetailed(LoadOptions{})
		if !errors.Is(err, ErrUnknownKey) {
			t.Fatalf("LoadDetailed() error = %v, want ErrUnknownKey", err)
		}
	})

	t.Run("private HTTPS is canonical", func(t *testing.T) {

		resetConfigEnv(t)
		t.Setenv("APP__OUTBOUND_AUTH__TOKEN_ENDPOINT", "HTTPS://AUTH.SERVICE.INTERNAL/oauth/token")
		t.Setenv("APP__OUTBOUND_AUTH__TOKEN_TARGET_CLASS", "private_https")
		t.Setenv("APP__OUTBOUND_AUTH__TOKEN_PRIVATE_HOST_SUFFIX", ".SERVICE.INTERNAL.")
		cfg, _, err := LoadDetailed(LoadOptions{})
		if err != nil {
			t.Fatalf("LoadDetailed() error = %v", err)
		}
		if cfg.OutboundAuth.TokenEndpoint != "https://auth.service.internal/oauth/token" {
			t.Fatalf("TokenEndpoint = %q", cfg.OutboundAuth.TokenEndpoint)
		}
		if cfg.OutboundAuth.TokenPrivateHostSuffix != "service.internal" {
			t.Fatalf("TokenPrivateHostSuffix = %q", cfg.OutboundAuth.TokenPrivateHostSuffix)
		}
	})

	joinedScopes := make([]string, 17)
	for index := range joinedScopes {
		joinedScopes[index] = strings.Repeat(string(rune('a'+index)), 256)
	}
	pureBounds := []struct {
		name   string
		mutate func(*OutboundAuthConfig)
	}{
		{name: "long dependency", mutate: func(cfg *OutboundAuthConfig) { cfg.Dependency = "a" + strings.Repeat("b", 64) }},
		{name: "long client id", mutate: func(cfg *OutboundAuthConfig) { cfg.ClientID = strings.Repeat("i", 513) }},
		{name: "client id control", mutate: func(cfg *OutboundAuthConfig) { cfg.ClientID = "client\u0085id" }},
		{name: "blank client secret", mutate: func(cfg *OutboundAuthConfig) { cfg.ClientSecret = "   " }},
		{name: "long client secret", mutate: func(cfg *OutboundAuthConfig) { cfg.ClientSecret = strings.Repeat("s", 4097) }},
		{name: "client secret control", mutate: func(cfg *OutboundAuthConfig) { cfg.ClientSecret = "secret\u0085value" }},
		{name: "relative endpoint", mutate: func(cfg *OutboundAuthConfig) { cfg.TokenEndpoint = "/oauth/token" }},
		{name: "endpoint user info", mutate: func(cfg *OutboundAuthConfig) { cfg.TokenEndpoint = "https://user@auth.example.com/token" }},
		{name: "endpoint fragment", mutate: func(cfg *OutboundAuthConfig) { cfg.TokenEndpoint += "#fragment" }},
		{name: "long endpoint", mutate: func(cfg *OutboundAuthConfig) {
			cfg.TokenEndpoint = "https://auth.example.com/" + strings.Repeat("a", 2048)
		}},
		{name: "private suffix missing", mutate: func(cfg *OutboundAuthConfig) { cfg.TokenTargetClass = outboundAuthPrivateHTTPS }},
		{name: "private suffix invalid", mutate: func(cfg *OutboundAuthConfig) {
			cfg.TokenTargetClass = outboundAuthPrivateHTTPS
			cfg.TokenPrivateHostSuffix = "-internal"
		}},
		{name: "too many scopes", mutate: func(cfg *OutboundAuthConfig) { cfg.Scopes = strings.Repeat("scope ", 33) }},
		{name: "long scope", mutate: func(cfg *OutboundAuthConfig) { cfg.Scopes = strings.Repeat("s", 257) }},
		{name: "joined scopes too long", mutate: func(cfg *OutboundAuthConfig) { cfg.Scopes = strings.Join(joinedScopes, " ") }},
		{name: "resource fragment", mutate: func(cfg *OutboundAuthConfig) { cfg.Resource += "#fragment" }},
		{name: "long resource", mutate: func(cfg *OutboundAuthConfig) { cfg.Resource = "urn:" + strings.Repeat("r", 2048) }},
		{name: "long audience", mutate: func(cfg *OutboundAuthConfig) { cfg.Resource = ""; cfg.Audience = strings.Repeat("a", 2049) }},
		{name: "audience control", mutate: func(cfg *OutboundAuthConfig) { cfg.Resource = ""; cfg.Audience = "payments\u0085api" }},
		{name: "plaintext resource authority", mutate: func(cfg *OutboundAuthConfig) { cfg.ResourceAuthority = "http://payments.example.com" }},
	}
	for _, test := range pureBounds {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cfg := validOutboundAuthSourceConfig()
			test.mutate(&cfg)
			if err := validateOutboundAuthConfig(&cfg); !errors.Is(err, ErrValidate) {
				t.Fatalf("validateOutboundAuthConfig() error = %v, want ErrValidate", err)
			}
		})
	}
}

func validOutboundAuthSourceConfig() OutboundAuthConfig {
	return OutboundAuthConfig{
		Dependency:           "payments",
		ClientID:             " client:id+ ",
		ClientSecret:         " secret:p@ss+ ",
		ClientAuthentication: outboundAuthClientAuthentication,
		TokenEndpoint:        "https://auth.example.com/oauth/token",
		TokenTargetClass:     outboundAuthExternalHTTPS,
		Scopes:               "payments.read payments.write",
		Resource:             "https://payments.example.com",
		ResourceAuthority:    "https://payments.example.com",
		AcquisitionTimeout:   2 * time.Second,
	}
}
