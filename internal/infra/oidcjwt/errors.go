package oidcjwt

import (
	"context"
	"errors"
)

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

// callerAborted reports whether err is the caller's own cancellation or deadline
// rather than an outcome this boundary decided.
//
// It is the one failure this package returns without a [Kind]: [Verifier.Verify]
// passes it back wrapped and otherwise unchanged, so each transport can report
// the caller's status instead of an authentication result. [KindOf] answers
// false for it, which is the same answer it gives an unclassified error, so
// every consumer that has to tell the caller apart from the credential asks this
// first.
func callerAborted(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// verificationReason names one failure category for the verification counter. It
// is the second table over [Kind], so it lives beside the first rather than
// beside recordVerification, its only caller.
//
// The returned set is closed and one label per Kind, which keeps cardinality
// fixed while leaving each cause separately alertable — untrusted_transport in
// particular, because it means the deployment's trusted_proxy_cidrs no longer
// matches the real peer and every request is failing, not that one client sent
// a bad header.
//
// canceled is the one label with no Kind behind it. [Verifier.Verify] returns
// the caller's own cancellation or deadline unchanged so each transport can
// answer with the caller's status rather than an authentication outcome, and
// that error carries no Kind. Counting it as invalid would report a client that
// hung up mid-refresh as a client that presented a bad credential, in the one
// series operators are told to expect noise in. Anything else arriving without a
// Kind is unclassified, and unavailable is the answer that does not blame the
// caller for it.
func verificationReason(err error) string {
	kind, ok := KindOf(err)
	if !ok {
		if callerAborted(err) {
			return "canceled"
		}
		return "unavailable"
	}
	// In [Kind] declaration order, like the message table above.
	switch kind {
	case KindMissing:
		return "missing"
	case KindMalformed:
		return "malformed"
	case KindOversize:
		return "oversize"
	case KindInvalid:
		return "invalid"
	case KindUnavailable:
		return "unavailable"
	case KindUntrustedTransport:
		return "untrusted_transport"
	default:
		return "unavailable"
	}
}

func failure(kind Kind) error {
	return &Error{kind: kind}
}

// NewError builds one sanitized authentication category.
//
// It is exported for the transport adapters' own tests: proving that the HTTP
// adapter maps every Kind to the right status and challenge needs each Kind
// constructible from outside this package, and the alternative is for those
// tests to mint real tokens and provider outages to reach a category. Production
// code has no reason to call it — a Verifier returns these categories itself,
// through the unexported failure above.
//
// A Kind outside the declared set is not screened here, and screening one would
// break a caller: every consumer already answers its own default arm with a
// fail-closed category, and TestDocumentedMetricReasonsMatchTheGuide depends on
// the unscreened behaviour outright, walking Kind values until [Error.Error]
// reaches its default arm to find the end of the declared run.
func NewError(kind Kind) error {
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
