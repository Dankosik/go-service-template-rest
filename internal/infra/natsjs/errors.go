package natsjs

import "errors"

// Every failure this package produces wraps one of the four sentinels below,
// and both composition roots route on them rather than on message text. The
// "Errors" section of doc.go owns what each one means to a caller and is the
// one place that states it; this file exists so a reader looking for the set
// has somewhere to find it.
var (
	ErrRejected  = errors.New("messaging operation rejected")
	ErrAmbiguous = errors.New("messaging operation outcome ambiguous")
	ErrDraining  = errors.New("messaging runtime draining")
	ErrTerminal  = errors.New("messaging runtime terminal failure")
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
