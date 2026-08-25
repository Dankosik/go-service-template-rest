package bearerauthn

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
)

func TestHTTPResolverPreservesBearerContract(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name        string
		headers     []string
		verifierErr error
		wantKind    Kind
		wantIs      error
		wantCalls   int64
		wantSub     string
	}{
		{name: "missing", wantKind: KindMissing},
		{name: "duplicate header", headers: []string{"Bearer one", "Bearer two"}, wantKind: KindMalformed},
		{name: "invalid", headers: []string{"Bearer token"}, verifierErr: NewError(KindInvalid), wantKind: KindInvalid, wantCalls: 1},
		{
			name:        "canceled",
			headers:     []string{"Bearer token"},
			verifierErr: fmt.Errorf("wait: %w", context.Canceled),
			wantIs:      context.Canceled,
			wantCalls:   1,
		},
		{name: "success", headers: []string{"Bearer token"}, wantCalls: 1, wantSub: "subject-1"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			verifier := &fakeVerifier{err: testCase.verifierErr}
			runtime := newTestRuntime(t, verifier)
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/private", http.NoBody)
			for _, value := range testCase.headers {
				request.Header.Add("Authorization", value)
			}
			principal, err := runtime.ResolveHTTP(t.Context(), bearerAuthInput(request))
			if verifier.calls.Load() != testCase.wantCalls {
				t.Fatalf("verifier calls = %d, want %d", verifier.calls.Load(), testCase.wantCalls)
			}
			if len(request.Header.Values("Authorization")) != 0 {
				t.Fatal("Authorization reached the handler-visible request")
			}
			if testCase.wantKind != 0 {
				requireKind(t, err, testCase.wantKind)
				return
			}
			if testCase.wantIs != nil {
				if !errors.Is(err, testCase.wantIs) {
					t.Fatalf("ResolveHTTP() error = %v, want %v", err, testCase.wantIs)
				}
				return
			}
			if err != nil || principal.Subject != testCase.wantSub {
				t.Fatalf("principal = %+v, error = %v", principal, err)
			}
		})
	}
}

func TestHTTPResolverIgnoresUnsupportedSchemesWithoutCounting(t *testing.T) {
	t.Parallel()
	verifier := &fakeVerifier{}
	runtime := newTestRuntime(t, verifier)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/private", http.NoBody)
	request.Header.Set("Authorization", "Bearer token")
	_, err := runtime.ResolveHTTP(t.Context(), &openapi3filter.AuthenticationInput{
		SecurityScheme: &openapi3.SecurityScheme{Type: "apiKey", Name: "X-API-Key"},
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request: request,
		},
	})
	if err == nil || verifier.calls.Load() != 0 {
		t.Fatalf("unsupported scheme error = %v, calls = %d", err, verifier.calls.Load())
	}
	if request.Header.Get("Authorization") == "" {
		t.Fatal("unsupported scheme removed Authorization")
	}
}
