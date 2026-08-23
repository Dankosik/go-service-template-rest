package postgreswebhook

import "errors"

var (
	ErrConfig   = errors.New("postgres webhook configuration invalid")
	ErrConflict = errors.New("postgres webhook intent conflict")
	ErrNotFound = errors.New("postgres webhook record not found")

	errDestinationDenied = errors.New("postgres webhook destination denied")
	errSecretUnavailable = errors.New("postgres webhook signing secret unavailable")
	errResponseLimit     = errors.New("postgres webhook response limit exceeded")
)
