package bootstrap

const (
	startupDependencyPostgres     = "postgres"
	startupPostgresProbeOperation = "postgres_probe"
	startupPostgresResolveStage   = "startup.resolve.postgres"
	startupPostgresProbeStage     = "startup.probe.postgres"
)

const (
	startupDependencyTelemetry       = "telemetry"
	startupDependencyNetworkPolicy   = "network_policy"
	startupDependencyIngressPolicy   = "ingress_policy"
	startupDependencyMetricsExposure = "metrics_exposure"
	startupDependencyEgressException = "egress_exception"
	startupDependencyModeFeatureOff  = "feature_off"
)

const (
	startupLogComponentStartupProbes = "startup_probes"
	startupLogComponentShutdown      = "shutdown"

	startupOperationTelemetryInit  = "telemetry_init"
	startupOperationTelemetryFlush = "telemetry_flush"
)
