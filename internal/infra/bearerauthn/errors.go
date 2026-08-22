package bearerauthn

import "errors"

// Kind is the finite, sanitized authentication failure taxonomy shared by the
// HTTP and gRPC adapters.
type Kind uint8

const (
	KindMissing Kind = iota + 1
	KindMalformed
	KindOversize
	KindInvalid
	KindUnavailable
)

type kindDetail struct {
	message string
	reason  string
}

var kindDetails = [...]kindDetail{
	KindMissing:     {message: "authentication credential is missing", reason: "missing"},
	KindMalformed:   {message: "authentication credential is malformed", reason: "malformed"},
	KindOversize:    {message: "authentication credential is too large", reason: "oversize"},
	KindInvalid:     {message: "authentication credential is invalid", reason: "invalid"},
	KindUnavailable: {message: "authentication trust is unavailable", reason: "unavailable"},
}

// Error contains no parser, token, key, endpoint, or dependency text.
type Error struct {
	kind Kind
}

func (e *Error) Error() string {
	if detail, ok := detailFor(e.kind); ok {
		return detail.message
	}
	return "authentication failed"
}

func detailFor(kind Kind) (kindDetail, bool) {
	if kind == 0 || int(kind) >= len(kindDetails) {
		return kindDetail{}, false
	}
	detail := kindDetails[kind]
	return detail, detail.message != "" && detail.reason != ""
}

func failure(kind Kind) error {
	return &Error{kind: kind}
}

// NewError builds one sanitized authentication category for transport contract
// tests and concrete trust engines. Production transport failures come from a
// [Runtime].
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
