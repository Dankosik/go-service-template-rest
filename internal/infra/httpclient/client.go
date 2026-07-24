// Package httpclient owns the transport-wide safety defaults for outbound HTTP.
// Provider adapters own authentication and must not put credentials in URLs.
package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	// Keep these aligned with the net/http defaults for the pinned Go toolchain.
	// A custom dialer is required so the public-address gate runs after DNS.
	defaultDialTimeout = 30 * time.Second
	defaultKeepAlive   = 30 * time.Second
)

// ErrTargetDenied reports a request or resolved address outside the client's fixed target.
var ErrTargetDenied = errors.New("outbound HTTP target denied")

// TargetClass selects the transport security policy for one fixed provider authority.
type TargetClass uint8

const (
	// ExternalHTTPS permits only HTTPS targets resolving to public addresses.
	ExternalHTTPS TargetClass = iota + 1
	// RailwayPrivateHTTP permits only HTTP targets under railway.internal.
	RailwayPrivateHTTP
)

// Config defines the required safety bounds for one provider authority.
type Config struct {
	DependencyName         string
	BaseURL                string
	TargetClass            TargetClass
	RequestTimeout         time.Duration
	ResponseHeaderTimeout  time.Duration
	MaxResponseHeaderBytes int64
	MaxResponseBodyBytes   int64
}

// Client owns one reusable HTTP client and its underlying connection pool.
type Client struct {
	baseURL    string
	httpClient *http.Client
	transport  *http.Transport
}

// ResponseTooLargeError reports a decoded response body exceeding its configured limit.
type ResponseTooLargeError struct {
	Limit int64
}

func (e *ResponseTooLargeError) Error() string {
	return fmt.Sprintf("outbound HTTP response body exceeds %d bytes", e.Limit)
}

// New builds one instrumented client for a fixed provider authority.
func New(cfg Config, meterProvider metric.MeterProvider) (*Client, error) {
	baseURL, err := validateConfig(cfg)
	if err != nil {
		return nil, err
	}

	baseTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errors.New("build outbound HTTP client: default transport has unexpected type")
	}
	transport := baseTransport.Clone()
	transport.Proxy = nil
	transport.ResponseHeaderTimeout = cfg.ResponseHeaderTimeout
	transport.MaxResponseHeaderBytes = cfg.MaxResponseHeaderBytes
	transport.TLSClientConfig = nil
	transport.DialTLSContext = nil

	if cfg.TargetClass == ExternalHTTPS {
		dialer := &net.Dialer{
			Timeout:        defaultDialTimeout,
			KeepAlive:      defaultKeepAlive,
			ControlContext: enforcePublicDialAddress,
		}
		transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			conn, dialErr := dialer.DialContext(ctx, network, address)
			if errors.Is(dialErr, ErrTargetDenied) {
				return nil, ErrTargetDenied
			}
			if dialErr != nil {
				return nil, fmt.Errorf("dial outbound HTTP target: %w", dialErr)
			}
			return conn, nil
		}
	}

	if meterProvider == nil {
		meterProvider = metricnoop.NewMeterProvider()
	}

	bounded := authorityTransport{
		base:      responseLimitTransport{base: transport, limit: cfg.MaxResponseBodyBytes},
		scheme:    baseURL.Scheme,
		authority: baseURL.Host,
	}
	instrumented := otelhttp.NewTransport(
		bounded,
		otelhttp.WithMeterProvider(meterProvider),
		otelhttp.WithPropagators(propagation.TraceContext{}),
		otelhttp.WithSpanOptions(trace.WithAttributes(
			attribute.String("dependency.name", strings.TrimSpace(cfg.DependencyName)),
		)),
	)

	return &Client{
		baseURL: baseURL.String(),
		httpClient: &http.Client{
			Transport: instrumented,
			Timeout:   cfg.RequestTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		transport: transport,
	}, nil
}

// Do sends a request through the client's fixed target and resource bounds.
func (c *Client) Do(request *http.Request) (*http.Response, error) {
	// #nosec G704 -- authorityTransport rejects requests outside the configured scheme and authority before dialing.
	response, err := c.httpClient.Do(request)
	if err != nil {
		return response, fmt.Errorf("send outbound HTTP request: %w", err)
	}
	return response, nil
}

// BaseURL returns the validated immutable provider base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// CloseIdleConnections closes this client's idle connection pool.
func (c *Client) CloseIdleConnections() {
	c.transport.CloseIdleConnections()
}

func validateConfig(cfg Config) (*url.URL, error) {
	if strings.TrimSpace(cfg.DependencyName) == "" {
		return nil, errors.New("build outbound HTTP client: dependency name is required")
	}
	if err := validateBounds(cfg); err != nil {
		return nil, err
	}

	baseURL, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
	if err != nil {
		return nil, errors.New("build outbound HTTP client: base URL is invalid")
	}
	if !baseURL.IsAbs() || baseURL.Opaque != "" || baseURL.Host == "" || baseURL.Hostname() == "" {
		return nil, errors.New("build outbound HTTP client: base URL must be absolute")
	}
	if baseURL.User != nil || baseURL.RawQuery != "" || baseURL.ForceQuery || baseURL.Fragment != "" {
		return nil, errors.New("build outbound HTTP client: base URL cannot contain user info, query, or fragment")
	}
	if err := validateTarget(baseURL, cfg.TargetClass); err != nil {
		return nil, err
	}

	baseURL.Scheme = strings.ToLower(baseURL.Scheme)
	baseURL.Host = strings.ToLower(baseURL.Host)
	return baseURL, nil
}

func validateBounds(cfg Config) error {
	if cfg.RequestTimeout <= 0 {
		return errors.New("build outbound HTTP client: request timeout must be positive")
	}
	if cfg.ResponseHeaderTimeout <= 0 {
		return errors.New("build outbound HTTP client: response header timeout must be positive")
	}
	if cfg.MaxResponseHeaderBytes <= 0 {
		return errors.New("build outbound HTTP client: response header limit must be positive")
	}
	if cfg.MaxResponseBodyBytes <= 0 {
		return errors.New("build outbound HTTP client: response body limit must be positive")
	}
	return nil
}

func validateTarget(baseURL *url.URL, targetClass TargetClass) error {
	switch targetClass {
	case ExternalHTTPS:
		if !strings.EqualFold(baseURL.Scheme, "https") {
			return errors.New("build outbound HTTP client: external target requires HTTPS")
		}
		if address, parseErr := netip.ParseAddr(baseURL.Hostname()); parseErr == nil && isForbiddenExternalAddress(address) {
			return ErrTargetDenied
		}
	case RailwayPrivateHTTP:
		if !strings.EqualFold(baseURL.Scheme, "http") {
			return errors.New("build outbound HTTP client: Railway private target requires HTTP")
		}
		hostname := strings.ToLower(strings.TrimSuffix(baseURL.Hostname(), "."))
		if !strings.HasSuffix(hostname, ".railway.internal") {
			return errors.New("build outbound HTTP client: Railway private target requires a railway.internal hostname")
		}
	default:
		return errors.New("build outbound HTTP client: target class is invalid")
	}
	return nil
}

type authorityTransport struct {
	base      http.RoundTripper
	scheme    string
	authority string
}

func (t authorityTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil || req.URL == nil ||
		req.URL.User != nil ||
		!strings.EqualFold(req.URL.Scheme, t.scheme) ||
		!strings.EqualFold(req.URL.Host, t.authority) {
		return nil, ErrTargetDenied
	}
	response, err := t.base.RoundTrip(req)
	if err != nil {
		return response, fmt.Errorf("send outbound HTTP request: %w", err)
	}
	return response, nil
}

type responseLimitTransport struct {
	base  http.RoundTripper
	limit int64
}

func (t responseLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(req)
	if response != nil && response.Body != nil {
		response.Body = &responseBody{
			ReadCloser: http.MaxBytesReader(nil, response.Body, t.limit),
			limit:      t.limit,
		}
	}
	if err != nil {
		return response, fmt.Errorf("receive outbound HTTP response: %w", err)
	}
	return response, nil
}

type responseBody struct {
	io.ReadCloser
	limit int64
}

func (b *responseBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		_ = b.Close()
		return n, &ResponseTooLargeError{Limit: b.limit}
	}
	if err == nil {
		return n, nil
	}
	if errors.Is(err, io.EOF) {
		return n, io.EOF
	}
	return n, fmt.Errorf("read outbound HTTP response body: %w", err)
}

func enforcePublicDialAddress(_ context.Context, _, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return ErrTargetDenied
	}
	resolved, err := netip.ParseAddr(host)
	if err != nil || isForbiddenExternalAddress(resolved) {
		return ErrTargetDenied
	}
	return nil
}

func isForbiddenExternalAddress(address netip.Addr) bool {
	address = address.Unmap()
	return !address.IsValid() ||
		!address.IsGlobalUnicast() ||
		address.IsPrivate() ||
		address.IsLoopback() ||
		address.IsLinkLocalUnicast() ||
		address.IsMulticast() ||
		address.IsUnspecified()
}
