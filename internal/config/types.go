package config

import (
	"log/slog"
	"time"
)

// Config is the immutable runtime snapshot built during startup.
type Config struct {
	App  AppConfig  `koanf:"app"`
	HTTP HTTPConfig `koanf:"http"`
	// profile:authn-oidc-jwt:start
	Authn AuthnConfig `koanf:"authn"`
	// profile:authn-oidc-jwt:end
	// profile:grpc:start
	GRPC GRPCConfig `koanf:"grpc"`
	// profile:grpc:end
	Health HealthConfig `koanf:"health"`
	Log    LogConfig    `koanf:"log"`
	// profile:messaging-nats-jetstream:start
	Messaging MessagingConfig `koanf:"messaging"`
	// profile:messaging-nats-jetstream:end
	Observability ObservabilityConfig `koanf:"observability"`
	Runtime       RuntimeConfig       `koanf:"runtime"`
	// profile:database-postgres:start
	Postgres PostgresConfig `koanf:"postgres"`
	// profile:database-postgres:end
	// profile:outbox-postgres:start
	Outbox OutboxConfig `koanf:"outbox"`
	// profile:outbox-postgres:end
}

// profile:messaging-nats-jetstream:start

type MessagingConfig struct {
	Enabled              bool                  `koanf:"enabled"`
	URLs                 string                `koanf:"urls"`
	CredentialsFile      string                `koanf:"credentials_file"`
	RootCAFile           string                `koanf:"root_ca_file"`
	AllowPlaintext       bool                  `koanf:"allow_plaintext"`
	AllowUnauthenticated bool                  `koanf:"allow_unauthenticated"`
	Stream               string                `koanf:"stream"`
	MaxPayloadBytes      int                   `koanf:"max_payload_bytes"`
	MaxPendingPublishes  int                   `koanf:"max_pending_publishes"`
	Worker               MessagingWorkerConfig `koanf:"worker"`
}

type MessagingWorkerConfig struct {
	Consumer             string        `koanf:"consumer"`
	FilterSubject        string        `koanf:"filter_subject"`
	DeadLetterSubject    string        `koanf:"dead_letter_subject"`
	MaxConcurrency       int           `koanf:"max_concurrency"`
	MaxDeliveryBytes     int           `koanf:"max_delivery_bytes"`
	HandlerTimeout       time.Duration `koanf:"handler_timeout"`
	RetryDelays          string        `koanf:"retry_delays"`
	DeadLetterRetryDelay time.Duration `koanf:"dead_letter_retry_delay"`
	DrainTimeout         time.Duration `koanf:"drain_timeout"`
}

// profile:messaging-nats-jetstream:end

// profile:authn-oidc-jwt:start

// AuthnConfig names the immutable issuer boundary. Cryptographic, cache, size,
// and time policy is fixed by the capability rather than provider-controlled
// configuration.
type AuthnConfig struct {
	Issuer            string `koanf:"issuer"`
	Audience          string `koanf:"audience"`
	TrustedProxyCIDRs string `koanf:"trusted_proxy_cidrs"`
}

// profile:authn-oidc-jwt:end

type AppConfig struct {
	Env     string `koanf:"env"`
	Version string `koanf:"version"`
	// Commit is the source revision this binary was built from. It defaults to
	// what the image build stamped in, and stays overridable for a platform that
	// knows the revision without driving the build.
	Commit string `koanf:"commit"`
	// InstanceID identifies this replica in exported telemetry. Empty resolves to
	// the hostname, which is the pod name on Kubernetes and the container id on
	// most other platforms; a platform that exposes neither should set it. Without
	// an instance identity every replica pushes the same resource, and cumulative
	// counters from different replicas collide into one series.
	InstanceID string `koanf:"instance_id"`
}

type HTTPConfig struct {
	Addr string `koanf:"addr"`
	// GracePeriod is the total time the platform allows between SIGTERM and
	// SIGKILL — terminationGracePeriodSeconds on Kubernetes, drainingSeconds on
	// Railway. Every teardown stage draws from it, so it is the one number that
	// bounds shutdown as a whole. ShutdownTimeout bounds only the HTTP drain;
	// closing diagnostics, joining background work, releasing pooled
	// dependencies, and flushing telemetry all still run after it.
	GracePeriod time.Duration `koanf:"grace_period"`
	// ShutdownTimeout bounds the HTTP drain, including the readiness
	// propagation delay in front of it. It is validated against GracePeriod so
	// the stages after the drain still have budget left.
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
	// MaxConnections bounds how many connections the API listener accepts at
	// once. MaxInFlight does not: it bounds handler execution, and every
	// accepted connection already costs a goroutine and its read and write
	// buffers before any middleware runs. Excess callers wait in the kernel
	// backlog, where they cost nothing. Validated against MaxInFlight so the cap
	// can never shed below the concurrency the service advertises. Zero accepts
	// without a bound.
	MaxConnections int `koanf:"max_connections"`
	// AccessLogHealthProbes re-enables access logging for /health/live and
	// /health/ready. It defaults to false because orchestrator probes generate
	// continuous no-signal log volume.
	AccessLogHealthProbes bool `koanf:"access_log_health_probes"`
}

// profile:grpc:start

type GRPCConfig struct {
	Server GRPCServerConfig `koanf:"server"`
}

type GRPCServerConfig struct {
	Enabled bool   `koanf:"enabled"`
	Addr    string `koanf:"addr"`

	// TransportSecurity is explicit whenever the server is enabled. "tls"
	// loads the configured certificate and key during bootstrap; "plaintext"
	// additionally requires AllowPlaintext so an operator cannot disable
	// transport security by changing one value.
	TransportSecurity string        `koanf:"transport_security"`
	AllowPlaintext    bool          `koanf:"allow_plaintext"`
	TLS               GRPCTLSConfig `koanf:"tls"`

	// MaxConnections bounds accepted HTTP/2 connections. MaxConcurrentRPCs is
	// the process-wide handler admission limit shared by unary and streaming
	// RPCs; MaxConcurrentStreams is the per-connection HTTP/2 stream limit.
	MaxConnections       int `koanf:"max_connections"`
	MaxConcurrentRPCs    int `koanf:"max_concurrent_rpcs"`
	MaxConcurrentStreams int `koanf:"max_concurrent_streams"`

	MaxHeaderListBytes     int  `koanf:"max_header_list_bytes"`
	MaxReceiveMessageBytes int  `koanf:"max_receive_message_bytes"`
	MaxSendMessageBytes    int  `koanf:"max_send_message_bytes"`
	AccessLogHealthChecks  bool `koanf:"access_log_health_checks"`
	// AccessLogSuccessSampleRate applies only to successful RPCs below
	// AccessLogSlowThreshold. Errors always remain visible while INFO logging is
	// enabled. The default of 1 preserves one completion record per business RPC.
	AccessLogSuccessSampleRate float64       `koanf:"access_log_success_sample_rate"`
	AccessLogSlowThreshold     time.Duration `koanf:"access_log_slow_threshold"`
	// TelemetryHealthChecks re-enables server-side OTel spans and duration
	// metrics for routine standard health polling.
	TelemetryHealthChecks bool `koanf:"telemetry_health_checks"`

	// UnaryTimeout caps how long one unary RPC may occupy a handler, and
	// StreamTimeout does the same for a stream. Both derive the RPC context's
	// deadline, so a caller deadline that is already earlier wins. Non-positive
	// disables the cap for that kind.
	//
	// UnaryTimeout defaults to http.request_timeout's value so one service
	// answers on one budget over both transports out of the box; it is a
	// separate key so a deployment can still move the two apart. StreamTimeout
	// is off by default because a long-lived stream's duration policy belongs to
	// the feature that owns the stream.
	UnaryTimeout  time.Duration `koanf:"unary_timeout"`
	StreamTimeout time.Duration `koanf:"stream_timeout"`

	// The liveness bounds, always on. None can end an RPC in progress: the idle
	// clock runs only while nothing is outstanding, and the ping bound closes
	// only when a ping goes unanswered, which means the peer is gone. Without
	// them a dead peer holds a listener slot for grpc-go's two-hour default.
	//
	// MinClientPingInterval and PermitPingWithoutStream bound what a client may
	// do rather than what this server does. grpc-go's defaults reject a ping
	// more often than every five minutes and any ping with no active stream,
	// which would disconnect this repository's own client half.
	MaxConnectionIdle       time.Duration `koanf:"max_connection_idle"`
	ServerPingInterval      time.Duration `koanf:"server_ping_interval"`
	ServerPingTimeout       time.Duration `koanf:"server_ping_timeout"`
	MinClientPingInterval   time.Duration `koanf:"min_client_ping_interval"`
	PermitPingWithoutStream bool          `koanf:"permit_ping_without_stream"`

	// MaxConnectionAge rotates connections and is off by default, because it is
	// the only bound here that ends work in progress: at this age the connection
	// is drained with GOAWAY and force-closed once the grace expires, cutting
	// every RPC and stream still running. Enable it behind an L4 balancer or any
	// hop that pins a caller to one replica for a connection's lifetime, and
	// accept that a stream outliving the age ends with UNAVAILABLE.
	//
	// The grace must be positive whenever an age is set: grpc-go reads a zero
	// grace as infinity, which would drain the connection and never close it.
	MaxConnectionAge      time.Duration `koanf:"max_connection_age"`
	MaxConnectionAgeGrace time.Duration `koanf:"max_connection_age_grace"`
}

type GRPCTLSConfig struct {
	CertFile string `koanf:"cert_file"`
	KeyFile  string `koanf:"key_file"`
}

// profile:grpc:end

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
	// OTLPEndpoint is the OTLP HTTP endpoint URL (http or https). A value with
	// no path is a collector root and serves both signals, resolving to
	// /v1/traces and /v1/metrics. A value that already carries a path names the
	// traces endpoint only, and metrics then need OTLPMetricsEndpoint.
	OTLPEndpoint string `koanf:"otlp_endpoint"`
	// OTLPMetricsEndpoint overrides where metrics are pushed. A missing path
	// defaults to /v1/metrics. Empty falls back to OTLPEndpoint when that names
	// a root, then to the standard OpenTelemetry endpoint variables.
	OTLPMetricsEndpoint string `koanf:"otlp_metrics_endpoint"`
	// OTLPHeaders is the credential for the collector, and covers both signals
	// because a collector credential belongs to the collector rather than to one
	// signal.
	OTLPHeaders string `koanf:"otlp_headers"`
}

// profile:database-postgres:start

// PostgresConfig defines PostgreSQL connection, migration, and pool settings.
type PostgresConfig struct {
	Enabled                   bool          `koanf:"enabled"`
	DSN                       string        `koanf:"dsn"`
	ConnectTimeout            time.Duration `koanf:"connect_timeout"`
	HealthcheckTimeout        time.Duration `koanf:"healthcheck_timeout"`
	MigrationTimeout          time.Duration `koanf:"migration_timeout"`
	MigrationStatementTimeout time.Duration `koanf:"migration_statement_timeout"`
	MigrationLockTimeout      time.Duration `koanf:"migration_lock_timeout"`
	MaxOpenConns              int           `koanf:"max_open_conns"`
	// MinIdleConns keeps connections open through quiet periods. At zero, pgx
	// closes every idle connection after its idle timeout, so a low-traffic
	// service pays TCP, TLS, and authentication on the first request of each
	// spike — once per connection the spike needs, all at the same moment.
	MinIdleConns int `koanf:"min_idle_conns"`
	// AcquireTimeout bounds waiting for a pooled connection. Nothing else does:
	// an acquire waits out whatever remains of the request budget, so a slow
	// database does not shed, it queues — and once every in-flight slot is held
	// by a connection waiter, requests that touch no database at all are shed
	// for capacity the database took. Validated against http.request_timeout so
	// a caller that waits still has budget left to run its query.
	AcquireTimeout  time.Duration `koanf:"acquire_timeout"`
	ConnMaxLifetime time.Duration `koanf:"conn_max_lifetime"`
	// StatementTimeout bounds every runtime statement server-side, and bounds
	// how long a session may sit idle inside a transaction. The request budget
	// only cancels client-side: if the cancel request itself is lost, which is
	// the failure a slow dependency tends to produce, an abandoned query keeps
	// running and keeps its locks. Validated against http.request_timeout so the
	// two are one budget rather than two numbers.
	StatementTimeout time.Duration `koanf:"statement_timeout"`
}

// profile:database-postgres:end

// profile:outbox-postgres:start

type OutboxConfig struct {
	Enabled      bool          `koanf:"enabled"`
	PollInterval time.Duration `koanf:"poll_interval"`
	// BatchSize is how many events one lease covers. It trades round trips per
	// event against peak relay memory, which is this many stored envelopes.
	BatchSize int `koanf:"batch_size"`
	// PublishConcurrency is how many events of a batch may be in the publisher
	// at once. The claim query hands out at most one ready event per ordering
	// key, so concurrency never reorders a key.
	PublishConcurrency  int           `koanf:"publish_concurrency"`
	PublishTimeout      time.Duration `koanf:"publish_timeout"`
	LeaseDuration       time.Duration `koanf:"lease_duration"`
	MaxAttempts         int           `koanf:"max_attempts"`
	RetryBase           time.Duration `koanf:"retry_base"`
	RetryMax            time.Duration `koanf:"retry_max"`
	ObservationInterval time.Duration `koanf:"observation_interval"`
	CleanupInterval     time.Duration `koanf:"cleanup_interval"`
	PublishedRetention  time.Duration `koanf:"published_retention"`
	CleanupBatchSize    int           `koanf:"cleanup_batch_size"`
	DrainTimeout        time.Duration `koanf:"drain_timeout"`
}

// profile:outbox-postgres:end
