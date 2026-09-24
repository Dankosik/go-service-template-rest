package bearerauthn

import (
	"fmt"
	"testing"
)

func TestKindOfInspectsWrappedSanitizedErrors(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("poison parser/provider detail: %w", NewError(KindUnavailable))
	requireKind(t, err, KindUnavailable)
}
