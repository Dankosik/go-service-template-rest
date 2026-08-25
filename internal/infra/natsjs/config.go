package natsjs

import (
	"fmt"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/messagingconfig"
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
}

// ValidateConfig rejects a client configuration this package cannot connect
// under. [Connect] applies it before touching the network, so a settings fault
// never reaches the broker.
//
// It is exported for the same reason [ValidateWorkerConfig] is: direct adapter
// callers receive the same fail-closed rules configuration loading applies.
func ValidateConfig(cfg Config) error {
	if len(cfg.URLs) == 0 {
		return fmt.Errorf("%w: messaging URLs are required", ErrRejected)
	}
	for _, raw := range cfg.URLs {
		if err := messagingconfig.ValidateURL(raw, cfg.AllowPlaintext); err != nil {
			return fmt.Errorf("%w: %w", ErrRejected, err)
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
	return nil
}

// validConsumerName keeps the adapter's local vocabulary while the pure leaf
// owns the rule shared with configuration loading.
func validConsumerName(value string) bool {
	return messagingconfig.ValidStreamOrConsumerName(value)
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
