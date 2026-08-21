package oauth2clientcredentials

// profile:outbound-auth-http:start

import (
	"fmt"
	"net/http"

	"github.com/example/go-service-template-rest/internal/infra/httpclient"
)

// HTTPClient is an authenticated bounded client. It exposes no credential API.
type HTTPClient struct {
	client *Client
	base   authorizedClient
}

type authorizedClient interface {
	DoWithAuthorization(
		request *http.Request,
		authorize httpclient.AttemptAuthorizer,
	) (*http.Response, error)
}

// HTTP binds this credential owner to one fixed-authority bounded client.
func (c *Client) HTTP(base *httpclient.Client) (*HTTPClient, error) {
	if !c.available() || base == nil {
		return nil, ErrInvalidConfiguration
	}
	return &HTTPClient{client: c, base: base}, nil
}

// Do authenticates each concrete retry attempt with the current valid token.
func (c *HTTPClient) Do(request *http.Request) (*http.Response, error) {
	if c == nil || c.client == nil || c.base == nil || request == nil || request.URL == nil ||
		hasAuthorization(request.Header) {
		return nil, ErrInvalidConfiguration
	}
	response, err := c.base.DoWithAuthorization(request, func(attempt *http.Request) error {
		if hasAuthorization(attempt.Header) {
			return ErrInvalidConfiguration
		}
		token, resolveErr := c.client.resolve(attempt.Context())
		if resolveErr != nil {
			return resolveErr
		}
		token.SetAuthHeader(attempt)
		return nil
	})
	if err != nil {
		return response, fmt.Errorf("send authenticated resource request: %w", err)
	}
	return response, nil
}

func hasAuthorization(header http.Header) bool {
	for name := range header {
		if http.CanonicalHeaderKey(name) == "Authorization" {
			return true
		}
	}
	return false
}

// profile:outbound-auth-http:end
