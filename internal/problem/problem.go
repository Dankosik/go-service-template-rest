// Package problem owns the problem envelope this repository publishes: the
// stable code, title, and type URI for every HTTP status any layer of a service
// answers with.
//
// It is a leaf on purpose, for the same reason internal/reqctx and
// internal/observability/otelconfig are. The transport adapter needs it to write
// the runtime's own fallback responses, and a feature package needs it to fill a
// generated Problem value for a status only the domain can produce — and the
// repository's depguard contract forbids a feature package from importing a
// concrete infra adapter, so neither may own the table both read.
//
// A copy of this table is what the reference example used to carry, and the copy
// drifted: a 409 fell through to the internal-error type, so a slug conflict
// advertised itself as a server fault while carrying status 409. That is exactly
// what a client keying its retry policy off `type` acts on.
//
// Nothing here performs I/O or depends on a wire contract. Each service maps
// these three strings onto its own generated Problem type; the correlation
// identifier for the body comes from internal/reqctx.
package problem

import (
	"net/http"
	"slices"
)

// Code is the stable machine-readable error code a client matches on.
type Code string

const (
	CodeBadRequest            Code = "bad_request"
	CodeUnauthorized          Code = "unauthorized"
	CodeForbidden             Code = "forbidden"
	CodeNotFound              Code = "not_found"
	CodeMethodNotAllowed      Code = "method_not_allowed"
	CodeConflict              Code = "conflict"
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

// Definition is one published problem class.
type Definition struct {
	Code    Code
	Title   string
	TypeURI string
	Status  int
}

// catalog is the single source of the envelope, and both lookups below resolve
// against it.
//
// It is a slice rather than a map so the reverse lookup is deterministic without
// depending on a uniqueness invariant a reader has to trust: two entries sharing
// a status would make a map-based For answer whichever iteration reached first,
// which is a wrong answer that looks right.
//
// 409, 422, and 429 are here because a domain layer produces them and the
// runtime cannot: no fallback path in internal/infra/http emits them, so they
// exist for an operation that declares them. A code with no matching
// `components/responses` entry in a service's contract is unreachable, not wrong.
var catalog = []Definition{
	{
		Code:    CodeBadRequest,
		Status:  http.StatusBadRequest,
		Title:   "bad request",
		TypeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.1",
	},
	{
		Code:    CodeUnauthorized,
		Status:  http.StatusUnauthorized,
		Title:   "unauthorized",
		TypeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.2",
	},
	{
		Code:    CodeForbidden,
		Status:  http.StatusForbidden,
		Title:   "forbidden",
		TypeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.4",
	},
	{
		Code:    CodeNotFound,
		Status:  http.StatusNotFound,
		Title:   "not found",
		TypeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.5",
	},
	{
		Code:    CodeMethodNotAllowed,
		Status:  http.StatusMethodNotAllowed,
		Title:   "method not allowed",
		TypeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.6",
	},
	{
		Code:    CodeConflict,
		Status:  http.StatusConflict,
		Title:   "conflict",
		TypeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.10",
	},
	{
		Code:    CodeRequestEntityTooLarge,
		Status:  http.StatusRequestEntityTooLarge,
		Title:   "request entity too large",
		TypeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.14",
	},
	// profile:authn-oidc-jwt:start
	{
		Code:    CodeRequestHeaderFieldsTooLarge,
		Status:  http.StatusRequestHeaderFieldsTooLarge,
		Title:   "request header fields too large",
		TypeURI: "https://www.rfc-editor.org/rfc/rfc6585#section-5",
	},
	// profile:authn-oidc-jwt:end
	{
		Code:    CodeUnprocessableContent,
		Status:  http.StatusUnprocessableEntity,
		Title:   "unprocessable content",
		TypeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.21",
	},
	{
		Code:   CodeTooManyRequests,
		Status: http.StatusTooManyRequests,
		Title:  "too many requests",
		// RFC 9110 stops at 426; 429 is still defined by RFC 6585.
		TypeURI: "https://www.rfc-editor.org/rfc/rfc6585#section-4",
	},
	{
		Code:    CodeInternalError,
		Status:  http.StatusInternalServerError,
		Title:   "internal server error",
		TypeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.6.1",
	},
	{
		Code:    CodeServiceUnavailable,
		Status:  http.StatusServiceUnavailable,
		Title:   "service unavailable",
		TypeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.6.4",
	},
	{
		Code:    CodeGatewayTimeout,
		Status:  http.StatusGatewayTimeout,
		Title:   "gateway timeout",
		TypeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.6.5",
	},
}

// For returns the definition published for status, and false for a status this
// repository describes no problem class for.
//
// The boolean is the point. A lookup whose failure mode is a plausible wrong
// answer — the internal-error type returned for an uncatalogued status — is
// worse than one that refuses: the caller has a status this repository does not
// describe, and must publish its own type for it.
func For(status int) (Definition, bool) {
	for _, definition := range catalog {
		if definition.Status == status {
			return definition, true
		}
	}
	return Definition{}, false
}

// ForCode returns the definition published for code, and false for a code this
// repository does not publish.
func ForCode(code Code) (Definition, bool) {
	for _, definition := range catalog {
		if definition.Code == code {
			return definition, true
		}
	}
	return Definition{}, false
}

// All returns every published definition, so a caller can render the catalog
// rather than restate it — a contract document, a client's error table, or the
// tests here that assert the catalog is internally consistent.
//
// The copy is not defensive habit. catalog is package state read by every
// problem response the service writes, so handing out the backing array would let
// one caller's sort or append change what every later response reports.
func All() []Definition {
	return slices.Clone(catalog)
}
