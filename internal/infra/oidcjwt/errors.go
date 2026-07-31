// Package oidcjwt authenticates strict OIDC JWT access tokens for the service
// transport adapters.
package oidcjwt

import "errors"

// Kind is the finite, sanitized authentication failure taxonomy.
type Kind uint8

const (
	KindMissing Kind = iota + 1
	KindMalformed
	KindOversize
	KindInvalid
	KindUnavailable
	KindUntrustedTransport
)

// Error contains no parser, token, key, endpoint, or dependency text.
type Error struct {
	kind Kind
}

func (e *Error) Error() string {
	switch e.kind {
	case KindMissing:
		return "authentication credential is missing"
	case KindMalformed:
		return "authentication credential is malformed"
	case KindOversize:
		return "authentication credential is too large"
	case KindInvalid:
		return "authentication credential is invalid"
	case KindUnavailable:
		return "authentication trust is unavailable"
	case KindUntrustedTransport:
		return "authentication transport is untrusted"
	default:
		return "authentication failed"
	}
}

func failure(kind Kind) error {
	return &Error{kind: kind}
}

// Failure reports one sanitized authentication category to a transport
// boundary. It exists so the HTTP adapter can prove every finite mapping
// without reproducing parser or provider internals.
func Failure(kind Kind) error {
	switch kind {
	case KindMissing,
		KindMalformed,
		KindOversize,
		KindInvalid,
		KindUnavailable,
		KindUntrustedTransport:
		return failure(kind)
	default:
		return failure(KindInvalid)
	}
}

// KindOf reports the sanitized category carried by err.
func KindOf(err error) (Kind, bool) {
	var target *Error
	if !errors.As(err, &target) {
		return 0, false
	}
	return target.kind, true
}
