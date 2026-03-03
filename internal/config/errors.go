package config

import "errors"

var (
	ErrLoad             = errors.New("config load")
	ErrParse            = errors.New("config parse")
	ErrValidate         = errors.New("config validate")
	ErrStrictUnknownKey = errors.New("config strict unknown key")
	ErrSecretPolicy     = errors.New("config secret policy")
	ErrDependencyInit   = errors.New("dependency init")
)

func ErrorType(err error) string {
	switch {
	case errors.Is(err, ErrStrictUnknownKey):
		return "strict_unknown_key"
	case errors.Is(err, ErrSecretPolicy):
		return "secret_policy"
	case errors.Is(err, ErrValidate):
		return "validate"
	case errors.Is(err, ErrParse):
		return "parse"
	case errors.Is(err, ErrDependencyInit):
		return "dependency_init"
	case errors.Is(err, ErrLoad):
		return "load"
	default:
		return "load"
	}
}
