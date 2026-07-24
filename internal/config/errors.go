package config

import "errors"

var (
	ErrLoad         = errors.New("config load")
	ErrParse        = errors.New("config parse")
	ErrValidate     = errors.New("config validate")
	ErrUnknownKey   = errors.New("config unknown key")
	ErrSecretPolicy = errors.New("config secret policy")
)

const (
	ErrorTypeLoad         = "load"
	ErrorTypeParse        = "parse"
	ErrorTypeValidate     = "validate"
	ErrorTypeUnknownKey   = "unknown_key"
	ErrorTypeSecretPolicy = "secret_policy"
	ErrorTypeUnknown      = "unknown"
)

func ErrorType(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrUnknownKey):
		return ErrorTypeUnknownKey
	case errors.Is(err, ErrSecretPolicy):
		return ErrorTypeSecretPolicy
	case errors.Is(err, ErrValidate):
		return ErrorTypeValidate
	case errors.Is(err, ErrParse):
		return ErrorTypeParse
	case errors.Is(err, ErrLoad):
		return ErrorTypeLoad
	default:
		return ErrorTypeUnknown
	}
}
