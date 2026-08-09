// Package failure defines transport-neutral failure identities: the code a
// client matches on, and the class chain an operator diagnoses with.
package failure

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"
)

// Code is the stable machine-readable failure code a client matches on.
type Code string

const (
	CodeBadRequest            Code = "bad_request"
	CodeUnauthorized          Code = "unauthorized"
	CodeForbidden             Code = "forbidden"
	CodeNotFound              Code = "not_found"
	CodeMethodNotAllowed      Code = "method_not_allowed"
	CodeAlreadyExists         Code = "already_exists"
	CodeRequestEntityTooLarge Code = "request_entity_too_large"
	// profile:authn-oidc-jwt:start
	CodeRequestHeaderFieldsTooLarge Code = "request_header_fields_too_large"
	// profile:authn-oidc-jwt:end
	CodeUnprocessableContent Code = "unprocessable_content"
	CodeTooManyRequests      Code = "too_many_requests"
	CodeInternalError        Code = "internal_error"
	CodeServiceUnavailable   Code = "service_unavailable"
	CodeGatewayTimeout       Code = "gateway_timeout"
)

// AllCodes returns every published code so a caller can walk the catalog rather
// than restate it. internal/problem's coverage test is the caller that matters:
// a code with no HTTP problem definition is rendered as a 500 by the fallback in
// internal/infra/http, which turns a classified 429 into a server fault.
//
// The list is maintained beside the constants above because Go cannot enumerate
// a named string type. That leaves one seam this list cannot close on its own — a
// constant added above and not added here — and internal/infra/grpc's
// TestEveryFailureCodeRendersAConformingReason is what covers it, by walking the
// constant block itself rather than any list.
func AllCodes() []Code {
	return []Code{
		CodeBadRequest,
		CodeUnauthorized,
		CodeForbidden,
		CodeNotFound,
		CodeMethodNotAllowed,
		CodeAlreadyExists,
		CodeRequestEntityTooLarge,
		// profile:authn-oidc-jwt:start
		CodeRequestHeaderFieldsTooLarge,
		// profile:authn-oidc-jwt:end
		CodeUnprocessableContent,
		CodeTooManyRequests,
		CodeInternalError,
		CodeServiceUnavailable,
		CodeGatewayTimeout,
	}
}

// SanitizedDetail is the caller-visible text for every failure a transport
// refused to describe: an unclassified handler error, a policy error, a
// recovered panic, and a classified answer whose mapper supplied no detail.
//
// One constant because those must stay indistinguishable to a caller. Text that
// separated them would report which internal path broke, which is exactly what
// the sanitization withholds. It is owned here rather than per transport for the
// same reason one transport owns one string: a caller that can reach both HTTP
// and gRPC would learn the distinction from the difference between them.
const SanitizedDetail = "request failed"

// AtCapacityDetail is the caller-visible text a transport's admission control
// answers with when a concurrency budget is exhausted.
//
// Only the explanation is shared. Each transport still chooses its own status —
// HTTP answers 503, gRPC answers RESOURCE_EXHAUSTED rather than UNAVAILABLE,
// which it reserves for a health-derived down state — because those vocabularies
// are not interchangeable and the shed reason is.
const AtCapacityDetail = "server is at capacity"

// PanicAttrs returns the log attributes describing a recovered panic: its class,
// its concrete type, and the stack it came from.
//
// Four packages recover panics — a background task, both transports, and the
// authn converters — and each answered its caller differently but described the
// panic identically, except that one of them silently omitted the class. One
// constructor is what makes that impossible: a site cannot forget an attribute
// it does not spell.
//
// stack is a parameter rather than a [runtime/debug.Stack] call made here
// because where it is taken is what makes it worth anything. Only a deferred
// function still holds the panicking goroutine's frames, and a helper that
// captured the stack on its caller's behalf would let a site that recovered
// earlier log an unrelated stack that still looks correct.
//
// Only the panic's type is published, never its value. A panic raised while
// parsing a token, a provider document, or a request body carries that input in
// its message, and a log is not a safer place for it than a response is.
func PanicAttrs(recovered any, stack []byte) []any {
	return []any{
		"panic.class", PanicClass(recovered),
		"panic.type", fmt.Sprintf("%T", recovered),
		"stack", string(stack),
	}
}

// classChainDepth bounds how far ClassChain walks. A chain this long is already
// past the point where another name helps, and the bound is what keeps a cyclic
// or pathological Unwrap from deciding the size of a log record.
const classChainDepth = 8

// opNameLimit bounds one [Op] name in a rendered chain. classChainDepth alone
// does not bound the record: eight links of unbounded text still is, and the
// two together are what keep a chain a fixed cost.
const opNameLimit = 64

// Op names the step err came from, so a record written at a sanitized boundary
// can say which step failed.
//
// It exists because [ClassChain] prints no message text, which leaves every
// fmt.Errorf layer rendering as "*fmt.wrapError": a handler that loads an author,
// renders a body, and stores a row produces the same record whichever of the
// three broke. Deriving the step from the text instead is not available — a
// wrapper's prefix cannot be told apart from a dependency's own, and pgx writes
// the DSN into one.
//
// The name must be a literal written in this repository. That, and not the type,
// is what makes it printable when err.Error() is not; interpolating a slug, an
// identifier, or anything else the request carried puts it in every log this
// boundary refused to leak into.
//
// fmt.Errorf stays correct everywhere else and needs no migration. It renders as
// its type, which is what every layer did before this existed.
func Op(name string, err error) error {
	if err == nil {
		return nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return err
	}
	return &opError{op: name, err: err}
}

type opError struct {
	op  string
	err error
}

func (e *opError) Error() string { return e.op + ": " + e.err.Error() }
func (e *opError) Unwrap() error { return e.err }

// ClassChain renders err's unwrap chain for the records a transport writes when
// it answers with a sanitized failure: the Go type of every layer, and the name
// of every layer built by [Op].
//
// The outermost type alone is not enough to act on: every wrapped failure in
// this repository goes through fmt.Errorf, so a record carrying only %T says
// "*fmt.wrapError" for a dead connection pool and for a nil map alike. The chain
// names the dependency underneath — "*fmt.wrapError -> *pgconn.PgError" is a
// database fault — and an operator can route on that.
//
// No message text is included, and that is the whole reason this exists rather
// than err.Error(). A handler's error can carry a credential, a DSN, or a token,
// and the canary tests in internal/infra/grpc assert that none of it reaches a
// log, span, or metric. The cost is real and worth stating: a chain built from
// errors.New renders as "*errors.errorString" and identifies nothing. A package
// whose failures must stay diagnosable publishes typed errors or sentinels,
// and why their faults survive this rendering. [Op] is what covers the rest.
//
// Joined errors are not expanded. errors.Join reports one type for the group,
// and walking every branch would turn a fan-out of cleanup failures into an
// unbounded record; the join's own type still says that is what happened.
func ClassChain(err error) string {
	if err == nil {
		return "nil"
	}

	classes := make([]string, 0, classChainDepth)
	for depth := 0; err != nil && depth < classChainDepth; depth++ {
		classes = append(classes, chainLink(err))
		err = errors.Unwrap(err)
	}
	if err != nil {
		classes = append(classes, "...")
	}
	return strings.Join(classes, " -> ")
}

// chainLink renders the layer err is, and never what it wraps.
//
// The type switch is deliberate where errors.As would be wrong: As searches the
// whole remaining chain, so an outer fmt.Errorf layer would render as the name of
// an [Op] several links below it and the chain would report that step twice.
func chainLink(err error) string {
	switch link := err.(type) { //nolint:errorlint // Rendering one layer is the whole contract; errors.As would search past it and repeat an inner Op's name on every layer above it.
	case *opError:
		// Byte length gates a rune slice: under the limit in bytes is under it in
		// runes too, so the common name costs no allocation, and the one that has
		// to be cut is not cut through a rune.
		if len(link.op) <= opNameLimit {
			return link.op
		}
		return string([]rune(link.op)[:opNameLimit]) + "…"
	default:
		return fmt.Sprintf("%T", err)
	}
}

// PanicClass names what a recovered value is, for the records the transports and
// the background supervisor write when they convert a panic into a failure.
//
// It is one function rather than one per recovery site because the label is what
// an operator alerts on: a runtime_error is a service defect every time, while a
// deliberate panic(error) from a library is frequently a control-flow choice, and
// an alert cannot separate them if each site invents its own vocabulary.
//
// The classes are closed on purpose. %T of the recovered value is the companion
// attribute and is unbounded — a panic can carry any type — so it names the cause
// while this bounds the series a dashboard groups by.
func PanicClass(recovered any) string {
	switch recovered.(type) {
	case nil:
		return "none"
	case runtime.Error:
		return "runtime_error"
	case error:
		return "error"
	case string:
		return "string"
	default:
		return "value"
	}
}

// Classification is the safe client-visible answer for one domain error.
type Classification struct {
	Code       Code
	Detail     string
	RetryAfter time.Duration
}

// Mapper classifies one service-owned error for any transport that exposes it.
type Mapper func(error) (Classification, bool)

// Classify runs mappers in order, skips nil entries, and returns the first match.
func Classify(err error, mappers []Mapper) (Classification, bool) {
	for _, classify := range mappers {
		if classify == nil {
			continue
		}
		if classified, ok := classify(err); ok {
			return classified, true
		}
	}
	return Classification{}, false
}
