// Package failure defines transport-neutral failure identities.
//
// failure.go holds the client-visible contract: [Code], the sanitized detail
// strings, and the [Classify] mapping from a service error to a
// [Classification]. error_chain.go holds what stays server-side when a
// transport refuses to describe an ordinary failure.
package failure

import (
	"slices"
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
	// profile:authn-bearer:start
	CodeRequestHeaderFieldsTooLarge Code = "request_header_fields_too_large"
	// profile:authn-bearer:end
	CodeUnprocessableContent Code = "unprocessable_content"
	CodeTooManyRequests      Code = "too_many_requests"
	// profile:http-idempotency-postgres:start
	CodeIdempotencyKeyMismatch    Code = "idempotency_key_mismatch"
	CodeIdempotencyUnavailable    Code = "idempotency_unavailable"
	CodeIdempotencyOutcomeUnknown Code = "idempotency_outcome_unknown"
	// profile:http-idempotency-postgres:end
	CodeInternalError      Code = "internal_error"
	CodeServiceUnavailable Code = "service_unavailable"
	CodeGatewayTimeout     Code = "gateway_timeout"
)

// AllCodes returns every published code so a caller can walk the catalog rather
// than restate it. internal/problem's coverage test is the caller that matters:
// a code with no HTTP problem definition is rendered as a 500 by the fallback in
// internal/infra/http, which turns a classified 429 into a server fault.
//
// The list is maintained beside the constants above because Go cannot enumerate
// a named string type. TestAllCodesEnumeratesEveryDeclaredConstant walks the
// constant block and closes the one seam this list cannot close on its own.
func AllCodes() []Code {
	return []Code{
		CodeBadRequest,
		CodeUnauthorized,
		CodeForbidden,
		CodeNotFound,
		CodeMethodNotAllowed,
		CodeAlreadyExists,
		CodeRequestEntityTooLarge,
		// profile:authn-bearer:start
		CodeRequestHeaderFieldsTooLarge,
		// profile:authn-bearer:end
		CodeUnprocessableContent,
		CodeTooManyRequests,
		// profile:http-idempotency-postgres:start
		CodeIdempotencyKeyMismatch,
		CodeIdempotencyUnavailable,
		CodeIdempotencyOutcomeUnknown,
		// profile:http-idempotency-postgres:end
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

// Classification is the safe client-visible answer for one domain error.
type Classification struct {
	Code       Code
	Detail     string
	RetryAfter time.Duration
}

// Mapper classifies one service-owned error for any transport that exposes it.
type Mapper func(error) (Classification, bool)

// Classify runs mappers in order, skips nil entries, and returns the first match
// carrying a published code. An unpublished match fails closed as unclassified.
func Classify(err error, mappers []Mapper) (Classification, bool) {
	for _, classify := range mappers {
		if classify == nil {
			continue
		}
		if classified, ok := classify(err); ok {
			if !slices.Contains(AllCodes(), classified.Code) {
				return Classification{}, false
			}
			return classified, true
		}
	}
	return Classification{}, false
}
