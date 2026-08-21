package oauth2clientcredentials

import (
	"context"
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

func TestProviderDelegatesParsingAndPublishesOnlySafeBearerState(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		clientID, clientSecret, ok := request.BasicAuth()
		if !ok || clientID != url.QueryEscape("client") || clientSecret != url.QueryEscape("secret") {
			t.Errorf("BasicAuth() = %q, %q, %t", clientID, clientSecret, ok)
		}
		if err := request.ParseForm(); err != nil {
			t.Errorf("ParseForm() error = %v", err)
		}
		if request.Form.Get("grant_type") != "client_credentials" ||
			request.Form.Get("scope") != "payments.read" || request.Form.Get("audience") != "payments" {
			t.Errorf("token form = %v", request.Form)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(response, `{"access_token":"opaque","token_type":"Bearer","expires_in":"86400","refresh_token":"discard-me","provider_extra":"discard-me"}`)
	}))
	t.Cleanup(server.Close)

	acquire := newProviderAcquirer(Config{
		TokenURL: server.URL, ClientID: "client", ClientSecret: "secret",
		Scopes: []string{"payments.read"}, Audience: "payments",
	}, server.Client())
	token, err := acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire() error = %v", err)
	}
	if token.AccessToken != "opaque" || token.TokenType != "Bearer" || token.RefreshToken != "" || token.Extra("provider_extra") != nil {
		t.Fatalf("published token = %#v", token)
	}
	if remaining := time.Until(token.Expiry); remaining < 23*time.Hour {
		t.Fatalf("token lifetime = %v, want provider value near 24h", remaining)
	}
}

func TestProviderErrorsAndUnusableTokensAreOpaque(t *testing.T) {
	const canary = "provider-secret-canary"
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		http.Error(response, canary, http.StatusBadRequest)
	}))
	t.Cleanup(server.Close)
	acquire := newProviderAcquirer(Config{TokenURL: server.URL, ClientID: "client", ClientSecret: "secret"}, server.Client())
	_, err := acquire(context.Background())
	if err == nil {
		t.Fatal("acquire() error = nil, want opaque ErrUnavailable")
	}
	if !errors.Is(err, ErrUnavailable) || strings.Contains(err.Error(), canary) {
		t.Fatalf("acquire() error = %q, want opaque ErrUnavailable", err)
	}
	if _, ok := errors.AsType[*oauth2.RetrieveError](err); ok {
		t.Fatal("raw oauth2.RetrieveError escaped")
	}

	for name, token := range map[string]*oauth2.Token{
		"missing expiry": {AccessToken: "opaque", TokenType: "Bearer"},
		"missing type":   {AccessToken: "opaque", Expiry: time.Now().Add(time.Hour)},
		"header newline": {AccessToken: "opaque\nleak", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := sanitizeToken(token); !errors.Is(err, ErrUnavailable) {
				t.Fatalf("sanitizeToken() error = %v, want ErrUnavailable", err)
			}
		})
	}
}
