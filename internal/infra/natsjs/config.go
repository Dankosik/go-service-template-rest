// Package natsjs provides the concrete NATS JetStream durable-messaging pack.
package natsjs

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	HeaderLimitBytes      = 8 << 10
	ResidentDeliveryLimit = 64 << 20

	operationTimeout = 5 * time.Second
)

var (
	ErrRejected  = errors.New("messaging operation rejected")
	ErrAmbiguous = errors.New("messaging operation outcome ambiguous")
	ErrCapacity  = errors.New("messaging producer capacity exhausted")
	ErrDraining  = errors.New("messaging runtime draining")
	ErrTerminal  = errors.New("messaging runtime terminal failure")
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

func validateConfig(cfg Config) error {
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
	if cfg.MaxConcurrency <= 0 || cfg.MaxDeliveryBytes <= 0 {
		return fmt.Errorf("%w: worker concurrency and delivery bytes must be positive", ErrRejected)
	}
	if cfg.MaxDeliveryBytes < maxPayloadBytes+HeaderLimitBytes {
		return fmt.Errorf("%w: delivery bound cannot contain the configured envelope", ErrRejected)
	}
	if cfg.MaxConcurrency > ResidentDeliveryLimit/cfg.MaxDeliveryBytes {
		return fmt.Errorf("%w: worker resident delivery bound exceeds %d bytes", ErrRejected, ResidentDeliveryLimit)
	}
	if cfg.HandlerTimeout <= 0 {
		return fmt.Errorf("%w: handler timeout must be positive", ErrRejected)
	}
	if len(cfg.RetryDelays) < 1 || len(cfg.RetryDelays) > 9 {
		return fmt.Errorf("%w: retry delays must contain one to nine values", ErrRejected)
	}
	for _, delay := range cfg.RetryDelays {
		if delay <= 0 {
			return fmt.Errorf("%w: retry delays must be positive", ErrRejected)
		}
	}
	if cfg.DeadLetterRetryDelay <= 0 {
		return fmt.Errorf("%w: dead-letter retry delay must be positive", ErrRejected)
	}
	return nil
}

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
