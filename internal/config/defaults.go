package config

import "github.com/example/go-service-template-rest/internal/observability/otelconfig"

// buildVersion is set by the production Docker build and remains "dev" for local builds.
var buildVersion = "dev"

func defaultValues() map[string]any {
	return map[string]any{
		"app.env":     "local",
		"app.version": buildVersion,

		"http.addr":                        ":8080",
		"http.shutdown_timeout":            "30s",
		"http.readiness_timeout":           "4s",
		"http.readiness_propagation_delay": "15s",
		"http.read_header_timeout":         "5s",
		"http.read_timeout":                "5s",
		"http.write_timeout":               "10s",
		"http.idle_timeout":                "60s",
		"http.max_header_bytes":            16 << 10,
		"http.max_body_bytes":              int64(1 << 20),

		"log.level": "info",

		"postgres.enabled":             false,
		"postgres.dsn":                 "",
		"postgres.connect_timeout":     "3s",
		"postgres.healthcheck_timeout": "3s",
		"postgres.max_open_conns":      25,
		"postgres.conn_max_lifetime":   "30m",

		"observability.otel.service_name":           "service",
		"observability.otel.traces_sampler":         otelconfig.DefaultTracesSampler,
		"observability.otel.traces_sampler_arg":     otelconfig.DefaultTracesSamplerArg,
		"observability.otel.exporter.otlp_endpoint": "",
		"observability.otel.exporter.otlp_headers":  "",
	}
}
