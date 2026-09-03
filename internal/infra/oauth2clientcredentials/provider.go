package oauth2clientcredentials

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/httpclient"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type doerRoundTripper struct {
	client *httpclient.Client
}

func (t doerRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return t.client.Do(request) //nolint:wrapcheck // The fixed-target client owns the safe public error.
}

func newNoRedirectHTTPClient(transport http.RoundTripper) *http.Client {
	return &http.Client{
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func newTokenHTTPClient(cfg Config) (*httpclient.Client, error) {
	client, err := httpclient.NewExternalHTTPS(cfg.TokenURL, httpclient.TransportLimits{
		ResponseHeaderTimeout:  defaultAcquisitionTimeout,
		MaxResponseHeaderBytes: maxTokenResponseHeaders,
		MaxInFlight:            maxTokenRequestsInFlight,
		AbsoluteBodyBytes:      maxTokenResponseBody,
	})
	if err != nil {
		return nil, ErrInvalidConfiguration
	}
	return client, nil
}

func newAcquirer(cfg Config, bounded *httpclient.Client) acquireToken {
	return newProviderAcquirer(cfg, newNoRedirectHTTPClient(doerRoundTripper{client: bounded}))
}

func newProviderAcquirer(cfg Config, tokenHTTP *http.Client) acquireToken {
	provider := clientcredentials.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		TokenURL:     cfg.TokenURL,
		Scopes:       cfg.Scopes,
		AuthStyle:    oauth2.AuthStyleInHeader,
	}
	return func(ctx context.Context) (*oauth2.Token, error) {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, tokenHTTP)
		return provider.Token(ctx)
	}
}

func sanitizeToken(token *oauth2.Token) (*oauth2.Token, error) {
	if token == nil || token.AccessToken == "" || token.Expiry.IsZero() ||
		!time.Now().Add(defaultEarlyExpiry).Before(token.Expiry) ||
		!strings.EqualFold(token.TokenType, "Bearer") || strings.ContainsAny(token.AccessToken, "\r\n") {
		return nil, ErrUnavailable
	}
	return &oauth2.Token{
		AccessToken: token.AccessToken,
		TokenType:   "Bearer",
		Expiry:      token.Expiry,
	}, nil
}
