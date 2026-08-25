package bearerauthn

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestIntrospectionDisclosureBoundary(t *testing.T) {
	t.Parallel()
	const canary = "peer-authn-redaction-canary"
	runtime := newTestRuntime(t, &fakeVerifier{err: errors.New(canary)})
	_, err := runtime.verifyCredential(t.Context(), []string{"Bearer token"}, transportHTTP)
	if rendered := fmt.Sprintf("%s %v %+v", err, err, err); strings.Contains(rendered, canary) {
		t.Fatalf("authentication error disclosed %q", canary)
	}
}
