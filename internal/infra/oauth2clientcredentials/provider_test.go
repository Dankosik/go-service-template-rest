package oauth2clientcredentials

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/httpclient"
)

func TestTokenEndpointAdmissionPreventsCredentialDisclosure(t *testing.T) {
	cfg := validTestConfig()
	client, err := newTokenHTTPClient(cfg, nil)
	if err != nil {
		t.Fatalf("newTokenHTTPClient() error = %v", err)
	}
	t.Cleanup(client.CloseIdleConnections)

	provider, err := newProvider(cfg, client, func() time.Time { return fixedProviderTime })
	if err != nil {
		t.Fatalf("newProvider() error = %v", err)
	}
	for _, endpoint := range []string{
		"http://127.0.0.1:1/oauth/token",
		"https://127.0.0.1:1/oauth/token",
		"https://user@127.0.0.1:1/oauth/token",
	} {
		provider.config.TokenEndpoint = endpoint
		token, acquireErr := provider.acquire(context.Background())
		if token != (accessToken{}) {
			t.Fatalf("acquire(%q) returned a token", endpoint)
		}
		assertFailureClass(t, acquireErr, FailureEndpointTrust)
		for _, forbidden := range []string{cfg.ClientID, cfg.ClientSecret} {
			if strings.Contains(acquireErr.Error(), forbidden) {
				t.Fatalf("acquire(%q) disclosed credential %q: %v", endpoint, forbidden, acquireErr)
			}
		}
	}

	invalid := cfg
	invalid.TokenEndpoint = "https://127.0.0.1/oauth/token"
	if _, err := newTokenHTTPClient(invalid, nil); err == nil {
		t.Fatal("newTokenHTTPClient() admitted an external private address")
	} else {
		assertFailureClass(t, err, FailureInvalidConfiguration)
	}
}

func TestProviderGrantRequestAndStrictResponse(t *testing.T) {
	t.Run("exact Basic form transaction", func(t *testing.T) {
		cfg := validTestConfig()
		var calls int
		client := doerFunc(func(request *http.Request) (*http.Response, error) {
			calls++
			if request.Method != http.MethodPost || request.URL.String() != cfg.TokenEndpoint {
				t.Fatalf("request = %s %s", request.Method, request.URL)
			}
			if request.Header.Get("Accept") != "application/json" ||
				request.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
				t.Fatalf("request headers = %#v", request.Header)
			}
			username, password, ok := request.BasicAuth()
			if !ok || username != url.QueryEscape(cfg.ClientID) || password != url.QueryEscape(cfg.ClientSecret) {
				t.Fatalf("Basic credential = %q, %q, %t", username, password, ok)
			}
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatal(err)
			}
			form, err := url.ParseQuery(string(body))
			if err != nil {
				t.Fatalf("ParseQuery() error = %v", err)
			}
			want := url.Values{
				"grant_type": {"client_credentials"},
				"scope":      {"payments.read payments.write"},
				"resource":   {cfg.Resource},
			}
			if form.Encode() != want.Encode() {
				t.Fatalf("grant form = %q, want %q", form.Encode(), want.Encode())
			}
			if strings.Contains(request.URL.String(), cfg.ClientID) || strings.Contains(request.URL.String(), cfg.ClientSecret) ||
				strings.Contains(string(body), cfg.ClientSecret) {
				t.Fatal("request disclosed a credential outside Basic authorization")
			}
			return providerResponse(http.StatusOK, "application/json", validTokenJSON()), nil
		})
		provider, err := newProvider(cfg, client, func() time.Time { return fixedProviderTime })
		if err != nil {
			t.Fatalf("newProvider() error = %v", err)
		}
		token, err := provider.acquire(context.Background())
		if err != nil {
			t.Fatalf("acquire() error = %v", err)
		}
		if calls != 1 || token.value != "abc.DEF_~+/==" || !token.expiresAt.Equal(fixedProviderTime.Add(time.Minute)) {
			t.Fatalf("calls=%d token=%#v", calls, token)
		}
	})

	validCases := []struct {
		name      string
		mediaType string
		body      string
	}{
		{name: "JSON parameters", mediaType: "application/json; charset=utf-8", body: validTokenJSON()},
		{name: "case-insensitive bearer", mediaType: "application/json", body: `{"access_token":"token","token_type":"bearer","expires_in":11}`},
		{name: "maximum lifetime", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer","expires_in":3600}`},
		{name: "scope omitted", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer","expires_in":60}`},
		{name: "empty refresh and additive field", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer","expires_in":60,"scope":"payments.write payments.read","refresh_token":"","extension":{"accepted":true}}`},
	}
	for _, test := range validCases {
		t.Run(test.name, func(t *testing.T) {
			token, calls, err := acquireScripted(t, test.mediaType, test.body, fixedProviderTime, fixedProviderTime)
			if err != nil || calls != 1 || token.value == "" {
				t.Fatalf("acquire() token=%#v error=%v calls=%d", token, err, calls)
			}
		})
	}

	invalidCases := []struct {
		name      string
		mediaType string
		body      string
		secondNow time.Time
	}{
		{name: "missing media type", body: validTokenJSON()},
		{name: "wrong media type", mediaType: "text/json", body: validTokenJSON()},
		{name: "oversized body", mediaType: "application/json", body: strings.Repeat("x", maxProviderBodyBytes+1)},
		{name: "array", mediaType: "application/json", body: `[]`},
		{name: "trailing object", mediaType: "application/json", body: validTokenJSON() + `{}`},
		{name: "duplicate field", mediaType: "application/json", body: `{"access_token":"one","access_token":"two","token_type":"Bearer","expires_in":60}`},
		{name: "missing access token", mediaType: "application/json", body: `{"token_type":"Bearer","expires_in":60}`},
		{name: "access token type", mediaType: "application/json", body: `{"access_token":1,"token_type":"Bearer","expires_in":60}`},
		{name: "long access token", mediaType: "application/json", body: fmt.Sprintf(`{"access_token":%q,"token_type":"Bearer","expires_in":60}`, strings.Repeat("a", maxAccessTokenBytes+1))},
		{name: "invalid access token grammar", mediaType: "application/json", body: `{"access_token":"abc=def","token_type":"Bearer","expires_in":60}`},
		{name: "missing token type", mediaType: "application/json", body: `{"access_token":"token","expires_in":60}`},
		{name: "unsupported token type", mediaType: "application/json", body: `{"access_token":"token","token_type":"MAC","expires_in":60}`},
		{name: "missing expiry", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer"}`},
		{name: "zero expiry", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer","expires_in":0}`},
		{name: "negative expiry", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer","expires_in":-1}`},
		{name: "signed expiry", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer","expires_in":+1}`},
		{name: "leading-zero expiry", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer","expires_in":01}`},
		{name: "fractional expiry", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer","expires_in":60.0}`},
		{name: "exponent expiry", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer","expires_in":6e1}`},
		{name: "string expiry", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer","expires_in":"60"}`},
		{name: "null expiry", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer","expires_in":null}`},
		{name: "overflow expiry", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer","expires_in":999999999999999999999999}`},
		{name: "long expiry", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer","expires_in":3601}`},
		{name: "expiry at margin", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer","expires_in":10}`},
		{name: "clock crosses margin after headers", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer","expires_in":11}`, secondNow: fixedProviderTime.Add(time.Second)},
		{name: "scope mismatch", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer","expires_in":60,"scope":"payments.read"}`},
		{name: "scope duplicate", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer","expires_in":60,"scope":"payments.read payments.read"}`},
		{name: "scope empty separator", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer","expires_in":60,"scope":"payments.read  payments.write"}`},
		{name: "scope type", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer","expires_in":60,"scope":[]}`},
		{name: "non-empty refresh token", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer","expires_in":60,"refresh_token":"refresh-canary"}`},
		{name: "refresh token type", mediaType: "application/json", body: `{"access_token":"token","token_type":"Bearer","expires_in":60,"refresh_token":null}`},
	}
	for _, test := range invalidCases {
		t.Run(test.name, func(t *testing.T) {
			secondNow := test.secondNow
			if secondNow.IsZero() {
				secondNow = fixedProviderTime
			}
			token, calls, err := acquireScripted(t, test.mediaType, test.body, fixedProviderTime, secondNow)
			if token != (accessToken{}) || calls != 1 {
				t.Fatalf("acquire() token=%#v calls=%d", token, calls)
			}
			assertFailureClass(t, err, FailureUnsupportedResponse)
			if strings.Contains(err.Error(), "refresh-canary") {
				t.Fatalf("acquire() disclosed response content: %v", err)
			}
		})
	}
}

func TestProviderFailureClassificationIsClosed(t *testing.T) {
	t.Run("response returned with transport error is closed", func(t *testing.T) {
		closed := false
		provider := mustProvider(t, doerFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{Body: closeTrackingBody{Reader: strings.NewReader(""), closed: &closed}}, errors.New(forbiddenCanary)
		}))
		_, err := provider.acquire(context.Background())
		assertFailureClass(t, err, FailureProviderUnavailable)
		if !closed {
			t.Fatal("acquire() did not close the response returned with a transport error")
		}
	})

	responseCases := []struct {
		name   string
		status int
		body   string
		want   FailureClass
	}{
		{name: "redirect", status: http.StatusFound, want: FailureEndpointTrust},
		{name: "request timeout", status: http.StatusRequestTimeout, want: FailureProviderUnavailable},
		{name: "rate limited", status: http.StatusTooManyRequests, want: FailureProviderUnavailable},
		{name: "provider failure", status: http.StatusServiceUnavailable, want: FailureProviderUnavailable},
		{name: "provider failure with oversized body", status: http.StatusServiceUnavailable, body: strings.Repeat("x", maxProviderBodyBytes+1) + forbiddenCanary, want: FailureProviderUnavailable},
		{name: "unauthorized", status: http.StatusUnauthorized, want: FailureClientRejected},
		{name: "forbidden", status: http.StatusForbidden, want: FailureClientRejected},
		{name: "invalid client", status: http.StatusBadRequest, body: `{"error":"invalid_client","error_description":"` + forbiddenCanary + `"}`, want: FailureClientRejected},
		{name: "invalid scope", status: http.StatusBadRequest, body: `{"error":"invalid_scope"}`, want: FailureGrantRejected},
		{name: "other client error", status: http.StatusUnprocessableEntity, body: forbiddenCanary, want: FailureGrantRejected},
		{name: "other client error with oversized body", status: http.StatusBadRequest, body: strings.Repeat("x", maxProviderBodyBytes+1) + forbiddenCanary, want: FailureGrantRejected},
		{name: "other success", status: http.StatusCreated, body: forbiddenCanary, want: FailureUnsupportedResponse},
		{name: "unknown status", status: 600, body: forbiddenCanary, want: FailureUnsupportedResponse},
	}
	for _, test := range responseCases {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			provider := mustProvider(t, doerFunc(func(*http.Request) (*http.Response, error) {
				calls++
				return providerResponse(test.status, "application/json", test.body), nil
			}))
			token, err := provider.acquire(context.Background())
			if token != (accessToken{}) || calls != 1 {
				t.Fatalf("acquire() token=%#v calls=%d", token, calls)
			}
			assertFailureClass(t, err, test.want)
		})
	}

	t.Run("provider retry delay remains sanitized", func(t *testing.T) {
		provider := mustProvider(t, doerFunc(func(*http.Request) (*http.Response, error) {
			response := providerResponse(http.StatusTooManyRequests, "application/json", forbiddenCanary)
			response.Header.Set("Retry-After", "3")
			return response, nil
		}))
		_, err := provider.acquire(context.Background())
		assertFailureClass(t, err, FailureProviderUnavailable)
		if got := providerCooldownDelay(err); got != 3*time.Second {
			t.Fatalf("provider cooldown = %s, want 3s", got)
		}
	})

	transportCases := []struct {
		name string
		ctx  func() context.Context
		err  error
		want FailureClass
	}{
		{name: "target denied", ctx: context.Background, err: fmt.Errorf("%s: %w", forbiddenCanary, httpclient.ErrTargetDenied), want: FailureEndpointTrust},
		{name: "unknown authority", ctx: context.Background, err: x509.UnknownAuthorityError{}, want: FailureEndpointTrust},
		{name: "invalid TLS record", ctx: context.Background, err: tls.RecordHeaderError{}, want: FailureEndpointTrust},
		{name: "transport deadline", ctx: context.Background, err: fmt.Errorf("%s: %w", forbiddenCanary, context.DeadlineExceeded), want: FailureProviderTimeout},
		{name: "provider deadline wins", ctx: func() context.Context { return canceledContext(context.DeadlineExceeded) }, err: fmt.Errorf("%s: %w", forbiddenCanary, httpclient.ErrTargetDenied), want: FailureProviderTimeout},
		{name: "owner canceled", ctx: func() context.Context { return canceledContext(context.Canceled) }, err: errors.New(forbiddenCanary), want: FailureProviderUnavailable},
		{name: "network failure", ctx: context.Background, err: errors.New(forbiddenCanary), want: FailureProviderUnavailable},
	}
	for _, test := range transportCases {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			provider := mustProvider(t, doerFunc(func(*http.Request) (*http.Response, error) {
				calls++
				return nil, test.err
			}))
			token, err := provider.acquire(test.ctx())
			if token != (accessToken{}) || calls != 1 {
				t.Fatalf("acquire() token=%#v calls=%d", token, calls)
			}
			assertFailureClass(t, err, test.want)
		})
	}
}

func validTokenJSON() string {
	return `{"access_token":"abc.DEF_~+/==","token_type":"Bearer","expires_in":60,"scope":"payments.write payments.read"}`
}

func acquireScripted(t *testing.T, mediaType, body string, times ...time.Time) (accessToken, int, error) {
	t.Helper()
	calls := 0
	provider := mustProvider(t, doerFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return providerResponse(http.StatusOK, mediaType, body), nil
	}))
	index := 0
	provider.now = func() time.Time {
		if index >= len(times) {
			return times[len(times)-1]
		}
		value := times[index]
		index++
		return value
	}
	token, err := provider.acquire(context.Background())
	return token, calls, err
}

func mustProvider(t *testing.T, client requestDoer) *provider {
	t.Helper()
	provider, err := newProvider(validTestConfig(), client, func() time.Time { return fixedProviderTime })
	if err != nil {
		t.Fatalf("newProvider() error = %v", err)
	}
	return provider
}
