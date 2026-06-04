package bootstrap

import "github.com/example/go-service-template-rest/internal/infra/telemetry"

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

var (
	startupPostgresDependencyLabels = newStartupDependencyProbeLabels(telemetry.StartupDependencyPostgres)
	startupRedisDependencyLabels    = newStartupDependencyProbeLabels(telemetry.StartupDependencyRedis)
	startupMongoDependencyLabels    = newStartupDependencyProbeLabels(telemetry.StartupDependencyMongo)
)

const (
	startupDependencyTelemetry       = telemetry.StartupDependencyTelemetry
	startupDependencyNetworkPolicy   = telemetry.StartupDependencyNetworkPolicy
	startupDependencyIngressPolicy   = telemetry.StartupDependencyIngressPolicy
	startupDependencyMetricsExposure = telemetry.StartupDependencyMetricsExposure
	startupDependencyEgressException = telemetry.StartupDependencyEgressException
)

const (
	startupDependencyModeDisabled                = telemetry.StartupDependencyModeDisabled
	startupDependencyModeCriticalFailClosed      = telemetry.StartupDependencyModeCriticalFailClosed
	startupDependencyModeCriticalFailDegraded    = telemetry.StartupDependencyModeCriticalFailDegraded
	startupDependencyModeOptionalFailOpen        = telemetry.StartupDependencyModeOptionalFailOpen
	startupDependencyModeFeatureOff              = telemetry.StartupDependencyModeFeatureOff
	startupDependencyModeDegradedReadOnlyOrStale = telemetry.StartupDependencyModeDegradedReadOnlyOrStale
)

const (
	startupLogComponentStartupProbes = "startup_probes"
	startupLogComponentShutdown      = "shutdown"

	startupOperationTelemetryInit  = "telemetry_init"
	startupOperationTelemetryFlush = "telemetry_flush"
)
