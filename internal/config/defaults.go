package config

func defaultValues() map[string]any {
	return map[string]any{
		"app.env":     "local",
		"app.version": "dev",

		"http.addr":                ":8080",
		"http.shutdown_timeout":    "10s",
		"http.read_header_timeout": "5s",
		"http.read_timeout":        "5s",
		"http.write_timeout":       "10s",
		"http.idle_timeout":        "60s",
		"http.max_header_bytes":    16 << 10,
		"http.max_body_bytes":      int64(1 << 20),

		"log.level": "info",

		"postgres.enabled":             false,
		"postgres.dsn":                 "",
		"postgres.connect_timeout":     "3s",
		"postgres.healthcheck_timeout": "3s",
		"postgres.max_open_conns":      25,
		"postgres.max_idle_conns":      10,
		"postgres.conn_max_lifetime":   "30m",

		"redis.enabled":                  false,
		"redis.mode":                     "cache",
		"redis.allow_store_mode":         false,
		"redis.addr":                     "127.0.0.1:6379",
		"redis.username":                 "",
		"redis.password":                 "",
		"redis.db":                       0,
		"redis.dial_timeout":             "2s",
		"redis.read_timeout":             "1s",
		"redis.write_timeout":            "1s",
		"redis.pool_size":                20,
		"redis.key_prefix":               "service",
		"redis.fresh_ttl":                "60s",
		"redis.stale_window":             "0s",
		"redis.negative_ttl":             "10s",
		"redis.ttl_jitter_percent":       10,
		"redis.enable_singleflight":      true,
		"redis.max_fallback_concurrency": 32,

		"mongo.enabled":                  false,
		"mongo.uri":                      "",
		"mongo.database":                 "app",
		"mongo.connect_timeout":          "5s",
		"mongo.server_selection_timeout": "3s",
		"mongo.max_pool_size":            100,

		"observability.otel.service_name":                  "service",
		"observability.otel.traces_sampler":                "parentbased_traceidratio",
		"observability.otel.traces_sampler_arg":            0.10,
		"observability.otel.exporter.otlp_endpoint":        "",
		"observability.otel.exporter.otlp_traces_endpoint": "",
		"observability.otel.exporter.otlp_headers":         "",
		"observability.otel.exporter.otlp_protocol":        "http/protobuf",

		"observability.metrics.enabled":             true,
		"observability.metrics.path":                "/metrics",
		"observability.grafana.enabled":             false,
		"observability.grafana.cloud_otlp_endpoint": "",

		"feature_flags.postgres_readiness_probe": true,
		"feature_flags.mongo_readiness_probe":    false,
		"feature_flags.redis_readiness_probe":    false,
	}
}

func knownConfigKeys() map[string]struct{} {
	keys := make(map[string]struct{}, len(defaultValues()))
	for key := range defaultValues() {
		keys[key] = struct{}{}
	}
	return keys
}
