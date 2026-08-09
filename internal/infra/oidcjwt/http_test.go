package oidcjwt

// Proof for http.go: the HTTP boundary's transport trust, credential removal,
// and one carrier class per Kind an operator can see.

import (
	"net/http"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3filter"
)

func TestHTTPAuthnBoundaryTransportAndIdentity(t *testing.T) {
	now := testNow
	key := loadTestRSAKey(t, testSigningKey)
	verifier := newTestVerifier(t, key)
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://service.example/protected", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("X-Forwarded-Proto", "https")
	request.Header.Set("Authorization", "Bearer "+signToken(t, key, "key-1", "at+jwt", validClaims(now)))
	input := &openapi3filter.AuthenticationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: request},
	}
	principal, err := verifier.ResolveHTTP(t.Context(), input)
	if err != nil ||
		principal.Issuer != testIssuer ||
		principal.Subject != "opaque-subject" ||
		principal.ClientID != "client-1" {
		t.Fatalf("ResolveHTTP() = (%+v, %v), want verified issuer, subject, and client ID", principal, err)
	}
	if request.Header.Get("Authorization") != "" {
		t.Fatal("handler-visible Authorization header was retained")
	}

	request.RemoteAddr = "198.51.100.10:1234"
	_, err = verifier.ResolveHTTP(t.Context(), input)
	requireKind(t, err, KindUntrustedTransport)
}

func TestHTTPAuthnBoundaryCarrierClasses(t *testing.T) {
	now := testNow
	trustedKey := loadTestRSAKey(t, testSigningKey)
	otherKey := loadTestRSAKey(t, testRotatedKey)
	valid := signToken(t, trustedKey, "key-1", "at+jwt", validClaims(now))
	invalid := signToken(t, otherKey, "key-1", "at+jwt", validClaims(now))
	staleClaims := validClaims(now)
	staleClaims.Expires = now.Add(MaxTokenLifetime).Unix()
	validAfterStaleCutoff := signToken(t, trustedKey, "key-1", "at+jwt", staleClaims)

	tests := []struct {
		name        string
		headers     []string
		remoteAddr  string
		stale       bool
		wantKind    Kind
		wantSubject string
	}{
		{
			name:        "valid",
			headers:     []string{"Bearer " + valid},
			wantSubject: "opaque-subject",
		},
		{
			name:     "missing",
			wantKind: KindMissing,
		},
		{
			name:     "malformed",
			headers:  []string{"Basic opaque-token"},
			wantKind: KindMalformed,
		},
		{
			name:     "duplicate",
			headers:  []string{"Bearer one", "Bearer two"},
			wantKind: KindMalformed,
		},
		{
			name:     "oversize",
			headers:  []string{"Bearer " + strings.Repeat("x", MaxTokenBytes+1)},
			wantKind: KindOversize,
		},
		{
			name:     "invalid",
			headers:  []string{"Bearer " + invalid},
			wantKind: KindInvalid,
		},
		{
			name:       "untrusted transport",
			headers:    []string{"Bearer " + valid},
			remoteAddr: "198.51.100.10:1234",
			wantKind:   KindUntrustedTransport,
		},
		{
			name:     "stale trust",
			headers:  []string{"Bearer " + validAfterStaleCutoff},
			stale:    true,
			wantKind: KindUnavailable,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			client := &scriptedClient{responses: initialResponses(t, trustedKey)}
			clock := newTestClock(now)
			verifier := requireTestVerifier(t, testVerifierOptions{now: clock.now, client: client})
			if testCase.stale {
				clock.advance(MaxKeySetAge)
			}

			request, err := http.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				"https://service.example/protected",
				nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			request.RemoteAddr = testCase.remoteAddr
			if request.RemoteAddr == "" {
				request.RemoteAddr = "127.0.0.1:1234"
			}
			request.Header.Set("X-Forwarded-Proto", "https")
			for _, value := range testCase.headers {
				request.Header.Add("Authorization", value)
			}
			input := &openapi3filter.AuthenticationInput{
				RequestValidationInput: &openapi3filter.RequestValidationInput{Request: request},
			}

			principal, err := verifier.ResolveHTTP(t.Context(), input)
			requireKind(t, err, testCase.wantKind)
			if principal.Subject != testCase.wantSubject {
				t.Fatalf("principal subject = %q, want %q", principal.Subject, testCase.wantSubject)
			}
			if testCase.wantKind != KindUntrustedTransport &&
				request.Header.Get("Authorization") != "" {
				t.Fatal("parsed Authorization header was retained")
			}
			if testCase.wantKind == KindMalformed || testCase.wantKind == KindOversize {
				requireProviderCalls(t, client, 0, "after a credential rejected at the boundary")
			}
		})
	}
}
