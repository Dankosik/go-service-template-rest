package config

import (
	"log/slog"
	"time"
)

// Config is the immutable runtime snapshot built during startup.
type Config struct {
	App           AppConfig           `koanf:"app"`
	HTTP          HTTPConfig          `koanf:"http"`
	Log           LogConfig           `koanf:"log"`
	Observability ObservabilityConfig `koanf:"observability"`
	Postgres      PostgresConfig      `koanf:"postgres"`
	Redis         RedisConfig         `koanf:"redis"`
	Mongo         MongoConfig         `koanf:"mongo"`
	ServiceAuth   ServiceAuthConfig   `koanf:"service_auth"`
	Redpanda      RedpandaConfig      `koanf:"redpanda"`
	Microlease    MicroleaseConfig    `koanf:"microlease"`
	Authority     AuthorityConfig     `koanf:"balance_usage_authority"`
	FeatureFlags  FeatureFlagsConfig  `koanf:"feature_flags"`
}

type AppConfig struct {
	Env     string `koanf:"env"`
	Version string `koanf:"version"`
}

type HTTPConfig struct {
	Addr                      string        `koanf:"addr"`
	ShutdownTimeout           time.Duration `koanf:"shutdown_timeout"`
	ReadinessTimeout          time.Duration `koanf:"readiness_timeout"`
	ReadinessPropagationDelay time.Duration `koanf:"readiness_propagation_delay"`
	ReadHeaderTimeout         time.Duration `koanf:"read_header_timeout"`
	ReadTimeout               time.Duration `koanf:"read_timeout"`
	WriteTimeout              time.Duration `koanf:"write_timeout"`
	IdleTimeout               time.Duration `koanf:"idle_timeout"`
	MaxHeaderBytes            int           `koanf:"max_header_bytes"`
	MaxBodyBytes              int64         `koanf:"max_body_bytes"`
}

type LogConfig struct {
	Level slog.Level `koanf:"level"`
}

type ObservabilityConfig struct {
	OTel OTelConfig `koanf:"otel"`
}

type OTelConfig struct {
	ServiceName      string             `koanf:"service_name"`
	TracesSampler    string             `koanf:"traces_sampler"`
	TracesSamplerArg float64            `koanf:"traces_sampler_arg"`
	Exporter         OTelExporterConfig `koanf:"exporter"`
}

type OTelExporterConfig struct {
	OTLPEndpoint       string `koanf:"otlp_endpoint"`
	OTLPTracesEndpoint string `koanf:"otlp_traces_endpoint"`
	OTLPHeaders        string `koanf:"otlp_headers"`
	OTLPProtocol       string `koanf:"otlp_protocol"`
}

type PostgresConfig struct {
	Enabled            bool          `koanf:"enabled"`
	DSN                string        `koanf:"dsn"`
	ConnectTimeout     time.Duration `koanf:"connect_timeout"`
	HealthcheckTimeout time.Duration `koanf:"healthcheck_timeout"`
	MaxOpenConns       int           `koanf:"max_open_conns"`
	MaxIdleConns       int           `koanf:"max_idle_conns"`
	ConnMaxLifetime    time.Duration `koanf:"conn_max_lifetime"`
}

type RedisConfig struct {
	Enabled                bool          `koanf:"enabled"`
	Mode                   string        `koanf:"mode"`
	AllowStoreMode         bool          `koanf:"allow_store_mode"`
	Addr                   string        `koanf:"addr"`
	Username               string        `koanf:"username"`
	Password               string        `koanf:"password"`
	DB                     int           `koanf:"db"`
	DialTimeout            time.Duration `koanf:"dial_timeout"`
	ReadTimeout            time.Duration `koanf:"read_timeout"`
	WriteTimeout           time.Duration `koanf:"write_timeout"`
	PoolSize               int           `koanf:"pool_size"`
	KeyPrefix              string        `koanf:"key_prefix"`
	FreshTTL               time.Duration `koanf:"fresh_ttl"`
	StaleWindow            time.Duration `koanf:"stale_window"`
	NegativeTTL            time.Duration `koanf:"negative_ttl"`
	TTLJitterPercent       int           `koanf:"ttl_jitter_percent"`
	EnableSingleflight     bool          `koanf:"enable_singleflight"`
	MaxFallbackConcurrency int           `koanf:"max_fallback_concurrency"`
}

type MongoConfig struct {
	Enabled                bool          `koanf:"enabled"`
	URI                    string        `koanf:"uri"`
	Database               string        `koanf:"database"`
	ConnectTimeout         time.Duration `koanf:"connect_timeout"`
	ServerSelectionTimeout time.Duration `koanf:"server_selection_timeout"`
	MaxPoolSize            int           `koanf:"max_pool_size"`
}

type ServiceAuthConfig struct {
	Enabled  bool   `koanf:"enabled"`
	Issuer   string `koanf:"issuer"`
	Audience string `koanf:"audience"`
	JWKSURL  string `koanf:"jwks_url"`
}

type RedpandaConfig struct {
	Enabled            bool          `koanf:"enabled"`
	Brokers            string        `koanf:"brokers"`
	TerminalTopic      string        `koanf:"terminal_topic"`
	CheckpointTopic    string        `koanf:"checkpoint_topic"`
	CloseTopic         string        `koanf:"close_topic"`
	BillingFactsTopic  string        `koanf:"billing_facts_topic"`
	ConsumerGroup      string        `koanf:"consumer_group"`
	HealthcheckTimeout time.Duration `koanf:"healthcheck_timeout"`
}

type MicroleaseConfig struct {
	Enabled                            bool          `koanf:"enabled"`
	WorkerEnabled                      bool          `koanf:"worker_enabled"`
	DefaultAdmissionState              string        `koanf:"default_admission_state"`
	MaxMicroleaseUSDAtoms              int64         `koanf:"max_microlease_usd_atoms"`
	AccountMicroleaseExposureCapAtoms  int64         `koanf:"account_microlease_exposure_cap_usd_atoms"`
	MinSafetyFloorUSDAtoms             int64         `koanf:"min_safety_floor_usd_atoms"`
	TTL                                time.Duration `koanf:"ttl"`
	DebitCutoffBeforeExpiry            time.Duration `koanf:"debit_cutoff_before_expiry"`
	TerminalDeadline                   time.Duration `koanf:"terminal_deadline"`
	StaleDebitWarningAge               time.Duration `koanf:"stale_debit_warning_age"`
	StaleDebitCriticalAge              time.Duration `koanf:"stale_debit_critical_age"`
	ReconciliationSLA                  time.Duration `koanf:"reconciliation_sla"`
	AdmissionControlRenewalInterval    time.Duration `koanf:"admission_control_renewal_interval"`
	AdmissionControlMaxStaleness       time.Duration `koanf:"admission_control_max_staleness"`
	RefillThresholdPercent             int           `koanf:"refill_threshold_percent"`
	MaxIssueTransactionDuration        time.Duration `koanf:"max_issue_transaction_duration"`
	MaxTerminalTransactionDuration     time.Duration `koanf:"max_terminal_transaction_duration"`
	MaxReconciliationScanBatchSize     int           `koanf:"max_reconciliation_scan_batch_size"`
	FirstRolloutRiskAcceptanceRecorded bool          `koanf:"first_rollout_risk_acceptance_recorded"`
}

type AuthorityConfig struct {
	Enabled                          bool          `koanf:"enabled"`
	Mode                             string        `koanf:"mode"`
	RequireWorkerReady               bool          `koanf:"require_worker_ready"`
	RequireRedpandaReady             bool          `koanf:"require_redpanda_ready"`
	RequireAdmissionControlFresh     bool          `koanf:"require_admission_control_fresh"`
	MaxAdmissionControlStaleness     time.Duration `koanf:"max_admission_control_staleness"`
	RejectRedisSpendAuthority        bool          `koanf:"reject_redis_spend_authority"`
	FailClosedWhenDependencyNotReady bool          `koanf:"fail_closed_when_dependency_not_ready"`
}

type FeatureFlagsConfig struct {
	PostgresReadinessProbe bool `koanf:"postgres_readiness_probe"`
	MongoReadinessProbe    bool `koanf:"mongo_readiness_probe"`
	RedisReadinessProbe    bool `koanf:"redis_readiness_probe"`
	RedpandaReadinessProbe bool `koanf:"redpanda_readiness_probe"`
}
