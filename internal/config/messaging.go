package config

import (
	"fmt"
	"net/url"
	"strings"
)

func validateMessagingConfig(cfg *MessagingConfig) error {
	cfg.URLs = strings.TrimSpace(cfg.URLs)
	cfg.CredentialsFile = strings.TrimSpace(cfg.CredentialsFile)
	cfg.RootCAFile = strings.TrimSpace(cfg.RootCAFile)
	cfg.Stream = strings.TrimSpace(cfg.Stream)
	if cfg.MaxPayloadBytes <= 0 {
		return fmt.Errorf("%w: messaging.max_payload_bytes must be positive", ErrValidate)
	}
	if cfg.MaxPendingPublishes <= 0 {
		return fmt.Errorf("%w: messaging.max_pending_publishes must be positive", ErrValidate)
	}
	if !cfg.Enabled {
		return nil
	}
	if cfg.URLs == "" {
		return fmt.Errorf("%w: messaging.urls cannot be empty when messaging is enabled", ErrValidate)
	}
	canonical := make([]string, 0)
	seen := make(map[string]struct{})
	for value := range strings.SplitSeq(cfg.URLs, ",") {
		value = strings.TrimSpace(value)
		parsed, err := url.Parse(value)
		if err != nil || parsed.Host == "" || parsed.User != nil {
			return fmt.Errorf("%w: messaging.urls contains an invalid URL or userinfo", ErrValidate)
		}
		switch parsed.Scheme {
		case secureTransportTLS, "wss":
		case "nats", "ws":
			if !cfg.AllowPlaintext {
				return fmt.Errorf("%w: plaintext messaging URL requires messaging.allow_plaintext", ErrValidate)
			}
		default:
			return fmt.Errorf("%w: messaging.urls contains an unsupported scheme", ErrValidate)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf("%w: messaging.urls contains a duplicate URL", ErrValidate)
		}
		seen[value] = struct{}{}
		canonical = append(canonical, value)
	}
	if len(canonical) == 0 {
		return fmt.Errorf("%w: messaging.urls cannot be empty when messaging is enabled", ErrValidate)
	}
	cfg.URLs = strings.Join(canonical, ",")
	if !cfg.AllowUnauthenticated && cfg.CredentialsFile == "" {
		return fmt.Errorf("%w: messaging.credentials_file is required", ErrValidate)
	}
	if !validMessagingName(cfg.Stream) {
		return fmt.Errorf("%w: messaging.stream is invalid", ErrValidate)
	}
	return nil
}

func validMessagingName(value string) bool {
	return value != "" && !strings.ContainsAny(value, " .*\\/>\t\r\n")
}
