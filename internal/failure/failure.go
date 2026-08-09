// Package failure defines transport-neutral client-visible failure identities.
package failure

import "time"

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
