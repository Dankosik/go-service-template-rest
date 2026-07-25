package config

import (
	"log/slog"
	"time"
)

// Config is the immutable runtime snapshot built during startup.
type Config struct {
	App           AppConfig           `koanf:"app"`
	HTTP          HTTPConfig          `koanf:"http"`
	Health        HealthConfig        `koanf:"health"`
	Log           LogConfig           `koanf:"log"`
	Observability ObservabilityConfig `koanf:"observability"`
	Runtime       RuntimeConfig       `koanf:"runtime"`
	// profile:database-postgres:start
	Postgres PostgresConfig `koanf:"postgres"`
	// profile:database-postgres:end
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
	// RequestTimeout is the per-request handler budget. The net/http server
	// timeouts are connection deadlines that never cancel the request context,
	// so this is the only bound on how long one handler may hold a goroutine
	// and its pooled resources.
	RequestTimeout time.Duration `koanf:"request_timeout"`
	WriteTimeout   time.Duration `koanf:"write_timeout"`
	IdleTimeout    time.Duration `koanf:"idle_timeout"`
	MaxHeaderBytes int           `koanf:"max_header_bytes"`
	MaxBodyBytes   int64         `koanf:"max_body_bytes"`
	// MaxInFlight bounds concurrent handler execution. Nothing else in the
	// request path does: RequestTimeout bounds how long one request may run,
	// not how many may run, so an arrival spike otherwise queues every request
	// on a finite resource until all of them spend the full budget waiting.
	// Zero disables shedding.
	MaxInFlight int `koanf:"max_in_flight"`
	// AccessLogHealthProbes re-enables access logging for /health/live and
	// /health/ready. It defaults to false because orchestrator probes generate
	// continuous no-signal log volume.
	AccessLogHealthProbes bool `koanf:"access_log_health_probes"`
}

// HealthConfig bounds how readiness state is refreshed. Probes are evaluated on
// this interval rather than per request, so a probe route can never consume the
// dependency capacity it reports on.
type HealthConfig struct {
	RefreshInterval time.Duration `koanf:"refresh_interval"`
	// FailureThreshold is how many consecutive failed evaluations flip
	// readiness. One slow round-trip must not evict an instance that is still
	// serving traffic.
	FailureThreshold int `koanf:"failure_threshold"`
}

// RuntimeConfig owns Go runtime limits that the toolchain does not derive on its
// own. GOMAXPROCS is deliberately absent: since Go 1.25 the runtime reads the
// cgroup CPU bandwidth limit and re-reads it periodically, and setting it here
// would disable that.
type RuntimeConfig struct {
	// MemoryLimitRatio is the fraction of the detected container memory limit
	// published to the garbage collector through debug.SetMemoryLimit. Zero
	// disables detection and leaves the limit at its default of math.MaxInt64.
	MemoryLimitRatio float64 `koanf:"memory_limit_ratio"`
}

type LogConfig struct {
	Level slog.Level `koanf:"level"`
}

type ObservabilityConfig struct {
	Metrics MetricsConfig `koanf:"metrics"`
	Pprof   PprofConfig   `koanf:"pprof"`
	OTel    OTelConfig    `koanf:"otel"`
}

type MetricsConfig struct {
	// Addr is the private diagnostics listener. Empty disables HTTP exposition.
	Addr string `koanf:"addr"`
}

// PprofConfig gates the runtime profile handlers on the diagnostics listener.
type PprofConfig struct {
	// Enabled serves net/http/pprof on the diagnostics listener. It defaults to
	// false: the profile handlers disclose heap contents and /debug/pprof/cmdline
	// discloses process arguments, and the diagnostics listener binds every
	// interface so Prometheus can reach it. Turning this on is a deployment
	// decision about who can reach that port, not a template default.
	Enabled bool `koanf:"enabled"`
}

type OTelConfig struct {
	ServiceName      string             `koanf:"service_name"`
	TracesSampler    string             `koanf:"traces_sampler"`
	TracesSamplerArg float64            `koanf:"traces_sampler_arg"`
	Exporter         OTelExporterConfig `koanf:"exporter"`
}

type OTelExporterConfig struct {
	// OTLPEndpoint is the full OTLP HTTP traces endpoint URL (http or https).
	// A missing path defaults to /v1/traces.
	OTLPEndpoint string `koanf:"otlp_endpoint"`
	OTLPHeaders  string `koanf:"otlp_headers"`
}

// profile:database-postgres:start
type PostgresConfig struct {
	Enabled                   bool          `koanf:"enabled"`
	DSN                       string        `koanf:"dsn"`
	ConnectTimeout            time.Duration `koanf:"connect_timeout"`
	HealthcheckTimeout        time.Duration `koanf:"healthcheck_timeout"`
	MigrationTimeout          time.Duration `koanf:"migration_timeout"`
	MigrationStatementTimeout time.Duration `koanf:"migration_statement_timeout"`
	MigrationLockTimeout      time.Duration `koanf:"migration_lock_timeout"`
	MaxOpenConns              int           `koanf:"max_open_conns"`
	ConnMaxLifetime           time.Duration `koanf:"conn_max_lifetime"`
	// StatementTimeout bounds every runtime statement server-side, and bounds
	// how long a session may sit idle inside a transaction. The request budget
	// only cancels client-side: if the cancel request itself is lost, which is
	// the failure a slow dependency tends to produce, an abandoned query keeps
	// running and keeps its locks. Validated against http.request_timeout so the
	// two are one budget rather than two numbers.
	StatementTimeout time.Duration `koanf:"statement_timeout"`
}

// profile:database-postgres:end
