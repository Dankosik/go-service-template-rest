package problem_test

import (
	"net/http"
	"testing"

	"github.com/example/go-service-template-rest/internal/failure"
	"github.com/example/go-service-template-rest/internal/problem"
)

func TestEveryPublishedCodeResolves(t *testing.T) {
	t.Parallel()

	for _, definition := range problem.All() {
		resolved, ok := problem.ForCode(definition.Code)
		if !ok {
			t.Fatalf("ForCode(%q) reported no definition", definition.Code)
		}
		if resolved != definition {
			t.Fatalf("ForCode(%q) = %+v, want %+v", definition.Code, resolved, definition)
		}
	}
}

func TestConflictAndAlreadyExistsShareTheHTTP409Envelope(t *testing.T) {
	t.Parallel()

	conflict, ok := problem.ForCode(problem.CodeConflict)
	if !ok {
		t.Fatal("conflict is missing")
	}
	alreadyExists, ok := problem.ForCode(problem.CodeAlreadyExists)
	if !ok {
		t.Fatal("already_exists is missing")
	}
	if conflict.Status != http.StatusConflict || alreadyExists.Status != http.StatusConflict ||
		conflict.Title != "conflict" || alreadyExists.Title != conflict.Title ||
		alreadyExists.TypeURI != conflict.TypeURI {
		t.Fatalf("409 definitions differ: conflict=%+v already_exists=%+v", conflict, alreadyExists)
	}
}

// TestCodesAreUnique is the same invariant from the other direction: a duplicated
// code would let two statuses answer with one client-visible identity.
func TestCodesAreUnique(t *testing.T) {
	t.Parallel()

	seen := make(map[problem.Code]int)
	for _, definition := range problem.All() {
		if existing, duplicate := seen[definition.Code]; duplicate {
			t.Fatalf("code %q is published for both status %d and %d", definition.Code, existing, definition.Status)
		}
		seen[definition.Code] = definition.Status
	}
}

func TestForCodeRefusesUnpublishedCode(t *testing.T) {
	t.Parallel()

	definition, ok := problem.ForCode("not_a_published_code")
	if ok || definition != (problem.Definition{}) {
		t.Fatalf("ForCode(unpublished) = (%+v, %t), want (zero, false)", definition, ok)
	}
}

// TestCatalogPublishesStableEnvelopes pins the remaining strings a client keys
// off. Conflict equivalence and the idempotency profile have focused tests.
func TestCatalogPublishesStableEnvelopes(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		code    problem.Code
		status  int
		title   string
		typeURI string
	}{
		{code: problem.CodeBadRequest, status: http.StatusBadRequest, title: "bad request", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.1"},
		{code: problem.CodeUnauthorized, status: http.StatusUnauthorized, title: "unauthorized", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.2"},
		{code: problem.CodeForbidden, status: http.StatusForbidden, title: "forbidden", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.4"},
		{code: problem.CodeNotFound, status: http.StatusNotFound, title: "not found", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.5"},
		{code: problem.CodeMethodNotAllowed, status: http.StatusMethodNotAllowed, title: "method not allowed", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.6"},
		{code: problem.CodeRequestEntityTooLarge, status: http.StatusRequestEntityTooLarge, title: "request entity too large", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.14"},
		// profile:authn-bearer:start
		{code: problem.CodeRequestHeaderFieldsTooLarge, status: http.StatusRequestHeaderFieldsTooLarge, title: "request header fields too large", typeURI: "https://www.rfc-editor.org/rfc/rfc6585#section-5"},
		// profile:authn-bearer:end
		{code: problem.CodeUnprocessableContent, status: http.StatusUnprocessableEntity, title: "unprocessable content", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.21"},
		{code: problem.CodeTooManyRequests, status: http.StatusTooManyRequests, title: "too many requests", typeURI: "https://www.rfc-editor.org/rfc/rfc6585#section-4"},
		{code: problem.CodeInternalError, status: http.StatusInternalServerError, title: "internal server error", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.6.1"},
		{code: problem.CodeServiceUnavailable, status: http.StatusServiceUnavailable, title: "service unavailable", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.6.4"},
		{code: problem.CodeGatewayTimeout, status: http.StatusGatewayTimeout, title: "gateway timeout", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.6.5"},
	} {
		definition := problem.ForCodeOrInternal(tt.code)
		if definition.Code != tt.code {
			t.Fatalf("ForCodeOrInternal(%q) code = %q", tt.code, definition.Code)
		}
		if definition.Status != tt.status || definition.Title != tt.title || definition.TypeURI != tt.typeURI {
			t.Fatalf("ForCodeOrInternal(%q) = %+v, want status=%d title=%q type=%q", tt.code, definition, tt.status, tt.title, tt.typeURI)
		}
	}
}

// TestForCodeOrInternalSubstitutesInternalError pins the one substitution this
// catalog is allowed to make. Its callers pass catalog constants, so an unknown
// code there is a programming error rather than a caller-supplied status — which
// is why For refuses for an uncatalogued status and this does not.
func TestForCodeOrInternalSubstitutesInternalError(t *testing.T) {
	t.Parallel()

	definition := problem.ForCodeOrInternal("not_a_published_code")
	if definition.Code != problem.CodeInternalError || definition.Status != http.StatusInternalServerError {
		t.Fatalf("ForCodeOrInternal(unpublished) = %+v, want the internal-error entry", definition)
	}
}

// TestCatalogCoversDomainStatuses names the three a domain layer produces and the
// runtime cannot, so removing one is a decision rather than an accident.
func TestCatalogCoversDomainStatuses(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		code   problem.Code
		status int
	}{
		{code: problem.CodeConflict, status: http.StatusConflict},
		{code: problem.CodeAlreadyExists, status: http.StatusConflict},
		{code: problem.CodeUnprocessableContent, status: http.StatusUnprocessableEntity},
		{code: problem.CodeTooManyRequests, status: http.StatusTooManyRequests},
	} {
		definition, ok := problem.ForCode(tt.code)
		if !ok {
			t.Fatalf("catalog is missing %q", tt.code)
		}
		if definition.Status != tt.status {
			t.Fatalf("%q status = %d, want %d", tt.code, definition.Status, tt.status)
		}
	}
}

// TestEveryFailureCodeHasAnHTTPEnvelope closes the one direction the catalog
// could not refuse loudly.
//
// internal/infra/http converts a classified failure.Code to problem.Code with a
// string conversion, which always succeeds. A code with no catalog entry
// therefore reaches ForCodeOrInternal, which substitutes the internal-error
// definition — so a domain that classified its error as 429 answers 503-shaped
// traffic with a 500, and nothing reports it. gRPC has no matching gap: the
// exhaustive linter holds mappedStatus to the whole set. This test is the HTTP
// side of that guarantee.
func TestEveryFailureCodeHasAnHTTPEnvelope(t *testing.T) {
	t.Parallel()

	for _, code := range failure.AllCodes() {
		definition, ok := problem.ForCode(problem.Code(code))
		if !ok {
			t.Errorf("failure code %q has no HTTP problem definition", code)
			continue
		}
		if definition.Status < http.StatusBadRequest {
			t.Errorf("failure code %q resolves to status %d, which is not a failure", code, definition.Status)
		}
	}
}

// TestAllReturnsACopy keeps a caller from editing the catalog every other reader
// resolves against.
func TestAllReturnsACopy(t *testing.T) {
	t.Parallel()

	published := problem.All()
	if len(published) == 0 {
		t.Fatal("All() returned no definitions")
	}
	published[0] = problem.Definition{Code: "mutated"}

	if problem.All()[0].Code == "mutated" {
		t.Fatal("All() exposed the package catalog to mutation")
	}
}
