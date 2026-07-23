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

func buildSnapshot(k *koanf.Koanf) (Config, error) {
	var cfg Config
	err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{
		DecoderConfig: &mapstructure.DecoderConfig{
			DecodeHook:       mapstructure.DecodeHookFuncType(decodeConfigValue),
			WeaklyTypedInput: false,
		},
	})
	if err != nil {
		return Config{}, fmt.Errorf("%w: %s", ErrParse, sanitizedSnapshotDecodeError(err))
	}

	normalizeConfigStrings(&cfg)
	return cfg, nil
}

func decodeConfigValue(_ reflect.Type, targetType reflect.Type, value any) (any, error) {
	switch {
	case targetType == durationType:
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
	case targetType == logLevelType:
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

	switch targetType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		integer, err := parseSignedInteger(value, targetType.Bits())
		if err != nil {
			return nil, newConfigValueDecodeError(err.Error())
		}
		converted := reflect.New(targetType).Elem()
		converted.SetInt(integer)
		return converted.Interface(), nil
	case reflect.Float64:
		number, err := parseFloat64(value)
		if err != nil {
			return nil, newConfigValueDecodeError(err.Error())
		}
		return number, nil
	case reflect.Bool:
		boolean, err := parseBool(value)
		if err != nil {
			return nil, newConfigValueDecodeError(err.Error())
		}
		return boolean, nil
	default:
		return value, nil
	}
}

func newConfigValueDecodeError(detail string) error {
	return &configValueDecodeError{detail: detail}
}

func sanitizedSnapshotDecodeError(err error) string {
	var valueErr *configValueDecodeError
	if !errors.As(err, &valueErr) {
		return "configuration values do not match the Config schema"
	}

	var decodeErr *mapstructure.DecodeError
	if errors.As(err, &decodeErr) && strings.TrimSpace(decodeErr.Name()) != "" {
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
	cfg.Observability.OTel.Exporter.OTLPTracesEndpoint = strings.TrimSpace(
		cfg.Observability.OTel.Exporter.OTLPTracesEndpoint,
	)
	cfg.Observability.OTel.Exporter.OTLPProtocol = strings.TrimSpace(
		cfg.Observability.OTel.Exporter.OTLPProtocol,
	)
}
