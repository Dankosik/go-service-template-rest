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

func configSectionValueIsMap(value any) bool {
	_, ok := value.(map[string]any)
	return ok
}
