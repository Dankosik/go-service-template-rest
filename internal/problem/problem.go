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

	"github.com/example/go-service-template-rest/internal/failure"
)

// Code is the stable machine-readable error code a client matches on.
type Code string

const (
	CodeBadRequest            Code = Code(failure.CodeBadRequest)
	CodeUnauthorized          Code = Code(failure.CodeUnauthorized)
	CodeForbidden             Code = Code(failure.CodeForbidden)
	CodeNotFound              Code = Code(failure.CodeNotFound)
	CodeMethodNotAllowed      Code = Code(failure.CodeMethodNotAllowed)
	CodeConflict              Code = "conflict"
	CodeAlreadyExists         Code = Code(failure.CodeAlreadyExists)
	CodeRequestEntityTooLarge Code = Code(failure.CodeRequestEntityTooLarge)
	// profile:authn-oidc-jwt:start
	CodeRequestHeaderFieldsTooLarge Code = Code(failure.CodeRequestHeaderFieldsTooLarge)
	// profile:authn-oidc-jwt:end
	CodeUnprocessableContent Code = Code(failure.CodeUnprocessableContent)
	CodeTooManyRequests      Code = Code(failure.CodeTooManyRequests)
	CodeInternalError        Code = Code(failure.CodeInternalError)
	CodeServiceUnavailable   Code = Code(failure.CodeServiceUnavailable)
	CodeGatewayTimeout       Code = Code(failure.CodeGatewayTimeout)
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
// It is a slice rather than a map so the reverse lookup stays deterministic: two
// entries sharing a status would make a map-based For answer whichever iteration
// reached first, which is a wrong answer that looks right.
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
		Title:   "conflict",
		TypeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.10",
	},
	{
		Code:    CodeAlreadyExists,
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
// The boolean is the point: returning the internal-error type for an
// uncatalogued status is a plausible wrong answer. A caller holding a status this
// repository does not describe must publish its own type for it.
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

// ForCodeOrInternal returns the definition published for code, substituting the
// internal-error entry for a code this repository does not publish.
//
// This is the lookup for a caller passing a constant from this catalog, where an
// unpublished code is a defect in that caller rather than a runtime condition.
// The internal-error envelope is then the only honest answer left: it must not
// claim a code nothing describes. A caller resolving a status it was handed uses
// For, which refuses instead — see its documentation for why a plausible wrong
// answer is the worse failure.
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
