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

// ResponseLimits are caller-supplied response-header guards.
type ResponseLimits struct {
	ResponseHeaderTimeout  time.Duration
	MaxResponseHeaderBytes int64
}

// NewExternalHTTPS builds a client for one public HTTPS authority.
func NewExternalHTTPS(baseURL string) (*Client, error) {
	return newClient(baseURL, targetPolicy{}, ResponseLimits{})
}

// NewExternalHTTPSWithLimits builds a public-HTTPS client with response-header guards.
func NewExternalHTTPSWithLimits(baseURL string, limits ResponseLimits) (*Client, error) {
	if err := validateResponseLimits(limits); err != nil {
		return nil, err
	}
	return newClient(baseURL, targetPolicy{}, limits)
}

// NewPrivateHTTPS builds a client for one HTTPS authority under privateSuffix.
func NewPrivateHTTPS(baseURL, privateSuffix string) (*Client, error) {
	suffix := privateHostSuffix(privateSuffix)
	if suffix == "" {
		return nil, errors.New("build outbound HTTP client: private DNS suffix is required")
	}
	return newClient(baseURL, targetPolicy{privateSuffix: suffix}, ResponseLimits{})
}

// NewPrivateHTTPSWithLimits builds a private-HTTPS client with response-header guards.
func NewPrivateHTTPSWithLimits(baseURL, privateSuffix string, limits ResponseLimits) (*Client, error) {
	if err := validateResponseLimits(limits); err != nil {
		return nil, err
	}
	suffix := privateHostSuffix(privateSuffix)
	if suffix == "" {
		return nil, errors.New("build outbound HTTP client: private DNS suffix is required")
	}
	return newClient(baseURL, targetPolicy{privateSuffix: suffix}, limits)
}

func validateResponseLimits(limits ResponseLimits) error {
	if limits.ResponseHeaderTimeout <= 0 || limits.MaxResponseHeaderBytes <= 0 {
		return errors.New("build outbound HTTP client: response limits must be positive")
	}
	return nil
}

func newClient(rawBaseURL string, policy targetPolicy, limits ResponseLimits) (*Client, error) {
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
	if limits.ResponseHeaderTimeout > 0 {
		transport.ResponseHeaderTimeout = limits.ResponseHeaderTimeout
	}
	if limits.MaxResponseHeaderBytes > 0 {
		transport.MaxResponseHeaderBytes = limits.MaxResponseHeaderBytes
	}
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

func loopbackTransport(client *Client) *http.Transport {
	sanitizer, ok := client.httpClient.Transport.(propagationSanitizer)
	if !ok {
		return nil
	}
	authority, ok := sanitizer.base.(authorityTransport)
	if !ok {
		return nil
	}
	transport, ok := authority.base.(*http.Transport)
	if !ok {
		return nil
	}
	return transport
}

// BindLoopbackTLS points dial and TLS roots at a loopback test server. It does
// not change Proxy or CheckRedirect. In-repo provider tests use this; production
// callers do not.
func BindLoopbackTLS(client *Client, source *http.Transport) {
	transport := loopbackTransport(client)
	if transport == nil || source == nil {
		return
	}
	transport.TLSClientConfig = source.TLSClientConfig.Clone()
	transport.DialContext = source.DialContext
	transport.ForceAttemptHTTP2 = source.ForceAttemptHTTP2
}

// ProxyDisabled reports whether the fixed-authority transport has proxy use
// disabled. In-repo provider tests use this; production callers do not.
func ProxyDisabled(client *Client) bool {
	transport := loopbackTransport(client)
	return transport != nil && transport.Proxy == nil
}

// RejectLoopbackTLSTrust drops test-server roots so a loopback TLS handshake
// fails closed. In-repo provider tests use this; production callers do not.
func RejectLoopbackTLSTrust(client *Client) {
	transport := loopbackTransport(client)
	if transport == nil || transport.TLSClientConfig == nil {
		return
	}
	transport.TLSClientConfig.RootCAs = nil
	transport.TLSClientConfig.InsecureSkipVerify = false
}
