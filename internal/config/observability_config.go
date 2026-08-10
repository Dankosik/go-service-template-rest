package config

import (
	"fmt"
	"strings"

	"github.com/example/go-service-template-rest/internal/observability/otelconfig"
)

type ObservabilityConfig struct {
	Metrics MetricsConfig `koanf:"metrics"`
	Pprof   PprofConfig   `koanf:"pprof"`
	OTel    OTelConfig    `koanf:"otel"`
}

type MetricsConfig struct {
	// Addr is the private diagnostics listener. Empty disables HTTP exposition.
	Addr string `koanf:"addr"`
}

// PprofConfig gates the runtime profile handlers on the diagnostics listener.
type PprofConfig struct {
	// Enabled serves net/http/pprof on the diagnostics listener. It defaults to
	// false: the profile handlers disclose heap contents and /debug/pprof/cmdline
	// discloses process arguments, and the diagnostics listener binds every
	// interface so Prometheus can reach it. Turning this on is a deployment
	// decision about who can reach that port, not a template default.
	Enabled bool `koanf:"enabled"`
}

type OTelConfig struct {
	ServiceName      string             `koanf:"service_name"`
	TracesSampler    string             `koanf:"traces_sampler"`
	TracesSamplerArg float64            `koanf:"traces_sampler_arg"`
	Exporter         OTelExporterConfig `koanf:"exporter"`
}

type OTelExporterConfig struct {
	// OTLPEndpoint is the OTLP HTTP endpoint URL (http or https). A value with
	// no path is a collector root and serves both signals, resolving to
	// /v1/traces and /v1/metrics. A value that already carries a path names the
	// traces endpoint only, and metrics then need OTLPMetricsEndpoint.
	OTLPEndpoint string `koanf:"otlp_endpoint"`
	// OTLPMetricsEndpoint overrides where metrics are pushed. A missing path
	// defaults to /v1/metrics. Empty falls back to OTLPEndpoint when that names
	// a root, then to the standard OpenTelemetry endpoint variables.
	OTLPMetricsEndpoint string `koanf:"otlp_metrics_endpoint"`
	// OTLPHeaders is the credential for the collector, and covers both signals
	// because a collector credential belongs to the collector rather than to one
	// signal.
	OTLPHeaders string `koanf:"otlp_headers"`
}

// observabilityDefaults holds the service_name value that scripts/init-module.sh
// rewrites for a derived service. It keys that rewrite on "service_name" alone
// rather than on the gofmt column beside it, so adding a longer key to this map
// cannot silently turn the rewrite into a no-op.
func observabilityDefaults() map[string]any {
	return map[string]any{
		// The diagnostics listener binds every interface because a metrics
		// scraper runs in another pod: a loopback default answers only inside
		// this network namespace, so the service reports healthy and exports
		// nothing. The port is not published by the container image, and
		// observability.pprof.enabled stays off by default.
		"observability.metrics.addr":  ":9090",
		"observability.pprof.enabled": false,

		"observability.otel.service_name":                   "service",
		"observability.otel.traces_sampler":                 otelconfig.DefaultTracesSampler,
		"observability.otel.traces_sampler_arg":             otelconfig.DefaultTracesSamplerArg,
		"observability.otel.exporter.otlp_endpoint":         "",
		"observability.otel.exporter.otlp_metrics_endpoint": "",
		"observability.otel.exporter.otlp_headers":          "",
	}
}

func validateObservabilityConfig(cfg *ObservabilityConfig) error {
	cfg.Metrics.Addr = strings.TrimSpace(cfg.Metrics.Addr)
	// An empty address is how an operator turns the diagnostics listener off.
	if cfg.Metrics.Addr != "" {
		if err := validateHostPortAddr("observability.metrics.addr", cfg.Metrics.Addr); err != nil {
			return err
		}
	}
	// pprof is published by the listener that address binds, so enabling it
	// without one asks for profiles nothing will ever serve.
	if cfg.Pprof.Enabled && cfg.Metrics.Addr == "" {
		return fmt.Errorf(
			"%w: observability.pprof.enabled requires observability.metrics.addr, which serves the diagnostics listener",
			ErrValidate,
		)
	}
	cfg.OTel.ServiceName = strings.TrimSpace(cfg.OTel.ServiceName)
	cfg.OTel.TracesSampler = strings.TrimSpace(cfg.OTel.TracesSampler)
	cfg.OTel.Exporter.OTLPEndpoint = strings.TrimSpace(cfg.OTel.Exporter.OTLPEndpoint)
	cfg.OTel.Exporter.OTLPMetricsEndpoint = strings.TrimSpace(cfg.OTel.Exporter.OTLPMetricsEndpoint)
	if cfg.OTel.ServiceName == "" {
		return fmt.Errorf("%w: observability.otel.service_name cannot be empty", ErrValidate)
	}
	return validateObservabilitySampler(cfg.OTel.TracesSampler, cfg.OTel.TracesSamplerArg)
}

// validateObservabilitySampler prefixes the shared sampler rules with the
// section that owns them, so a rejection names the setting an operator can edit.
func validateObservabilitySampler(sampler string, samplerArg float64) error {
	if err := otelconfig.ValidateTraceSampler(sampler, samplerArg); err != nil {
		return fmt.Errorf("%w: observability.otel.%w", ErrValidate, err)
	}
	return nil
}
