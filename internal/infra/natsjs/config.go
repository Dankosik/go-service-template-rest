package natsjs

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	HeaderLimitBytes      = 8 << 10
	ResidentDeliveryLimit = 64 << 20

	operationTimeout = 5 * time.Second
	// maxHeaderValueBytes bounds every identity carried in a message header, so
	// the encoded envelope stays inside HeaderLimitBytes.
	maxHeaderValueBytes = 256
	// maxRetryDelays keeps the configured delay sequence, and so a message's
	// attempt budget, small enough to stay operator-legible.
	maxRetryDelays = 9
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

type WorkerConfig struct {
	Consumer             string
	FilterSubject        string
	DeadLetterSubject    string
	MaxConcurrency       int
	MaxDeliveryBytes     int
	HandlerTimeout       time.Duration
	RetryDelays          []time.Duration
	DeadLetterRetryDelay time.Duration
}

// ValidateConfig rejects a client configuration this package cannot connect
// under. [Connect] applies it before touching the network, so a settings fault
// never reaches the broker.
//
// It is exported for the same reason [ValidateWorkerConfig] is: internal/config
// restates these rules and cannot import this package, so the composition root
// is the only place that can hold both and prove they still describe the same
// configuration. See the comment above validateMessagingConfig in
// internal/config/messaging.go for that arrangement and the test that pins it.
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

func ValidateWorkerConfig(cfg WorkerConfig, maxPayloadBytes int) error {
	if !validConsumerName(cfg.Consumer) {
		return fmt.Errorf("%w: invalid durable consumer", ErrRejected)
	}
	if !validSubject(cfg.FilterSubject, true) {
		return fmt.Errorf("%w: invalid filter subject", ErrRejected)
	}
	if !validSubject(cfg.DeadLetterSubject, false) {
		return fmt.Errorf("%w: invalid dead-letter subject", ErrRejected)
	}
	if subjectMatches(cfg.FilterSubject, cfg.DeadLetterSubject) {
		return fmt.Errorf("%w: source filter and dead-letter subject overlap", ErrRejected)
	}
	if cfg.MaxConcurrency <= 0 {
		return fmt.Errorf("%w: max concurrency must be positive, got %d", ErrRejected, cfg.MaxConcurrency)
	}
	if cfg.MaxDeliveryBytes <= 0 {
		return fmt.Errorf("%w: max delivery bytes must be positive, got %d", ErrRejected, cfg.MaxDeliveryBytes)
	}
	if cfg.MaxDeliveryBytes < maxPayloadBytes+HeaderLimitBytes {
		return fmt.Errorf(
			"%w: max delivery bytes (%d) cannot contain the configured envelope (%d payload + %d header)",
			ErrRejected, cfg.MaxDeliveryBytes, maxPayloadBytes, HeaderLimitBytes,
		)
	}
	// Every concurrent handler can hold one whole delivery, so the worker's peak
	// resident wire data is the product of the two.
	if cfg.MaxConcurrency > ResidentDeliveryLimit/cfg.MaxDeliveryBytes {
		return fmt.Errorf(
			"%w: max concurrency (%d) times max delivery bytes (%d) exceeds the %d byte resident limit",
			ErrRejected, cfg.MaxConcurrency, cfg.MaxDeliveryBytes, ResidentDeliveryLimit,
		)
	}
	if cfg.HandlerTimeout <= 0 {
		return fmt.Errorf("%w: handler timeout must be positive, got %s", ErrRejected, cfg.HandlerTimeout)
	}
	if len(cfg.RetryDelays) < 1 || len(cfg.RetryDelays) > maxRetryDelays {
		return fmt.Errorf(
			"%w: retry delays must contain one to %d values, got %d",
			ErrRejected, maxRetryDelays, len(cfg.RetryDelays),
		)
	}
	for index, delay := range cfg.RetryDelays {
		if delay <= 0 {
			return fmt.Errorf("%w: retry delay %d must be positive, got %s", ErrRejected, index, delay)
		}
	}
	if cfg.DeadLetterRetryDelay <= 0 {
		return fmt.Errorf(
			"%w: dead-letter retry delay must be positive, got %s",
			ErrRejected, cfg.DeadLetterRetryDelay,
		)
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

// subjectMatches reports whether subject falls inside filter under NATS subject
// wildcards, which is why it is not a string comparison: "*" matches exactly one
// token and ">" matches every remaining one, so "orders.*" and "orders.>" both
// cover "orders.created". [ValidateWorkerConfig] uses it to reject a
// dead-letter subject the worker's own filter would consume, which would make
// every dead-lettered message a delivery to itself.
func subjectMatches(filter, subject string) bool {
	filterParts := strings.Split(filter, ".")
	subjectParts := strings.Split(subject, ".")
	for index, part := range filterParts {
		if part == ">" {
			return true
		}
		if index >= len(subjectParts) {
			return false
		}
		if part != "*" && part != subjectParts[index] {
			return false
		}
	}
	return len(filterParts) == len(subjectParts)
}
