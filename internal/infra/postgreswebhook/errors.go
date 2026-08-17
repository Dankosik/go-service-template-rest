package postgreswebhook

import "errors"

var (
	ErrConfig            = errors.New("postgres webhook configuration invalid")
	ErrConflict          = errors.New("postgres webhook intent conflict")
	ErrNotFound          = errors.New("postgres webhook record not found")
	ErrPrivacyDeleted    = errors.New("postgres webhook content privacy deleted")
	ErrClockRegression   = errors.New("postgres webhook clock regression")
	ErrStaleAttempt      = errors.New("postgres webhook attempt is stale")
	ErrDestinationDenied = errors.New("postgres webhook destination denied")
	ErrSecretUnavailable = errors.New("postgres webhook signing secret unavailable")
	ErrResponseLimit     = errors.New("postgres webhook response limit exceeded")
	ErrDrainUnsafe       = errors.New("postgres webhook drain cleanup unsafe")
)
