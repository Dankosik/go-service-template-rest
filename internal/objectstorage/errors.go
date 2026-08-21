package objectstorage

import "errors"

var (
	ErrInvalid        = errors.New("object storage input is invalid")
	ErrTooLarge       = errors.New("object storage object is too large")
	ErrBusy           = errors.New("object storage is busy")
	ErrNotFound       = errors.New("object storage object was not found")
	ErrAlreadyExists  = errors.New("object storage object already exists")
	ErrIntegrity      = errors.New("object storage integrity check failed")
	ErrOutcomeUnknown = errors.New("object storage outcome is unknown")
)
