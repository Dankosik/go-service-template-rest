package config

import (
	"fmt"
	"net/url"
	"strings"
)

type MessagingConfig struct {
	URLs                 string                `koanf:"urls"`
	CredentialsFile      string                `koanf:"credentials_file"`
	RootCAFile           string                `koanf:"root_ca_file"`
	AllowPlaintext       bool                  `koanf:"allow_plaintext"`
	AllowUnauthenticated bool                  `koanf:"allow_unauthenticated"`
	Stream               string                `koanf:"stream"`
	MaxPayloadBytes      int                   `koanf:"max_payload_bytes"`
	Worker               MessagingWorkerConfig `koanf:"worker"`
}

type MessagingWorkerConfig struct {
	Consumer          string `koanf:"consumer"`
	FilterSubject     string `koanf:"filter_subject"`
	DeadLetterSubject string `koanf:"dead_letter_subject"`
	MaxConcurrency    int    `koanf:"max_concurrency"`
}

func messagingDefaults() map[string]any {
	return map[string]any{
		"messaging.urls":                       "",
		"messaging.credentials_file":           "",
		"messaging.root_ca_file":               "",
		"messaging.allow_plaintext":            false,
		"messaging.allow_unauthenticated":      false,
		"messaging.stream":                     "",
		"messaging.max_payload_bytes":          256 << 10,
		"messaging.worker.consumer":            "",
		"messaging.worker.filter_subject":      "",
		"messaging.worker.dead_letter_subject": "",
		"messaging.worker.max_concurrency":     8,
	}
}

// validateMessagingConfig restates the rules natsjs.ValidateConfig applies, so
// an operator's settings are rejected at load time instead of at connect. That
// copy is not a shortcut and cannot be removed by importing the adapter:
// depguard's config_no_runtime_owners rule forbids internal/config from
// importing any repository runtime package, so that loading configuration does
// not link a broker client into every binary that merely reads it.
//
// The copy survives only because the composition root — cmd/worker/internal/
// bootstrap, the one package that wires both — pins it from outside;
// TestMessagingConfigRulesMatchAdapter there feeds the same values through both
// validators and fails when they disagree.
//
// The two sides deliberately differ in one direction only: this one also
// canonicalizes — it trims, rejects duplicate URLs, and rewrites cfg.URLs — so
// it is strictly the stricter of the two. A rule added here that rejects
// something natsjs accepts is fine; one that accepts something natsjs rejects
// moves a startup failure to connect time.
func validateMessagingConfig(cfg *MessagingConfig) error {
	cfg.URLs = strings.TrimSpace(cfg.URLs)
	cfg.CredentialsFile = strings.TrimSpace(cfg.CredentialsFile)
	cfg.RootCAFile = strings.TrimSpace(cfg.RootCAFile)
	cfg.Stream = strings.TrimSpace(cfg.Stream)
	if cfg.MaxPayloadBytes <= 0 {
		return fmt.Errorf("%w: messaging.max_payload_bytes must be positive", ErrValidate)
	}
	if cfg.URLs == "" {
		return nil
	}
	canonicalURLs, err := canonicalizeMessagingURLs(cfg.URLs, cfg.AllowPlaintext)
	if err != nil {
		return err
	}
	cfg.URLs = canonicalURLs
	if !cfg.AllowUnauthenticated && cfg.CredentialsFile == "" {
		return fmt.Errorf("%w: messaging.credentials_file is required", ErrValidate)
	}
	if !validMessagingStreamName(cfg.Stream) {
		return fmt.Errorf("%w: messaging.stream is invalid", ErrValidate)
	}
	return nil
}

func canonicalizeMessagingURLs(values string, allowPlaintext bool) (string, error) {
	canonical := make([]string, 0)
	seen := make(map[string]struct{})
	for value := range strings.SplitSeq(values, ",") {
		value = strings.TrimSpace(value)
		parsed, err := url.Parse(value)
		if err != nil || parsed.Host == "" || parsed.User != nil {
			return "", fmt.Errorf("%w: messaging.urls contains an invalid URL or userinfo", ErrValidate)
		}
		switch parsed.Scheme {
		case secureTransportTLS, "wss":
		case "nats", "ws":
			if !allowPlaintext {
				return "", fmt.Errorf("%w: plaintext messaging URL requires messaging.allow_plaintext", ErrValidate)
			}
		default:
			return "", fmt.Errorf("%w: messaging.urls contains an unsupported scheme", ErrValidate)
		}
		if _, duplicate := seen[value]; duplicate {
			return "", fmt.Errorf("%w: messaging.urls contains a duplicate URL", ErrValidate)
		}
		seen[value] = struct{}{}
		canonical = append(canonical, value)
	}
	if len(canonical) == 0 {
		return "", fmt.Errorf("%w: messaging.urls cannot be empty when messaging is enabled", ErrValidate)
	}
	return strings.Join(canonical, ","), nil
}

// validMessagingStreamName is the same rule as natsjs.validConsumerName, which
// the adapter applies to both the stream and the durable consumer. Keep the two
// in step; see the comment above validateMessagingConfig for why there are two.
func validMessagingStreamName(value string) bool {
	return value != "" && !strings.ContainsAny(value, " .*\\/>\t\r\n")
}
