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

type snapshotContractValue struct {
	source any
	want   any
}

func sameSnapshotValue(value any) snapshotContractValue {
	return snapshotContractValue{source: value, want: value}
}

func TestSnapshotContract(t *testing.T) {
	t.Parallel()

	contractValues := snapshotContractValues()
	knownKeys := configLeafKeysFromType(t, reflect.TypeFor[Config](), "")
	slices.Sort(knownKeys)
	contractKeys := sortedStringSetKeys(contractValues)
	if !slices.Equal(contractKeys, knownKeys) {
		t.Fatalf("snapshot contract keys = %v, want known config keys %v", contractKeys, knownKeys)
	}

	sourceValues := make(map[string]any, len(contractValues))
	for key, value := range contractValues {
		sourceValues[key] = value.source
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

	for _, key := range knownKeys {
		if diff := cmp.Diff(contractValues[key].want, observedValues[key]); diff != "" {
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

func snapshotContractValues() map[string]snapshotContractValue {
	return map[string]snapshotContractValue{
		"app.env":         sameSnapshotValue("stage"),
		"app.version":     sameSnapshotValue("v-snapshot-test"),
		"app.commit":      sameSnapshotValue("c0ffee-snapshot-test"),
		"app.instance_id": sameSnapshotValue("instance-snapshot-test"),

		"http.addr":                        sameSnapshotValue(":18080"),
		"http.grace_period":                {source: "61s", want: 61 * time.Second},
		"http.shutdown_timeout":            {source: "31s", want: 31 * time.Second},
		"http.readiness_timeout":           {source: "4s", want: 4 * time.Second},
		"http.readiness_propagation_delay": {source: "16s", want: 16 * time.Second},
		"http.read_header_timeout":         {source: "6s", want: 6 * time.Second},
		"http.read_timeout":                {source: "7s", want: 7 * time.Second},
		"http.request_timeout":             {source: "9s", want: 9 * time.Second},
		"http.write_timeout":               {source: "11s", want: 11 * time.Second},
		"http.idle_timeout":                {source: "61s", want: 61 * time.Second},
		"http.max_header_bytes":            sameSnapshotValue(20 << 10),
		"http.max_body_bytes":              sameSnapshotValue(int64(2 << 20)),
		"http.max_in_flight":               sameSnapshotValue(512),
		"http.max_connections":             sameSnapshotValue(1024),
		"http.access_log_health_probes":    sameSnapshotValue(true),

		// profile:authn-bearer:start
		"authn.issuer":   sameSnapshotValue("https://issuer.snapshot.example"),
		"authn.audience": sameSnapshotValue("snapshot-api"),
		// profile:authn-oidc-jwt:start
		"authn.token_profile": sameSnapshotValue("resource-server"),
		// profile:authn-oidc-jwt:end
		// profile:authn-oidc-introspection:start
		"authn.introspection_endpoint":            sameSnapshotValue("https://idp.snapshot.example/oauth/introspect"),
		"authn.introspection_target_class":        sameSnapshotValue("external-https"),
		"authn.introspection_private_host_suffix": sameSnapshotValue(""),
		"authn.introspection_client_id":           sameSnapshotValue("snapshot-rs"),
		"authn.introspection_client_secret":       sameSnapshotValue(" snapshot-introspection-secret "),
		// profile:authn-oidc-introspection:end
		// profile:authn-bearer:end

		// profile:grpc:start
		"grpc.server.enabled":            sameSnapshotValue(true),
		"grpc.server.addr":               sameSnapshotValue(":19091"),
		"grpc.server.transport_security": sameSnapshotValue("tls"),
		"grpc.server.tls.cert_file":      sameSnapshotValue("/run/secrets/snapshot.crt"),
		"grpc.server.tls.key_file":       sameSnapshotValue("/run/secrets/snapshot.key"),
		"grpc.server.tls.client_ca_file": sameSnapshotValue("/run/secrets/snapshot-clients.pem"),
		// profile:grpc:end

		"health.refresh_interval":  {source: "3s", want: 3 * time.Second},
		"health.failure_threshold": sameSnapshotValue(5),

		"log.level": {source: "warn", want: slog.LevelWarn},

		// profile:messaging-nats-jetstream:start
		"messaging.urls":                       sameSnapshotValue("tls://nats.snapshot.example:4222"),
		"messaging.credentials_file":           sameSnapshotValue("/run/secrets/nats.creds"),
		"messaging.root_ca_file":               sameSnapshotValue("/run/secrets/nats-ca.pem"),
		"messaging.allow_plaintext":            sameSnapshotValue(false),
		"messaging.allow_unauthenticated":      sameSnapshotValue(false),
		"messaging.stream":                     sameSnapshotValue("EVENTS"),
		"messaging.max_payload_bytes":          sameSnapshotValue(300 << 10),
		"messaging.worker.consumer":            sameSnapshotValue("snapshot-worker"),
		"messaging.worker.filter_subject":      sameSnapshotValue("events.snapshot.>"),
		"messaging.worker.dead_letter_subject": sameSnapshotValue("dead.snapshot"),
		"messaging.worker.max_concurrency":     sameSnapshotValue(9),
		// profile:messaging-nats-jetstream:end

		"runtime.memory_limit_ratio": sameSnapshotValue(0.75),

		// profile:database-postgres:start
		"postgres.enabled":        sameSnapshotValue(true),
		"postgres.dsn":            sameSnapshotValue("postgres://app:secret@db:5432/app?sslmode=disable"),
		"postgres.max_open_conns": sameSnapshotValue(26),
		// profile:database-postgres:end

		// profile:http-idempotency-postgres:start
		"http_idempotency.retention": {source: "24h", want: 24 * time.Hour},
		// profile:http-idempotency-postgres:end

		// profile:jobs-postgres:start
		"jobs.max_workers": sameSnapshotValue(7),
		// profile:jobs-postgres:end

		// profile:webhooks-durable:start
		"webhooks.enabled":        sameSnapshotValue(true),
		"webhooks.endpoints":      sameSnapshotValue(`{"endpoints":[]}`),
		"webhooks.static_secrets": sameSnapshotValue(`{"entries":[]}`),
		// profile:webhooks-durable:end
		// profile:inbound-webhooks-standard:start
		"inbound_webhooks.endpoints":      sameSnapshotValue(`{"endpoints":[]}`),
		"inbound_webhooks.static_secrets": sameSnapshotValue(`{"entries":[]}`),
		// profile:inbound-webhooks-standard:end

		// profile:object-storage:start
		"object_storage.provider":              sameSnapshotValue("amazon_s3"),
		"object_storage.endpoint":              sameSnapshotValue(""),
		"object_storage.region":                sameSnapshotValue("us-east-1"),
		"object_storage.bucket":                sameSnapshotValue("examplebucket"),
		"object_storage.expected_bucket_owner": sameSnapshotValue("123456789012"),
		"object_storage.credential_source":     sameSnapshotValue("aws_default"),
		"object_storage.max_object_bytes":      {source: 10485760, want: int64(10485760)},
		// profile:object-storage:end

		"observability.metrics.addr":                        sameSnapshotValue("127.0.0.1:19090"),
		"observability.pprof.enabled":                       sameSnapshotValue(true),
		"observability.otel.service_name":                   sameSnapshotValue("snapshot-service"),
		"observability.otel.traces_sampler":                 sameSnapshotValue("always_on"),
		"observability.otel.traces_sampler_arg":             sameSnapshotValue(0.25),
		"observability.otel.exporter.otlp_metrics_endpoint": sameSnapshotValue("https://sentinel-metrics.example/v1/metrics"),
		"observability.otel.exporter.otlp_endpoint":         sameSnapshotValue("https://otel.example.com:4318"),
		"observability.otel.exporter.otlp_headers":          sameSnapshotValue("authorization=Bearer snapshot"),
	}
}

func sortedStringSetKeys[V any](values map[string]V) []string {
	return slices.Sorted(maps.Keys(values))
}
