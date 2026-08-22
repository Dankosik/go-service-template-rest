package bearerauthn

import (
	"fmt"
	"strings"
	"testing"
)

func TestKindOfInspectsWrappedSanitizedErrors(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("poison parser/provider detail: %w", NewError(KindUnavailable))
	requireKind(t, err, KindUnavailable)
	if strings.Contains(NewError(KindInvalid).Error(), "parser") {
		t.Fatal("sanitized error leaked diagnostic text")
	}
}
