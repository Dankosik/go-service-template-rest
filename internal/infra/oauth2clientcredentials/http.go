package oauth2clientcredentials

// profile:outbound-auth-http:start

import (
	"fmt"
	"net/http"

	"github.com/example/go-service-template-rest/internal/infra/httpclient"
	"golang.org/x/oauth2"
)

// HTTPClient is an authenticated fixed-target client. It exposes no credential API.
type HTTPClient struct {
	client *http.Client
}

// HTTP binds this credential owner to one fixed-authority client.
func (c *Client) HTTP(base *httpclient.Client) (*HTTPClient, error) {
	if !c.available() || base == nil {
		return nil, ErrInvalidConfiguration
	}
	return &HTTPClient{client: &http.Client{Transport: &oauth2.Transport{
		Source: clientTokenSource{client: c},
		Base:   doerRoundTripper{client: base},
	}}}, nil
}

// Do authenticates one request copy with the current valid token.
func (c *HTTPClient) Do(request *http.Request) (*http.Response, error) {
	if c == nil || c.client == nil || request == nil || request.URL == nil || hasAuthorization(request.Header) {
		return nil, ErrInvalidConfiguration
	}
	response, err := c.client.Do(request)
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
