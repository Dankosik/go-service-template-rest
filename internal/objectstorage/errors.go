package objectstorage

import "errors"

type ErrorKind string

const (
	KindInvalid            ErrorKind = "invalid"
	KindTooLarge           ErrorKind = "too_large"
	KindBusy               ErrorKind = "busy"
	KindNotFound           ErrorKind = "not_found"
	KindPreconditionFailed ErrorKind = "precondition_failed"
	KindDenied             ErrorKind = "denied"
	KindTemporary          ErrorKind = "temporary"
	KindIntegrityFailed    ErrorKind = "integrity_failed"
	KindCancelled          ErrorKind = "cancelled"
	KindDeadlineExceeded   ErrorKind = "deadline_exceeded"
	KindOutcomeUnknown     ErrorKind = "outcome_unknown"
	KindInternal           ErrorKind = "internal"
)

type kindError struct {
	kind ErrorKind
}

func (err kindError) Error() string {
	return string(err.kind)
}

// NewError returns a stable provider-neutral error for kind.
func NewError(kind ErrorKind) error {
	if !validErrorKind(kind) {
		kind = KindInternal
	}
	return kindError{kind: kind}
}

// Kind reports a stable provider-neutral error kind.
func Kind(err error) ErrorKind {
	if kindErr, ok := errors.AsType[kindError](err); ok {
		return kindErr.kind
	}
	return KindInternal
}

func validErrorKind(kind ErrorKind) bool {
	switch kind {
	case KindInvalid, KindTooLarge, KindBusy, KindNotFound, KindPreconditionFailed,
		KindDenied, KindTemporary, KindIntegrityFailed, KindCancelled,
		KindDeadlineExceeded, KindOutcomeUnknown, KindInternal:
		return true
	default:
		return false
	}
}
