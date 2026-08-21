package config

import (
	"errors"
	"testing"
)

func TestValidateMessagingConfig(t *testing.T) {
	valid := MessagingConfig{
		URLs:            " tls://one.example:4222 , tls://two.example:4222 ",
		CredentialsFile: " /run/secrets/nats.creds ", Stream: " EVENTS ",
		MaxPayloadBytes: 256 << 10,
	}
	if err := validateMessagingConfig(&valid); err != nil {
		t.Fatalf("validateMessagingConfig(valid) error = %v", err)
	}
	if valid.URLs != "tls://one.example:4222,tls://two.example:4222" || valid.Stream != "EVENTS" {
		t.Fatalf("canonical messaging config = URLs %q, stream %q", valid.URLs, valid.Stream)
	}

	for name, mutate := range map[string]func(*MessagingConfig){
		"userinfo":            func(cfg *MessagingConfig) { cfg.URLs = "tls://user@one.example:4222" },
		"plaintext":           func(cfg *MessagingConfig) { cfg.URLs = "nats://one.example:4222" },
		"duplicate URL":       func(cfg *MessagingConfig) { cfg.URLs = "tls://one.example:4222,tls://one.example:4222" },
		"missing credentials": func(cfg *MessagingConfig) { cfg.CredentialsFile = "" },
		"invalid stream":      func(cfg *MessagingConfig) { cfg.Stream = "EVENTS.BAD" },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := valid
			mutate(&cfg)
			if err := validateMessagingConfig(&cfg); !errors.Is(err, ErrValidate) {
				t.Fatalf("validateMessagingConfig() error = %v, want ErrValidate", err)
			}
		})
	}

	disabled := MessagingConfig{MaxPayloadBytes: 1}
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
