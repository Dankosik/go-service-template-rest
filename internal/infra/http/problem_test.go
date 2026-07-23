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
