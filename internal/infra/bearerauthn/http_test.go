package bearerauthn

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
)

func TestHTTPResolverPreservesBearerContract(t *testing.T) {
	t.Parallel()

	boundary := strings.Repeat("x", MaxTokenBytes)
	for _, testCase := range []struct {
		name      string
		headers   []string
		scheme    *openapi3.SecurityScheme
		err       error
		wantKind  Kind
		wantCalls int64
		wantSub   string
	}{
		{name: "missing", wantKind: KindMissing},
		{name: "duplicate header", headers: []string{"Bearer one", "Bearer two"}, wantKind: KindMalformed},
		{name: "wrong scheme", headers: []string{"Basic token"}, wantKind: KindMalformed},
		{name: "whitespace", headers: []string{" Bearer token"}, wantKind: KindMalformed},
		{name: "empty", headers: []string{"Bearer "}, wantKind: KindMalformed},
		{name: "oversize", headers: []string{"Bearer " + strings.Repeat("x", MaxTokenBytes+1)}, wantKind: KindOversize},
		{name: "size boundary", headers: []string{"Bearer " + boundary}, wantCalls: 1, wantSub: "subject-1"},
		{name: "scheme case", headers: []string{"bearer token"}, wantCalls: 1, wantSub: "subject-1"},
		{name: "invalid", headers: []string{"Bearer token"}, err: NewError(KindInvalid), wantKind: KindInvalid, wantCalls: 1},
		{name: "unavailable", headers: []string{"Bearer token"}, err: NewError(KindUnavailable), wantKind: KindUnavailable, wantCalls: 1},
		{name: "canceled", headers: []string{"Bearer token"}, err: fmt.Errorf("wait: %w", context.Canceled), wantCalls: 1},
		{name: "deadline", headers: []string{"Bearer token"}, err: fmt.Errorf("wait: %w", context.DeadlineExceeded), wantCalls: 1},
		{name: "success", headers: []string{"Bearer token"}, wantCalls: 1, wantSub: "subject-1"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			verifier := &fakeVerifier{err: testCase.err}
			runtime := newTestRuntime(t, verifier)
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/private", http.NoBody)
			for _, value := range testCase.headers {
				request.Header.Add("Authorization", value)
			}
			input := bearerAuthInput(request)
			if testCase.scheme != nil {
				input.SecurityScheme = testCase.scheme
			}
			principal, err := runtime.ResolveHTTP(t.Context(), input)
			if verifier.calls.Load() != testCase.wantCalls {
				t.Fatalf("verifier calls = %d, want %d", verifier.calls.Load(), testCase.wantCalls)
			}
			if request.Header.Get("Authorization") != "" {
				t.Fatal("Authorization reached the handler-visible request")
			}
			if testCase.wantKind != 0 {
				requireKind(t, err, testCase.wantKind)
				return
			}
			if testCase.err != nil {
				if err == nil {
					t.Fatal("ResolveHTTP() error = nil, want context failure")
				}
				return
			}
			if err != nil || principal.Subject != testCase.wantSub {
				t.Fatalf("principal = %+v, error = %v", principal, err)
			}
		})
	}
}

func TestHTTPPublishesPrincipalWithoutForwardedTransportPolicy(t *testing.T) {
	t.Parallel()
	runtime := newTestRuntime(t, &fakeVerifier{
		result: Result{Principal: reqctx.Principal{Issuer: "https://issuer.example.com", Subject: "subject-1", ClientID: "client-1"}},
	})
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/private", http.NoBody)
	request.Header.Set("Authorization", "Bearer token")
	principal, err := runtime.ResolveHTTP(t.Context(), bearerAuthInput(request))
	if err != nil {
		t.Fatalf("ResolveHTTP() error = %v", err)
	}
	if principal.Subject != "subject-1" || principal.ClientID != "client-1" {
		t.Fatalf("principal = %+v", principal)
	}
	if request.Header.Get("Authorization") != "" {
		t.Fatal("Authorization reached the handler-visible request")
	}
	ctx := reqctx.ContextWithPrincipal(request.Context(), principal)
	if stored, ok := reqctx.PrincipalFromContext(ctx); !ok ||
		stored.Issuer != principal.Issuer || stored.Subject != principal.Subject || stored.ClientID != principal.ClientID {
		t.Fatalf("PrincipalFromContext() = %+v, %v", stored, ok)
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
