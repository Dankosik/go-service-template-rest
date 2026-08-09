package article

import (
	"errors"
	"fmt"
	"testing"

	"github.com/example/go-service-template-rest/internal/failure"
)

func TestClassifyErrorUsesStableFailureIdentity(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		err      error
		wantCode failure.Code
		wantText string
	}{
		{err: ErrNotFound, wantCode: failure.CodeNotFound, wantText: "article was not found"},
		{err: ErrAlreadyExists, wantCode: failure.CodeAlreadyExists, wantText: "an article with this slug already exists"},
		{err: ErrInvalid, wantCode: failure.CodeBadRequest, wantText: "request is malformed or invalid"},
	} {
		got, ok := ClassifyError(fmt.Errorf("operation failed: %w", testCase.err))
		if !ok || got.Code != testCase.wantCode || got.Detail != testCase.wantText {
			t.Errorf("ClassifyError(%v) = (%+v, %t), want %q/%q", testCase.err, got, ok, testCase.wantCode, testCase.wantText)
		}
	}

	if got, ok := ClassifyError(errors.New("dependency secret")); ok || got != (failure.Classification{}) {
		t.Fatalf("ClassifyError(unknown) = (%+v, %t), want (zero, false)", got, ok)
	}
}
