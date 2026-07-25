package bootstrap

import "errors"

var errDependencyInit = errors.New("dependency init")

const (
	startupDependencyTelemetry      = "telemetry"
	startupDependencyNetworkPolicy  = "network_policy"
	startupDependencyIngressPolicy  = "ingress_policy"
	startupDependencyModeFeatureOff = "feature_off"
	startupDependencyModeConfigured = "configured"
)

const (
	startupLogComponentStartupProbes = "startup_probes"
	startupLogComponentShutdown      = "shutdown"

	startupOperationTelemetryInit  = "telemetry_init"
	startupOperationTelemetryFlush = "telemetry_flush"
)
