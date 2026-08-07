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
		// Also the sentinel TestDocumentedMetricReasonsMatchTheGuide walks to. It
		// enumerates Kind values until one answers here, and that is how the
		// operator guide is held to the full set without a second list of them.
		return "authentication failed"
	}
}

func failure(kind Kind) error {
	return &Error{kind: kind}
}

// Failure builds one sanitized authentication category.
//
// It is exported for the transport adapters' own tests: proving that the HTTP
// adapter maps every Kind to the right status and challenge needs each Kind
// constructible from outside this package, and the alternative is for those
// tests to mint real tokens and provider outages to reach a category. Production
// code has no reason to call it — a Verifier returns these categories itself.
//
// A Kind outside the declared set is not screened here, and screening one would
// break a caller: every consumer already answers its own default arm with a
// fail-closed category, and TestDocumentedMetricReasonsMatchTheGuide depends on
// the unscreened behaviour outright, walking Kind values until [Error.Error]
// reaches its default arm to find the end of the declared run.
func Failure(kind Kind) error {
	return failure(kind)
}

// KindOf reports the sanitized category carried by err.
func KindOf(err error) (Kind, bool) {
	var target *Error
	if !errors.As(err, &target) {
		return 0, false
	}
	return target.kind, true
}
