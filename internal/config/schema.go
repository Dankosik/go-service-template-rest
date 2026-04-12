package config

import (
	"reflect"
	"strings"
)

func knownConfigKeys() map[string]struct{} {
	leafKeys, _ := configSchemaKeySets(reflect.TypeOf(Config{}), "")
	return leafKeys
}

func knownConfigSections() map[string]struct{} {
	_, sectionKeys := configSchemaKeySets(reflect.TypeOf(Config{}), "")
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

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
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
			for nestedKey := range nestedLeafKeys {
				leafKeys[nestedKey] = struct{}{}
			}
			for nestedKey := range nestedSectionKeys {
				sectionKeys[nestedKey] = struct{}{}
			}
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
	for i := 0; i < typ.NumField(); i++ {
		if strings.TrimSpace(typ.Field(i).Tag.Get("koanf")) != "" {
			return true
		}
	}
	return false
}

func configSectionValueIsMap(value any) bool {
	_, ok := value.(map[string]any)
	return ok
}
