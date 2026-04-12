package config

import (
	"reflect"
	"strings"
)

func knownConfigKeys() map[string]struct{} {
	return configSchemaLeafKeySet(reflect.TypeOf(Config{}), "")
}

func configSchemaLeafKeySet(typ reflect.Type, prefix string) map[string]struct{} {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	keys := make(map[string]struct{})
	if typ.Kind() != reflect.Struct {
		return keys
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
			for nestedKey := range configSchemaLeafKeySet(field.Type, key) {
				keys[nestedKey] = struct{}{}
			}
			continue
		}
		keys[key] = struct{}{}
	}

	return keys
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
