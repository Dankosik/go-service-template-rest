package config

import (
	"fmt"
	"maps"
	"math"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"
)

// validateConfig is pure computation over an in-memory snapshot: no I/O, and it
// measures at 0ms in the startup log. Cancellation is observed once by the
// caller before this runs rather than between every rule.
//
// Each section's rules live in its own <section>_config.go beside this one, so a
// section that a build profile removes leaves with its file. This file keeps the
// order they run in and the helpers more than one of them shares. A rule that
// spans two sections goes to whichever section depends on the other, and takes
// that section's other half as a parameter — postgres against the request
// budget, outbox against postgres — so no rule outlives the section it is about.
//
// There is deliberately no rule tying health.refresh_interval to
// http.readiness_timeout. The readiness handler answers from cached state and
// performs no I/O, so its budget bounds nothing the refresher does; the interval
// only has to be small relative to the orchestrator's own probe period, which
// this service cannot see.
func validateConfig(cfg *Config, unknownKeys []string) error {
	if unknown := findUnknownKeys(unknownKeys); len(unknown) > 0 {
		return fmt.Errorf("%w: unknown keys: %s", ErrUnknownKey, strings.Join(unknown, ", "))
	}

	if err := validateAppConfig(&cfg.App); err != nil {
		return err
	}
	if err := validateHTTPConfig(&cfg.HTTP); err != nil {
		return err
	}
	// profile:authn-bearer:start
	if err := validateAuthnConfig(&cfg.Authn); err != nil {
		return err
	}
	// profile:authn-bearer:end
	// profile:outbound-auth-oauth2-client-credentials:start
	if err := validateOutboundAuthConfig(&cfg.OutboundAuth); err != nil {
		return err
	}
	// profile:outbound-auth-oauth2-client-credentials:end
	// profile:grpc:start
	if err := validateGRPCConfig(&cfg.GRPC); err != nil {
		return err
	}
	// profile:grpc:end
	if err := validateHealthConfig(cfg.Health); err != nil {
		return err
	}
	if err := validateRuntimeConfig(cfg.Runtime); err != nil {
		return err
	}
	// profile:messaging-nats-jetstream:start
	if err := validateMessagingConfig(&cfg.Messaging); err != nil {
		return err
	}
	// profile:messaging-nats-jetstream:end

	// profile:database-postgres:start
	if err := validatePostgres(cfg.Postgres, cfg.HTTP); err != nil {
		return err
	}
	// profile:database-postgres:end
	// profile:jobs-postgres:start
	if err := validateJobs(cfg.Jobs, cfg.Postgres); err != nil {
		return err
	}
	// profile:jobs-postgres:end
	// profile:webhooks-durable:start
	if err := validateWebhooks(cfg.Webhooks, cfg.Postgres, cfg.Jobs); err != nil {
		return err
	}
	// profile:webhooks-durable:end
	// profile:object-storage:start
	if err := validateObjectStorage(&cfg.ObjectStorage); err != nil {
		return err
	}
	// profile:object-storage:end

	return validateObservabilityConfig(&cfg.Observability)
}

func validateAppConfig(cfg *AppConfig) error {
	cfg.Env = strings.TrimSpace(cfg.Env)
	cfg.Version = strings.TrimSpace(cfg.Version)
	cfg.Commit = strings.TrimSpace(cfg.Commit)
	cfg.InstanceID = strings.TrimSpace(cfg.InstanceID)
	if cfg.Env == "" {
		return fmt.Errorf("%w: app.env cannot be empty", ErrValidate)
	}
	if cfg.Version == "" {
		return fmt.Errorf("%w: app.version cannot be empty", ErrValidate)
	}
	// Rejected rather than defaulted, because the honest unknown is already the
	// build default: an empty value would publish a resource attribute and a
	// diagnostic field that read as a lost identifier rather than an unstamped
	// build.
	if cfg.Commit == "" {
		return fmt.Errorf("%w: app.commit cannot be empty", ErrValidate)
	}
	return nil
}

func validateHealthConfig(cfg HealthConfig) error {
	if err := validateDurationRange("health.refresh_interval", cfg.RefreshInterval, 100*time.Millisecond, time.Minute); err != nil {
		return err
	}
	return validateIntRange("health.failure_threshold", cfg.FailureThreshold, 1, 100)
}

func validateRuntimeConfig(cfg RuntimeConfig) error {
	if math.IsNaN(cfg.MemoryLimitRatio) || math.IsInf(cfg.MemoryLimitRatio, 0) {
		return fmt.Errorf("%w: runtime.memory_limit_ratio must be finite", ErrValidate)
	}
	if cfg.MemoryLimitRatio < 0 || cfg.MemoryLimitRatio > 1 {
		return fmt.Errorf("%w: runtime.memory_limit_ratio must be in range [0,1]", ErrValidate)
	}
	return nil
}

func findUnknownKeys(keys []string) []string {
	unknownSet := make(map[string]struct{})
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		unknownSet[key] = struct{}{}
	}
	return slices.Sorted(maps.Keys(unknownSet))
}

func validateDurationRange(name string, value time.Duration, lowerBound time.Duration, upperBound time.Duration) error {
	if value < lowerBound || value > upperBound {
		return fmt.Errorf("%w: %s must be in range [%s,%s]", ErrValidate, name, lowerBound, upperBound)
	}
	return nil
}

// validateIntRange is validateDurationRange for a whole-number bound. Both name
// the configuration leaf an operator sets and print the range they enforced, so
// a rejection says what to change it to.
func validateIntRange[T ~int | ~int64](name string, value T, lowerBound T, upperBound T) error {
	if value < lowerBound || value > upperBound {
		return fmt.Errorf("%w: %s must be in range [%d,%d]", ErrValidate, name, lowerBound, upperBound)
	}
	return nil
}

// validateHostPortAddr rejects what a listener address cannot be. The numeric
// port is the rule net.SplitHostPort alone does not enforce: it accepts a
// service name, which resolves against /etc/services at bind time rather than
// here, and port 0, which binds an arbitrary port nothing can be told to reach.
// Whether an empty address is allowed belongs to the section, which knows what
// leaving the listener unbound means there.
func validateHostPortAddr(name string, addr string) error {
	_, rawPort, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("%w: %s must be host:port", ErrValidate, name)
	}
	port, err := strconv.ParseUint(rawPort, 10, 16)
	if err != nil || port == 0 {
		return fmt.Errorf("%w: %s port must be in range [1,65535]", ErrValidate, name)
	}
	return nil
}
