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
	// KindLifetime is a token this service refuses for how long it was issued
	// for, not for anything wrong with it. It sits beside KindInvalid because a
	// caller cannot tell the two apart and must not: both transports answer it
	// exactly as they answer KindInvalid. It exists for the operator, who
	// otherwise reads a provider configured to mint hour-long access tokens as a
	// signature failure. MaxTokenLifetime owns what the limit buys.
	KindLifetime
	KindUnavailable
	KindUntrustedTransport
)

type kindDetail struct {
	message string
	reason  string
}

var kindDetails = [...]kindDetail{
	KindMissing:            {message: "authentication credential is missing", reason: "missing"},
	KindMalformed:          {message: "authentication credential is malformed", reason: "malformed"},
	KindOversize:           {message: "authentication credential is too large", reason: "oversize"},
	KindInvalid:            {message: "authentication credential is invalid", reason: "invalid"},
	KindLifetime:           {message: "authentication credential was issued for longer than this service accepts", reason: "lifetime"},
	KindUnavailable:        {message: "authentication trust is unavailable", reason: "unavailable"},
	KindUntrustedTransport: {message: "authentication transport is untrusted", reason: "untrusted_transport"},
}

// errUnclassified stands for any error reaching this boundary without a [Kind].
// It is never returned: metrics.go passes it to verificationReason to learn the
// label such an error records under, so that label is prebuilt like the rest.
var errUnclassified = errors.New("unclassified authentication failure")

// Error contains no parser, token, key, endpoint, or dependency text.
type Error struct {
	kind Kind
}

func (e *Error) Error() string {
	detail, ok := detailFor(e.kind)
	if !ok {
		return "authentication failed"
	}
	return detail.message
}

func detailFor(kind Kind) (kindDetail, bool) {
	if kind == 0 || int(kind) >= len(kindDetails) {
		return kindDetail{}, false
	}
	detail := kindDetails[kind]
	return detail, detail.message != "" && detail.reason != ""
}

// callerAborted reports whether err is the caller's own cancellation or deadline
// rather than an outcome this boundary decided.
//
// It is the one failure this package returns without a [Kind], so each transport
// can report the caller's status instead of an authentication result. [KindOf]
// answers false for it and for an unclassified error alike, which is why every
// consumer that has to tell the caller apart from the credential asks this first.
func callerAborted(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// verificationReason names one bounded failure category for the verification
// counter. untrusted_transport and lifetime stay distinct because both can fail
// every request at once and point operators at deployment policy rather than a
// caller's credential.
//
// canceled is the one label with no Kind behind it, because verifyToken passes a
// caller's own cancellation back unchanged. Counting that as invalid would report
// a client that hung up mid-refresh as one that presented a bad credential, in
// the series operators are told to expect noise in. Anything else arriving
// without a Kind is unclassified, and unavailable is the answer that does not
// blame the caller for it.
func verificationReason(err error) string {
	kind, ok := KindOf(err)
	if !ok {
		if callerAborted(err) {
			return "canceled"
		}
		return "unavailable"
	}
	detail, ok := detailFor(kind)
	if !ok {
		return "unavailable"
	}
	return detail.reason
}

func failure(kind Kind) error {
	return &Error{kind: kind}
}

// NewError builds one sanitized authentication category.
//
// It lets sibling transport contract tests reach every status mapping without
// minting real tokens and provider outages. Production failures come from a
// Verifier through failure.
func NewError(kind Kind) error {
	return failure(kind)
}

// KindOf reports the sanitized category carried by err.
func KindOf(err error) (Kind, bool) {
	target, ok := errors.AsType[*Error](err)
	if !ok {
		return 0, false
	}
	return target.kind, true
}
