package config

import (
	"log/slog"
	"maps"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
)

func TestBuildSnapshotMapsEveryKnownConfigLeafKey(t *testing.T) {
	t.Parallel()

	sourceValues := sentinelConfigSourceValues()
	knownKeys := configLeafKeysFromType(t, reflect.TypeFor[Config](), "")
	sort.Strings(knownKeys)
	sourceKeys := sortedStringSetKeys(sourceValues)
	if !reflect.DeepEqual(sourceKeys, knownKeys) {
		t.Fatalf("sentinel source keys = %v, want known config keys %v", sourceKeys, knownKeys)
	}

	k := koanf.New(keyDelimiter)
	if err := k.Load(confmap.Provider(sourceValues, keyDelimiter), nil); err != nil {
		t.Fatalf("load sentinel config source: %v", err)
	}

	cfg, _, err := buildSnapshot(k)
	if err != nil {
		t.Fatalf("buildSnapshot() error = %v", err)
	}

	observedValues := flattenConfigSnapshotValues(t, reflect.ValueOf(cfg), "")
	observedKeys := sortedStringSetKeys(observedValues)
	if !reflect.DeepEqual(observedKeys, knownKeys) {
		t.Fatalf("flattened Config keys = %v, want known config keys %v", observedKeys, knownKeys)
	}

	expectedValues := expectedSentinelSnapshotValues()
	expectedKeys := sortedStringSetKeys(expectedValues)
	if !reflect.DeepEqual(expectedKeys, knownKeys) {
		t.Fatalf("expected sentinel keys = %v, want known config keys %v", expectedKeys, knownKeys)
	}

	for _, key := range knownKeys {
		if got, want := observedValues[key], expectedValues[key]; !reflect.DeepEqual(got, want) {
			t.Fatalf("buildSnapshot() value for %s = %#v (%T), want %#v (%T)", key, got, got, want, want)
		}
	}
}

func TestKnownConfigSectionsMatchSnapshotTags(t *testing.T) {
	t.Parallel()

	knownSections := sortedStringSetKeys(knownConfigSections())

	tagSections := configSectionKeysFromType(t, reflect.TypeFor[Config](), "")
	sort.Strings(tagSections)
	if !reflect.DeepEqual(knownSections, tagSections) {
		t.Fatalf("knownConfigSections() = %v, want Config koanf section keys %v", knownSections, tagSections)
	}
}

func TestDefaultValuesAreSubsetOfKnownConfigKeys(t *testing.T) {
	t.Parallel()

	knownKeys := make(map[string]struct{})
	for _, key := range configLeafKeysFromType(t, reflect.TypeFor[Config](), "") {
		knownKeys[key] = struct{}{}
	}
	for key := range defaultValues() {
		if _, ok := knownKeys[key]; !ok {
			t.Fatalf("defaultValues() contains %q, which is not a known Config koanf leaf key", key)
		}
	}
}

func configLeafKeysFromType(t *testing.T, typ reflect.Type, prefix string) []string {
	t.Helper()

	if typ.Kind() != reflect.Struct {
		t.Fatalf("configLeafKeysFromType(%s) called with non-struct type", typ)
	}

	keys := make([]string, 0)
	for field := range typ.Fields() {
		tag := strings.TrimSpace(field.Tag.Get("koanf"))
		if tag == "" || tag == "-" {
			t.Fatalf("%s.%s must declare a concrete koanf tag", typ.Name(), field.Name)
		}

		key := tag
		if prefix != "" {
			key = prefix + keyDelimiter + tag
		}

		if hasKoanfTaggedFields(field.Type) {
			keys = append(keys, configLeafKeysFromType(t, field.Type, key)...)
			continue
		}
		keys = append(keys, key)
	}
	return keys
}

func configSectionKeysFromType(t *testing.T, typ reflect.Type, prefix string) []string {
	t.Helper()

	if typ.Kind() != reflect.Struct {
		t.Fatalf("configSectionKeysFromType(%s) called with non-struct type", typ)
	}

	keys := make([]string, 0)
	for field := range typ.Fields() {
		tag := strings.TrimSpace(field.Tag.Get("koanf"))
		if tag == "" || tag == "-" {
			t.Fatalf("%s.%s must declare a concrete koanf tag", typ.Name(), field.Name)
		}

		key := tag
		if prefix != "" {
			key = prefix + keyDelimiter + tag
		}

		if !hasKoanfTaggedFields(field.Type) {
			continue
		}
		keys = append(keys, key)
		keys = append(keys, configSectionKeysFromType(t, field.Type, key)...)
	}
	return keys
}

func hasKoanfTaggedFields(typ reflect.Type) bool {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return false
	}
	for field := range typ.Fields() {
		if strings.TrimSpace(field.Tag.Get("koanf")) != "" {
			return true
		}
	}
	return false
}

func flattenConfigSnapshotValues(t *testing.T, value reflect.Value, prefix string) map[string]any {
	t.Helper()

	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			t.Fatalf("flattenConfigSnapshotValues(%s) got nil pointer", value.Type())
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		t.Fatalf("flattenConfigSnapshotValues(%s) called with non-struct value", value.Type())
	}

	typ := value.Type()
	values := make(map[string]any)
	for field := range typ.Fields() {
		tag := strings.TrimSpace(field.Tag.Get("koanf"))
		if tag == "" || tag == "-" {
			t.Fatalf("%s.%s must declare a concrete koanf tag", typ.Name(), field.Name)
		}

		key := tag
		if prefix != "" {
			key = prefix + keyDelimiter + tag
		}

		fieldValue := value.FieldByIndex(field.Index)
		if hasKoanfTaggedFields(field.Type) {
			maps.Copy(values, flattenConfigSnapshotValues(t, fieldValue, key))
			continue
		}
		values[key] = fieldValue.Interface()
	}
	return values
}

func sentinelConfigSourceValues() map[string]any {
	return map[string]any{
		"app.env":         "stage",
		"app.version":     "v-snapshot-test",
		"app.commit":      "c0ffee-snapshot-test",
		"app.instance_id": "instance-snapshot-test",

		"http.addr":                        ":18080",
		"http.grace_period":                "61s",
		"http.shutdown_timeout":            "31s",
		"http.readiness_timeout":           "4s",
		"http.readiness_propagation_delay": "16s",
		"http.read_header_timeout":         "6s",
		"http.read_timeout":                "7s",
		"http.request_timeout":             "9s",
		"http.write_timeout":               "11s",
		"http.idle_timeout":                "61s",
		"http.max_header_bytes":            20 << 10,
		"http.max_body_bytes":              int64(2 << 20),
		"http.max_in_flight":               512,
		"http.max_connections":             1024,
		"http.access_log_health_probes":    true,

		// profile:grpc:start
		"grpc.server.enabled":                        true,
		"grpc.server.addr":                           ":19091",
		"grpc.server.transport_security":             "tls",
		"grpc.server.allow_plaintext":                false,
		"grpc.server.tls.cert_file":                  "/run/secrets/snapshot.crt",
		"grpc.server.tls.key_file":                   "/run/secrets/snapshot.key",
		"grpc.server.max_connections":                2048,
		"grpc.server.max_concurrent_rpcs":            384,
		"grpc.server.max_concurrent_streams":         128,
		"grpc.server.max_header_list_bytes":          24 << 10,
		"grpc.server.max_receive_message_bytes":      6 << 20,
		"grpc.server.max_send_message_bytes":         7 << 20,
		"grpc.server.access_log_health_checks":       true,
		"grpc.server.access_log_success_sample_rate": 0.25,
		"grpc.server.access_log_slow_threshold":      "750ms",
		"grpc.server.telemetry_health_checks":        true,
		// profile:grpc:end

		"health.refresh_interval":  "3s",
		"health.failure_threshold": 5,

		"log.level": "warn",

		"runtime.memory_limit_ratio": 0.75,

		// profile:database-postgres:start
		"postgres.enabled":                     true,
		"postgres.dsn":                         "postgres://app:secret@db:5432/app?sslmode=disable",
		"postgres.connect_timeout":             "17s",
		"postgres.healthcheck_timeout":         "18s",
		"postgres.migration_timeout":           "19m",
		"postgres.migration_statement_timeout": "3m",
		"postgres.migration_lock_timeout":      "19s",
		"postgres.max_open_conns":              26,
		"postgres.min_idle_conns":              3,
		"postgres.acquire_timeout":             "1500ms",
		"postgres.conn_max_lifetime":           "45m",
		"postgres.statement_timeout":           "7s",
		// profile:database-postgres:end

		"observability.metrics.addr":                        "127.0.0.1:19090",
		"observability.pprof.enabled":                       true,
		"observability.otel.service_name":                   "snapshot-service",
		"observability.otel.traces_sampler":                 "always_on",
		"observability.otel.traces_sampler_arg":             0.25,
		"observability.otel.exporter.otlp_metrics_endpoint": "https://sentinel-metrics.example/v1/metrics",
		"observability.otel.exporter.otlp_endpoint":         "https://otel.example.com:4318",
		"observability.otel.exporter.otlp_headers":          "authorization=Bearer snapshot",
	}
}

func expectedSentinelSnapshotValues() map[string]any {
	return map[string]any{
		"app.env":         "stage",
		"app.version":     "v-snapshot-test",
		"app.commit":      "c0ffee-snapshot-test",
		"app.instance_id": "instance-snapshot-test",

		"http.addr":                        ":18080",
		"http.grace_period":                61 * time.Second,
		"http.shutdown_timeout":            31 * time.Second,
		"http.readiness_timeout":           4 * time.Second,
		"http.readiness_propagation_delay": 16 * time.Second,
		"http.read_header_timeout":         6 * time.Second,
		"http.read_timeout":                7 * time.Second,
		"http.request_timeout":             9 * time.Second,
		"http.write_timeout":               11 * time.Second,
		"http.idle_timeout":                61 * time.Second,
		"http.max_header_bytes":            20 << 10,
		"http.max_body_bytes":              int64(2 << 20),
		"http.max_in_flight":               512,
		"http.max_connections":             1024,
		"http.access_log_health_probes":    true,

		// profile:grpc:start
		"grpc.server.enabled":                        true,
		"grpc.server.addr":                           ":19091",
		"grpc.server.transport_security":             "tls",
		"grpc.server.allow_plaintext":                false,
		"grpc.server.tls.cert_file":                  "/run/secrets/snapshot.crt",
		"grpc.server.tls.key_file":                   "/run/secrets/snapshot.key",
		"grpc.server.max_connections":                2048,
		"grpc.server.max_concurrent_rpcs":            384,
		"grpc.server.max_concurrent_streams":         128,
		"grpc.server.max_header_list_bytes":          24 << 10,
		"grpc.server.max_receive_message_bytes":      6 << 20,
		"grpc.server.max_send_message_bytes":         7 << 20,
		"grpc.server.access_log_health_checks":       true,
		"grpc.server.access_log_success_sample_rate": 0.25,
		"grpc.server.access_log_slow_threshold":      750 * time.Millisecond,
		"grpc.server.telemetry_health_checks":        true,
		// profile:grpc:end

		"health.refresh_interval":  3 * time.Second,
		"health.failure_threshold": 5,

		"log.level": slog.LevelWarn,

		"runtime.memory_limit_ratio": 0.75,

		// profile:database-postgres:start
		"postgres.enabled":                     true,
		"postgres.dsn":                         "postgres://app:secret@db:5432/app?sslmode=disable",
		"postgres.connect_timeout":             17 * time.Second,
		"postgres.healthcheck_timeout":         18 * time.Second,
		"postgres.migration_timeout":           19 * time.Minute,
		"postgres.migration_statement_timeout": 3 * time.Minute,
		"postgres.migration_lock_timeout":      19 * time.Second,
		"postgres.max_open_conns":              26,
		"postgres.min_idle_conns":              3,
		"postgres.acquire_timeout":             1500 * time.Millisecond,
		"postgres.conn_max_lifetime":           45 * time.Minute,
		"postgres.statement_timeout":           7 * time.Second,
		// profile:database-postgres:end

		"observability.metrics.addr":                        "127.0.0.1:19090",
		"observability.pprof.enabled":                       true,
		"observability.otel.service_name":                   "snapshot-service",
		"observability.otel.traces_sampler":                 "always_on",
		"observability.otel.traces_sampler_arg":             0.25,
		"observability.otel.exporter.otlp_metrics_endpoint": "https://sentinel-metrics.example/v1/metrics",
		"observability.otel.exporter.otlp_endpoint":         "https://otel.example.com:4318",
		"observability.otel.exporter.otlp_headers":          "authorization=Bearer snapshot",
	}
}

func sortedStringSetKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
