// Package runtimeopts is what more than one composition root would otherwise
// answer separately: the adapter options a loaded configuration maps onto, the
// startup steps two of them install identically, and the teardown arithmetic all
// three of them charge their grace period for.
//
// Only what a second binary already needs lives here. Each binary owns its own
// startup flow, its own readiness and drain, and every option only they build;
// what they cannot own separately is the meaning of a configured value, because
// a field added to an adapter and to only two of three call sites is a binary
// that quietly runs without it. Keeping the mapping here is why there is one
// place to add it.
//
// [InstallTelemetry], [DiagnosticsServer], [ListenDiagnostics], and the teardown
// primitives are here on the same reasoning rather than as exceptions to it. Each
// replaced copies that had already drifted: a degraded exporter logged with
// different fields, a diagnostics join spelled once as a type and once inline,
// and a teardown stage that on two of three binaries handed its cleanup an
// already-expired context.
//
// It sits under cmd/ rather than internal/ because mapping configuration onto
// concrete adapters, and bounding a process teardown, are composition — which
// internal/config and the adapters deliberately do not own.
package runtimeopts

import (
	"io"
	"log/slog"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/observability/logctx"
)

// Resource builds the identity both signals attach to what they export.
// instanceID is resolved once per process with telemetry.ResolveInstanceID, so a
// span and a metric from the same replica carry the same resource.
func Resource(cfg config.Config, instanceID string) telemetry.ResourceConfig {
	return telemetry.ResourceConfig{
		ServiceName:       cfg.Observability.OTel.ServiceName,
		ServiceVersion:    cfg.App.Version,
		ServiceCommit:     cfg.App.Commit,
		ServiceInstanceID: instanceID,
		DeploymentEnv:     cfg.App.Env,
	}
}

// Tracing builds the tracing setup options.
func Tracing(cfg config.Config, instanceID string) telemetry.TracingConfig {
	return telemetry.TracingConfig{
		Resource:         Resource(cfg, instanceID),
		TracesSampler:    cfg.Observability.OTel.TracesSampler,
		TracesSamplerArg: cfg.Observability.OTel.TracesSamplerArg,
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
		Resource: Resource(cfg, instanceID),
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

// Logger builds the process logger every binary writes through, carrying
// [LoggerFields] and whatever extra attributes the binary adds to them.
//
// All three composition roots spelled this line, two of them identically and the
// third differing only by the component field [LoggerFields] already documents a
// binary as appending. Appending it is the variation, so it is a parameter here
// rather than a reason to write the construction again.
func Logger(out io.Writer, cfg config.Config, extra ...any) *slog.Logger {
	return logctx.NewProcessLogger(out, cfg.Log.Level).With(append(LoggerFields(cfg), extra...)...)
}
