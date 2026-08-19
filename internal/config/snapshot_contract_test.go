package config

import (
	"log/slog"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
)

func TestSnapshotContract(t *testing.T) {
	t.Parallel()

	sourceValues := sentinelConfigSourceValues()
	knownKeys := configLeafKeysFromType(t, reflect.TypeFor[Config](), "")
	slices.Sort(knownKeys)
	sourceKeys := sortedStringSetKeys(sourceValues)
	if !slices.Equal(sourceKeys, knownKeys) {
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
	if !slices.Equal(observedKeys, knownKeys) {
		t.Fatalf("flattened Config keys = %v, want known config keys %v", observedKeys, knownKeys)
	}

	expectedValues := expectedSentinelSnapshotValues()
	expectedKeys := sortedStringSetKeys(expectedValues)
	if !slices.Equal(expectedKeys, knownKeys) {
		t.Fatalf("expected sentinel keys = %v, want known config keys %v", expectedKeys, knownKeys)
	}

	for _, key := range knownKeys {
		if diff := cmp.Diff(expectedValues[key], observedValues[key]); diff != "" {
			t.Fatalf("buildSnapshot() value for %s mismatch (-want +got):\n%s", key, diff)
		}
	}
}

func TestKnownConfigSectionsMatchSnapshotTags(t *testing.T) {
	t.Parallel()

	knownSections := sortedStringSetKeys(knownConfigSections())

	tagSections := configSectionKeysFromType(t, reflect.TypeFor[Config](), "")
	slices.Sort(tagSections)
	if !slices.Equal(knownSections, tagSections) {
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

// TestEveryKnownConfigKeyHasADefault holds the direction
// [TestDefaultValuesAreSubsetOfKnownConfigKeys] does not. That one proves a
// default names a real leaf; this one proves a leaf was not forgotten.
//
// A section wired into Config but never given its maps.Copy in defaultValues
// loads at the Go zero value, and whether anything notices depends on whether
// that section's validator happens to reject zero. For a bool, an optional
// string, or a wide-open numeric range it notices nothing, and the service runs
// on a value nobody chose.
func TestEveryKnownConfigKeyHasADefault(t *testing.T) {
	t.Parallel()

	defaults := defaultValues()
	activeOnly := map[string]struct{}{
		// profile:http-idempotency-postgres:start
		"http_idempotency.retention": {},
		// profile:http-idempotency-postgres:end
	}
	missing := make([]string, 0)
	for _, key := range configLeafKeysFromType(t, reflect.TypeFor[Config](), "") {
		if _, defaulted := defaults[key]; defaulted {
			if _, mustBeExplicit := activeOnly[key]; mustBeExplicit {
				t.Errorf("defaultValues() contains active-only key %q", key)
			}
			continue
		}
		if _, mustBeExplicit := activeOnly[key]; mustBeExplicit {
			continue
		}
		missing = append(missing, key)
	}
	slices.Sort(missing)
	if len(missing) > 0 {
		t.Fatalf(
			"these Config leaf keys have no defaultValues() entry: %v; add the section's maps.Copy "+
				"to defaultValues, or state here why the operator has to supply it",
			missing,
		)
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

		// profile:authn-oidc-jwt:start
		"authn.issuer":              "https://issuer.snapshot.example",
		"authn.audience":            "snapshot-api",
		"authn.trusted_proxy_cidrs": "192.0.2.0/24",
		// profile:authn-oidc-jwt:end

		// profile:outbound-auth-oauth2-client-credentials:start
		"outbound_auth.dependency":                "snapshot-dependency",
		"outbound_auth.client_id":                 " snapshot-client:id ",
		"outbound_auth.client_secret":             " snapshot-client-secret ",
		"outbound_auth.client_authentication":     "client_secret_basic",
		"outbound_auth.token_endpoint":            "https://auth.snapshot.example/oauth/token",
		"outbound_auth.token_target_class":        "private_https",
		"outbound_auth.token_private_host_suffix": "snapshot.internal",
		"outbound_auth.scopes":                    "snapshot.read snapshot.write",
		"outbound_auth.resource":                  "https://resource.snapshot.example",
		"outbound_auth.audience":                  "",
		"outbound_auth.resource_authority":        "https://resource.snapshot.example",
		"outbound_auth.acquisition_timeout":       "4s",
		// profile:outbound-auth-oauth2-client-credentials:end

		// profile:grpc:start
		"grpc.server.enabled":                        true,
		"grpc.server.addr":                           ":19091",
		"grpc.server.transport_security":             "tls",
		"grpc.server.allow_plaintext":                false,
		"grpc.server.tls.cert_file":                  "/run/secrets/snapshot.crt",
		"grpc.server.tls.key_file":                   "/run/secrets/snapshot.key",
		"grpc.server.tls.client_ca_file":             "/run/secrets/snapshot-clients.pem",
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
		"grpc.server.unary_timeout":                  "9s",
		"grpc.server.stream_timeout":                 "45s",
		"grpc.server.max_connection_idle":            "16m",
		"grpc.server.server_ping_interval":           "2m",
		"grpc.server.server_ping_timeout":            "21s",
		"grpc.server.min_client_ping_interval":       "11s",
		"grpc.server.permit_ping_without_stream":     true,
		"grpc.server.max_connection_age":             "50s",
		"grpc.server.max_connection_age_grace":       "12s",
		// profile:grpc:end

		"health.refresh_interval":  "3s",
		"health.failure_threshold": 5,

		"log.level": "warn",

		// profile:messaging-nats-jetstream:start
		"messaging.enabled":                        true,
		"messaging.urls":                           "tls://nats.snapshot.example:4222",
		"messaging.credentials_file":               "/run/secrets/nats.creds",
		"messaging.root_ca_file":                   "/run/secrets/nats-ca.pem",
		"messaging.allow_plaintext":                false,
		"messaging.allow_unauthenticated":          false,
		"messaging.stream":                         "EVENTS",
		"messaging.min_stream_replicas":            3,
		"messaging.min_stream_retention":           "72h",
		"messaging.max_payload_bytes":              300 << 10,
		"messaging.max_pending_publishes":          17,
		"messaging.worker.consumer":                "snapshot-worker",
		"messaging.worker.filter_subject":          "events.snapshot.>",
		"messaging.worker.dead_letter_subject":     "dead.snapshot",
		"messaging.worker.max_concurrency":         9,
		"messaging.worker.max_delivery_bytes":      2 << 20,
		"messaging.worker.handler_timeout":         "31s",
		"messaging.worker.retry_delays":            "2s,7s",
		"messaging.worker.dead_letter_retry_delay": "41s",
		"messaging.worker.drain_timeout":           "21s",
		// profile:messaging-nats-jetstream:end

		"runtime.memory_limit_ratio": 0.75,

		// profile:database-postgres:start
		"postgres.enabled":        true,
		"postgres.dsn":            "postgres://app:secret@db:5432/app?sslmode=disable",
		"postgres.max_open_conns": 26,
		// profile:database-postgres:end

		// profile:http-idempotency-postgres:start
		"http_idempotency.retention": "24h",
		// profile:http-idempotency-postgres:end

		// profile:jobs-postgres:start
		"jobs.max_workers": 7,
		// profile:jobs-postgres:end

		// profile:webhooks-durable:start
		"webhooks.enabled":        true,
		"webhooks.endpoints":      `{"endpoints":[]}`,
		"webhooks.static_secrets": `{"entries":[]}`,
		// profile:webhooks-durable:end

		// profile:object-storage:start
		"object_storage.provider":                   "amazon_s3",
		"object_storage.endpoint":                   "https://s3.us-east-1.amazonaws.com",
		"object_storage.region":                     "us-east-1",
		"object_storage.bucket":                     "examplebucket",
		"object_storage.access_key_id":              "snapshot-access-key",
		"object_storage.secret_access_key":          "snapshot-secret-key",
		"object_storage.session_token":              "snapshot-session-token",
		"object_storage.expected_bucket_owner":      "123456789012",
		"object_storage.max_object_bytes":           10485760,
		"object_storage.multipart_chunk_bytes":      5242880,
		"object_storage.max_active_operations":      2,
		"object_storage.max_operation_duration":     "1s",
		"object_storage.max_presign_lifetime":       "1m",
		"object_storage.max_response_header_bytes":  1024,
		"object_storage.max_control_response_bytes": 1024,
		"object_storage.max_working_memory_bytes":   62_149_760,
		// profile:object-storage:end

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

		// profile:authn-oidc-jwt:start
		"authn.issuer":              "https://issuer.snapshot.example",
		"authn.audience":            "snapshot-api",
		"authn.trusted_proxy_cidrs": "192.0.2.0/24",
		// profile:authn-oidc-jwt:end

		// profile:outbound-auth-oauth2-client-credentials:start
		"outbound_auth.dependency":                "snapshot-dependency",
		"outbound_auth.client_id":                 " snapshot-client:id ",
		"outbound_auth.client_secret":             " snapshot-client-secret ",
		"outbound_auth.client_authentication":     "client_secret_basic",
		"outbound_auth.token_endpoint":            "https://auth.snapshot.example/oauth/token",
		"outbound_auth.token_target_class":        "private_https",
		"outbound_auth.token_private_host_suffix": "snapshot.internal",
		"outbound_auth.scopes":                    "snapshot.read snapshot.write",
		"outbound_auth.resource":                  "https://resource.snapshot.example",
		"outbound_auth.audience":                  "",
		"outbound_auth.resource_authority":        "https://resource.snapshot.example",
		"outbound_auth.acquisition_timeout":       4 * time.Second,
		// profile:outbound-auth-oauth2-client-credentials:end

		// profile:grpc:start
		"grpc.server.enabled":                        true,
		"grpc.server.addr":                           ":19091",
		"grpc.server.transport_security":             "tls",
		"grpc.server.allow_plaintext":                false,
		"grpc.server.tls.cert_file":                  "/run/secrets/snapshot.crt",
		"grpc.server.tls.key_file":                   "/run/secrets/snapshot.key",
		"grpc.server.tls.client_ca_file":             "/run/secrets/snapshot-clients.pem",
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
		"grpc.server.unary_timeout":                  9 * time.Second,
		"grpc.server.stream_timeout":                 45 * time.Second,
		"grpc.server.max_connection_idle":            16 * time.Minute,
		"grpc.server.server_ping_interval":           2 * time.Minute,
		"grpc.server.server_ping_timeout":            21 * time.Second,
		"grpc.server.min_client_ping_interval":       11 * time.Second,
		"grpc.server.permit_ping_without_stream":     true,
		"grpc.server.max_connection_age":             50 * time.Second,
		"grpc.server.max_connection_age_grace":       12 * time.Second,
		// profile:grpc:end

		"health.refresh_interval":  3 * time.Second,
		"health.failure_threshold": 5,

		"log.level": slog.LevelWarn,

		// profile:messaging-nats-jetstream:start
		"messaging.enabled":                        true,
		"messaging.urls":                           "tls://nats.snapshot.example:4222",
		"messaging.credentials_file":               "/run/secrets/nats.creds",
		"messaging.root_ca_file":                   "/run/secrets/nats-ca.pem",
		"messaging.allow_plaintext":                false,
		"messaging.allow_unauthenticated":          false,
		"messaging.stream":                         "EVENTS",
		"messaging.min_stream_replicas":            3,
		"messaging.min_stream_retention":           72 * time.Hour,
		"messaging.max_payload_bytes":              300 << 10,
		"messaging.max_pending_publishes":          17,
		"messaging.worker.consumer":                "snapshot-worker",
		"messaging.worker.filter_subject":          "events.snapshot.>",
		"messaging.worker.dead_letter_subject":     "dead.snapshot",
		"messaging.worker.max_concurrency":         9,
		"messaging.worker.max_delivery_bytes":      2 << 20,
		"messaging.worker.handler_timeout":         31 * time.Second,
		"messaging.worker.retry_delays":            "2s,7s",
		"messaging.worker.dead_letter_retry_delay": 41 * time.Second,
		"messaging.worker.drain_timeout":           21 * time.Second,
		// profile:messaging-nats-jetstream:end

		"runtime.memory_limit_ratio": 0.75,

		// profile:database-postgres:start
		"postgres.enabled":        true,
		"postgres.dsn":            "postgres://app:secret@db:5432/app?sslmode=disable",
		"postgres.max_open_conns": 26,
		// profile:database-postgres:end

		// profile:http-idempotency-postgres:start
		"http_idempotency.retention": 24 * time.Hour,
		// profile:http-idempotency-postgres:end

		// profile:jobs-postgres:start
		"jobs.max_workers": 7,
		// profile:jobs-postgres:end

		// profile:webhooks-durable:start
		"webhooks.enabled":        true,
		"webhooks.endpoints":      `{"endpoints":[]}`,
		"webhooks.static_secrets": `{"entries":[]}`,
		// profile:webhooks-durable:end

		// profile:object-storage:start
		"object_storage.provider":                   "amazon_s3",
		"object_storage.endpoint":                   "https://s3.us-east-1.amazonaws.com",
		"object_storage.region":                     "us-east-1",
		"object_storage.bucket":                     "examplebucket",
		"object_storage.access_key_id":              "snapshot-access-key",
		"object_storage.secret_access_key":          "snapshot-secret-key",
		"object_storage.session_token":              "snapshot-session-token",
		"object_storage.expected_bucket_owner":      "123456789012",
		"object_storage.max_object_bytes":           int64(10485760),
		"object_storage.multipart_chunk_bytes":      int64(5242880),
		"object_storage.max_active_operations":      2,
		"object_storage.max_operation_duration":     time.Second,
		"object_storage.max_presign_lifetime":       time.Minute,
		"object_storage.max_response_header_bytes":  int64(1024),
		"object_storage.max_control_response_bytes": int64(1024),
		"object_storage.max_working_memory_bytes":   int64(62_149_760),
		// profile:object-storage:end

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
	return slices.Sorted(maps.Keys(values))
}
