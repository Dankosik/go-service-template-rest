package config

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Dankosik/billing-service/internal/observability/otelconfig"
	"github.com/knadh/koanf/v2"
)

type validationOptions struct {
	Strict                bool
	AdditionalUnknownKeys []string
}

type validationResult struct {
	UnknownKeyWarnings []string
}

func validateConfig(ctx context.Context, k *koanf.Koanf, cfg *Config, opts validationOptions) (validationResult, error) {
	if err := checkValidateContext(ctx); err != nil {
		return validationResult{}, err
	}

	result, err := validateUnknownConfigKeys(k, opts)
	if err != nil {
		return validationResult{}, err
	}
	if err := checkValidateContext(ctx); err != nil {
		return result, err
	}

	if err := validateAppConfig(cfg.App); err != nil {
		return result, err
	}
	if err := validateHTTPConfig(cfg.HTTP); err != nil {
		return result, err
	}
	if err := checkValidateContext(ctx); err != nil {
		return result, err
	}

	if err := validateDatastoreConfig(*cfg); err != nil {
		return result, err
	}
	if err := validateAuthorityRuntimeConfig(*cfg); err != nil {
		return result, err
	}
	if err := validateMicroleaseRuntimeConfig(*cfg); err != nil {
		return result, err
	}
	if err := validateReadinessProbeBudgets(*cfg); err != nil {
		return result, err
	}
	if err := checkValidateContext(ctx); err != nil {
		return result, err
	}

	if err := validateObservabilityConfig(cfg.Observability); err != nil {
		return result, err
	}

	return result, nil
}

func validateUnknownConfigKeys(k *koanf.Koanf, opts validationOptions) (validationResult, error) {
	unknownKeys := findUnknownKeys(k, opts.AdditionalUnknownKeys)
	if len(unknownKeys) == 0 {
		return validationResult{}, nil
	}
	if opts.Strict {
		return validationResult{}, fmt.Errorf("%w: unknown keys: %s", ErrStrictUnknownKey, strings.Join(unknownKeys, ", "))
	}
	return validationResult{UnknownKeyWarnings: unknownKeys}, nil
}

func validateAppConfig(cfg AppConfig) error {
	if strings.TrimSpace(cfg.Env) == "" {
		return fmt.Errorf("%w: app.env cannot be empty", ErrValidate)
	}
	if strings.TrimSpace(cfg.Version) == "" {
		return fmt.Errorf("%w: app.version cannot be empty", ErrValidate)
	}
	return nil
}

func validateHTTPConfig(cfg HTTPConfig) error {
	if strings.TrimSpace(cfg.Addr) == "" {
		return fmt.Errorf("%w: http.addr cannot be empty", ErrValidate)
	}
	if err := validateDurationRange("http.shutdown_timeout", cfg.ShutdownTimeout, time.Second, 10*time.Minute); err != nil {
		return err
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
	if err := validateDurationRange("http.write_timeout", cfg.WriteTimeout, 100*time.Millisecond, 10*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("http.idle_timeout", cfg.IdleTimeout, 100*time.Millisecond, 24*time.Hour); err != nil {
		return err
	}
	if err := validateHTTPReadinessWriteTimeout(cfg); err != nil {
		return err
	}
	if err := validateHTTPShutdownBudget(cfg); err != nil {
		return err
	}
	if cfg.MaxHeaderBytes <= 0 {
		return fmt.Errorf("%w: http.max_header_bytes must be > 0", ErrValidate)
	}
	if cfg.MaxBodyBytes <= 0 {
		return fmt.Errorf("%w: http.max_body_bytes must be > 0", ErrValidate)
	}
	return nil
}

func validateDatastoreConfig(cfg Config) error {
	if err := validatePostgres(cfg.Postgres); err != nil {
		return err
	}
	if err := validateRedis(cfg.Redis); err != nil {
		return err
	}
	if err := validateMongo(cfg.Mongo); err != nil {
		return err
	}
	return nil
}

func validateMicroleaseRuntimeConfig(cfg Config) error {
	if err := validateServiceAuth(cfg.ServiceAuth); err != nil {
		return err
	}
	if err := validateRedpanda(cfg.Redpanda); err != nil {
		return err
	}
	if err := validateMicrolease(cfg.Microlease); err != nil {
		return err
	}
	if cfg.Microlease.Enabled || cfg.Microlease.WorkerEnabled {
		if !cfg.Postgres.Enabled {
			return fmt.Errorf("%w: postgres.enabled must be true when microlease runtime is enabled", ErrValidate)
		}
		if !cfg.ServiceAuth.Enabled {
			return fmt.Errorf("%w: service_auth.enabled must be true when microlease runtime is enabled", ErrValidate)
		}
		if !cfg.Redpanda.Enabled {
			return fmt.Errorf("%w: redpanda.enabled must be true when microlease runtime is enabled", ErrValidate)
		}
		if cfg.Redis.Enabled {
			return fmt.Errorf("%w: redis.enabled must remain false for the first microlease runtime target", ErrValidate)
		}
	}
	return nil
}

func validateAuthorityRuntimeConfig(cfg Config) error {
	if err := validateAuthority(cfg.Authority); err != nil {
		return err
	}
	if cfg.Authority.RequireAdmissionControlFresh &&
		cfg.Authority.MaxAdmissionControlStaleness > cfg.Microlease.AdmissionControlMaxStaleness {
		return fmt.Errorf("%w: balance_usage_authority.max_admission_control_staleness must not exceed microlease.admission_control_max_staleness", ErrValidate)
	}
	if !cfg.Authority.Enabled {
		return nil
	}
	if !cfg.Postgres.Enabled {
		return fmt.Errorf("%w: postgres.enabled must be true when balance_usage_authority.enabled=true", ErrValidate)
	}
	if !cfg.ServiceAuth.Enabled {
		return fmt.Errorf("%w: service_auth.enabled must be true when balance_usage_authority.enabled=true", ErrValidate)
	}
	if !cfg.Microlease.Enabled {
		return fmt.Errorf("%w: microlease.enabled must be true when balance_usage_authority.enabled=true", ErrValidate)
	}
	if cfg.Authority.RequireWorkerReady && !cfg.Microlease.WorkerEnabled {
		return fmt.Errorf("%w: microlease.worker_enabled must be true when balance_usage_authority requires worker readiness", ErrValidate)
	}
	if cfg.Authority.RequireRedpandaReady && !cfg.Redpanda.Enabled {
		return fmt.Errorf("%w: redpanda.enabled must be true when balance_usage_authority requires Redpanda readiness", ErrValidate)
	}
	if cfg.Authority.RejectRedisSpendAuthority && cfg.Redis.Enabled {
		return fmt.Errorf("%w: redis.enabled must remain false for balance/usage spend authority", ErrValidate)
	}
	return nil
}

func validateAuthority(cfg AuthorityConfig) error {
	switch cfg.Mode {
	case "inert_expand", "shadow_no_spend", "internal_cohort", "migrated", "rollback":
	default:
		return fmt.Errorf("%w: balance_usage_authority.mode must be one of [inert_expand,shadow_no_spend,internal_cohort,migrated,rollback]", ErrValidate)
	}
	if !cfg.Enabled && cfg.Mode != "inert_expand" {
		return fmt.Errorf("%w: balance_usage_authority.mode must be inert_expand while disabled", ErrValidate)
	}
	if err := validateDurationRange("balance_usage_authority.max_admission_control_staleness", cfg.MaxAdmissionControlStaleness, time.Second, 5*time.Minute); err != nil {
		return err
	}
	if cfg.Enabled && !cfg.FailClosedWhenDependencyNotReady {
		return fmt.Errorf("%w: balance_usage_authority.fail_closed_when_dependency_not_ready must be true when enabled", ErrValidate)
	}
	if cfg.Enabled && !cfg.RejectRedisSpendAuthority {
		return fmt.Errorf("%w: balance_usage_authority.reject_redis_spend_authority must be true when enabled", ErrValidate)
	}
	return nil
}

func validateServiceAuth(cfg ServiceAuthConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(cfg.Issuer) == "" {
		return fmt.Errorf("%w: service_auth.issuer is required when service_auth.enabled=true", ErrValidate)
	}
	if strings.TrimSpace(cfg.Audience) == "" {
		return fmt.Errorf("%w: service_auth.audience is required when service_auth.enabled=true", ErrValidate)
	}
	if strings.TrimSpace(cfg.JWKSURL) == "" {
		return fmt.Errorf("%w: service_auth.jwks_url is required when service_auth.enabled=true", ErrValidate)
	}
	if !strings.HasPrefix(cfg.JWKSURL, "https://") && !strings.HasPrefix(cfg.JWKSURL, "http://") {
		return fmt.Errorf("%w: service_auth.jwks_url must be http or https", ErrValidate)
	}
	return nil
}

func validateRedpanda(cfg RedpandaConfig) error {
	if err := validateDurationRange("redpanda.healthcheck_timeout", cfg.HealthcheckTimeout, 100*time.Millisecond, 10*time.Second); err != nil {
		return err
	}
	if !cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(cfg.Brokers) == "" {
		return fmt.Errorf("%w: redpanda.brokers is required when redpanda.enabled=true", ErrValidate)
	}
	for broker := range strings.SplitSeq(cfg.Brokers, ",") {
		if err := validateHostPortWithNumericTCPPort("redpanda.brokers", strings.TrimSpace(broker)); err != nil {
			return err
		}
	}
	for _, topic := range []struct {
		key   string
		value string
	}{
		{key: "redpanda.terminal_topic", value: cfg.TerminalTopic},
		{key: "redpanda.checkpoint_topic", value: cfg.CheckpointTopic},
		{key: "redpanda.close_topic", value: cfg.CloseTopic},
		{key: "redpanda.billing_facts_topic", value: cfg.BillingFactsTopic},
		{key: "redpanda.consumer_group", value: cfg.ConsumerGroup},
	} {
		if strings.TrimSpace(topic.value) == "" {
			return fmt.Errorf("%w: %s cannot be empty when redpanda.enabled=true", ErrValidate, topic.key)
		}
	}
	return nil
}

func validateMicrolease(cfg MicroleaseConfig) error {
	if cfg.DefaultAdmissionState != "fail_closed" && cfg.DefaultAdmissionState != "strict" && cfg.DefaultAdmissionState != "throttle" && cfg.DefaultAdmissionState != "open" {
		return fmt.Errorf("%w: microlease.default_admission_state must be one of [fail_closed,strict,throttle,open]", ErrValidate)
	}
	if !cfg.Enabled && cfg.DefaultAdmissionState != "fail_closed" {
		return fmt.Errorf("%w: microlease.default_admission_state must be fail_closed while microlease.enabled=false", ErrValidate)
	}
	if err := validateMicroleaseMoneyAndDurations(cfg); err != nil {
		return err
	}
	if err := validateMicroleaseRuntimeLimits(cfg); err != nil {
		return err
	}
	if !cfg.FirstRolloutRiskAcceptanceRecorded {
		if cfg.MaxMicroleaseUSDAtoms > 100_000_000 || cfg.AccountMicroleaseExposureCapAtoms > 200_000_000 {
			return fmt.Errorf("%w: first rollout microlease caps exceed approved budget without risk acceptance", ErrValidate)
		}
	}
	return nil
}

func validateMicroleaseMoneyAndDurations(cfg MicroleaseConfig) error {
	if err := validateInt64Range("microlease.max_microlease_usd_atoms", cfg.MaxMicroleaseUSDAtoms, 1, 100_000_000); err != nil {
		return err
	}
	if err := validateInt64Range("microlease.account_microlease_exposure_cap_usd_atoms", cfg.AccountMicroleaseExposureCapAtoms, 1, 200_000_000); err != nil {
		return err
	}
	if err := validateInt64Range("microlease.min_safety_floor_usd_atoms", cfg.MinSafetyFloorUSDAtoms, 5_000_000, 200_000_000); err != nil {
		return err
	}
	if err := validateDurationRange("microlease.ttl", cfg.TTL, time.Second, 5*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("microlease.debit_cutoff_before_expiry", cfg.DebitCutoffBeforeExpiry, time.Second, cfg.TTL-time.Nanosecond); err != nil {
		return err
	}
	if err := validateDurationRange("microlease.terminal_deadline", cfg.TerminalDeadline, time.Second, 10*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("microlease.stale_debit_warning_age", cfg.StaleDebitWarningAge, time.Second, 10*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("microlease.stale_debit_critical_age", cfg.StaleDebitCriticalAge, time.Second, 30*time.Minute); err != nil {
		return err
	}
	if cfg.StaleDebitWarningAge >= cfg.StaleDebitCriticalAge {
		return fmt.Errorf("%w: microlease.stale_debit_warning_age must be less than microlease.stale_debit_critical_age", ErrValidate)
	}
	if err := validateDurationRange("microlease.reconciliation_sla", cfg.ReconciliationSLA, time.Minute, 30*time.Minute); err != nil {
		return err
	}
	return nil
}

func validateMicroleaseRuntimeLimits(cfg MicroleaseConfig) error {
	if err := validateDurationRange("microlease.admission_control_renewal_interval", cfg.AdmissionControlRenewalInterval, time.Second, cfg.AdmissionControlMaxStaleness); err != nil {
		return err
	}
	if err := validateDurationRange("microlease.admission_control_max_staleness", cfg.AdmissionControlMaxStaleness, time.Second, 5*time.Minute); err != nil {
		return err
	}
	if err := validateIntRange("microlease.refill_threshold_percent", cfg.RefillThresholdPercent, 1, 100); err != nil {
		return err
	}
	if err := validateDurationRange("microlease.max_issue_transaction_duration", cfg.MaxIssueTransactionDuration, time.Millisecond, 250*time.Millisecond); err != nil {
		return err
	}
	if err := validateDurationRange("microlease.max_terminal_transaction_duration", cfg.MaxTerminalTransactionDuration, time.Millisecond, time.Second); err != nil {
		return err
	}
	if err := validateIntRange("microlease.max_reconciliation_scan_batch_size", cfg.MaxReconciliationScanBatchSize, 1, 1000); err != nil {
		return err
	}
	return nil
}

func validateObservabilityConfig(cfg ObservabilityConfig) error {
	if strings.TrimSpace(cfg.OTel.ServiceName) == "" {
		return fmt.Errorf("%w: observability.otel.service_name cannot be empty", ErrValidate)
	}
	if err := validateSampler(cfg.OTel.TracesSampler, cfg.OTel.TracesSamplerArg); err != nil {
		return err
	}
	return validateOTLPExporter(cfg.OTel.Exporter)
}

func findUnknownKeys(k *koanf.Koanf, additionalUnknownKeys []string) []string {
	knownKeys := knownConfigKeys()
	knownSections := knownConfigSections()
	unknownSet := make(map[string]struct{})
	unknown := make([]string, 0)
	for _, key := range additionalUnknownKeys {
		if strings.TrimSpace(key) == "" {
			continue
		}
		if _, ok := unknownSet[key]; ok {
			continue
		}
		unknownSet[key] = struct{}{}
		unknown = append(unknown, key)
	}
	for _, key := range k.Keys() {
		if _, ok := knownKeys[key]; ok {
			continue
		}

		if _, ok := knownSections[key]; ok && configSectionValueIsMap(k.Get(key)) {
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
	if err := validateIntRange("postgres.max_open_conns", cfg.MaxOpenConns, 1, 500); err != nil {
		return err
	}
	if err := validateIntRange("postgres.max_idle_conns", cfg.MaxIdleConns, 0, cfg.MaxOpenConns); err != nil {
		return err
	}
	if err := validateDurationRange("postgres.conn_max_lifetime", cfg.ConnMaxLifetime, time.Minute, 24*time.Hour); err != nil {
		return err
	}

	return nil
}

func validateRedis(cfg RedisConfig) error {
	mode, err := validateRedisMode(cfg)
	if err != nil {
		return err
	}
	if err := validateRedisEndpoint(cfg); err != nil {
		return err
	}
	if err := validateRedisStoreMode(cfg, mode); err != nil {
		return err
	}
	if err := validateRedisLimits(cfg); err != nil {
		return err
	}
	return nil
}

func validateRedisMode(cfg RedisConfig) (string, error) {
	mode := cfg.ModeValue()
	if mode == "" {
		return "", fmt.Errorf("%w: redis.mode cannot be empty", ErrValidate)
	}
	if mode != RedisModeCache && mode != RedisModeStore {
		return "", fmt.Errorf("%w: redis.mode must be one of [cache,store]", ErrValidate)
	}
	return mode, nil
}

func validateRedisEndpoint(cfg RedisConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(cfg.Addr) == "" {
		return fmt.Errorf("%w: redis.addr is required when redis.enabled=true", ErrValidate)
	}
	return validateHostPortWithNumericTCPPort("redis.addr", cfg.Addr)
}

func validateRedisStoreMode(cfg RedisConfig, mode string) error {
	if mode != RedisModeStore {
		return nil
	}
	// ARCH-008: v1 only supports guard/reject behavior for store mode.
	if !cfg.AllowStoreMode {
		return fmt.Errorf("%w: redis.mode=store is blocked unless redis.allow_store_mode=true", ErrValidate)
	}
	if cfg.StaleWindow != 0 {
		return fmt.Errorf("%w: redis.stale_window must be 0 when redis.mode=store", ErrValidate)
	}
	return nil
}

func validateRedisLimits(cfg RedisConfig) error {
	if err := validateIntRange("redis.db", cfg.DB, 0, 15); err != nil {
		return err
	}
	if err := validateDurationRange("redis.dial_timeout", cfg.DialTimeout, 50*time.Millisecond, 5*time.Second); err != nil {
		return err
	}
	if err := validateDurationRange("redis.read_timeout", cfg.ReadTimeout, 50*time.Millisecond, 5*time.Second); err != nil {
		return err
	}
	if err := validateDurationRange("redis.write_timeout", cfg.WriteTimeout, 50*time.Millisecond, 5*time.Second); err != nil {
		return err
	}
	if err := validateIntRange("redis.pool_size", cfg.PoolSize, 1, 1000); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.KeyPrefix) == "" {
		return fmt.Errorf("%w: redis.key_prefix cannot be empty", ErrValidate)
	}
	if err := validateDurationRange("redis.fresh_ttl", cfg.FreshTTL, time.Second, 15*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("redis.stale_window", cfg.StaleWindow, 0, 5*time.Minute); err != nil {
		return err
	}
	if err := validateDurationRange("redis.negative_ttl", cfg.NegativeTTL, time.Second, 60*time.Second); err != nil {
		return err
	}
	if err := validateIntRange("redis.ttl_jitter_percent", cfg.TTLJitterPercent, 0, 30); err != nil {
		return err
	}
	return validateIntRange("redis.max_fallback_concurrency", cfg.MaxFallbackConcurrency, 1, 256)
}

func validateMongo(cfg MongoConfig) error {
	if cfg.Enabled && strings.TrimSpace(cfg.URI) == "" {
		return fmt.Errorf("%w: mongo.uri is required when mongo.enabled=true", ErrSecretPolicy)
	}
	if cfg.Enabled {
		if _, err := MongoProbeAddress(cfg.URI); err != nil {
			return fmt.Errorf("mongo.uri must contain a valid probe target: %w", err)
		}
		if strings.TrimSpace(cfg.Database) == "" {
			return fmt.Errorf("%w: mongo.database is required when mongo.enabled=true", ErrValidate)
		}
	}

	if err := validateDurationRange("mongo.connect_timeout", cfg.ConnectTimeout, 100*time.Millisecond, 15*time.Second); err != nil {
		return err
	}
	if err := validateDurationRange("mongo.server_selection_timeout", cfg.ServerSelectionTimeout, 100*time.Millisecond, 10*time.Second); err != nil {
		return err
	}
	if err := validateIntRange("mongo.max_pool_size", cfg.MaxPoolSize, 1, 1000); err != nil {
		return err
	}

	return nil
}

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

func validateReadinessProbeBudgets(cfg Config) error {
	budgets := cfg.ReadinessProbeBudgets()
	var aggregate time.Duration
	names := make([]string, 0, len(budgets))
	for _, probe := range budgets {
		aggregate += probe.Budget
		names = append(names, probe.ConfigKey)
	}
	if cfg.HTTP.ReadinessTimeout < aggregate {
		return fmt.Errorf(
			"%w: http.readiness_timeout must be >= aggregate sequential readiness probe budget (%s = %s)",
			ErrValidate,
			strings.Join(names, " + "),
			aggregate,
		)
	}
	return nil
}

func validateSampler(sampler string, samplerArg float64) error {
	if !otelconfig.TraceSamplerSupported(sampler) {
		return fmt.Errorf("%w: observability.otel.traces_sampler is unsupported", ErrValidate)
	}

	if !otelconfig.TraceSamplerArgFinite(samplerArg) {
		return fmt.Errorf("%w: observability.otel.traces_sampler_arg must be finite", ErrValidate)
	}
	if !otelconfig.TraceSamplerArgInRange(samplerArg) {
		return fmt.Errorf("%w: observability.otel.traces_sampler_arg must be in range [0,1]", ErrValidate)
	}
	return nil
}

func validateOTLPExporter(cfg OTelExporterConfig) error {
	protocol := otelconfig.NormalizeOTLPProtocol(cfg.OTLPProtocol)
	if protocol == "" {
		return nil
	}

	if !otelconfig.OTLPProtocolSupported(protocol) {
		return fmt.Errorf("%w: observability.otel.exporter.otlp_protocol must be %s", ErrValidate, otelconfig.OTLPProtocolHTTPProtobuf)
	}
	return nil
}

func validateDurationRange(name string, value time.Duration, lowerBound time.Duration, upperBound time.Duration) error {
	if value < lowerBound || value > upperBound {
		return fmt.Errorf("%w: %s must be in range [%s,%s]", ErrValidate, name, lowerBound, upperBound)
	}
	return nil
}

func validateIntRange(name string, value int, lowerBound int, upperBound int) error {
	if value < lowerBound || value > upperBound {
		return fmt.Errorf("%w: %s must be in range [%d,%d]", ErrValidate, name, lowerBound, upperBound)
	}
	return nil
}

func validateInt64Range(name string, value int64, lowerBound int64, upperBound int64) error {
	if value < lowerBound || value > upperBound {
		return fmt.Errorf("%w: %s must be in range [%d,%d]", ErrValidate, name, lowerBound, upperBound)
	}
	return nil
}

func validateHostPortWithNumericTCPPort(name string, address string) error {
	host, port, err := net.SplitHostPort(strings.TrimSpace(address))
	if err != nil {
		return fmt.Errorf("%w: %s must be host:port", ErrValidate, name)
	}
	if strings.TrimSpace(host) == "" {
		return fmt.Errorf("%w: %s must include non-empty host", ErrValidate, name)
	}
	if err := validateNumericTCPPort(port); err != nil {
		return fmt.Errorf("%w: %s must include numeric TCP port in range [1,65535]", ErrValidate, name)
	}
	return nil
}

func validateNumericTCPPort(port string) error {
	value, err := strconv.ParseUint(port, 10, 32)
	if err != nil || value == 0 || value > 65535 {
		return fmt.Errorf("port must be numeric TCP port in range [1,65535]")
	}
	return nil
}

const (
	mongodbScheme    = "mongodb://"
	mongodbSRVScheme = "mongodb+srv://"
	defaultMongoPort = "27017"
)

// MongoProbeAddress extracts a probe-ready host:port from a MongoDB URI.
func MongoProbeAddress(rawURI string) (string, error) {
	uri := strings.TrimSpace(rawURI)
	if uri == "" {
		return "", mongoProbeAddressError("empty mongo uri")
	}
	if uri != rawURI {
		return "", mongoProbeAddressError("mongo uri must not include surrounding whitespace")
	}

	lower := strings.ToLower(uri)
	var hostPart string
	switch {
	case strings.HasPrefix(lower, mongodbScheme):
		hostPart = uri[len(mongodbScheme):]
	case strings.HasPrefix(lower, mongodbSRVScheme):
		hostPart = uri[len(mongodbSRVScheme):]
	default:
		return "", mongoProbeAddressError("unsupported mongo uri scheme")
	}

	if hostPart == "" {
		return "", mongoProbeAddressError("empty mongo host section")
	}
	if slash := strings.Index(hostPart, "/"); slash >= 0 {
		hostPart = hostPart[:slash]
	}
	if at := strings.LastIndex(hostPart, "@"); at >= 0 {
		hostPart = hostPart[at+1:]
	}
	if hostPart == "" {
		return "", mongoProbeAddressError("empty mongo host section")
	}
	if strings.Contains(hostPart, ",") {
		return "", mongoProbeAddressError("mongo seedlists are not supported by guard-only probe path")
	}

	firstHost := strings.TrimSpace(hostPart)
	if firstHost == "" {
		return "", mongoProbeAddressError("empty mongo host")
	}

	return normalizeMongoProbeAddress(firstHost)
}

func normalizeMongoProbeAddress(host string) (string, error) {
	trimmed := strings.TrimSpace(host)
	if trimmed == "" {
		return "", mongoProbeAddressError("empty mongo host")
	}

	if address, ok, err := normalizeSplitMongoProbeAddress(trimmed); ok || err != nil {
		return address, err
	}
	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, "?") {
		return "", mongoProbeAddressError("invalid mongo host")
	}

	if strings.ContainsAny(trimmed, "[]") {
		return normalizeBracketedMongoProbeAddress(trimmed)
	}

	if strings.Count(trimmed, ":") > 1 {
		return normalizeBareMongoIPv6ProbeAddress(trimmed)
	}

	if strings.Contains(trimmed, ":") {
		return "", mongoProbeAddressError("invalid mongo host")
	}

	return net.JoinHostPort(trimmed, defaultMongoPort), nil
}

func normalizeSplitMongoProbeAddress(trimmed string) (string, bool, error) {
	parsedHost, port, ok := splitMongoHostPort(trimmed)
	if !ok {
		return "", false, nil
	}
	if strings.TrimSpace(parsedHost) == "" {
		return "", true, mongoProbeAddressError("empty mongo host")
	}
	if strings.ContainsAny(parsedHost, "[]") {
		return "", true, mongoProbeAddressError("invalid mongo host")
	}
	if strings.HasPrefix(trimmed, "[") || strings.Contains(parsedHost, ":") {
		if err := validateMongoIPv6Literal(parsedHost); err != nil {
			return "", true, err
		}
	}
	if err := validateNumericTCPPort(port); err != nil {
		return "", true, mongoProbeAddressError("invalid mongo TCP port")
	}
	return trimmed, true, nil
}

func splitMongoHostPort(trimmed string) (string, string, bool) {
	host, port, err := net.SplitHostPort(trimmed)
	return host, port, err == nil
}

func normalizeBracketedMongoProbeAddress(trimmed string) (string, error) {
	if !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
		return "", mongoProbeAddressError("invalid mongo host")
	}
	bracketedHost := strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]")
	if bracketedHost == "" || strings.ContainsAny(bracketedHost, "[]") {
		return "", mongoProbeAddressError("invalid mongo host")
	}
	if err := validateMongoIPv6Literal(bracketedHost); err != nil {
		return "", err
	}
	return net.JoinHostPort(bracketedHost, defaultMongoPort), nil
}

func normalizeBareMongoIPv6ProbeAddress(trimmed string) (string, error) {
	if err := validateMongoIPv6Literal(trimmed); err != nil {
		return "", err
	}
	return net.JoinHostPort(trimmed, defaultMongoPort), nil
}

func validateMongoIPv6Literal(host string) error {
	addr, err := netip.ParseAddr(host)
	if err != nil || !addr.Is6() {
		return mongoProbeAddressError("invalid mongo host")
	}
	return nil
}

func mongoProbeAddressError(detail string) error {
	return fmt.Errorf("%w: %s", ErrValidate, detail)
}
