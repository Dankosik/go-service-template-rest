package oauth2clientcredentials

import (
	"context"
	"errors"
)

type authError struct {
	class FailureClass
	cause error
}

func (e *authError) Error() string {
	if message, ok := failureMessages[e.class]; ok {
		return message
	}
	return "outbound authentication failed"
}

func (e *authError) Unwrap() error {
	if e.class == FailureCallerCanceled &&
		(errors.Is(e.cause, context.Canceled) || errors.Is(e.cause, context.DeadlineExceeded)) {
		return e.cause
	}
	return nil
}

func failure(class FailureClass) error {
	return &authError{class: class}
}

func callerFailure(cause error) error {
	switch {
	case errors.Is(cause, context.DeadlineExceeded):
		cause = context.DeadlineExceeded
	case errors.Is(cause, context.Canceled):
		cause = context.Canceled
	default:
		return failure(FailureCallerCanceled)
	}
	return &authError{class: FailureCallerCanceled, cause: cause}
}

// FailureClassOf reports the sanitized category carried by err.
func FailureClassOf(err error) (FailureClass, bool) {
	target, ok := errors.AsType[*authError](err)
	if !ok || !validFailureClass(target.class) {
		return "", false
	}
	return target.class, true
}
