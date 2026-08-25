package bearerauthn

import (
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/reqctx"
)

func TestRuntimeRejectsInvalidVerifierResult(t *testing.T) {
	t.Parallel()

	expiresAt := time.Unix(1_900_003_600, 0)
	for _, testCase := range []struct {
		name   string
		result Result
	}{
		{name: "empty"},
		{
			name: "missing expiry",
			result: Result{
				Principal: reqctx.Principal{Issuer: "https://issuer.example.com", Subject: "subject-1"},
			},
		},
		{
			name: "missing issuer",
			result: Result{
				Principal: reqctx.Principal{Subject: "subject-1"},
				ExpiresAt: expiresAt,
			},
		},
		{
			name: "missing identity",
			result: Result{
				Principal: reqctx.Principal{Issuer: "https://issuer.example.com"},
				ExpiresAt: expiresAt,
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			runtime := newTestRuntime(t, &fakeVerifier{result: testCase.result, returnResult: true})
			verified, err := runtime.verifyCredential(t.Context(), []string{"Bearer token"}, transportHTTP)
			requireKind(t, err, KindUnavailable)
			if verified.Principal.Issuer != "" || verified.Principal.Subject != "" ||
				verified.Principal.ClientID != "" || len(verified.Principal.Scopes) != 0 ||
				!verified.ExpiresAt.IsZero() {
				t.Fatalf("verified result = %+v, want zero", verified)
			}
		})
	}
}
