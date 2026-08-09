// Package runtimeopts maps one loaded configuration onto the adapter options
// more than one composition root builds from it.
//
// Only mappings a second binary already needs live here. `cmd/service`,
// `cmd/worker`, and `cmd/outbox-relay` each own their own startup flow, their own
// readiness and drain, and every option only they build; what they cannot own
// separately is the meaning of a configured value, because a field added to an
// adapter and to only two of three call sites is a binary that quietly runs
// without it. Keeping the mapping here is why there is one place to add it.
//
// It sits under cmd/ rather than internal/ because mapping configuration onto
// concrete adapters is composition, which internal/config and the adapters
// deliberately do not own.
package runtimeopts

import (
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

// Tracing builds the tracing setup options. instanceID is resolved once per
// process with telemetry.ResolveInstanceID and passed to both signals, so a span
// and a metric from the same replica carry the same resource identity.
func Tracing(cfg config.Config, instanceID string) telemetry.TracingConfig {
	return telemetry.TracingConfig{
		ServiceName:       cfg.Observability.OTel.ServiceName,
		ServiceVersion:    cfg.App.Version,
		ServiceCommit:     cfg.App.Commit,
		ServiceInstanceID: instanceID,
		DeploymentEnv:     cfg.App.Env,
		TracesSampler:     cfg.Observability.OTel.TracesSampler,
		TracesSamplerArg:  cfg.Observability.OTel.TracesSamplerArg,
		Exporter: telemetry.TraceExporterConfig{
			OTLPEndpoint: cfg.Observability.OTel.Exporter.OTLPEndpoint,
			OTLPHeaders:  cfg.Observability.OTel.Exporter.OTLPHeaders,
		},
	}
}

// Metrics builds the metrics setup options against the same resource identity
// [Tracing] was given.
func Metrics(cfg config.Config, instanceID string) telemetry.MetricsConfig {
	return telemetry.MetricsConfig{
		ServiceName:       cfg.Observability.OTel.ServiceName,
		ServiceVersion:    cfg.App.Version,
		ServiceCommit:     cfg.App.Commit,
		ServiceInstanceID: instanceID,
		DeploymentEnv:     cfg.App.Env,
		Exporter: telemetry.MetricExporterConfig{
			OTLPEndpoint:       cfg.Observability.OTel.Exporter.OTLPMetricsEndpoint,
			SharedOTLPEndpoint: cfg.Observability.OTel.Exporter.OTLPEndpoint,
			OTLPHeaders:        cfg.Observability.OTel.Exporter.OTLPHeaders,
		},
	}
}

// LoggerFields are the service identity attributes every process record carries.
// A binary appends its own component field to them.
func LoggerFields(cfg config.Config) []any {
	return []any{
		"service.name", cfg.Observability.OTel.ServiceName,
		"service.version", cfg.App.Version,
		"deployment.environment.name", cfg.App.Env,
	}
}
