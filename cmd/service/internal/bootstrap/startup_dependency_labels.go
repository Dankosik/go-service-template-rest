package bootstrap

const (
	startupDependencyPostgres      = "postgres"
	startupDependencyRedis         = "redis"
	startupDependencyMongo         = "mongo"
	startupDependencyTelemetry     = "telemetry"
	startupDependencyNetworkPolicy = "network_policy"
	startupDependencyIngressPolicy = "ingress_policy"
	startupDependencyEgressPolicy  = "egress_exception"
)

const (
	startupDependencyModeDisabled                = "disabled"
	startupDependencyModeCriticalFailClosed      = "critical_fail_closed"
	startupDependencyModeCriticalFailDegraded    = "critical_fail_degraded"
	startupDependencyModeOptionalFailOpen        = "optional_fail_open"
	startupDependencyModeFeatureOff              = "feature_off"
	startupDependencyModeDegradedReadOnlyOrStale = "degraded_read_only_or_stale"
	startupDependencyModeStore                   = "store"
	startupDependencyModeCache                   = "cache"
)

const (
	startupLogComponentStartupProbes = "startup_probes"

	startupOperationTelemetryInit  = "telemetry_init"
	startupOperationTelemetryFlush = "telemetry_shutdown"
	startupOperationPostgresProbe  = "postgres_probe"
	startupOperationRedisProbe     = "redis_probe"
	startupOperationMongoProbe     = "mongo_probe"
)

const (
	startupResolveStagePostgres = "startup.resolve.postgres"
	startupResolveStageRedis    = "startup.resolve.redis"
	startupResolveStageMongo    = "startup.resolve.mongo"
	startupProbeStagePostgres   = "startup.probe.postgres"
	startupProbeStageRedis      = "startup.probe.redis"
	startupProbeStageMongo      = "startup.probe.mongo"
	startupProbeNamePostgres    = "postgres_startup_probe"
	startupProbeNameRedis       = "redis_startup_probe"
	startupProbeNameMongo       = "mongo_startup_probe"
)
