package objectstorage

import "errors"

var (
	// ErrInvalid reports a key, option, or ttl the port does not accept.
	ErrInvalid = errors.New("object storage input is invalid")
	// ErrTooLarge reports an object above the configured size limit.
	ErrTooLarge = errors.New("object storage object is too large")
	// ErrBusy reports that the store's concurrency limit is reached; retry later.
	ErrBusy = errors.New("object storage is busy")
	// ErrNotFound reports that no object exists under the key.
	ErrNotFound = errors.New("object storage object was not found")
	// ErrAlreadyExists reports a create-only upload whose key is already taken.
	ErrAlreadyExists = errors.New("object storage object already exists")
	// ErrIntegrity reports stored or transferred bytes that fail verification.
	ErrIntegrity = errors.New("object storage integrity check failed")
	// ErrOutcomeUnknown reports a mutation the provider may have applied;
	// establish the outcome before retrying.
	ErrOutcomeUnknown = errors.New("object storage outcome is unknown")
)
