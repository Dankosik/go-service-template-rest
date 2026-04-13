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

	"github.com/example/go-service-template-rest/internal/observability/otelconfig"
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
	result := validationResult{}
	if err := checkValidateContext(ctx); err != nil {
		return validationResult{}, err
	}

	unknownKeys := findUnknownKeys(k, opts.AdditionalUnknownKeys)
	if len(unknownKeys) > 0 {
		if opts.Strict {
			return validationResult{}, fmt.Errorf("%w: unknown keys: %s", ErrStrictUnknownKey, strings.Join(unknownKeys, ", "))
		}
		result.UnknownKeyWarnings = unknownKeys
	}
	if err := checkValidateContext(ctx); err != nil {
		return result, err
	}

	if strings.TrimSpace(cfg.App.Env) == "" {
		return result, fmt.Errorf("%w: app.env cannot be empty", ErrValidate)
	}
	if strings.TrimSpace(cfg.App.Version) == "" {
		return result, fmt.Errorf("%w: app.version cannot be empty", ErrValidate)
	}
	if strings.TrimSpace(cfg.HTTP.Addr) == "" {
		return result, fmt.Errorf("%w: http.addr cannot be empty", ErrValidate)
	}

	if err := validateDurationRange("http.shutdown_timeout", cfg.HTTP.ShutdownTimeout, time.Second, 10*time.Minute); err != nil {
		return result, err
	}
	if err := validateDurationRange("http.readiness_timeout", cfg.HTTP.ReadinessTimeout, 100*time.Millisecond, 30*time.Second); err != nil {
		return result, err
	}
	if err := validateDurationRange("http.readiness_propagation_delay", cfg.HTTP.ReadinessPropagationDelay, 0, cfg.HTTP.ShutdownTimeout); err != nil {
		return result, err
	}
	if err := validateDurationRange("http.read_header_timeout", cfg.HTTP.ReadHeaderTimeout, 100*time.Millisecond, 5*time.Minute); err != nil {
		return result, err
	}
	if err := validateDurationRange("http.read_timeout", cfg.HTTP.ReadTimeout, 100*time.Millisecond, 5*time.Minute); err != nil {
		return result, err
	}
	if err := validateDurationRange("http.write_timeout", cfg.HTTP.WriteTimeout, 100*time.Millisecond, 10*time.Minute); err != nil {
		return result, err
	}
	if err := validateDurationRange("http.idle_timeout", cfg.HTTP.IdleTimeout, 100*time.Millisecond, 24*time.Hour); err != nil {
		return result, err
	}
	if err := validateHTTPReadinessWriteTimeout(cfg.HTTP); err != nil {
		return result, err
	}
	if err := validateHTTPShutdownBudget(cfg.HTTP); err != nil {
		return result, err
	}
	if cfg.HTTP.MaxHeaderBytes <= 0 {
		return result, fmt.Errorf("%w: http.max_header_bytes must be > 0", ErrValidate)
	}
	if cfg.HTTP.MaxBodyBytes <= 0 {
		return result, fmt.Errorf("%w: http.max_body_bytes must be > 0", ErrValidate)
	}
	if err := checkValidateContext(ctx); err != nil {
		return result, err
	}

	if err := validatePostgres(cfg.Postgres); err != nil {
		return result, err
	}
	if err := validateRedis(cfg.Redis); err != nil {
		return result, err
	}
	if err := validateMongo(cfg.Mongo); err != nil {
		return result, err
	}
	if err := validateReadinessProbeBudgets(*cfg); err != nil {
		return result, err
	}
	if err := checkValidateContext(ctx); err != nil {
		return result, err
	}

	if strings.TrimSpace(cfg.Observability.OTel.ServiceName) == "" {
		return result, fmt.Errorf("%w: observability.otel.service_name cannot be empty", ErrValidate)
	}
	if err := validateSampler(cfg.Observability.OTel.TracesSampler, cfg.Observability.OTel.TracesSamplerArg); err != nil {
		return result, err
	}
	if err := validateOTLPExporter(cfg.Observability.OTel.Exporter); err != nil {
		return result, err
	}

	return result, nil
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
	mode := cfg.ModeValue()
	if mode == "" {
		return fmt.Errorf("%w: redis.mode cannot be empty", ErrValidate)
	}
	if mode != RedisModeCache && mode != RedisModeStore {
		return fmt.Errorf("%w: redis.mode must be one of [cache,store]", ErrValidate)
	}

	if cfg.Enabled {
		if strings.TrimSpace(cfg.Addr) == "" {
			return fmt.Errorf("%w: redis.addr is required when redis.enabled=true", ErrValidate)
		}
		if err := validateHostPortWithNumericTCPPort("redis.addr", cfg.Addr); err != nil {
			return err
		}
	}

	if mode == RedisModeStore {
		// ARCH-008: v1 only supports guard/reject behavior for store mode.
		if !cfg.AllowStoreMode {
			return fmt.Errorf("%w: redis.mode=store is blocked unless redis.allow_store_mode=true", ErrValidate)
		}
		if cfg.StaleWindow != 0 {
			return fmt.Errorf("%w: redis.stale_window must be 0 when redis.mode=store", ErrValidate)
		}
	}

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
	if err := validateIntRange("redis.max_fallback_concurrency", cfg.MaxFallbackConcurrency, 1, 256); err != nil {
		return err
	}

	return nil
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

	if parsedHost, port, err := net.SplitHostPort(trimmed); err == nil {
		if strings.TrimSpace(parsedHost) == "" {
			return "", mongoProbeAddressError("empty mongo host")
		}
		if strings.ContainsAny(parsedHost, "[]") {
			return "", mongoProbeAddressError("invalid mongo host")
		}
		if strings.HasPrefix(trimmed, "[") || strings.Contains(parsedHost, ":") {
			if err := validateMongoIPv6Literal(parsedHost); err != nil {
				return "", err
			}
		}
		if err := validateNumericTCPPort(port); err != nil {
			return "", mongoProbeAddressError("invalid mongo TCP port")
		}
		return trimmed, nil
	}
	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, "?") {
		return "", mongoProbeAddressError("invalid mongo host")
	}

	if strings.ContainsAny(trimmed, "[]") {
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

	if strings.Count(trimmed, ":") > 1 {
		if err := validateMongoIPv6Literal(trimmed); err != nil {
			return "", err
		}
		return net.JoinHostPort(trimmed, defaultMongoPort), nil
	}

	if strings.Contains(trimmed, ":") {
		return "", mongoProbeAddressError("invalid mongo host")
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
