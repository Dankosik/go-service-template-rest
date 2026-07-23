package bootstrap

type startupDependencyProbeLabels struct {
	dependency   string
	operation    string
	resolveStage string
	probeStage   string
}

func newStartupDependencyProbeLabels(dependency string) startupDependencyProbeLabels {
	return startupDependencyProbeLabels{
		dependency:   dependency,
		operation:    dependency + "_probe",
		resolveStage: "startup.resolve." + dependency,
		probeStage:   "startup.probe." + dependency,
	}
}

var startupPostgresDependencyLabels = newStartupDependencyProbeLabels("postgres")

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
