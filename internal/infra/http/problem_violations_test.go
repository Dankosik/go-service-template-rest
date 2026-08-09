package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
)

// draftSchema mirrors the shape a real contract rejects on: a required member,
// a bounded string, and a pattern.
func draftSchema() *openapi3.Schema {
	return &openapi3.Schema{
		Type:     &openapi3.Types{"object"},
		Required: []string{"slug", "title"},
		Properties: map[string]*openapi3.SchemaRef{
			"slug":  {Value: &openapi3.Schema{Type: &openapi3.Types{"string"}, Pattern: "^[a-z][a-z0-9-]*$"}},
			"title": {Value: openapi3.NewStringSchema().WithMaxLength(8)},
		},
	}
}

// schemaRejection returns the error the library itself produces, rather than a
// hand-built SchemaError. The path this code reads — JSONPointer — is filled in
// by unexported state, so a fabricated value would prove nothing about the real
// rejection.
func schemaRejection(tb testing.TB, value map[string]any, opts ...openapi3.SchemaValidationOption) error {
	tb.Helper()

	err := draftSchema().VisitJSON(value, opts...)
	if err == nil {
		tb.Fatalf("VisitJSON(%v) returned no error, want a rejection to extract", value)
	}
	//nolint:wrapcheck // The library's own error is the subject under test; wrapping it would hide the shape requestViolations has to walk.
	return err
}

func TestRequestViolationsNamesTheMemberAndNotItsValue(t *testing.T) {
	t.Parallel()

	const submitted = "NOT_A_SLUG_secret"

	violations := requestViolations(schemaRejection(t, map[string]any{"slug": submitted, "title": "ok"}))

	if len(violations) != 1 {
		t.Fatalf("violations = %+v, want the one member that failed", violations)
	}
	if violations[0].Field != "/slug" {
		t.Errorf("field = %q, want the RFC 6901 pointer %q", violations[0].Field, "/slug")
	}
	if violations[0].Reason == "" {
		t.Error("reason is empty, want the constraint that failed")
	}
	// The whole safety argument for publishing any of this. SchemaError carries
	// the offending value in Value and the library failure in Origin; reading
	// either is what this assertion exists to catch.
	if strings.Contains(violations[0].Reason, submitted) {
		t.Errorf("reason = %q, want no trace of the submitted value", violations[0].Reason)
	}
}

// TestRequestViolationsReportsEveryFailedMember covers the shape a caller
// actually wants: fix one field, resubmit, get told about the next one is the
// loop this avoids.
func TestRequestViolationsReportsEveryFailedMember(t *testing.T) {
	t.Parallel()

	violations := requestViolations(schemaRejection(
		t,
		map[string]any{"slug": "Bad Slug", "title": "far too long to pass"},
		openapi3.MultiErrors(),
	))

	fields := violationFields(violations)
	if len(fields) != 2 {
		t.Fatalf("fields = %v, want both failed members", fields)
	}
	for _, want := range []string{"/slug", "/title"} {
		if !slices.Contains(fields, want) {
			t.Errorf("fields = %v, want %q among them", fields, want)
		}
	}
}

// TestRequestViolationsBoundsWhatOneRejectionCanPublish keeps a deeply invalid
// body from deciding the size of a response and a log record.
func TestRequestViolationsBoundsWhatOneRejectionCanPublish(t *testing.T) {
	t.Parallel()

	properties := map[string]*openapi3.SchemaRef{}
	value := map[string]any{}
	for _, name := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l"} {
		properties[name] = &openapi3.SchemaRef{Value: openapi3.NewStringSchema()}
		value[name] = 1
	}
	schema := &openapi3.Schema{Type: &openapi3.Types{"object"}, Properties: properties}

	err := schema.VisitJSON(value, openapi3.MultiErrors())
	if err == nil {
		t.Fatal("VisitJSON() returned no error, want a rejection per member")
	}

	if got := len(requestViolations(err)); got != maxViolations {
		t.Fatalf("violations = %d, want them capped at %d", got, maxViolations)
	}
}

// TestRequestViolationsNamesAParameterThatFailedBeforeItsSchema covers the
// rejection with no SchemaError under it: `?limit=abc` against an integer is
// refused by the decoder, and the caller still has to be told which parameter.
func TestRequestViolationsNamesAParameterThatFailedBeforeItsSchema(t *testing.T) {
	t.Parallel()

	const submitted = "abc-secret"
	requestErr := &openapi3filter.RequestError{
		Input:       nil,
		Parameter:   &openapi3.Parameter{Name: "limit", In: "query"},
		RequestBody: nil,
		Reason:      "path is not convertible to primitive",
		// The decoder's own error, which writes the offending value into its
		// text. Reaching for it instead of Reason is what this asserts against.
		Err: errors.New(`strconv.ParseInt: parsing "` + submitted + `": invalid syntax`),
	}

	violations := requestViolations(requestErr)

	if len(violations) != 1 {
		t.Fatalf("violations = %+v, want the one parameter that failed", violations)
	}
	if violations[0].Field != "query.limit" {
		t.Errorf("field = %q, want %q", violations[0].Field, "query.limit")
	}
	if violations[0].Reason != "path is not convertible to primitive" {
		t.Errorf("reason = %q, want the validator's own literal", violations[0].Reason)
	}
	if strings.Contains(violations[0].Reason, submitted) {
		t.Errorf("reason = %q, want no trace of the submitted value", violations[0].Reason)
	}
}

// TestRequestViolationsIgnoresACredentialRejection keeps the extension member
// from turning a 401 into a report about the caller's token.
func TestRequestViolationsIgnoresACredentialRejection(t *testing.T) {
	t.Parallel()

	const leak = "token sk-live-abcdef is not in the allow list"
	securityErr := &openapi3filter.SecurityRequirementsError{
		SecurityRequirements: openapi3.SecurityRequirements{},
		Errors:               []error{errors.New(leak)},
	}

	if got := requestViolations(securityErr); len(got) != 0 {
		t.Fatalf("violations = %+v, want none for a credential rejection", got)
	}
}

// TestRejectedRequestRecordNamesTheFieldsButNotTheReasons splits the audience.
//
// An operator holding a spike of 400s needs the field to know which caller to go
// look at, and the field is contract vocabulary with bounded cardinality. The
// reason can carry a key the caller sent, so it goes to that caller in the
// response and stays out of the log, where it would be unbounded-cardinality
// text nobody queries.
func TestRejectedRequestRecordNamesTheFieldsButNotTheReasons(t *testing.T) {
	t.Parallel()

	var logged bytes.Buffer
	err := schemaRejection(t, map[string]any{"slug": "Bad Slug", "title": "ok"})

	logStrictRequestError(newTestServiceLogger(&logged), httptest.NewRequestWithContext(
		context.Background(), http.MethodPost, "/api/v1/articles", nil,
	), err)

	var record struct {
		Message       string   `json:"msg"`
		ErrorChain    string   `json:"error_chain"`
		InvalidFields []string `json:"invalid_fields"`
	}
	if decodeErr := json.Unmarshal(logged.Bytes(), &record); decodeErr != nil {
		t.Fatalf("decode record %q: %v", logged.String(), decodeErr)
	}
	if record.Message != "http_request_rejected" {
		t.Errorf("msg = %q, want the shared snake_case event name", record.Message)
	}
	if !slices.Equal(record.InvalidFields, []string{"/slug"}) {
		t.Errorf("invalid_fields = %v, want [/slug]", record.InvalidFields)
	}
	if record.ErrorChain == "" {
		t.Error("error_chain is empty, want the rejection's class chain")
	}
	if strings.Contains(logged.String(), "reason") {
		t.Errorf("record carries a violation reason: %s", logged.String())
	}
}

func TestRequestViolationsHandlesNothingToReport(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "nil", err: nil},
		{name: "unrelated error", err: errors.New("reading failed")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := requestViolations(tc.err); len(got) != 0 {
				t.Fatalf("violations = %+v, want none", got)
			}
		})
	}
}
