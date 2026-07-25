package config

import "github.com/example/go-service-template-rest/internal/observability/otelconfig"

// buildVersion is set by the production Docker build and remains "dev" for local builds.
var buildVersion = "dev"

func defaultValues() map[string]any {
	return map[string]any{
		"app.env":     "local",
		"app.version": buildVersion,

		"http.addr":              ":8080",
		"http.shutdown_timeout":  "30s",
		"http.readiness_timeout": "4s",
		// readiness_propagation_delay holds the drain open long enough for a load
		// balancer to notice /health/ready failing before connections stop being
		// accepted. It is production-shaped on purpose, so running the binary
		// directly waits it out on SIGTERM; env/.env.example sets 0s for local
		// iteration, and docs/railway-deployment-profile.md does the arithmetic
		// against the platform's draining window.
		"http.readiness_propagation_delay": "15s",
		"http.read_header_timeout":         "5s",
		"http.read_timeout":                "5s",
		// request_timeout stays below write_timeout so the budget expires while
		// the connection can still carry the 504 that reports it.
		"http.request_timeout":  "8s",
		"http.write_timeout":    "10s",
		"http.idle_timeout":     "60s",
		"http.max_header_bytes": 16 << 10,
		"http.max_body_bytes":   int64(1 << 20),
		// max_in_flight is deliberately larger than postgres.max_open_conns:
		// shedding must engage after the pool is saturated and requests start
		// queueing, not before it is used.
		"http.max_in_flight":            256,
		"http.access_log_health_probes": false,

		"health.refresh_interval":  "2s",
		"health.failure_threshold": 3,

		"log.level": "info",

		"runtime.memory_limit_ratio": 0.9,

		// The diagnostics listener binds every interface because a metrics
		// scraper runs in another pod: a loopback default answers only inside
		// this network namespace, so the service reports healthy and exports
		// nothing. The port is not published by the container image, and
		// observability.pprof.enabled stays off by default.
		"observability.metrics.addr":  ":9090",
		"observability.pprof.enabled": false,

		// profile:database-postgres:start
		"postgres.enabled":                     false,
		"postgres.dsn":                         "",
		"postgres.connect_timeout":             "3s",
		"postgres.healthcheck_timeout":         "3s",
		"postgres.migration_timeout":           "5m",
		"postgres.migration_statement_timeout": "2m",
		"postgres.migration_lock_timeout":      "15s",
		"postgres.max_open_conns":              25,
		"postgres.conn_max_lifetime":           "30m",
		"postgres.statement_timeout":           "8s",
		// profile:database-postgres:end

		"observability.otel.service_name":           "service",
		"observability.otel.traces_sampler":         otelconfig.DefaultTracesSampler,
		"observability.otel.traces_sampler_arg":     otelconfig.DefaultTracesSamplerArg,
		"observability.otel.exporter.otlp_endpoint": "",
		"observability.otel.exporter.otlp_headers":  "",
	}
}
