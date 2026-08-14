package postgreswebhook

import "errors"

var (
	ErrConfig            = errors.New("postgres webhook configuration invalid")
	ErrConflict          = errors.New("postgres webhook intent conflict")
	ErrPrivacyDeleted    = errors.New("postgres webhook content privacy deleted")
	ErrClockRegression   = errors.New("postgres webhook clock regression")
	ErrStaleAttempt      = errors.New("postgres webhook attempt is stale")
	ErrDestinationDenied = errors.New("postgres webhook destination denied")
	ErrDrainUnsafe       = errors.New("postgres webhook drain cleanup unsafe")
)
