// Package httpclient owns the transport-wide safety defaults for outbound HTTP.
// Provider adapters own authentication and must not put credentials in URLs.
//
// The transport deliberately ignores HTTP_PROXY/HTTPS_PROXY: a proxy resolves and
// dials on the client's behalf, bypassing the post-DNS public-address gate that
// makes the fixed-authority guarantee meaningful. A service that must egress
// through a mandatory proxy uses a plain net/http client instead.
package httpclient

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"

	"github.com/example/go-service-template-rest/internal/observability/correlationpolicy"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

const (
	// Keep these aligned with the net/http defaults for the pinned Go toolchain.
	// A custom dialer is required so the public-address gate runs after DNS.
	defaultDialTimeout = 30 * time.Second
	defaultKeepAlive   = 30 * time.Second
)

// Client owns one reusable HTTP client and its underlying connection pool.
type Client struct {
	baseURL    string
	httpClient *http.Client
	transport  *http.Transport
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
	// Both are set explicitly, because the clone carries net/http's defaults for
	// a general-purpose client and this one is pinned to a single authority. See
	// Config.MaxConnsPerHost and Config.MaxIdleConnsPerHost.
	transport.MaxConnsPerHost = cfg.MaxConnsPerHost
	transport.MaxIdleConnsPerHost = cmp.Or(cfg.MaxIdleConnsPerHost, cfg.MaxConnsPerHost)

	dialer := &net.Dialer{
		Timeout:   defaultDialTimeout,
		KeepAlive: defaultKeepAlive,
		ControlContext: func(_ context.Context, _, address string, _ syscall.RawConn) error {
			return enforceDialAddress(cfg.TargetClass, address)
		},
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

	if meterProvider == nil {
		meterProvider = metricnoop.NewMeterProvider()
	}

	bounded := authorityTransport{
		base:      responseLimitTransport{base: transport, limit: cfg.MaxResponseBodyBytes},
		scheme:    baseURL.Scheme,
		authority: baseURL.Host,
	}
	// Instrumentation sits inside retries so each attempt is its own span and
	// metric sample. The fixed-authority and response-size bounds remain innermost
	// and therefore apply to every attempt.
	var instrumented http.RoundTripper = otelhttp.NewTransport(
		bounded,
		otelhttp.WithMeterProvider(meterProvider),
		otelhttp.WithPropagators(correlationpolicy.NewPropagator(cfg.Propagation, requestIDHeader)),
		otelhttp.WithSpanOptions(trace.WithAttributes(
			attribute.String("dependency.name", strings.TrimSpace(cfg.DependencyName)),
		)),
	)
	sanitized := propagationSanitizer{base: instrumented}
	var roundTripper http.RoundTripper = sanitized
	// profile:authn-oidc-jwt:start
	if cfg.DisableInstrumentation {
		roundTripper = bounded
	}
	// profile:authn-oidc-jwt:end
	if cfg.Retry.enabled() {
		roundTripper = retryTransport{base: roundTripper, policy: cfg.Retry}
	}

	return &Client{
		baseURL: baseURL.String(),
		httpClient: &http.Client{
			Transport: roundTripper,
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
	if hasBlankIdempotencyKey(request) {
		return nil, errors.New("send outbound HTTP request: Idempotency-Key must not be blank")
	}
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
