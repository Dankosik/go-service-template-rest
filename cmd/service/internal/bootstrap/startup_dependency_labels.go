package bootstrap

type startupDependencyProbeLabels struct {
	dependency   string
	operation    string
	resolveStage string
	probeStage   string
	probeName    string
}

func newStartupDependencyProbeLabels(dependency string) startupDependencyProbeLabels {
	return startupDependencyProbeLabels{
		dependency:   dependency,
		operation:    dependency + "_probe",
		resolveStage: "startup.resolve." + dependency,
		probeStage:   "startup.probe." + dependency,
		probeName:    dependency + "_startup_probe",
	}
}

var (
	startupPostgresDependencyLabels = newStartupDependencyProbeLabels("postgres")
	startupRedisDependencyLabels    = newStartupDependencyProbeLabels("redis")
	startupMongoDependencyLabels    = newStartupDependencyProbeLabels("mongo")
)

const (
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
)
