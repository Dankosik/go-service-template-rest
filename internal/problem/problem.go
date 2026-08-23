// Package problem owns the HTTP problem envelope this repository publishes: the
// stable code, title, and type URI for every HTTP status any layer of a service
// answers with.
//
// Feature packages classify their errors with internal/failure; this package
// adds the HTTP-only status, title, and type URI used to render that identity.
// It is the single copy on purpose: a second one drifts, and a status advertising
// the wrong type URI is what a client keys its retry policy off.
//
// Each service maps these three strings onto its own generated Problem type; the
// correlation identifier for the body comes from internal/reqctx.
package problem

import (
	"net/http"
	"slices"

	"github.com/samber/lo"

	"github.com/example/go-service-template-rest/internal/failure"
)

// Code is the stable machine-readable error code a client matches on.
type Code string

const conflictTypeURI = "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.10"

const (
	CodeBadRequest            Code = Code(failure.CodeBadRequest)
	CodeUnauthorized          Code = Code(failure.CodeUnauthorized)
	CodeForbidden             Code = Code(failure.CodeForbidden)
	CodeNotFound              Code = Code(failure.CodeNotFound)
	CodeMethodNotAllowed      Code = Code(failure.CodeMethodNotAllowed)
	CodeConflict              Code = "conflict"
	CodeAlreadyExists         Code = Code(failure.CodeAlreadyExists)
	CodeRequestEntityTooLarge Code = Code(failure.CodeRequestEntityTooLarge)
	// profile:authn-bearer:start
	CodeRequestHeaderFieldsTooLarge Code = Code(failure.CodeRequestHeaderFieldsTooLarge)
	// profile:authn-bearer:end
	CodeUnprocessableContent Code = Code(failure.CodeUnprocessableContent)
	CodeTooManyRequests      Code = Code(failure.CodeTooManyRequests)
	// profile:http-idempotency-postgres:start
	CodeIdempotencyKeyMismatch    Code = Code(failure.CodeIdempotencyKeyMismatch)
	CodeIdempotencyUnavailable    Code = Code(failure.CodeIdempotencyUnavailable)
	CodeIdempotencyOutcomeUnknown Code = Code(failure.CodeIdempotencyOutcomeUnknown)
	// profile:http-idempotency-postgres:end
	CodeInternalError      Code = Code(failure.CodeInternalError)
	CodeServiceUnavailable Code = Code(failure.CodeServiceUnavailable)
	CodeGatewayTimeout     Code = Code(failure.CodeGatewayTimeout)
)

// Definition is one published problem class.
type Definition struct {
	Code    Code
	Title   string
	TypeURI string
	Status  int
}

// catalog is the single source of the envelope.
//
// 409, 422, and 429 are here because a domain layer produces them and no fallback
// path in internal/infra/http does. A code with no matching
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
		Title:   string(CodeConflict),
		TypeURI: conflictTypeURI,
	},
	{
		Code:    CodeAlreadyExists,
		Status:  http.StatusConflict,
		Title:   string(CodeConflict),
		TypeURI: conflictTypeURI,
	},
	{
		Code:    CodeRequestEntityTooLarge,
		Status:  http.StatusRequestEntityTooLarge,
		Title:   "request entity too large",
		TypeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.14",
	},
	// profile:authn-bearer:start
	{
		Code:    CodeRequestHeaderFieldsTooLarge,
		Status:  http.StatusRequestHeaderFieldsTooLarge,
		Title:   "request header fields too large",
		TypeURI: "https://www.rfc-editor.org/rfc/rfc6585#section-5",
	},
	// profile:authn-bearer:end
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
	// profile:http-idempotency-postgres:start
	{
		Code:    CodeIdempotencyKeyMismatch,
		Status:  http.StatusUnprocessableEntity,
		Title:   "unprocessable content",
		TypeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.21",
	},
	{
		Code:    CodeIdempotencyUnavailable,
		Status:  http.StatusServiceUnavailable,
		Title:   "service unavailable",
		TypeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.6.4",
	},
	{
		Code:    CodeIdempotencyOutcomeUnknown,
		Status:  http.StatusServiceUnavailable,
		Title:   "service unavailable",
		TypeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.6.4",
	},
	// profile:http-idempotency-postgres:end
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

// ForCode returns the definition published for code, and false for a code this
// repository does not publish.
func ForCode(code Code) (Definition, bool) {
	return lo.Find(catalog, func(definition Definition) bool { return definition.Code == code })
}

// ForCodeOrInternal returns the definition published for code, substituting the
// internal-error entry for a code this repository does not publish.
//
// This is the lookup for a caller passing a constant from this catalog, where an
// unpublished code is a defect in that caller rather than a runtime condition.
// The internal-error envelope is then the only honest answer left: it must not
// claim a code nothing describes.
func ForCodeOrInternal(code Code) Definition {
	if definition, ok := ForCode(code); ok {
		return definition
	}
	internalError, _ := ForCode(CodeInternalError)
	return internalError
}

// All returns every published definition, so a caller can render the catalog
// rather than restate it.
//
// catalog is package state read by every problem response the service writes, so
// handing out the backing array would let one caller's sort or append change what
// every later response reports.
func All() []Definition {
	return slices.Clone(catalog)
}
