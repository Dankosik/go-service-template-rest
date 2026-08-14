package httpx

import (
	"errors"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/samber/lo"
)

// A rejected request publishes at most this many violations, each with a reason
// at most this long.
//
// Both bounds exist because this is the one path where text derived from a
// request reaches a response and a log. A body can nest deep enough to produce a
// violation per leaf, and the caller only needs enough to find the mistake.
const (
	maxViolations      = 10
	maxViolationReason = 200
)

// fieldViolation is one contract-authored reason a request failed validation.
//
// Field is an RFC 6901 pointer for a body member and a `location.name` pair for
// a parameter, so a caller can tell `/slug` in the body from `query.slug`.
type fieldViolation struct {
	Field  string
	Reason string
}

// requestViolations renders why the OpenAPI validator rejected a request, using
// only what the contract and the validator wrote.
//
// The rule that governs every branch below: a value the caller sent never
// appears. openapi3.SchemaError carries the offending value in Value and the
// underlying library failure in Origin, and neither is read here — Reason is,
// because kin-openapi documents it as the field that deliberately excludes the
// value. Everything else comes from the contract: a property name, a parameter
// name, a schema keyword.
//
// That leaves one class of caller-derived text: a reason naming an unexpected
// property echoes back a key the caller itself sent. It goes to that same caller
// and is bounded by maxViolationReason, which is why it is acceptable here and
// why the bound is not optional.
//
// An empty result means the rejection had nothing field-shaped behind it — a
// failed security requirement, an unreadable body — and the caller gets the
// unchanged detail-free problem.
func requestViolations(err error) []fieldViolation {
	violations := make([]fieldViolation, 0, maxViolations)
	return appendViolations(violations, err, "")
}

func appendViolations(violations []fieldViolation, err error, field string) []fieldViolation {
	if err == nil || len(violations) >= maxViolations {
		return violations
	}

	// A failed security requirement is about the credential, not a field, and
	// its wrapped resolver errors are dependency text this boundary refuses to
	// publish. handleGeneratedRequestError already answers it with a challenge.
	if _, ok := errors.AsType[*openapi3filter.SecurityRequirementsError](err); ok {
		return violations
	}

	if requestErr, ok := errors.AsType[*openapi3filter.RequestError](err); ok {
		field := parameterField(requestErr)
		nested := appendViolations(violations, requestErr.Err, field)
		if len(nested) > len(violations) || field == "" {
			return nested
		}
		// A parameter that failed before schema validation — `?limit=abc` against
		// an integer — carries no SchemaError, so the walk above finds nothing and
		// the caller would be told only that something was invalid. RequestError's
		// own Reason is a literal from the validator, which makes it the one
		// remaining answer that is safe to publish.
		return append(nested, fieldViolation{
			Field:  field,
			Reason: boundedReason(requestErrorReason(requestErr)),
		})
	}

	// MultiError is how the validator reports every failed member of one body at
	// once, and is why maxViolations exists.
	//
	// It has to be tested before SchemaError and cannot be reordered. MultiError
	// implements As by scanning its own members and matching the first one, so
	// asking "is this a SchemaError" of a group answers yes and hands back member
	// zero — which silently reports one violation for a body that failed five.
	if multi, ok := errors.AsType[openapi3.MultiError](err); ok {
		for _, branch := range multi {
			violations = appendViolations(violations, branch, field)
			if len(violations) >= maxViolations {
				break
			}
		}
		return violations
	}

	if schemaErr, ok := errors.AsType[*openapi3.SchemaError](err); ok {
		return append(violations, schemaViolation(schemaErr, field))
	}

	return violations
}

func schemaViolation(schemaErr *openapi3.SchemaError, fallbackField string) fieldViolation {
	field := fallbackField
	if pointer := schemaErr.JSONPointer(); len(pointer) > 0 {
		field = "/" + strings.Join(pointer, "/")
	}

	// Reason first: it is the only member kin-openapi documents as value-free.
	// SchemaField names the keyword that failed and is the contract's own
	// vocabulary, which is a usable answer when Reason is empty.
	reason := strings.TrimSpace(schemaErr.Reason)
	if reason == "" {
		reason = strings.TrimSpace(schemaErr.SchemaField)
	}
	if reason == "" {
		reason = "does not match the contract"
	}

	return fieldViolation{Field: field, Reason: boundedReason(reason)}
}

// parameterField names the parameter a RequestError is about, and the empty
// string for a request body — whose members name themselves through their
// JSON pointer.
func parameterField(requestErr *openapi3filter.RequestError) string {
	if requestErr.Parameter == nil {
		return ""
	}
	if requestErr.Parameter.In == "" {
		return requestErr.Parameter.Name
	}
	return requestErr.Parameter.In + "." + requestErr.Parameter.Name
}

// requestErrorReason reports why the validator rejected a parameter, using only
// its own Reason. requestErr.Err is deliberately not read: it is the decoder's
// error, and strconv writes the offending value into its text.
func requestErrorReason(requestErr *openapi3filter.RequestError) string {
	if reason := strings.TrimSpace(requestErr.Reason); reason != "" {
		return reason
	}
	return "does not match the contract"
}

func boundedReason(reason string) string {
	// Byte length gates a rune slice: under the limit in bytes is under it in
	// runes too, so the ordinary reason costs no allocation and the one that has
	// to be cut is not cut through a rune.
	if len(reason) <= maxViolationReason {
		return reason
	}
	return string([]rune(reason)[:maxViolationReason]) + "…"
}

// violationFields renders violations for a log record.
func violationFields(violations []fieldViolation) []string {
	if len(violations) == 0 {
		return nil
	}
	return lo.FilterMap(violations, func(violation fieldViolation, _ int) (string, bool) {
		return violation.Field, violation.Field != ""
	})
}
