// Package httpclient builds one fixed-authority HTTP client per dependency.
// Provider adapters own authentication, operation budgets, response handling,
// retries, errors, telemetry, and lifecycle wiring.
package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"syscall"
	"time"
)

const (
	defaultDialTimeout = 30 * time.Second
	defaultKeepAlive   = 30 * time.Second
)

var (
	// ErrSaturated means the dependency request limit was full while the caller was still live.
	ErrSaturated = errors.New("outbound HTTP dependency is saturated")
	// ErrResponseTooLarge means the response exceeded its admitted decoded-body limit.
	ErrResponseTooLarge = errors.New("outbound HTTP response is too large")
	// ErrOperationTimeout means an operation-specific deadline expired before its parent context.
	ErrOperationTimeout = errors.New("outbound HTTP operation timed out")
)

// Client is a reusable fixed-authority HTTP request Doer.
type Client struct {
	baseURL           string
	httpClient        *http.Client
	requests          chan struct{}
	absoluteBodyBytes int64
}

// TransportLimits are mandatory provider-wide safety ceilings.
type TransportLimits struct {
	ResponseHeaderTimeout  time.Duration
	MaxResponseHeaderBytes int64
	MaxInFlight            int
	AbsoluteBodyBytes      int64
}

// OperationPolicy narrows one non-streaming operation below its transport ceilings.
type OperationPolicy struct {
	Timeout      time.Duration
	MaxBodyBytes int64
}

// NewExternalHTTPS builds a bounded client for one public HTTPS authority.
func NewExternalHTTPS(baseURL string, limits TransportLimits) (*Client, error) {
	if err := validateTransportLimits(limits); err != nil {
		return nil, err
	}
	return newClient(baseURL, targetPolicy{}, limits)
}

// NewPrivateHTTPS builds a bounded client for one HTTPS authority under privateSuffix.
func NewPrivateHTTPS(baseURL, privateSuffix string, limits TransportLimits) (*Client, error) {
	if err := validateTransportLimits(limits); err != nil {
		return nil, err
	}
	suffix := privateHostSuffix(privateSuffix)
	if suffix == "" {
		return nil, errors.New("build outbound HTTP client: private DNS suffix is required")
	}
	return newClient(baseURL, targetPolicy{privateSuffix: suffix}, limits)
}

func validateTransportLimits(limits TransportLimits) error {
	if limits.ResponseHeaderTimeout <= 0 || limits.MaxResponseHeaderBytes <= 0 ||
		limits.MaxInFlight <= 0 || limits.AbsoluteBodyBytes <= 0 {
		return errors.New("build outbound HTTP client: transport limits must be positive")
	}
	return nil
}

func newClient(rawBaseURL string, policy targetPolicy, limits TransportLimits) (*Client, error) {
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
	transport.ResponseHeaderTimeout = limits.ResponseHeaderTimeout
	transport.MaxResponseHeaderBytes = limits.MaxResponseHeaderBytes
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
		baseURL:           baseURL.String(),
		requests:          make(chan struct{}, limits.MaxInFlight),
		absoluteBodyBytes: limits.AbsoluteBodyBytes,
		httpClient: &http.Client{
			Transport: roundTripper,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

// Do sends one non-streaming request under the provider-wide limits.
func (c *Client) Do(request *http.Request) (*http.Response, error) {
	return c.do(request, c.absoluteBodyBytes, nil, func() error {
		if request == nil {
			return nil
		}
		return request.Context().Err()
	})
}

// DoWithPolicy sends one non-streaming request under a smaller operation budget.
func (c *Client) DoWithPolicy(request *http.Request, policy OperationPolicy) (*http.Response, error) {
	if request == nil || request.URL == nil {
		return nil, errors.New("send outbound HTTP request: request URL is required")
	}
	if policy.Timeout <= 0 || policy.MaxBodyBytes <= 0 || policy.MaxBodyBytes > c.absoluteBodyBytes {
		return nil, errors.New("send outbound HTTP request: operation policy must be positive and within transport limits")
	}

	operationCtx, cancel := context.WithTimeoutCause(request.Context(), policy.Timeout, ErrOperationTimeout)
	attempt := request.Clone(operationCtx)
	return c.do(attempt, policy.MaxBodyBytes, cancel, func() error {
		if err := request.Context().Err(); err != nil {
			return fmt.Errorf("parent request: %w", err)
		}
		if errors.Is(context.Cause(operationCtx), ErrOperationTimeout) {
			return ErrOperationTimeout
		}
		return nil
	})
}

func (c *Client) do(request *http.Request, maxBodyBytes int64, done func(), contextError func() error) (*http.Response, error) {
	if request == nil || request.URL == nil {
		if done != nil {
			done()
		}
		return nil, errors.New("send outbound HTTP request: request URL is required")
	}
	if err := request.Context().Err(); err != nil {
		if done != nil {
			done()
		}
		return nil, fmt.Errorf("send outbound HTTP request: %w", err)
	}
	select {
	case c.requests <- struct{}{}:
	default:
		if done != nil {
			done()
		}
		if err := request.Context().Err(); err != nil {
			return nil, fmt.Errorf("send outbound HTTP request: %w", err)
		}
		return nil, fmt.Errorf("send outbound HTTP request: %w", ErrSaturated)
	}

	var once sync.Once
	finish := func() {
		once.Do(func() {
			<-c.requests
			if done != nil {
				done()
			}
		})
	}

	// #nosec G704 -- authorityTransport pins scheme/authority and the dialer checks the resolved address.
	response, err := c.httpClient.Do(request)
	if err != nil {
		finish()
		if contextErr := contextError(); contextErr != nil {
			err = contextErr
		}
		return response, fmt.Errorf("send outbound HTTP request: %w", err)
	}
	if response.ContentLength > maxBodyBytes {
		_ = response.Body.Close()
		finish()
		return nil, fmt.Errorf("send outbound HTTP request: %w", ErrResponseTooLarge)
	}
	response.Body = &boundedBody{
		body:         response.Body,
		remaining:    maxBodyBytes,
		complete:     finish,
		contextError: contextError,
	}
	return response, nil
}

type boundedBody struct {
	body         io.ReadCloser
	remaining    int64
	tooLarge     bool
	complete     func()
	contextError func() error
}

func (b *boundedBody) Read(buffer []byte) (int, error) {
	if b.tooLarge {
		return 0, ErrResponseTooLarge
	}
	if b.remaining < int64(len(buffer)) {
		buffer = buffer[:int(b.remaining)+1]
	}
	n, err := b.body.Read(buffer)
	if int64(n) > b.remaining {
		n = int(b.remaining)
		b.remaining = 0
		b.tooLarge = true
		b.complete()
		return n, ErrResponseTooLarge
	}
	b.remaining -= int64(n)
	if err != nil {
		b.complete()
		if contextErr := b.contextError(); contextErr != nil {
			err = contextErr
		}
	}
	return n, err
}

func (b *boundedBody) Close() error {
	err := b.body.Close()
	b.complete()
	if err != nil {
		return fmt.Errorf("close outbound HTTP response body: %w", err)
	}
	return nil
}

// BaseURL returns the validated provider base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// CloseIdleConnections closes this client's idle connection pool.
func (c *Client) CloseIdleConnections() {
	c.httpClient.CloseIdleConnections()
}
