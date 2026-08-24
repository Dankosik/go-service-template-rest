package config

import (
	"errors"
	"strings"
	"testing"
)

func TestOutboundAuthConfigIsThePortableMinimum(t *testing.T) {
	valid := OutboundAuthConfig{
		TokenURL:     "HTTPS://AUTH.EXAMPLE.COM/oauth/token",
		ClientID:     "client",
		ClientSecret: "secret",
		Scopes:       "payments.write   payments.read",
	}
	for name, mutate := range map[string]func(*OutboundAuthConfig){
		"missing token URL":     func(cfg *OutboundAuthConfig) { cfg.TokenURL = "" },
		"plaintext token URL":   func(cfg *OutboundAuthConfig) { cfg.TokenURL = "http://auth.example.com/token" },
		"missing client ID":     func(cfg *OutboundAuthConfig) { cfg.ClientID = "" },
		"missing client secret": func(cfg *OutboundAuthConfig) { cfg.ClientSecret = "" },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := valid
			mutate(&cfg)
			if err := validateOutboundAuthConfig(&cfg, "integrations.billing.oauth"); !errors.Is(err, ErrValidate) {
				t.Fatalf("validateOutboundAuthConfig() error = %v, want ErrValidate", err)
			}
		})
	}

	if err := validateOutboundAuthConfig(&valid, "integrations.billing.oauth"); err != nil {
		t.Fatalf("validateOutboundAuthConfig() error = %v", err)
	}
	if valid.TokenURL != "https://auth.example.com/oauth/token" || valid.Scopes != "payments.write payments.read" {
		t.Fatalf("canonical config = %#v", valid)
	}
}

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
func TestRetiredOutboundAuthEnvironmentKeyIsUnknown(t *testing.T) {
	resetConfigEnv(t)
	const canary = "retired-outbound-auth-canary"
	t.Setenv("APP__OUTBOUND_AUTH__CLIENT_SECRET", canary)
	_, _, err := LoadDetailed(LoadOptions{})
	if !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("LoadDetailed() error = %v, want ErrUnknownKey", err)
	}
	if strings.Contains(err.Error(), canary) {
		t.Fatalf("LoadDetailed() disclosed retired secret: %v", err)
	}
}
