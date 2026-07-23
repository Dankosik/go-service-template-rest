package config

import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/knadh/koanf/v2"
)

var (
	durationType = reflect.TypeFor[time.Duration]()
	logLevelType = reflect.TypeFor[slog.Level]()
)

type configValueDecodeError struct {
	detail string
}

func (e *configValueDecodeError) Error() string {
	return e.detail
}

func buildSnapshot(k *koanf.Koanf) (Config, []string, error) {
	var cfg Config
	var metadata mapstructure.Metadata
	err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{
		DecoderConfig: &mapstructure.DecoderConfig{
			DecodeHook:       mapstructure.DecodeHookFuncType(decodeConfigValue),
			Metadata:         &metadata,
			WeaklyTypedInput: false,
		},
	})
	if err != nil {
		return Config{}, nil, fmt.Errorf("%w: %s", ErrParse, sanitizedSnapshotDecodeError(err))
	}

	normalizeConfigStrings(&cfg)
	return cfg, expandUnusedConfigKeys(metadata.Unused, k.Keys()), nil
}

func expandUnusedConfigKeys(unusedKeys, sourceLeafKeys []string) []string {
	expanded := make([]string, 0, len(unusedKeys))
	for _, unusedKey := range unusedKeys {
		prefix := unusedKey + keyDelimiter
		foundLeaf := false
		for _, sourceKey := range sourceLeafKeys {
			if sourceKey == unusedKey || strings.HasPrefix(sourceKey, prefix) {
				expanded = append(expanded, sourceKey)
				foundLeaf = true
			}
		}
		if !foundLeaf {
			expanded = append(expanded, unusedKey)
		}
	}
	return expanded
}

func decodeConfigValue(_ reflect.Type, targetType reflect.Type, value any) (any, error) {
	switch targetType {
	case durationType:
		raw, ok := value.(string)
		if !ok {
			return nil, newConfigValueDecodeError("duration must be a string with a unit")
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil, newConfigValueDecodeError("duration is empty")
		}
		duration, err := parseDuration(raw)
		if err != nil {
			return nil, newConfigValueDecodeError(err.Error())
		}
		return duration, nil
	case logLevelType:
		raw, ok := value.(string)
		if !ok {
			return nil, newConfigValueDecodeError("log level must be a string")
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil, newConfigValueDecodeError("log level is empty")
		}
		var level slog.Level
		if err := level.UnmarshalText([]byte(raw)); err != nil {
			return nil, newConfigValueDecodeError("invalid log level")
		}
		return level, nil
	}

	kind := targetType.Kind()
	if kind >= reflect.Int && kind <= reflect.Int64 {
		integer, err := parseSignedInteger(value, targetType.Bits())
		if err != nil {
			return nil, newConfigValueDecodeError(err.Error())
		}
		converted := reflect.New(targetType).Elem()
		converted.SetInt(integer)
		return converted.Interface(), nil
	}
	if kind == reflect.Float64 {
		number, err := parseFloat64(value)
		if err != nil {
			return nil, newConfigValueDecodeError(err.Error())
		}
		return number, nil
	}
	if kind == reflect.Bool {
		boolean, err := parseBool(value)
		if err != nil {
			return nil, newConfigValueDecodeError(err.Error())
		}
		return boolean, nil
	}
	return value, nil
}

func newConfigValueDecodeError(detail string) error {
	return &configValueDecodeError{detail: detail}
}

func sanitizedSnapshotDecodeError(err error) string {
	valueErr, ok := errors.AsType[*configValueDecodeError](err)
	if !ok {
		return "configuration values do not match the Config schema"
	}

	if decodeErr, ok := errors.AsType[*mapstructure.DecodeError](err); ok && strings.TrimSpace(decodeErr.Name()) != "" {
		return fmt.Sprintf("%s has invalid value: %s", decodeErr.Name(), valueErr.detail)
	}
	return valueErr.detail
}

func normalizeConfigStrings(cfg *Config) {
	cfg.App.Env = strings.TrimSpace(cfg.App.Env)
	cfg.App.Version = strings.TrimSpace(cfg.App.Version)
	cfg.HTTP.Addr = strings.TrimSpace(cfg.HTTP.Addr)
	cfg.Observability.OTel.ServiceName = strings.TrimSpace(cfg.Observability.OTel.ServiceName)
	cfg.Observability.OTel.TracesSampler = strings.TrimSpace(cfg.Observability.OTel.TracesSampler)
	cfg.Observability.OTel.Exporter.OTLPEndpoint = strings.TrimSpace(cfg.Observability.OTel.Exporter.OTLPEndpoint)
}
