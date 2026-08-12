package problem_test

import (
	"net/http"
	"testing"

	"github.com/example/go-service-template-rest/internal/problem"
)

func TestHTTPIdempotencyProblemCatalog(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		code    problem.Code
		status  int
		title   string
		typeURI string
	}{
		{problem.CodeIdempotencyKeyMismatch, http.StatusUnprocessableEntity, "unprocessable content", "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.21"},
		{problem.CodeIdempotencyInProgress, http.StatusConflict, "conflict", "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.10"},
		{problem.CodeIdempotencyKeyExpired, http.StatusConflict, "conflict", "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.10"},
		{problem.CodeIdempotencyUnavailable, http.StatusServiceUnavailable, "service unavailable", "https://www.rfc-editor.org/rfc/rfc9110#section-15.6.4"},
		{problem.CodeIdempotencyOutcomeUnknown, http.StatusServiceUnavailable, "service unavailable", "https://www.rfc-editor.org/rfc/rfc9110#section-15.6.4"},
		{problem.CodeIdempotencyResultTooLarge, http.StatusInternalServerError, "internal server error", "https://www.rfc-editor.org/rfc/rfc9110#section-15.6.1"},
	} {
		definition, ok := problem.ForCode(testCase.code)
		if !ok {
			t.Fatalf("ForCode(%q) reported no definition", testCase.code)
		}
		if definition.Status != testCase.status || definition.Title != testCase.title || definition.TypeURI != testCase.typeURI {
			t.Fatalf("ForCode(%q) = %+v", testCase.code, definition)
		}
	}
}
