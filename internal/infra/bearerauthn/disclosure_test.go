package bearerauthn

import (
	"fmt"
	"strings"
	"testing"
)

func TestIntrospectionDisclosureBoundary(t *testing.T) {
	t.Parallel()
	canaries := []string{"token-canary", "secret-canary", "endpoint-canary"}
	for _, kind := range []Kind{KindInvalid, KindUnavailable, KindMissing, KindMalformed, KindOversize} {
		err := fmt.Errorf("wrap: %w", NewError(kind))
		rendered := fmt.Sprintf("%s %v %+v", err, err, err)
		for _, canary := range canaries {
			if strings.Contains(rendered, canary) {
				t.Fatalf("kind %v disclosed %q", kind, canary)
			}
		}
	}
}
