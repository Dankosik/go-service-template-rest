package config

import (
	"fmt"
	"strings"

	"github.com/example/go-service-template-rest/internal/messagingconfig"
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

// validateMessagingConfig canonicalizes the operator-facing representation and
// applies the pure rules shared with the NATS adapter. Disabled transport stays
// a valid service snapshot; binaries that require it reject that state.
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
	if !messagingconfig.ValidStreamOrConsumerName(cfg.Stream) {
		return fmt.Errorf("%w: messaging.stream is invalid", ErrValidate)
	}
	return nil
}

func canonicalizeMessagingURLs(values string, allowPlaintext bool) (string, error) {
	canonical := make([]string, 0)
	seen := make(map[string]struct{})
	for value := range strings.SplitSeq(values, ",") {
		value = strings.TrimSpace(value)
		if err := messagingconfig.ValidateURL(value, allowPlaintext); err != nil {
			return "", fmt.Errorf("%w: messaging.urls: %w", ErrValidate, err)
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
