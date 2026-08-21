package natsjs

import (
	"fmt"
	"strings"
	"time"
)

const (
	ResidentDeliveryLimit = 64 << 20
	// maxRetryDelays keeps the configured delay sequence, and so a message's
	// attempt budget, small enough to stay operator-legible.
	maxRetryDelays = 9
)

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

// DefaultWorkerConfig keeps delivery policy out of operator configuration.
// Reopen these defaults only for a measured handler or recovery requirement.
func DefaultWorkerConfig(consumer, filterSubject, deadLetterSubject string, maxConcurrency, maxPayloadBytes int) WorkerConfig {
	return WorkerConfig{
		Consumer: consumer, FilterSubject: filterSubject, DeadLetterSubject: deadLetterSubject,
		MaxConcurrency: maxConcurrency, MaxDeliveryBytes: maxPayloadBytes + HeaderLimitBytes,
		HandlerTimeout: 30 * time.Second,
		RetryDelays: []time.Duration{
			time.Second,
			5 * time.Second,
			30 * time.Second,
			2 * time.Minute,
		},
		DeadLetterRetryDelay: 30 * time.Second,
	}
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
