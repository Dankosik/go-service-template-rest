package config

import (
	"errors"
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
		"audience and resource": func(cfg *OutboundAuthConfig) { cfg.Audience, cfg.Resource = "payments", "payments" },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := valid
			mutate(&cfg)
			if err := validateOutboundAuthConfig(&cfg); !errors.Is(err, ErrValidate) {
				t.Fatalf("validateOutboundAuthConfig() error = %v, want ErrValidate", err)
			}
		})
	}

	if err := validateOutboundAuthConfig(&valid); err != nil {
		t.Fatalf("validateOutboundAuthConfig() error = %v", err)
	}
	if valid.TokenURL != "https://auth.example.com/oauth/token" || valid.Scopes != "payments.write payments.read" {
		t.Fatalf("canonical config = %#v", valid)
	}
}
