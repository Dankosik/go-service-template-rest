package natsjs

import "errors"

// A refusal or fault this package decides wraps one of the sentinels below; a
// context's own cancellation and a collaborator's error, such as domain-event
// validation, keep their identity instead. A caller branches with errors.Is;
// message text is for people and may change.
var (
	// ErrRejected is a definite refusal: the operation did not take effect.
	ErrRejected = errors.New("messaging operation rejected")
	// ErrAmbiguous is a publish whose outcome is unknown: the broker may have
	// stored it. Retry with the same event identity so broker deduplication
	// can recognize it.
	ErrAmbiguous = errors.New("messaging operation outcome ambiguous")
	// ErrDraining accompanies [ErrRejected] when the client refuses new work
	// because it is shutting down.
	ErrDraining = errors.New("messaging runtime draining")
	// ErrTerminal is a fault that stops the worker, and with it the process.
	ErrTerminal = errors.New("messaging runtime terminal failure")
)

type permanentError struct{ err error }

func (e permanentError) Error() string { return e.err.Error() }
func (e permanentError) Unwrap() error { return e.err }

// Permanent marks bytes that retrying unchanged cannot make processable.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return permanentError{err: err}
}

func IsPermanent(err error) bool {
	_, ok := errors.AsType[permanentError](err)
	return ok
}
