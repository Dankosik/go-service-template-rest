package oauth2clientcredentials

// profile:outbound-auth-http:start

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/example/go-service-template-rest/internal/infra/httpclient"
)

// HTTPClient attaches one operation-fixed credential through the bounded
// client's real attempt path.
type HTTPClient struct {
	client *Client
	base   *httpclient.Client
}

// NewHTTPClient binds one credential owner to its fixed resource client.
func NewHTTPClient(client *Client, base *httpclient.Client) (*HTTPClient, error) {
	if client == nil || base == nil || base.BaseURL() != client.config.ResourceAuthority {
		return nil, failure(FailureInvalidConfiguration)
	}
	return &HTTPClient{client: client, base: base}, nil
}

// Do sends one authenticated logical resource operation.
func (c *HTTPClient) Do(request *http.Request) (*http.Response, error) {
	if request == nil || request.URL == nil || hasAuthorization(request.Header) {
		return nil, failure(FailureInvalidConfiguration)
	}
	token, err := c.client.resolve(request.Context())
	if err != nil {
		return nil, err
	}
	response, err := c.base.DoWithAuthorization(request, func(attempt *http.Request) error {
		if hasAuthorization(attempt.Header) {
			return failure(FailureInvalidConfiguration)
		}
		value, authorizationErr := token.authorization()
		if authorizationErr != nil {
			return authorizationErr
		}
		attempt.Header.Set("Authorization", "Bearer "+value)
		return nil
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return response, callerFailure(err)
		}
		return response, fmt.Errorf("send authenticated resource request: %w", err)
	}
	if response != nil {
		switch response.StatusCode {
		case http.StatusUnauthorized:
			c.client.telemetry.recordResourceRejection(request.Context(), transportHTTP, resultUnauthenticated)
		case http.StatusForbidden:
			c.client.telemetry.recordResourceRejection(request.Context(), transportHTTP, resultForbidden)
		}
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
