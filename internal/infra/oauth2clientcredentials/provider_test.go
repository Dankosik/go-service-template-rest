package oauth2clientcredentials

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestProviderDelegatesParsingAndPublishesOnlySafeBearerState(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		clientID, clientSecret, ok := request.BasicAuth()
		if !ok || clientID != url.QueryEscape("client") || clientSecret != url.QueryEscape("secret") {
			t.Errorf("BasicAuth() = %q, %q, %t", clientID, clientSecret, ok)
		}
		if err := request.ParseForm(); err != nil {
			t.Errorf("ParseForm() error = %v", err)
		}
		if request.Form.Get("grant_type") != "client_credentials" || request.Form.Get("scope") != "payments.read" {
			t.Errorf("token form = %v", request.Form)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(response, "{\"access_token\":\"opaque\",\"token_type\":\"Bearer\",\"expires_in\":\"3600\",\"refresh_token\":\"discard-me\",\"provider_extra\":\"discard-me\"}")
	}))
	t.Cleanup(server.Close)

	client := newClient(newProviderAcquirer(Config{
		TokenURL: server.URL, ClientID: "client", ClientSecret: "secret", Scopes: []string{"payments.read"},
	}, server.Client()), nil)
	t.Cleanup(client.Close)
	token, err := (clientTokenSource{client: client}).Token()
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if token.AccessToken != "opaque" || token.TokenType != "Bearer" || token.RefreshToken != "" || token.Extra("provider_extra") != nil {
		t.Fatalf("published token = %#v", token)
	}
	if remaining := time.Until(token.Expiry); remaining < 50*time.Minute {
		t.Fatalf("token lifetime = %v, want provider value near 1h", remaining)
	}
}

func TestProviderErrorsAndUnusableTokensAreOpaque(t *testing.T) {
	const canary = "provider-secret-canary"
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		http.Error(response, canary, http.StatusBadRequest)
	}))
	t.Cleanup(server.Close)
	client := newClient(newProviderAcquirer(
		Config{TokenURL: server.URL, ClientID: "client", ClientSecret: "secret"},
		server.Client(),
	), nil)
	t.Cleanup(client.Close)
	_, err := (clientTokenSource{client: client}).Token()
	if err == nil {
		t.Fatal("Token() error = nil, want opaque ErrUnavailable")
	}
	if !errors.Is(err, ErrUnavailable) || strings.Contains(err.Error(), canary) {
		t.Fatalf("Token() error = %q, want opaque ErrUnavailable", err)
	}
	if _, ok := errors.AsType[*oauth2.RetrieveError](err); ok {
		t.Fatal("raw oauth2.RetrieveError escaped")
	}

	for name, token := range map[string]*oauth2.Token{
		"missing expiry": {AccessToken: "opaque", TokenType: "Bearer"},
		"missing type":   {AccessToken: "opaque", Expiry: time.Now().Add(time.Hour)},
		"header newline": {AccessToken: "opaque\nleak", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}, //nolint:gosec // Rejected test fixture.
		"expires soon":   {AccessToken: "opaque", TokenType: "Bearer", Expiry: time.Now().Add(defaultEarlyExpiry / 2)},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := sanitizeToken(token); !errors.Is(err, ErrUnavailable) {
				t.Fatalf("sanitizeToken() error = %v, want ErrUnavailable", err)
			}
		})
	}
}

func TestProviderDoesNotFollowRedirect(t *testing.T) {
	var calls int
	tokenHTTP := newNoRedirectHTTPClient(roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{
			StatusCode: http.StatusTemporaryRedirect,
			Header:     http.Header{"Location": []string{"/redirected"}},
			Body:       http.NoBody,
			Request:    request,
		}, nil
	}))
	client := newClient(newProviderAcquirer(Config{ //nolint:gosec // Obvious local test fixture, not a live credential.
		TokenURL: "https://auth.example.com/token", ClientID: "client", ClientSecret: "secret",
	}, tokenHTTP), nil)
	t.Cleanup(client.Close)

	if _, err := (clientTokenSource{client: client}).Token(); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Token() error = %v, want ErrUnavailable", err)
	}
	if calls != 1 {
		t.Fatalf("provider calls = %d, want 1", calls)
	}
}
