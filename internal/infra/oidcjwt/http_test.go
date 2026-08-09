package oidcjwt

// Proof for http.go: the HTTP boundary's transport trust, credential removal,
// and one carrier class per Kind an operator can see.

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestHTTPAuthnBoundaryTransportAndIdentity(t *testing.T) {
	now := testNow
	key := loadTestRSAKey(t, testSigningKey)
	verifier := newTestVerifier(t, key)
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://service.example/protected", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("X-Forwarded-Proto", "https")
	request.Header.Set("Authorization", "Bearer "+signToken(t, key, "key-1", "at+jwt", validClaims(now)))
	input := bearerAuthInput(request)
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

// TestHTTPAuthnBoundaryAnswersOnlyItsOwnSecurityScheme covers the requirement a
// contract declares that this Verifier does not implement.
//
// The validator calls the resolver once per scheme in every security
// requirement until one is met, so a second declared scheme reaches ResolveHTTP
// as itself. Two things must not follow. An access token must not prove that
// other scheme's requirement — nothing about a bearer JWT says anything about an
// API key. And the credential must survive, because the bearer requirement is
// asked afterwards and ResolveHTTP strips the header it reads: consuming it for
// a requirement this boundary declined would refuse a valid caller as one who
// sent nothing.
func TestHTTPAuthnBoundaryAnswersOnlyItsOwnSecurityScheme(t *testing.T) {
	key := loadTestRSAKey(t, testSigningKey)
	credential := "Bearer " + signToken(t, key, "key-1", "at+jwt", validClaims(testNow))

	tests := []struct {
		name   string
		scheme *openapi3.SecurityScheme
	}{
		{name: "api key", scheme: otherSchemeAuthInput(&http.Request{}).SecurityScheme},
		// Same type as the scheme this boundary implements, different scheme
		// token: the case that fails if only one of the two fields is read.
		{name: "http basic", scheme: &openapi3.SecurityScheme{Type: "http", Scheme: "basic"}},
		{name: "oauth2", scheme: &openapi3.SecurityScheme{Type: "oauth2"}},
		{name: "no declaration at all", scheme: nil},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			verifier := newTestVerifier(t, key)
			request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://service.example/protected", http.NoBody)
			if err != nil {
				t.Fatal(err)
			}
			request.RemoteAddr = "127.0.0.1:1234"
			request.Header.Set("X-Forwarded-Proto", "https")
			request.Header.Set("Authorization", credential)

			declined := bearerAuthInput(request)
			declined.SecuritySchemeName = "otherAuth"
			declined.SecurityScheme = testCase.scheme

			principal, err := verifier.ResolveHTTP(t.Context(), declined)
			if !errors.Is(err, errUnsupportedSecurityScheme) {
				t.Fatalf("ResolveHTTP() error = %v, want an unsupported security scheme", err)
			}
			if kind, counted := KindOf(err); counted {
				t.Fatalf("a declined requirement was reported as verification outcome %v", kind)
			}
			if principal.Subject != "" {
				t.Fatalf("principal = %+v, want none", principal)
			}
			if request.Header.Get("Authorization") != credential {
				t.Fatal("a declined requirement consumed the credential")
			}

			// The sequence, not just the refusal: this is the call the validator
			// makes next, and it is the one the old ordering broke.
			principal, err = verifier.ResolveHTTP(t.Context(), bearerAuthInput(request))
			if err != nil || principal.Subject != "opaque-subject" {
				t.Fatalf(
					"ResolveHTTP() after a declined requirement = (%+v, %v), want the verified caller",
					principal,
					err,
				)
			}
		})
	}
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

			request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://service.example/protected", http.NoBody)
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
			principal, err := verifier.ResolveHTTP(t.Context(), bearerAuthInput(request))
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
