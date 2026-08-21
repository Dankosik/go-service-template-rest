package oauth2clientcredentials

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/httpclient"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	defaultAcquisitionTimeout = 5 * time.Second
	maxProviderHeaderBytes    = 32 << 10
	maxProviderBodyBytes      = 64 << 10
)

type doerRoundTripper struct {
	client *httpclient.Client
}

func (t doerRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return t.client.Do(request) //nolint:wrapcheck // The bounded client owns the safe public error.
}

func newTokenHTTPClient(cfg Config) (*httpclient.Client, error) {
	client, err := httpclient.New(httpclient.Config{
		DependencyName:         "oauth-token",
		BaseURL:                cfg.TokenURL,
		TargetClass:            cfg.TokenTargetClass,
		OneAttempt:             true,
		DisableInstrumentation: true,
		PrivateHostSuffix:      cfg.TokenPrivateHostSuffix,
		RequestTimeout:         defaultAcquisitionTimeout,
		ResponseHeaderTimeout:  defaultAcquisitionTimeout,
		MaxResponseHeaderBytes: maxProviderHeaderBytes,
		MaxResponseBodyBytes:   maxProviderBodyBytes,
		MaxConnsPerHost:        1,
		MaxIdleConnsPerHost:    1,
	}, nil)
	if err != nil {
		return nil, ErrInvalidConfiguration
	}
	return client, nil
}

func newAcquirer(cfg Config, bounded *httpclient.Client) acquireToken {
	return newProviderAcquirer(cfg, &http.Client{
		Transport: doerRoundTripper{client: bounded},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	})
}

func newProviderAcquirer(cfg Config, tokenHTTP *http.Client) acquireToken {
	params := make(url.Values, 1)
	if cfg.Audience != "" {
		params.Set("audience", cfg.Audience)
	}
	if cfg.Resource != "" {
		params.Set("resource", cfg.Resource)
	}
	provider := clientcredentials.Config{
		ClientID:       cfg.ClientID,
		ClientSecret:   cfg.ClientSecret,
		TokenURL:       cfg.TokenURL,
		Scopes:         cfg.Scopes,
		EndpointParams: params,
		AuthStyle:      oauth2.AuthStyleInHeader,
	}
	return func(ctx context.Context) (*oauth2.Token, error) {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, tokenHTTP)
		token, err := provider.Token(ctx)
		if err != nil {
			return nil, ErrUnavailable
		}
		return sanitizeToken(token)
	}
}

func sanitizeToken(token *oauth2.Token) (*oauth2.Token, error) {
	if token == nil || token.Expiry.IsZero() || !token.Valid() ||
		!strings.EqualFold(token.TokenType, "Bearer") || strings.ContainsAny(token.AccessToken, "\r\n") {
		return nil, ErrUnavailable
	}
	return &oauth2.Token{
		AccessToken: token.AccessToken,
		TokenType:   "Bearer",
		Expiry:      token.Expiry,
	}, nil
}
