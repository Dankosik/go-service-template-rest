package httpx

import (
	"net/http"
	"testing"

	"github.com/example/go-service-template-rest/internal/problem"
)

func TestProblemDefinitionForPublishesCatalogEntries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code    problem.Code
		status  int
		title   string
		typeURI string
	}{
		{code: problem.CodeBadRequest, status: http.StatusBadRequest, title: "bad request", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.1"},
		{code: problem.CodeNotFound, status: http.StatusNotFound, title: "not found", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.5"},
		{code: problem.CodeMethodNotAllowed, status: http.StatusMethodNotAllowed, title: "method not allowed", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.6"},
		{code: problem.CodeRequestEntityTooLarge, status: http.StatusRequestEntityTooLarge, title: "request entity too large", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.14"},
		{code: problem.CodeInternalError, status: http.StatusInternalServerError, title: "internal server error", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.6.1"},
	}

	for _, tt := range tests {
		definition := problemDefinitionFor(tt.code)
		if definition.Code != tt.code {
			t.Fatalf("problemDefinitionFor(%q) code = %q", tt.code, definition.Code)
		}
		if definition.Status != tt.status || definition.Title != tt.title || definition.TypeURI != tt.typeURI {
			t.Fatalf("problemDefinitionFor(%q) = %+v, want status=%d title=%q type=%q", tt.code, definition, tt.status, tt.title, tt.typeURI)
		}
	}
}

// TestProblemDefinitionForSubstitutesInternalError pins the one substitution this
// adapter is allowed to make. Its callers pass catalog constants, so an unknown
// code here is a programming error rather than a caller-supplied status — which
// is why problem.For refuses for an uncatalogued status and this does not.
func TestProblemDefinitionForSubstitutesInternalError(t *testing.T) {
	t.Parallel()

	definition := problemDefinitionFor("unknown")
	if definition.Code != problem.CodeInternalError || definition.Status != http.StatusInternalServerError {
		t.Fatalf("problemDefinitionFor(\"unknown\") = %+v, want the internal-error entry", definition)
	}
}
