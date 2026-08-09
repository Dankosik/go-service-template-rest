package oidcjwt

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
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

// lastKind ends the declared run. metrics.go walks to it to build one attribute
// set per category from verificationReason, which is what keeps that function
// the only table over [Kind] rather than one of several.
//
// It names the last category rather than riding the iota above, because the
// iotamixing linter does not allow a constant with a right-hand side to share
// that block. So this is the one thing here that a new Kind does not update on
// its own, and TestVerificationSetsCoverEveryReason is what fails when it is
// left behind: that test finds the end of the run independently, through
// [Error.Error]'s default arm.
const lastKind = KindUntrustedTransport

// errUnclassified stands for any error reaching this boundary without a [Kind].
// It is never returned: metrics.go passes it to verificationReason to learn the
// label such an error records under, so that label is prebuilt like the rest.
var errUnclassified = errors.New("unclassified authentication failure")

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
	case KindLifetime:
		return "authentication credential was issued for longer than this service accepts"
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
// It is the one failure this package returns without a [Kind], so each transport
// can report the caller's status instead of an authentication result. [KindOf]
// answers false for it and for an unclassified error alike, which is why every
// consumer that has to tell the caller apart from the credential asks this first.
func callerAborted(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// verificationReason names one failure category for the verification counter. It
// is the second table over [Kind], so it lives beside the first rather than
// beside recordVerification, its only caller.
//
// One label per Kind keeps cardinality fixed while leaving each cause separately
// alertable, and two of them earn their separation by naming a deployment rather
// than a caller: untrusted_transport means trusted_proxy_cidrs no longer matches
// the real peer, and lifetime means the issuer mints access tokens longer-lived
// than MaxTokenLifetime. Both fail every request at once, so folding either into
// invalid would send an operator looking at signatures and keys.
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
	case KindLifetime:
		return "lifetime"
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

// logRecoveredPanic reports a panic this package converted into a sanitized
// failure, which the metric alone cannot locate: it says a panic occurred, and
// this says where the service defect is.
//
// Only the panic's type is published, never its value: a panic raised while
// parsing a token or a provider document can carry either in its message, and
// providerError's redaction rule covers logs as much as errors. The stack is
// safe by the same reading — it names functions and files, not values — and it
// is the only record of where the panic came from, because both converters
// answer before any transport recovery could see it.
func logRecoveredPanic(ctx context.Context, log *slog.Logger, operation string, recovered any) {
	if log == nil {
		log = slog.Default()
	}
	log.ErrorContext(
		ctx,
		"authn_panic_recovered",
		"component", "authn",
		"authn.operation", operation,
		"panic.type", fmt.Sprintf("%T", recovered),
		"stack", string(debug.Stack()),
	)
}

// NewError builds one sanitized authentication category.
//
// It is exported for the transport adapters' own tests, which otherwise have to
// mint real tokens and provider outages to reach every Kind their status mapping
// must answer. Production code has no reason to call it: a Verifier returns
// these categories itself, through the unexported failure above.
//
// A Kind outside the declared set is deliberately not screened.
// TestDocumentedMetricReasonsMatchTheGuide depends on that, walking Kind values
// until [Error.Error] reaches its default arm to find the end of the declared
// run, and every consumer already answers its own default arm fail-closed.
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
