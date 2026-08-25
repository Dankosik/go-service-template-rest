// Package messagingconfig owns the pure NATS client configuration rules shared
// by configuration loading and the runtime adapter.
//
// profile:messaging-nats-jetstream:start
package messagingconfig

import (
	"errors"
	"net/url"
	"strings"
)

// ValidateURL rejects a broker URL that the NATS adapter cannot use.
func ValidateURL(raw string, allowPlaintext bool) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return errors.New("messaging URL is invalid")
	}
	if parsed.User != nil {
		return errors.New("messaging URL userinfo is forbidden")
	}
	switch parsed.Scheme {
	case "tls", "wss":
		return nil
	case "nats", "ws":
		if !allowPlaintext {
			return errors.New("plaintext messaging URL requires explicit opt-in")
		}
		return nil
	default:
		return errors.New("messaging URL scheme is unsupported")
	}
}

// ValidStreamOrConsumerName reports whether value is one NATS stream or
// durable-consumer token.
func ValidStreamOrConsumerName(value string) bool {
	return value != "" && !strings.ContainsAny(value, " .*\\/>\t\r\n")
}

// profile:messaging-nats-jetstream:end
