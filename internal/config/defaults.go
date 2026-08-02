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

// profile:grpc:start

// DefaultGRPCServerConfig returns the canonical production defaults without
// reading files or environment variables. Runtime loading and benchmark-only
// composition consume the same value so their transport bounds cannot drift.
func DefaultGRPCServerConfig() GRPCServerConfig {
	return GRPCServerConfig{
		Enabled:           false,
		Addr:              "",
		TransportSecurity: "",
		AllowPlaintext:    false,
		TLS: GRPCTLSConfig{
			CertFile: "",
			KeyFile:  "",
		},
		MaxConnections:             4096,
		MaxConcurrentRPCs:          256,
		MaxConcurrentStreams:       100,
		MaxHeaderListBytes:         16 << 10,
		MaxReceiveMessageBytes:     4 << 20,
		MaxSendMessageBytes:        4 << 20,
		AccessLogHealthChecks:      false,
		AccessLogSuccessSampleRate: 1,
		AccessLogSlowThreshold:     0,
		TelemetryHealthChecks:      false,
	}
}

// profile:grpc:end

func defaultValues() map[string]any {
	// profile:grpc:start
	grpcServer := DefaultGRPCServerConfig()
	// profile:grpc:end

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

		// profile:authn-oidc-jwt:start
		// Trust policy has no executable placeholder. OIDC profiles fail config
		// validation until deployment supplies every required value.
		"authn.issuer":              "",
		"authn.audience":            "",
		"authn.trusted_proxy_cidrs": "",
		// profile:authn-oidc-jwt:end

		// profile:grpc:start
		// gRPC is compiled into the enabled template profile but remains
		// disabled until a service chooses an address and an explicit transport
		// security mode.
		"grpc.server.enabled":                        grpcServer.Enabled,
		"grpc.server.addr":                           grpcServer.Addr,
		"grpc.server.transport_security":             grpcServer.TransportSecurity,
		"grpc.server.allow_plaintext":                grpcServer.AllowPlaintext,
		"grpc.server.tls.cert_file":                  grpcServer.TLS.CertFile,
		"grpc.server.tls.key_file":                   grpcServer.TLS.KeyFile,
		"grpc.server.max_connections":                grpcServer.MaxConnections,
		"grpc.server.max_concurrent_rpcs":            grpcServer.MaxConcurrentRPCs,
		"grpc.server.max_concurrent_streams":         grpcServer.MaxConcurrentStreams,
		"grpc.server.max_header_list_bytes":          grpcServer.MaxHeaderListBytes,
		"grpc.server.max_receive_message_bytes":      grpcServer.MaxReceiveMessageBytes,
		"grpc.server.max_send_message_bytes":         grpcServer.MaxSendMessageBytes,
		"grpc.server.access_log_health_checks":       grpcServer.AccessLogHealthChecks,
		"grpc.server.access_log_success_sample_rate": grpcServer.AccessLogSuccessSampleRate,
		"grpc.server.access_log_slow_threshold":      grpcServer.AccessLogSlowThreshold.String(),
		"grpc.server.telemetry_health_checks":        grpcServer.TelemetryHealthChecks,
		// profile:grpc:end

		"health.refresh_interval":  "2s",
		"health.failure_threshold": 3,

		"log.level": "info",

		// profile:messaging-nats-jetstream:start
		"messaging.enabled":                        false,
		"messaging.urls":                           "",
		"messaging.credentials_file":               "",
		"messaging.root_ca_file":                   "",
		"messaging.allow_plaintext":                false,
		"messaging.allow_unauthenticated":          false,
		"messaging.stream":                         "",
		"messaging.max_payload_bytes":              256 << 10,
		"messaging.max_pending_publishes":          64,
		"messaging.worker.consumer":                "",
		"messaging.worker.filter_subject":          "",
		"messaging.worker.dead_letter_subject":     "",
		"messaging.worker.max_concurrency":         8,
		"messaging.worker.max_delivery_bytes":      1 << 20,
		"messaging.worker.handler_timeout":         "30s",
		"messaging.worker.retry_delays":            "1s,5s,30s,2m",
		"messaging.worker.dead_letter_retry_delay": "30s",
		"messaging.worker.drain_timeout":           "20s",
		// profile:messaging-nats-jetstream:end

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
		// profile:database-postgres:end

		// profile:outbox-postgres:start
		"outbox.enabled":              false,
		"outbox.poll_interval":        "500ms",
		"outbox.publish_timeout":      "10s",
		"outbox.lease_duration":       "30s",
		"outbox.max_attempts":         10,
		"outbox.retry_base":           "1s",
		"outbox.retry_max":            "5m",
		"outbox.observation_interval": "5s",
		"outbox.cleanup_interval":     "1m",
		"outbox.published_retention":  "168h",
		"outbox.cleanup_batch_size":   1000,
		"outbox.drain_timeout":        "20s",
		// profile:outbox-postgres:end

		"observability.otel.service_name":                   "service",
		"observability.otel.traces_sampler":                 otelconfig.DefaultTracesSampler,
		"observability.otel.traces_sampler_arg":             otelconfig.DefaultTracesSamplerArg,
		"observability.otel.exporter.otlp_endpoint":         "",
		"observability.otel.exporter.otlp_metrics_endpoint": "",
		"observability.otel.exporter.otlp_headers":          "",
	}
}
