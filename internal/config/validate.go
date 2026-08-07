package config

import (
	"fmt"
	"math"
	"net"

	// profile:authn-oidc-jwt:start
	"net/netip"
	"net/url"

	// profile:authn-oidc-jwt:end
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/observability/otelconfig"
)

// validateConfig is pure computation over an in-memory snapshot: no I/O, and it
// measures at 0ms in the startup log. Cancellation is observed once by the
// caller before this runs rather than between every rule.
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
	// profile:authn-oidc-jwt:start
	if err := validateAuthnConfig(&cfg.Authn); err != nil {
		return err
	}
	// profile:authn-oidc-jwt:end
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
	if err := validatePostgres(cfg.Postgres); err != nil {
		return err
	}
	// profile:database-postgres:end
	// profile:outbox-postgres:start
	if err := validateOutbox(cfg.Outbox, cfg.Postgres); err != nil {
		return err
	}
	// profile:outbox-postgres:end

	if err := validateObservabilityConfig(&cfg.Observability); err != nil {
		return err
	}
	return validateCrossSectionBudgets(*cfg)
}

// profile:authn-oidc-jwt:start

// validateAuthnConfig restates the trust rules that oidcjwt.NewPolicy enforces,
// so a bad issuer, audience, or proxy CIDR stops configuration load instead of
// surfacing at authn startup. The duplication is forced rather than chosen:
// oidcjwt's validProviderURL owns why it cannot be removed, and
// TestPolicyRulesMatchConfigValidation is what fails the build when a rule
// tightened on one side is not tightened on the other.
//
// It also rewrites what it accepts, which is more than the trimming the other
// validators here do: trusted_proxy_cidrs is stored back in masked canonical
// form, so 10.0.0.5/8 is held as 10.0.0.0/8. That rewritten string is the one
// startup_authn.go hands NewPolicy, which masks again and so agrees; a reader
// changing either side is changing a value the other one parses.
func validateAuthnConfig(cfg *AuthnConfig) error {
	cfg.Issuer = strings.TrimSpace(cfg.Issuer)
	cfg.Audience = strings.TrimSpace(cfg.Audience)
	cfg.TrustedProxyCIDRs = strings.TrimSpace(cfg.TrustedProxyCIDRs)

	if !validAuthnIssuerURL(cfg.Issuer) {
		return fmt.Errorf(
			"%w: authn.issuer must be an absolute HTTPS URL without user info, query, or fragment",
			ErrValidate,
		)
	}
	if cfg.Audience == "" {
		return fmt.Errorf("%w: authn.audience cannot be empty", ErrValidate)
	}

	canonical := make([]string, 0)
	seen := make(map[netip.Prefix]struct{})
	for value := range strings.SplitSeq(cfg.TrustedProxyCIDRs, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return fmt.Errorf("%w: authn.trusted_proxy_cidrs contains an invalid CIDR", ErrValidate)
		}
		prefix = prefix.Masked()
		if _, exists := seen[prefix]; exists {
			return fmt.Errorf("%w: authn.trusted_proxy_cidrs contains a duplicate CIDR", ErrValidate)
		}
		seen[prefix] = struct{}{}
		canonical = append(canonical, prefix.String())
	}
	if len(canonical) == 0 {
		return fmt.Errorf("%w: authn.trusted_proxy_cidrs must contain at least one CIDR", ErrValidate)
	}
	cfg.TrustedProxyCIDRs = strings.Join(canonical, ",")
	return nil
}

// validAuthnIssuerURL is the copy of oidcjwt's validProviderURL this package is
// forced to hold, kept term for term and in the same order so the two can be read
// against each other line by line. Tighten one and you must tighten the other.
func validAuthnIssuerURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	return err == nil &&
		parsed != nil &&
		parsed.IsAbs() &&
		strings.EqualFold(parsed.Scheme, "https") &&
		parsed.Host != "" &&
		parsed.Hostname() != "" &&
		parsed.Opaque == "" &&
		parsed.User == nil &&
		parsed.RawQuery == "" &&
		!parsed.ForceQuery &&
		parsed.Fragment == ""
}

// profile:authn-oidc-jwt:end

// validateCrossSectionBudgets rejects settings that are each individually legal
// but incoherent together. A budget that only holds inside one section is not a
// budget: it is a number that happens to look like one.
func validateCrossSectionBudgets(cfg Config) error {
	// profile:database-postgres:start
	if cfg.Postgres.Enabled {
		if cfg.Postgres.StatementTimeout > cfg.HTTP.RequestTimeout {
			return fmt.Errorf(
				"%w: postgres.statement_timeout must be <= http.request_timeout (%s)",
				ErrValidate,
				cfg.HTTP.RequestTimeout,
			)
		}
		// Strictly less, not at most: a caller that spends the whole request
		// budget waiting for a connection has nothing left to run a query with,
		// which makes the wait indistinguishable from the unbounded one this
		// budget replaced.
		if cfg.Postgres.AcquireTimeout >= cfg.HTTP.RequestTimeout {
			return fmt.Errorf(
				"%w: postgres.acquire_timeout must be < http.request_timeout (%s) so a caller that waited still has budget to query",
				ErrValidate,
				cfg.HTTP.RequestTimeout,
			)
		}
	}
	// profile:database-postgres:end

	// There is deliberately no rule tying health.refresh_interval to
	// http.readiness_timeout. The readiness handler answers from cached state and
	// performs no I/O, so its budget bounds nothing the refresher does; the
	// interval only has to be small relative to the orchestrator's own probe
	// period, which this service cannot see.
	if !cfg.Observability.Pprof.Enabled {
		return nil
	}
	if cfg.Observability.Metrics.Addr == "" {
		return fmt.Errorf(
			"%w: observability.pprof.enabled requires observability.metrics.addr, which serves the diagnostics listener",
			ErrValidate,
		)
	}
	return nil
}

func validateHealthConfig(cfg HealthConfig) error {
	if err := validateDurationRange("health.refresh_interval", cfg.RefreshInterval, 100*time.Millisecond, time.Minute); err != nil {
		return err
	}
	if cfg.FailureThreshold < 1 || cfg.FailureThreshold > 100 {
		return fmt.Errorf("%w: health.failure_threshold must be in range [1,100]", ErrValidate)
	}
	return nil
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

func validateHTTPConfig(cfg *HTTPConfig) error {
	cfg.Addr = strings.TrimSpace(cfg.Addr)
	if cfg.Addr == "" {
		return fmt.Errorf("%w: http.addr cannot be empty", ErrValidate)
	}
	if err := validateDurationRange("http.grace_period", cfg.GracePeriod, time.Second, 10*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("http.shutdown_timeout", cfg.ShutdownTimeout, time.Second, 10*time.Minute); err != nil {
		return err
	}
	// The drain is one stage inside the grace period, never the whole of it. How
	// much the stages after it need is owned by the composition root, which knows
	// their ceilings; see validateShutdownGraceBudget.
	if cfg.ShutdownTimeout > cfg.GracePeriod {
		return fmt.Errorf(
			"%w: http.shutdown_timeout must be <= http.grace_period (%s)",
			ErrValidate,
			cfg.GracePeriod,
		)
	}
	if err := validateDurationRange("http.readiness_timeout", cfg.ReadinessTimeout, 100*time.Millisecond, 30*time.Second); err != nil {
		return err
	}
	if err := validateDurationRange("http.readiness_propagation_delay", cfg.ReadinessPropagationDelay, 0, cfg.ShutdownTimeout); err != nil {
		return err
	}
	if err := validateDurationRange("http.read_header_timeout", cfg.ReadHeaderTimeout, 100*time.Millisecond, 5*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("http.read_timeout", cfg.ReadTimeout, 100*time.Millisecond, 5*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("http.request_timeout", cfg.RequestTimeout, 100*time.Millisecond, 10*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("http.write_timeout", cfg.WriteTimeout, 100*time.Millisecond, 10*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("http.idle_timeout", cfg.IdleTimeout, 100*time.Millisecond, 24*time.Hour); err != nil {
		return err
	}
	if err := validateHTTPReadinessWriteTimeout(*cfg); err != nil {
		return err
	}
	if err := validateHTTPRequestWriteTimeout(*cfg); err != nil {
		return err
	}
	if err := validateHTTPShutdownBudget(*cfg); err != nil {
		return err
	}
	if err := validateHTTPCapacityBounds(*cfg); err != nil {
		return err
	}
	return nil
}

// validateHTTPCapacityBounds owns the four settings that decide how much work
// and how many callers one process admits. They are grouped because they are
// read together — see validateHTTPConnectionCeiling for the one relationship
// between them that a range check cannot express.
func validateHTTPCapacityBounds(cfg HTTPConfig) error {
	if cfg.MaxHeaderBytes <= 0 {
		return fmt.Errorf("%w: http.max_header_bytes must be > 0", ErrValidate)
	}
	if cfg.MaxBodyBytes <= 0 {
		return fmt.Errorf("%w: http.max_body_bytes must be > 0", ErrValidate)
	}
	if cfg.MaxInFlight < 0 || cfg.MaxInFlight > 100_000 {
		return fmt.Errorf("%w: http.max_in_flight must be in range [0,100000]", ErrValidate)
	}
	if cfg.MaxConnections < 0 || cfg.MaxConnections > 1_000_000 {
		return fmt.Errorf("%w: http.max_connections must be in range [0,1000000]", ErrValidate)
	}
	return validateHTTPConnectionCeiling(cfg)
}

// validateHTTPConnectionCeiling keeps the accept cap above the concurrency the
// service advertises.
//
// The two bound different things and only make sense together. max_in_flight is
// how many requests may execute a handler; max_connections is how many callers
// may be attached at all. Setting the second below the first means the service
// cannot reach its own concurrency ceiling, and the excess callers do not get
// the 503 with a Retry-After that shedding would have given them — they wait in
// the kernel backlog and time out at connect, which is a worse signal for a
// client and an invisible one for an operator.
func validateHTTPConnectionCeiling(cfg HTTPConfig) error {
	if cfg.MaxConnections == 0 || cfg.MaxConnections >= cfg.MaxInFlight {
		return nil
	}
	return fmt.Errorf(
		"%w: http.max_connections must be >= http.max_in_flight (%d)",
		ErrValidate,
		cfg.MaxInFlight,
	)
}

// profile:grpc:start

func validateGRPCConfig(cfg *GRPCConfig) error {
	server := &cfg.Server
	server.Addr = strings.TrimSpace(server.Addr)
	server.TransportSecurity = strings.ToLower(strings.TrimSpace(server.TransportSecurity))
	server.TLS.CertFile = strings.TrimSpace(server.TLS.CertFile)
	server.TLS.KeyFile = strings.TrimSpace(server.TLS.KeyFile)

	if err := validateGRPCCapacityBounds(*server); err != nil {
		return err
	}
	// grpcx.validateConfig re-proves this rate and this threshold when the server
	// is built, so a direct package user cannot recover an unbounded default.
	// Changing either rule here means changing it there too;
	// internal/infra/grpc/config_parity_test.go holds the two to one answer.
	if math.IsNaN(server.AccessLogSuccessSampleRate) ||
		math.IsInf(server.AccessLogSuccessSampleRate, 0) ||
		server.AccessLogSuccessSampleRate < 0 ||
		server.AccessLogSuccessSampleRate > 1 {
		return fmt.Errorf(
			"%w: grpc.server.access_log_success_sample_rate must be finite and in range [0,1]",
			ErrValidate,
		)
	}
	if server.AccessLogSlowThreshold < 0 {
		return fmt.Errorf(
			"%w: grpc.server.access_log_slow_threshold must be non-negative",
			ErrValidate,
		)
	}
	switch server.TransportSecurity {
	case "", "plaintext", secureTransportTLS:
	default:
		return fmt.Errorf(
			"%w: grpc.server.transport_security must be one of plaintext or tls",
			ErrValidate,
		)
	}
	if !server.Enabled {
		return nil
	}
	if err := validateGRPCAddress(server.Addr); err != nil {
		return err
	}
	if err := validateEnabledGRPCTransport(server); err != nil {
		return err
	}
	// profile:authn-oidc-jwt:start
	if err := validateGRPCAuthnTransport(server); err != nil {
		return err
	}
	// profile:authn-oidc-jwt:end
	return nil
}

func validateEnabledGRPCTransport(server *GRPCServerConfig) error {
	switch server.TransportSecurity {
	case "plaintext":
		if !server.AllowPlaintext {
			return fmt.Errorf(
				"%w: grpc.server.allow_plaintext must be true when transport_security is plaintext",
				ErrValidate,
			)
		}
		if server.TLS.CertFile != "" || server.TLS.KeyFile != "" {
			return fmt.Errorf(
				"%w: grpc.server.tls cert_file and key_file must be empty when transport_security is plaintext",
				ErrValidate,
			)
		}
	case secureTransportTLS:
		if server.AllowPlaintext {
			return fmt.Errorf(
				"%w: grpc.server.allow_plaintext must be false when transport_security is tls",
				ErrValidate,
			)
		}
		if server.TLS.CertFile == "" || server.TLS.KeyFile == "" {
			return fmt.Errorf(
				"%w: grpc.server.tls cert_file and key_file are required when transport_security is tls",
				ErrValidate,
			)
		}
	default:
		return fmt.Errorf(
			"%w: grpc.server.transport_security is required when grpc.server.enabled is true",
			ErrValidate,
		)
	}
	return nil
}

// profile:authn-oidc-jwt:start

func validateGRPCAuthnTransport(server *GRPCServerConfig) error {
	if server.TransportSecurity != secureTransportTLS {
		return fmt.Errorf(
			"%w: authn OIDC profile requires grpc.server.transport_security=tls",
			ErrValidate,
		)
	}
	return nil
}

// profile:authn-oidc-jwt:end

func validateGRPCCapacityBounds(cfg GRPCServerConfig) error {
	if cfg.MaxConnections <= 0 || cfg.MaxConnections > 1_000_000 {
		return fmt.Errorf("%w: grpc.server.max_connections must be in range [1,1000000]", ErrValidate)
	}
	if cfg.MaxConcurrentRPCs <= 0 || cfg.MaxConcurrentRPCs > 100_000 {
		return fmt.Errorf("%w: grpc.server.max_concurrent_rpcs must be in range [1,100000]", ErrValidate)
	}
	if cfg.MaxConcurrentStreams <= 0 || cfg.MaxConcurrentStreams > 100_000 {
		return fmt.Errorf("%w: grpc.server.max_concurrent_streams must be in range [1,100000]", ErrValidate)
	}
	if cfg.MaxHeaderListBytes <= 0 || cfg.MaxHeaderListBytes > math.MaxInt32 {
		return fmt.Errorf(
			"%w: grpc.server.max_header_list_bytes must be in range [1,%d]",
			ErrValidate,
			math.MaxInt32,
		)
	}
	if cfg.MaxReceiveMessageBytes <= 0 || cfg.MaxReceiveMessageBytes > math.MaxInt32 {
		return fmt.Errorf(
			"%w: grpc.server.max_receive_message_bytes must be in range [1,%d]",
			ErrValidate,
			math.MaxInt32,
		)
	}
	if cfg.MaxSendMessageBytes <= 0 || cfg.MaxSendMessageBytes > math.MaxInt32 {
		return fmt.Errorf(
			"%w: grpc.server.max_send_message_bytes must be in range [1,%d]",
			ErrValidate,
			math.MaxInt32,
		)
	}

	connectionCapacity := uint64(cfg.MaxConnections) * uint64(cfg.MaxConcurrentStreams)
	if uint64(cfg.MaxConcurrentRPCs) > connectionCapacity {
		return fmt.Errorf(
			"%w: grpc.server.max_concurrent_rpcs must be <= max_connections * max_concurrent_streams (%d)",
			ErrValidate,
			connectionCapacity,
		)
	}
	return nil
}

func validateGRPCAddress(addr string) error {
	if addr == "" {
		return fmt.Errorf("%w: grpc.server.addr cannot be empty when grpc.server.enabled is true", ErrValidate)
	}
	_, rawPort, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("%w: grpc.server.addr must be host:port", ErrValidate)
	}
	port, err := strconv.ParseUint(rawPort, 10, 16)
	if err != nil || port == 0 {
		return fmt.Errorf("%w: grpc.server.addr port must be in range [1,65535]", ErrValidate)
	}
	return nil
}

// profile:grpc:end

func validateObservabilityConfig(cfg *ObservabilityConfig) error {
	cfg.Metrics.Addr = strings.TrimSpace(cfg.Metrics.Addr)
	if cfg.Metrics.Addr != "" {
		_, rawPort, err := net.SplitHostPort(cfg.Metrics.Addr)
		if err != nil {
			return fmt.Errorf("%w: observability.metrics.addr must be host:port", ErrValidate)
		}
		port, err := strconv.ParseUint(rawPort, 10, 16)
		if err != nil || port == 0 {
			return fmt.Errorf("%w: observability.metrics.addr port must be in range [1,65535]", ErrValidate)
		}
	}
	cfg.OTel.ServiceName = strings.TrimSpace(cfg.OTel.ServiceName)
	cfg.OTel.TracesSampler = strings.TrimSpace(cfg.OTel.TracesSampler)
	cfg.OTel.Exporter.OTLPEndpoint = strings.TrimSpace(cfg.OTel.Exporter.OTLPEndpoint)
	cfg.OTel.Exporter.OTLPMetricsEndpoint = strings.TrimSpace(cfg.OTel.Exporter.OTLPMetricsEndpoint)
	if cfg.OTel.ServiceName == "" {
		return fmt.Errorf("%w: observability.otel.service_name cannot be empty", ErrValidate)
	}
	return validateObservabilitySampler(cfg.OTel.TracesSampler, cfg.OTel.TracesSamplerArg)
}

// validateObservabilitySampler prefixes the shared sampler rules with the
// section that owns them, so a rejection names the setting an operator can edit.
func validateObservabilitySampler(sampler string, samplerArg float64) error {
	if err := otelconfig.ValidateTraceSampler(sampler, samplerArg); err != nil {
		return fmt.Errorf("%w: observability.otel.%w", ErrValidate, err)
	}
	return nil
}

func findUnknownKeys(keys []string) []string {
	unknownSet := make(map[string]struct{})
	unknown := make([]string, 0)
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, ok := unknownSet[key]; ok {
			continue
		}
		unknownSet[key] = struct{}{}
		unknown = append(unknown, key)
	}
	sort.Strings(unknown)
	return unknown
}

// profile:database-postgres:start
func validatePostgres(cfg PostgresConfig) error {
	if cfg.Enabled && strings.TrimSpace(cfg.DSN) == "" {
		return fmt.Errorf("%w: postgres.dsn is required when postgres.enabled=true", ErrSecretPolicy)
	}

	if err := validateDurationRange("postgres.connect_timeout", cfg.ConnectTimeout, 100*time.Millisecond, 10*time.Second); err != nil {
		return err
	}
	if err := validateDurationRange("postgres.healthcheck_timeout", cfg.HealthcheckTimeout, 100*time.Millisecond, 10*time.Second); err != nil {
		return err
	}
	if err := validateDurationRange("postgres.migration_timeout", cfg.MigrationTimeout, time.Second, time.Hour); err != nil {
		return err
	}
	if err := validateDurationRange(
		"postgres.migration_statement_timeout",
		cfg.MigrationStatementTimeout,
		100*time.Millisecond,
		cfg.MigrationTimeout,
	); err != nil {
		return err
	}
	if err := validateDurationRange(
		"postgres.migration_lock_timeout",
		cfg.MigrationLockTimeout,
		100*time.Millisecond,
		cfg.MigrationTimeout,
	); err != nil {
		return err
	}
	if cfg.MigrationLockTimeout >= cfg.MigrationTimeout {
		return fmt.Errorf(
			"%w: postgres.migration_lock_timeout must be less than postgres.migration_timeout to reserve cleanup time",
			ErrValidate,
		)
	}
	if err := validateDurationRange("postgres.acquire_timeout", cfg.AcquireTimeout, 10*time.Millisecond, 30*time.Second); err != nil {
		return err
	}
	if cfg.MaxOpenConns < 1 || cfg.MaxOpenConns > 500 {
		return fmt.Errorf("%w: postgres.max_open_conns must be in range [1,500]", ErrValidate)
	}
	if cfg.MinIdleConns < 0 || cfg.MinIdleConns > cfg.MaxOpenConns {
		return fmt.Errorf(
			"%w: postgres.min_idle_conns must be in range [0,postgres.max_open_conns] (%d)",
			ErrValidate,
			cfg.MaxOpenConns,
		)
	}
	if err := validateDurationRange("postgres.conn_max_lifetime", cfg.ConnMaxLifetime, time.Minute, 24*time.Hour); err != nil {
		return err
	}
	if err := validateDurationRange("postgres.statement_timeout", cfg.StatementTimeout, 100*time.Millisecond, 10*time.Minute); err != nil {
		return err
	}

	return nil
}

// profile:database-postgres:end

// profile:outbox-postgres:start

// Copies of the outbox ceilings, restated so an operator is rejected at load time
// instead of at relay startup. postgresoutbox owns the values and the reasoning;
// its ceiling block in relay_config.go says why depguard makes this a copy rather
// than an import, and cmd/outbox-relay/internal/bootstrap holds the two sides
// together.
//
// A relay budget added to validateOutbox below must be added to
// ValidateRelayConfig too. Nothing here can check that; the composition root's
// walk over OutboxConfig is what fails when only one side learns about it.
//
// The lease relation below is this package's own, not a copy: it also charges
// the acquire and statement budgets a RelayConfig never carries, so it stays
// the stricter of the two by design and is pinned to nothing.
const (
	outboxPublisherJoinTimeout  = time.Second
	outboxMaxBatchSize          = 1000
	outboxMaxPublishConcurrency = 256
	outboxMaxAttempts           = 100
	outboxMaxCleanupBatchSize   = 10_000
)

func validateOutbox(cfg OutboxConfig, postgres PostgresConfig) error {
	if cfg.Enabled && !postgres.Enabled {
		return fmt.Errorf("%w: outbox.enabled requires postgres.enabled", ErrValidate)
	}
	// Claim, finalization, and periodic maintenance are separate statements that
	// share the pool with health checks and, in an API process, with request
	// traffic. A single-connection pool leaves none of them any headroom.
	if cfg.Enabled && postgres.MaxOpenConns < 2 {
		return fmt.Errorf("%w: outbox.enabled requires postgres.max_open_conns >= 2", ErrValidate)
	}
	for _, duration := range []struct {
		name  string
		value time.Duration
	}{
		{name: "outbox.poll_interval", value: cfg.PollInterval},
		{name: "outbox.publish_timeout", value: cfg.PublishTimeout},
		{name: "outbox.lease_duration", value: cfg.LeaseDuration},
		{name: "outbox.retry_base", value: cfg.RetryBase},
		{name: "outbox.retry_max", value: cfg.RetryMax},
		{name: "outbox.observation_interval", value: cfg.ObservationInterval},
		{name: "outbox.cleanup_interval", value: cfg.CleanupInterval},
		{name: "outbox.published_retention", value: cfg.PublishedRetention},
		{name: "outbox.drain_timeout", value: cfg.DrainTimeout},
	} {
		if duration.value <= 0 {
			return fmt.Errorf("%w: %s must be positive", ErrValidate, duration.name)
		}
	}
	for _, bound := range []struct {
		name string
		value,
		high int
	}{
		{name: "outbox.max_attempts", value: cfg.MaxAttempts, high: outboxMaxAttempts},
		{name: "outbox.batch_size", value: cfg.BatchSize, high: outboxMaxBatchSize},
		{name: "outbox.publish_concurrency", value: cfg.PublishConcurrency, high: outboxMaxPublishConcurrency},
		{name: "outbox.cleanup_batch_size", value: cfg.CleanupBatchSize, high: outboxMaxCleanupBatchSize},
	} {
		if bound.value < 1 || bound.value > bound.high {
			return fmt.Errorf("%w: %s must be in range [1,%d]", ErrValidate, bound.name, bound.high)
		}
	}
	if cfg.RetryMax < cfg.RetryBase {
		return fmt.Errorf("%w: outbox.retry_max must be >= outbox.retry_base", ErrValidate)
	}
	if !durationExceedsSum(
		cfg.LeaseDuration,
		cfg.PublishTimeout,
		outboxPublisherJoinTimeout,
		postgres.AcquireTimeout,
		postgres.StatementTimeout,
	) {
		return fmt.Errorf(
			"%w: outbox.lease_duration must exceed publish, publisher-join, postgres acquire, and statement budgets",
			ErrValidate,
		)
	}
	return nil
}

func durationExceedsSum(total time.Duration, parts ...time.Duration) bool {
	for _, part := range parts {
		if total <= part {
			return false
		}
		total -= part
	}
	return true
}

// profile:outbox-postgres:end

func validateHTTPShutdownBudget(cfg HTTPConfig) error {
	effectiveDrainBudget := cfg.ShutdownTimeout - cfg.ReadinessPropagationDelay
	if effectiveDrainBudget <= 0 {
		return fmt.Errorf("%w: http.readiness_propagation_delay must be less than http.shutdown_timeout", ErrValidate)
	}
	if cfg.WriteTimeout > effectiveDrainBudget {
		return fmt.Errorf(
			"%w: http.write_timeout must be <= effective drain budget after readiness propagation (%s)",
			ErrValidate,
			effectiveDrainBudget,
		)
	}
	return nil
}

func validateHTTPReadinessWriteTimeout(cfg HTTPConfig) error {
	if cfg.ReadinessTimeout > cfg.WriteTimeout {
		return fmt.Errorf("%w: http.readiness_timeout must be <= http.write_timeout", ErrValidate)
	}
	return nil
}

// validateHTTPRequestWriteTimeout keeps the handler budget inside the response
// write deadline. A request budget larger than http.write_timeout expires only
// after the connection can no longer carry a response, so the timeout would be
// reported to the client as a dropped connection instead of a 504.
func validateHTTPRequestWriteTimeout(cfg HTTPConfig) error {
	if cfg.RequestTimeout > cfg.WriteTimeout {
		return fmt.Errorf("%w: http.request_timeout must be <= http.write_timeout", ErrValidate)
	}
	return nil
}

func validateDurationRange(name string, value time.Duration, lowerBound time.Duration, upperBound time.Duration) error {
	if value < lowerBound || value > upperBound {
		return fmt.Errorf("%w: %s must be in range [%s,%s]", ErrValidate, name, lowerBound, upperBound)
	}
	return nil
}
