// Package httpclient builds one fixed-authority HTTP client per dependency.
// Provider adapters own authentication, operation budgets, response handling,
// retries, errors, telemetry, and lifecycle wiring.
package httpclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

const (
	defaultDialTimeout = 30 * time.Second
	defaultKeepAlive   = 30 * time.Second
)

// Client is a reusable fixed-authority HTTP request Doer.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewExternalHTTPS builds a client for one public HTTPS authority.
func NewExternalHTTPS(baseURL string) (*Client, error) {
	return newClient(baseURL, targetPolicy{})
}

// NewPrivateHTTPS builds a client for one HTTPS authority under privateSuffix.
func NewPrivateHTTPS(baseURL, privateSuffix string) (*Client, error) {
	suffix := privateHostSuffix(privateSuffix)
	if suffix == "" {
		return nil, errors.New("build outbound HTTP client: private DNS suffix is required")
	}
	return newClient(baseURL, targetPolicy{privateSuffix: suffix})
}

func newClient(rawBaseURL string, policy targetPolicy) (*Client, error) {
	baseURL, err := validateTarget(rawBaseURL, policy)
	if err != nil {
		return nil, err
	}

	baseTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errors.New("build outbound HTTP client: default transport has unexpected type")
	}
	transport := baseTransport.Clone()
	transport.Proxy = nil
	dialer := &net.Dialer{
		Timeout:   defaultDialTimeout,
		KeepAlive: defaultKeepAlive,
		ControlContext: func(_ context.Context, _, address string, _ syscall.RawConn) error {
			return enforceDialAddress(policy, address)
		},
	}
	transport.DialContext = dialer.DialContext

	roundTripper := propagationSanitizer{base: authorityTransport{
		base:      transport,
		scheme:    baseURL.Scheme,
		authority: baseURL.Host,
	}}
	return &Client{
		baseURL: baseURL.String(),
		httpClient: &http.Client{
			Transport: roundTripper,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

// Do sends request only to the configured authority.
func (c *Client) Do(request *http.Request) (*http.Response, error) {
	// #nosec G704 -- authorityTransport pins scheme/authority and the dialer checks the resolved address.
	response, err := c.httpClient.Do(request)
	if err != nil {
		return response, fmt.Errorf("send outbound HTTP request: %w", err)
	}
	return response, nil
}

// BaseURL returns the validated provider base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// CloseIdleConnections closes this client's idle connection pool.
func (c *Client) CloseIdleConnections() {
	c.httpClient.CloseIdleConnections()
}
