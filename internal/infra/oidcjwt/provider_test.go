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

	_, err = fetchDocument(t.Context(), requestClientFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(strings.Repeat("x", maxProviderBody+1))),
		}, nil
	}), "https://issuer.example/document")
	if err == nil {
		t.Fatal("fetchDocument() accepted an oversized provider document")
	}

	got, err := fetchDocument(t.Context(), requestClientFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("document"))}, nil
	}), "https://issuer.example/document")
	if err != nil || string(got) != "document" {
		t.Fatalf("fetchDocument() = %q, %v", got, err)
	}
	if _, err := fetchDocument(t.Context(), requestClientFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: failingReadCloser{}}, nil
	}), "https://issuer.example/document"); err == nil {
		t.Fatal("fetchDocument() accepted a provider body read failure")
	}
	if _, err := fetchDocument(t.Context(), requestClientFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("unexpected provider request")
	}), "://invalid"); err == nil {
		t.Fatal("fetchDocument() accepted an invalid target")
	}
}

func TestJWKSClientRejectsOversizedDocumentBeforeDecode(t *testing.T) {
	t.Parallel()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://issuer.example/jwks", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	transport := jwksRoundTripper{client: requestClientFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(strings.Repeat("x", maxProviderBody+1))),
		}, nil
	})}
	response, err := transport.RoundTrip(request)
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	if response != nil || err == nil {
		t.Fatalf("RoundTrip() = response %#v, error %v; want bounded rejection", response, err)
	}
}

func TestJWKSClientBoundsAndReconstructsProviderResponse(t *testing.T) {
	t.Parallel()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://issuer.example/jwks", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	for name, client := range map[string]requestClient{
		"empty response": requestClientFunc(func(*http.Request) (*http.Response, error) {
			return nil, nil //nolint:nilnil // Invalid provider response under test.
		}),
		"empty body": requestClientFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK}, nil
		}),
		"read failure": requestClientFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: failingReadCloser{}}, nil
		}),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			response, err := (jwksRoundTripper{client: client}).RoundTrip(request)
			if response != nil && response.Body != nil {
				_ = response.Body.Close()
			}
			if err == nil {
				t.Fatal("RoundTrip() error = nil")
			}
		})
	}

	response, err := (jwksRoundTripper{client: requestClientFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"keys":[]}`))}, nil
	})}).RoundTrip(request)
	if err != nil || response == nil || response.Body == nil {
		t.Fatalf("RoundTrip() response = %#v, error %v", response, err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	body, err := io.ReadAll(response.Body)
	if err != nil || string(body) != `{"keys":[]}` || response.ContentLength != int64(len(body)) {
		t.Fatalf("reconstructed body = %q, length %d, error %v", body, response.ContentLength, err)
	}
}

func TestProviderClientConstructionRejectsInvalidAuthority(t *testing.T) {
	t.Parallel()
	if _, err := discoverJWKSURI(t.Context(), Policy{issuer: ":"}); err == nil {
		t.Fatal("discoverJWKSURI() accepted an invalid issuer")
	}
	if client, closeIdle, err := newJWKSClient(":"); err == nil || client != nil || closeIdle != nil {
		t.Fatalf("newJWKSClient(invalid) = client %#v, close %t, error %v", client, closeIdle != nil, err)
	}
	client, closeIdle, err := newJWKSClient("https://keys.example.com/jwks")
	if err != nil || client == nil || closeIdle == nil {
		t.Fatalf("newJWKSClient(valid) = client %#v, close %t, error %v", client, closeIdle != nil, err)
	}
	closeIdle()
}

type requestClientFunc func(*http.Request) (*http.Response, error)

func (f requestClientFunc) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}

type failingReadCloser struct{}

func (failingReadCloser) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (failingReadCloser) Close() error             { return nil }
