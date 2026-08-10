package natsjs

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	HeaderLimitBytes = 8 << 10

	operationTimeout = 5 * time.Second
	// maxHeaderValueBytes bounds every identity carried in a message header, so
	// the encoded envelope stays inside HeaderLimitBytes.
	maxHeaderValueBytes = 256
)

type Config struct {
	URLs                 []string
	CredentialsFile      string
	RootCAFile           string
	AllowPlaintext       bool
	AllowUnauthenticated bool
	Stream               string
	MaxPayloadBytes      int
	MaxPendingPublishes  int
}

// ValidateConfig rejects a client configuration this package cannot connect
// under. [Connect] applies it before touching the network, so a settings fault
// never reaches the broker.
//
// It is exported for the same reason [ValidateWorkerConfig] is: internal/config
// restates these rules and cannot import this package, so the composition root
// is the only place that can hold both and prove they still describe the same
// configuration. See the comment above validateMessagingConfig in
// internal/config/messaging_config.go for that arrangement and the test that
// pins it.
func ValidateConfig(cfg Config) error {
	if len(cfg.URLs) == 0 {
		return fmt.Errorf("%w: messaging URLs are required", ErrRejected)
	}
	for _, raw := range cfg.URLs {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Host == "" {
			return fmt.Errorf("%w: invalid messaging URL", ErrRejected)
		}
		if parsed.User != nil {
			return fmt.Errorf("%w: messaging URL userinfo is forbidden", ErrRejected)
		}
		if !cfg.AllowPlaintext && parsed.Scheme != "tls" && parsed.Scheme != "wss" {
			return fmt.Errorf("%w: plaintext messaging URL requires explicit opt-in", ErrRejected)
		}
		switch parsed.Scheme {
		case "nats", "tls", "ws", "wss":
		default:
			return fmt.Errorf("%w: unsupported messaging URL scheme", ErrRejected)
		}
	}
	if !cfg.AllowUnauthenticated && strings.TrimSpace(cfg.CredentialsFile) == "" {
		return fmt.Errorf("%w: credentials file is required", ErrRejected)
	}
	if !validConsumerName(cfg.Stream) {
		return fmt.Errorf("%w: invalid source stream", ErrRejected)
	}
	if cfg.MaxPayloadBytes <= 0 {
		return fmt.Errorf("%w: max payload bytes must be positive", ErrRejected)
	}
	if cfg.MaxPendingPublishes <= 0 {
		return fmt.Errorf("%w: max pending publishes must be positive", ErrRejected)
	}
	return nil
}

// validConsumerName bounds a stream or durable-consumer name to what NATS
// accepts as one token: no subject separators, wildcards, path characters, or
// whitespace. internal/config restates this rule as validMessagingStreamName —
// see the comment above validateMessagingConfig there for why it must, and for
// the test that keeps the two in step.
func validConsumerName(value string) bool {
	if value == "" {
		return false
	}
	return !strings.ContainsAny(value, " .*\\/>\t\r\n")
}

func validSubject(value string, wildcards bool) bool {
	if value == "" || strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") || strings.ContainsAny(value, " \t\r\n") {
		return false
	}
	parts := strings.Split(value, ".")
	for index, part := range parts {
		if part == "" {
			return false
		}
		if part == ">" {
			if !wildcards || index != len(parts)-1 {
				return false
			}
			continue
		}
		if part == "*" {
			if !wildcards {
				return false
			}
			continue
		}
		if strings.ContainsAny(part, "*>") {
			return false
		}
	}
	return true
}
