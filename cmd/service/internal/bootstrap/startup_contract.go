package bootstrap

import "errors"

var errDependencyInit = errors.New("dependency init")

const (
	startupDependencyTelemetry       = "telemetry"
	startupDependencyNetworkPolicy   = "network_policy"
	startupDependencyIngressPolicy   = "ingress_policy"
	startupDependencyEgressException = "egress_exception"
	startupDependencyModeFeatureOff  = "feature_off"
)

const (
	startupLogComponentStartupProbes = "startup_probes"
	startupLogComponentShutdown      = "shutdown"

	startupOperationTelemetryInit  = "telemetry_init"
	startupOperationTelemetryFlush = "telemetry_flush"
)
