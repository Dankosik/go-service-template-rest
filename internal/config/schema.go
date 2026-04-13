package config

import (
	"maps"
	"reflect"
	"strings"
)

func knownConfigKeys() map[string]struct{} {
	leafKeys, _ := configSchemaKeySets(reflect.TypeFor[Config](), "")
	return leafKeys
}

func knownConfigSections() map[string]struct{} {
	_, sectionKeys := configSchemaKeySets(reflect.TypeFor[Config](), "")
	return sectionKeys
}

func configSchemaKeySets(typ reflect.Type, prefix string) (map[string]struct{}, map[string]struct{}) {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	leafKeys := make(map[string]struct{})
	sectionKeys := make(map[string]struct{})
	if typ.Kind() != reflect.Struct {
		return leafKeys, sectionKeys
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
			nestedLeafKeys, nestedSectionKeys := configSchemaKeySets(field.Type, key)
			maps.Copy(leafKeys, nestedLeafKeys)
			maps.Copy(sectionKeys, nestedSectionKeys)
			continue
		}
		leafKeys[key] = struct{}{}
	}

	return leafKeys, sectionKeys
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
