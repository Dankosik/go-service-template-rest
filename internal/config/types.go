package config

import (
	"log/slog"
	"time"
)

// Config is the immutable runtime snapshot built during startup.
//
// Each section's type, defaults, and validation live together in its own
// <section>_config.go, so a section a build profile removes leaves with that
// file rather than being cut out of three shared ones. What stays below is what
// a dedicated file would not pay for: sections that are both always present and
// small enough to read in place. HTTP and Observability are always present too
// and still have their own files, because size, not removability, is what earns
// one.
type Config struct {
	App  AppConfig  `koanf:"app"`
	HTTP HTTPConfig `koanf:"http"`
	// profile:authn-oidc-jwt:start
	Authn AuthnConfig `koanf:"authn"`
	// profile:authn-oidc-jwt:end
	// profile:outbound-auth-oauth2-client-credentials:start
	OutboundAuth OutboundAuthConfig `koanf:"outbound_auth"`
	// profile:outbound-auth-oauth2-client-credentials:end
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
	// profile:http-idempotency-postgres:start
	HTTPIdempotency HTTPIdempotencyConfig `koanf:"http_idempotency"`
	// profile:http-idempotency-postgres:end
	// profile:jobs-postgres:start
	Jobs JobsConfig `koanf:"jobs"`
	// profile:jobs-postgres:end
	// profile:webhooks-durable:start
	Webhooks WebhooksConfig `koanf:"webhooks"`
	// profile:webhooks-durable:end
	// profile:outbox-postgres:start
	Outbox OutboxConfig `koanf:"outbox"`
	// profile:outbox-postgres:end
	// profile:object-storage:start
	ObjectStorage ObjectStorageConfig `koanf:"object_storage"`
	// profile:object-storage:end
}

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
