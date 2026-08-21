package oauth2clientcredentials

// profile:outbound-auth-http:start

import (
	"fmt"
	"net/http"

	"github.com/example/go-service-template-rest/internal/infra/httpclient"
)

// HTTPClient is an authenticated fixed-target client. It exposes no credential API.
type HTTPClient struct {
	client *Client
	base   resourceDoer
}

type resourceDoer interface {
	Do(request *http.Request) (*http.Response, error)
}

// HTTP binds this credential owner to one fixed-authority client.
func (c *Client) HTTP(base *httpclient.Client) (*HTTPClient, error) {
	if !c.available() || base == nil {
		return nil, ErrInvalidConfiguration
	}
	return &HTTPClient{client: c, base: base}, nil
}

// Do authenticates one request copy with the current valid token.
func (c *HTTPClient) Do(request *http.Request) (*http.Response, error) {
	if c == nil || c.client == nil || c.base == nil || request == nil || request.URL == nil ||
		hasAuthorization(request.Header) {
		return nil, ErrInvalidConfiguration
	}
	token, err := c.client.resolve(request.Context())
	if err != nil {
		return nil, err
	}
	attempt := request.Clone(request.Context())
	attempt.Header = request.Header.Clone()
	token.SetAuthHeader(attempt)
	response, err := c.base.Do(attempt)
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
