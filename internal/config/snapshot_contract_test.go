package config

import (
	"context"
	"log/slog"
	"maps"
	"path/filepath"
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
	knownKeys := sortedStringSetKeys(knownConfigKeys())
	sourceKeys := sortedStringSetKeys(sourceValues)
	if !reflect.DeepEqual(sourceKeys, knownKeys) {
		t.Fatalf("sentinel source keys = %v, want known config keys %v", sourceKeys, knownKeys)
	}

	k := koanf.New(keyDelimiter)
	if err := k.Load(confmap.Provider(sourceValues, keyDelimiter), nil); err != nil {
		t.Fatalf("load sentinel config source: %v", err)
	}

	cfg, err := buildSnapshot(k)
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

func TestKnownConfigKeysMatchSnapshotTags(t *testing.T) {
	t.Parallel()

	knownKeys := sortedStringSetKeys(knownConfigKeys())

	tagKeys := configLeafKeysFromType(t, reflect.TypeFor[Config](), "")
	sort.Strings(tagKeys)
	if !reflect.DeepEqual(knownKeys, tagKeys) {
		t.Fatalf("knownConfigKeys() = %v, want Config koanf leaf keys %v", knownKeys, tagKeys)
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

	knownKeys := knownConfigKeys()
	for key := range defaultValues() {
		if _, ok := knownKeys[key]; !ok {
			t.Fatalf("defaultValues() contains %q, which is not a known Config koanf leaf key", key)
		}
	}
}

func TestDefaultConfigYAMLMatchesCodeDefaults(t *testing.T) {
	t.Parallel()

	defaultKoanf := koanf.New(keyDelimiter)
	if err := defaultKoanf.Load(confmap.Provider(defaultValues(), keyDelimiter), nil); err != nil {
		t.Fatalf("load code defaults: %v", err)
	}
	defaultSnapshot, err := buildSnapshot(defaultKoanf)
	if err != nil {
		t.Fatalf("buildSnapshot(defaultValues()) error = %v", err)
	}

	yamlKoanf := koanf.New(keyDelimiter)
	path := filepath.Join("..", "..", "env", "config", "default.yaml")
	if err := loadConfigFile(context.Background(), yamlKoanf, path, configFilePolicyLocal); err != nil {
		t.Fatalf("load default config yaml: %v", err)
	}

	defaultKeys := sortedStringSetKeys(defaultValues())
	yamlKeys := yamlKoanf.Keys()
	sort.Strings(yamlKeys)
	if !reflect.DeepEqual(yamlKeys, defaultKeys) {
		t.Fatalf("env/config/default.yaml keys = %v, want code default keys %v", yamlKeys, defaultKeys)
	}

	yamlSnapshot, err := buildSnapshot(yamlKoanf)
	if err != nil {
		t.Fatalf("buildSnapshot(env/config/default.yaml) error = %v", err)
	}
	if !reflect.DeepEqual(yamlSnapshot, defaultSnapshot) {
		t.Fatalf("env/config/default.yaml snapshot = %+v, want code defaults %+v", yamlSnapshot, defaultSnapshot)
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
		"app.env":     "stage",
		"app.version": "v-snapshot-test",

		"http.addr":                        ":18080",
		"http.shutdown_timeout":            "31s",
		"http.readiness_timeout":           "4s",
		"http.readiness_propagation_delay": "16s",
		"http.read_header_timeout":         "6s",
		"http.read_timeout":                "7s",
		"http.write_timeout":               "11s",
		"http.idle_timeout":                "61s",
		"http.max_header_bytes":            20 << 10,
		"http.max_body_bytes":              int64(2 << 20),

		"log.level": "warn",

		"postgres.enabled":             true,
		"postgres.dsn":                 "postgres://app:secret@db:5432/app?sslmode=disable",
		"postgres.connect_timeout":     "17s",
		"postgres.healthcheck_timeout": "18s",
		"postgres.max_open_conns":      26,
		"postgres.conn_max_lifetime":   "45m",

		"observability.otel.service_name":                  "snapshot-service",
		"observability.otel.traces_sampler":                "always_on",
		"observability.otel.traces_sampler_arg":            0.25,
		"observability.otel.exporter.otlp_endpoint":        "https://otel.example.com:4318",
		"observability.otel.exporter.otlp_traces_endpoint": "https://otel.example.com:4318/v1/traces",
		"observability.otel.exporter.otlp_headers":         "authorization=Bearer snapshot",
		"observability.otel.exporter.otlp_protocol":        "grpc",

		"feature_flags.postgres_readiness_probe": false,
	}
}

func expectedSentinelSnapshotValues() map[string]any {
	return map[string]any{
		"app.env":     "stage",
		"app.version": "v-snapshot-test",

		"http.addr":                        ":18080",
		"http.shutdown_timeout":            31 * time.Second,
		"http.readiness_timeout":           4 * time.Second,
		"http.readiness_propagation_delay": 16 * time.Second,
		"http.read_header_timeout":         6 * time.Second,
		"http.read_timeout":                7 * time.Second,
		"http.write_timeout":               11 * time.Second,
		"http.idle_timeout":                61 * time.Second,
		"http.max_header_bytes":            20 << 10,
		"http.max_body_bytes":              int64(2 << 20),

		"log.level": slog.LevelWarn,

		"postgres.enabled":             true,
		"postgres.dsn":                 "postgres://app:secret@db:5432/app?sslmode=disable",
		"postgres.connect_timeout":     17 * time.Second,
		"postgres.healthcheck_timeout": 18 * time.Second,
		"postgres.max_open_conns":      26,
		"postgres.conn_max_lifetime":   45 * time.Minute,

		"observability.otel.service_name":                  "snapshot-service",
		"observability.otel.traces_sampler":                "always_on",
		"observability.otel.traces_sampler_arg":            0.25,
		"observability.otel.exporter.otlp_endpoint":        "https://otel.example.com:4318",
		"observability.otel.exporter.otlp_traces_endpoint": "https://otel.example.com:4318/v1/traces",
		"observability.otel.exporter.otlp_headers":         "authorization=Bearer snapshot",
		"observability.otel.exporter.otlp_protocol":        "grpc",

		"feature_flags.postgres_readiness_probe": false,
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
