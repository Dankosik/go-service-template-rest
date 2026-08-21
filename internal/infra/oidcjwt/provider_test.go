package oidcjwt

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDiscoveryValidation(t *testing.T) {
	t.Parallel()
	policy := testPolicy(t)
	valid := `{"issuer":"` + testIssuer + `","jwks_uri":"https://keys.example.com/jwks.json"}`
	if got, err := validateDiscovery([]byte(valid), policy); err != nil || got != "https://keys.example.com/jwks.json" {
		t.Fatalf("validateDiscovery() = %q, %v", got, err)
	}
	for _, document := range []string{
		`{"issuer":"https://other.example","jwks_uri":"https://keys.example.com/jwks.json"}`,
		`{"issuer":"` + testIssuer + `","jwks_uri":"http://keys.example.com/jwks.json"}`,
		`not-json`,
	} {
		if _, err := validateDiscovery([]byte(document), policy); err == nil {
			t.Fatalf("validateDiscovery(%q) succeeded", document)
		}
	}
}

func TestFetchDocumentPreservesCallerCancellationAndRedactsProviderContent(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := fetchDocument(ctx, requestClientFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("secret provider detail")
	}), "https://issuer.example/document")
	if !errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "secret") {
		t.Fatalf("fetchDocument() error = %v", err)
	}

	_, err = fetchDocument(t.Context(), requestClientFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(strings.NewReader("secret body"))}, nil
	}), "https://issuer.example/document")
	if err == nil || strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "502") {
		t.Fatalf("fetchDocument() error = %v", err)
	}
}

type requestClientFunc func(*http.Request) (*http.Response, error)

func (f requestClientFunc) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}
