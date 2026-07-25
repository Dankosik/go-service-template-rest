package httpx

import (
	"net/http"
	"testing"
)

func TestProblemCatalogDefinitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code    problemCode
		status  int
		title   string
		typeURI string
	}{
		{code: problemCodeBadRequest, status: http.StatusBadRequest, title: "bad request", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.1"},
		{code: problemCodeNotFound, status: http.StatusNotFound, title: "not found", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.5"},
		{code: problemCodeMethodNotAllowed, status: http.StatusMethodNotAllowed, title: "method not allowed", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.6"},
		{code: problemCodeRequestEntityTooLarge, status: http.StatusRequestEntityTooLarge, title: "request entity too large", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.14"},
		{code: problemCodeInternalError, status: http.StatusInternalServerError, title: "internal server error", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.6.1"},
	}

	for _, tt := range tests {
		code, definition := problemDefinitionFor(tt.code)
		if code != tt.code {
			t.Fatalf("problemDefinitionFor(%q) code = %q", tt.code, code)
		}
		if definition.status != tt.status || definition.title != tt.title || definition.typeURI != tt.typeURI {
			t.Fatalf("problemDefinitionFor(%q) = %+v, want status=%d title=%q type=%q", tt.code, definition, tt.status, tt.title, tt.typeURI)
		}
	}

	code, definition := problemDefinitionFor("unknown")
	if code != problemCodeInternalError || definition.status != http.StatusInternalServerError {
		t.Fatalf("unknown problem definition = (%q, %+v), want internal error", code, definition)
	}
}

// TestProblemTypeURIResolvesEveryCatalogedStatus is the regression anchor for the
// defect that made this helper worth fixing: a second, hand-maintained list of
// codes behind ProblemTypeURI meant a status present in the catalog resolved to
// the internal-error type. Reading the catalog itself is what keeps the two from
// drifting again.
func TestProblemTypeURIResolvesEveryCatalogedStatus(t *testing.T) {
	t.Parallel()

	_, internal := problemDefinitionFor(problemCodeInternalError)

	for code, definition := range problemCatalog {
		typeURI, ok := ProblemTypeURI(definition.status)
		if !ok {
			t.Fatalf("ProblemTypeURI(%d) reported no type for cataloged code %q", definition.status, code)
		}
		if typeURI != definition.typeURI {
			t.Fatalf("ProblemTypeURI(%d) = %q, want %q", definition.status, typeURI, definition.typeURI)
		}
		if definition.status != http.StatusInternalServerError && typeURI == internal.typeURI {
			t.Fatalf("ProblemTypeURI(%d) returned the internal-error type for code %q", definition.status, code)
		}
	}
}

// TestProblemTypeURIStatusesAreUnique keeps the reverse lookup deterministic. Two
// codes sharing a status would make ProblemTypeURI answer whichever the map
// iteration reached first.
func TestProblemTypeURIStatusesAreUnique(t *testing.T) {
	t.Parallel()

	seen := make(map[int]problemCode, len(problemCatalog))
	for code, definition := range problemCatalog {
		if existing, duplicate := seen[definition.status]; duplicate {
			t.Fatalf("status %d is published by both %q and %q", definition.status, existing, code)
		}
		seen[definition.status] = code
	}
}

// TestProblemTypeURIRefusesUncatalogedStatus pins the behavior that replaced the
// silent wrong answer.
func TestProblemTypeURIRefusesUncatalogedStatus(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusTeapot, http.StatusPaymentRequired, http.StatusGone, 0} {
		typeURI, ok := ProblemTypeURI(status)
		if ok {
			t.Fatalf("ProblemTypeURI(%d) = %q, true; want refusal for an uncataloged status", status, typeURI)
		}
		if typeURI != "" {
			t.Fatalf("ProblemTypeURI(%d) returned %q with ok=false", status, typeURI)
		}
	}
}

// TestProblemCatalogCoversDomainStatuses names the three a domain layer produces
// and the runtime cannot, so removing one is a decision rather than an accident.
func TestProblemCatalogCoversDomainStatuses(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		code   problemCode
		status int
	}{
		{code: problemCodeConflict, status: http.StatusConflict},
		{code: problemCodeUnprocessableContent, status: http.StatusUnprocessableEntity},
		{code: problemCodeTooManyRequests, status: http.StatusTooManyRequests},
	} {
		definition, ok := problemCatalog[tt.code]
		if !ok {
			t.Fatalf("problem catalog is missing %q", tt.code)
		}
		if definition.status != tt.status {
			t.Fatalf("%q status = %d, want %d", tt.code, definition.status, tt.status)
		}
	}
}
