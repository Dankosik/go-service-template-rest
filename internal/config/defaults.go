package config

import "github.com/example/go-service-template-rest/internal/observability/otelconfig"

// buildVersion is set by the production Docker build and remains "dev" for local builds.
var buildVersion = "dev"

// buildCommit is the source revision the binary was built from, stamped by the
// production Docker build.
//
// It is a linker variable rather than something read back from
// runtime/debug.ReadBuildInfo: the image build runs with -buildvcs=false, because
// .dockerignore keeps .git out of the build context, so the binary carries no
// vcs.revision of its own. Without this the commit existed only as an OCI image
// label — unreachable from inside the container, and unjoinable to a span, a
// metric, or a log line.
var buildCommit = "unknown"

func defaultValues() map[string]any {
	return map[string]any{
		"app.env":         "local",
		"app.version":     buildVersion,
		"app.commit":      buildCommit,
		"app.instance_id": "",

		"http.addr": ":8080",
		// grace_period matches railway.toml's drainingSeconds, the platform this
		// repository ships a deployment profile for. Kubernetes grants 30s by
		// default, so a deployment there sets both this and shutdown_timeout;
		// leaving them mismatched is what validateShutdownGraceBudget rejects at
		// startup rather than letting a SIGKILL discover it.
		"http.grace_period": "45s",
		// shutdown_timeout bounds the drain only. It is 25s rather than the whole
		// grace period because the teardown after the drain — diagnostics,
		// background join, pool release, telemetry flush — needs the remaining 17s,
		// and the flush that records how all three went is what runs last.
		"http.shutdown_timeout":  "25s",
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
		"http.max_in_flight": 256,
		// max_connections is well above max_in_flight, because the two bound
		// different things: shedding answers 503 with a Retry-After, while this
		// leaves a caller in the kernel backlog with no answer at all. The
		// headroom is what keeps the informative rejection the common one, and
		// this the backstop against a connection flood costing a goroutine and
		// two buffers apiece until the process is OOM-killed.
		"http.max_connections":          4096,
		"http.access_log_health_probes": false,
		// idempotency_outcome_timeout matches postgres.acquire_timeout: recording an
		// outcome is one short statement, and a store that cannot take it in a second
		// is not going to take it in ten. Overshooting costs handler goroutines that
		// are already holding an in-flight slot with nobody waiting on the answer.
		"http.idempotency_outcome_timeout": "1s",

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
		// min_idle_conns keeps a small warm floor so a low-traffic service does
		// not pay a full connect handshake on the first request after a quiet
		// period, once per connection the arriving burst needs.
		"postgres.min_idle_conns": 2,
		// acquire_timeout is a small fraction of http.request_timeout on purpose:
		// a caller that has waited this long is queued behind a saturated pool,
		// and telling it so in a second beats spending the rest of its budget in
		// a queue and answering 504. What is left of the budget is the query's.
		"postgres.acquire_timeout":   "1s",
		"postgres.conn_max_lifetime": "30m",
		"postgres.statement_timeout": "8s",
		// 24h covers every client retry policy worth honoring, and the sweep keeps
		// the table proportional to one day of unsafe requests rather than to the
		// lifetime of the service.
		"postgres.idempotency_retention":      "24h",
		"postgres.idempotency_sweep_interval": "5m",
		// profile:database-postgres:end

		"observability.otel.service_name":                   "service",
		"observability.otel.traces_sampler":                 otelconfig.DefaultTracesSampler,
		"observability.otel.traces_sampler_arg":             otelconfig.DefaultTracesSamplerArg,
		"observability.otel.exporter.otlp_endpoint":         "",
		"observability.otel.exporter.otlp_metrics_endpoint": "",
		"observability.otel.exporter.otlp_headers":          "",
	}
}
