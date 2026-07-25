package problem_test

import (
	"net/http"
	"testing"

	"github.com/example/go-service-template-rest/internal/problem"
)

// TestForResolvesEveryPublishedStatus is the regression anchor for the defect
// that made this package worth extracting: a second, hand-maintained table meant
// a status the catalog published resolved to the internal-error type, so a 409
// advertised itself as a server fault while carrying status 409.
func TestForResolvesEveryPublishedStatus(t *testing.T) {
	t.Parallel()

	internalError, ok := problem.ForCode(problem.CodeInternalError)
	if !ok {
		t.Fatal("catalog is missing the internal-error entry every fallback path depends on")
	}

	for _, definition := range problem.All() {
		resolved, ok := problem.For(definition.Status)
		if !ok {
			t.Fatalf("For(%d) reported no definition for published code %q", definition.Status, definition.Code)
		}
		if resolved != definition {
			t.Fatalf("For(%d) = %+v, want %+v", definition.Status, resolved, definition)
		}
		if definition.Status != http.StatusInternalServerError && resolved.TypeURI == internalError.TypeURI {
			t.Fatalf("For(%d) returned the internal-error type for code %q", definition.Status, definition.Code)
		}
	}
}

// TestStatusesAreUnique keeps the reverse lookup deterministic. Two entries
// sharing a status would make For answer whichever the scan reached first, which
// is a wrong answer that looks right.
func TestStatusesAreUnique(t *testing.T) {
	t.Parallel()

	seen := make(map[int]problem.Code)
	for _, definition := range problem.All() {
		if existing, duplicate := seen[definition.Status]; duplicate {
			t.Fatalf("status %d is published by both %q and %q", definition.Status, existing, definition.Code)
		}
		seen[definition.Status] = definition.Code
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

// TestForRefusesUnpublishedStatus pins the behavior that replaced the silent
// wrong answer.
func TestForRefusesUnpublishedStatus(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusTeapot, http.StatusPaymentRequired, http.StatusGone, 0} {
		definition, ok := problem.For(status)
		if ok {
			t.Fatalf("For(%d) = %+v, true; want refusal for an unpublished status", status, definition)
		}
		if definition != (problem.Definition{}) {
			t.Fatalf("For(%d) returned %+v with ok=false", status, definition)
		}
	}
}

func TestForCodeRefusesUnpublishedCode(t *testing.T) {
	t.Parallel()

	definition, ok := problem.ForCode("not_a_published_code")
	if ok || definition != (problem.Definition{}) {
		t.Fatalf("ForCode(unpublished) = (%+v, %t), want (zero, false)", definition, ok)
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
