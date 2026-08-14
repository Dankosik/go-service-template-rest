package config

import (
	"maps"
	"reflect"
	"strings"
)

func knownConfigSections() map[string]struct{} {
	return configSchemaSections(reflect.TypeFor[Config](), "")
}

func configSchemaSections(typ reflect.Type, prefix string) map[string]struct{} {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	sectionKeys := make(map[string]struct{})
	if typ.Kind() != reflect.Struct {
		return sectionKeys
	}

	for field := range typ.Fields() {
		tag := strings.TrimSpace(field.Tag.Get("koanf"))
		if tag == "" || tag == "-" {
			continue
		}

		key := tag
		if prefix != "" {
			key = prefix + keyDelimiter + tag
		}

		if configSchemaHasTaggedFields(field.Type) {
			sectionKeys[key] = struct{}{}
			maps.Copy(sectionKeys, configSchemaSections(field.Type, key))
		}
	}

	return sectionKeys
}

func configSchemaHasTaggedFields(typ reflect.Type) bool {
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

// removeSectionScalarOverridesInPlace drops every key that names a known section
// but carries a scalar, and reports the keys it dropped. A scalar there would
// otherwise replace the whole subtree during the merge, silently discarding the
// defaults beneath it.
//
// Each merge source calls this on its own tree before merging, so the rule holds
// no matter which source supplied the scalar: load_file.go for a YAML file and
// load_koanf.go for the namespaced environment.
func removeSectionScalarOverridesInPlace(values map[string]any) []string {
	return removeSectionScalarOverridesRecursive(values, "", knownConfigSections())
}

func removeSectionScalarOverridesRecursive(values map[string]any, prefix string, knownSections map[string]struct{}) []string {
	sectionScalarOverrideKeys := make([]string, 0)
	for key, value := range values {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + keyDelimiter + key
		}

		nested, isMap := value.(map[string]any)
		if _, ok := knownSections[fullKey]; ok && !isMap {
			sectionScalarOverrideKeys = append(sectionScalarOverrideKeys, fullKey)
			delete(values, key)
			continue
		}
		if !isMap {
			continue
		}
		sectionScalarOverrideKeys = append(sectionScalarOverrideKeys, removeSectionScalarOverridesRecursive(nested, fullKey, knownSections)...)
	}
	return sectionScalarOverrideKeys
}
