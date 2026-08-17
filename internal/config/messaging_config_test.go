package config

import (
	"errors"
	"testing"
	"time"
)

func TestValidateMessagingConfig(t *testing.T) {
	valid := MessagingConfig{
		Enabled:             true,
		URLs:                " tls://one.example:4222 , tls://two.example:4222 ",
		CredentialsFile:     " /run/secrets/nats.creds ",
		Stream:              " EVENTS ",
		MinStreamReplicas:   3,
		MinStreamRetention:  24 * time.Hour,
		MaxPayloadBytes:     256 << 10,
		MaxPendingPublishes: 64,
	}
	if err := validateMessagingConfig(&valid); err != nil {
		t.Fatalf("validateMessagingConfig(valid) error = %v", err)
	}
	if valid.URLs != "tls://one.example:4222,tls://two.example:4222" || valid.Stream != "EVENTS" {
		t.Fatalf("canonical messaging config = URLs %q, stream %q", valid.URLs, valid.Stream)
	}

	cases := map[string]func(*MessagingConfig){
		"missing URLs":        func(cfg *MessagingConfig) { cfg.URLs = "" },
		"userinfo":            func(cfg *MessagingConfig) { cfg.URLs = "tls://user@one.example:4222" },
		"plaintext":           func(cfg *MessagingConfig) { cfg.URLs = "nats://one.example:4222" },
		"duplicate URL":       func(cfg *MessagingConfig) { cfg.URLs = "tls://one.example:4222,tls://one.example:4222" },
		"missing credentials": func(cfg *MessagingConfig) { cfg.CredentialsFile = "" },
		"invalid stream":      func(cfg *MessagingConfig) { cfg.Stream = "EVENTS.BAD" },
		"missing replicas":    func(cfg *MessagingConfig) { cfg.MinStreamReplicas = 0 },
		"too many replicas":   func(cfg *MessagingConfig) { cfg.MinStreamReplicas = 6 },
		"missing retention":   func(cfg *MessagingConfig) { cfg.MinStreamRetention = 0 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := valid
			mutate(&cfg)
			if err := validateMessagingConfig(&cfg); !errors.Is(err, ErrValidate) {
				t.Fatalf("validateMessagingConfig() error = %v, want ErrValidate", err)
			}
		})
	}

	disabled := MessagingConfig{MaxPayloadBytes: 1, MaxPendingPublishes: 1}
	if err := validateMessagingConfig(&disabled); err != nil {
		t.Fatalf("validateMessagingConfig(disabled) error = %v", err)
	}

	local := valid
	local.URLs = "nats://127.0.0.1:4222"
	local.CredentialsFile = ""
	local.AllowPlaintext = true
	local.AllowUnauthenticated = true
	if err := validateMessagingConfig(&local); err != nil {
		t.Fatalf("validateMessagingConfig(local escape hatches) error = %v", err)
	}
}
