// Package httpclient owns the transport-wide safety defaults for outbound HTTP.
// Provider adapters own authentication and must not put credentials in URLs.
//
// The transport deliberately ignores HTTP_PROXY/HTTPS_PROXY: a proxy resolves
// and dials on the client's behalf, which would bypass the post-DNS
// public-address gate that makes the fixed-authority guarantee meaningful. A
// service that must egress through a mandatory proxy uses a plain net/http
// client in its provider adapter instead.
package httpclient

import (
	"cmp"
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

var (
	// netip.Addr.IsGlobalUnicast intentionally follows the IP protocol's broad
	// definition, not IANA's globally-reachable registry. These are the
	// special-purpose ranges it would otherwise admit.
	nonPublicIPv4Prefixes = [...]netip.Prefix{
		netip.MustParsePrefix("100.64.0.0/10"),
		netip.MustParsePrefix("192.0.0.0/24"),
		netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("192.88.99.0/24"),
		netip.MustParsePrefix("198.18.0.0/15"),
		netip.MustParsePrefix("198.51.100.0/24"),
		netip.MustParsePrefix("203.0.113.0/24"),
		netip.MustParsePrefix("240.0.0.0/4"),
	}
	globallyReachableIPv4SpecialPrefixes = [...]netip.Prefix{
		netip.MustParsePrefix("192.0.0.9/32"),
		netip.MustParsePrefix("192.0.0.10/32"),
	}
	allocatedGlobalIPv6Prefix = netip.MustParsePrefix("2000::/3")
	publicNAT64Prefix         = netip.MustParsePrefix("64:ff9b::/96")
	nonPublicIPv6Prefixes     = [...]netip.Prefix{
		netip.MustParsePrefix("2001::/23"),
		netip.MustParsePrefix("2001:db8::/32"),
		netip.MustParsePrefix("2002::/16"),
		netip.MustParsePrefix("3fff::/20"),
	}
	globallyReachableIPv6SpecialPrefixes = [...]netip.Prefix{
		netip.MustParsePrefix("2001:1::1/128"),
		netip.MustParsePrefix("2001:1::2/128"),
		netip.MustParsePrefix("2001:1::3/128"),
		netip.MustParsePrefix("2001:3::/32"),
		netip.MustParsePrefix("2001:4:112::/48"),
		netip.MustParsePrefix("2001:20::/28"),
		netip.MustParsePrefix("2001:30::/28"),
	}
)

// TargetClass selects the transport security policy for one fixed provider authority.
type TargetClass uint8

const (
	// ExternalHTTPS permits only HTTPS targets resolving to public addresses.
	ExternalHTTPS TargetClass = iota + 1
	// PrivateHTTP permits only plaintext HTTP targets under the platform's
	// private DNS zone, named by Config.PrivateHostSuffix. Transport security
	// is the platform's private network, not TLS.
	PrivateHTTP
)

// There is deliberately no default private DNS zone. A platform-specific
// default in an otherwise platform-neutral package is how a "generic" component
// quietly becomes single-platform: it silently succeeds on the platform it was
// written for and fails with a confusing message everywhere else. The caller
// names its own zone, for example ".railway.internal", ".svc.cluster.local", or
// ".internal".

// Config defines the required safety bounds for one provider authority.
type Config struct {
	DependencyName string
	BaseURL        string
	TargetClass    TargetClass
	// profile:authn-oidc-jwt:start
	// DisableInstrumentation prevents fixed identity-provider endpoints and
	// credential-adjacent request metadata from entering the general outbound
	// trace path. It is reserved for the OIDC trust bootstrap.
	DisableInstrumentation bool
	// profile:authn-oidc-jwt:end
	// PrivateHostSuffix is the required hostname suffix for PrivateHTTP
	// targets, and is required for them. It is ignored for ExternalHTTPS.
	PrivateHostSuffix      string
	RequestTimeout         time.Duration
	ResponseHeaderTimeout  time.Duration
	MaxResponseHeaderBytes int64
	MaxResponseBodyBytes   int64
	// MaxConnsPerHost bounds concurrent connections to the fixed authority, and
	// is the bulkhead between one slow provider and the rest of the service.
	//
	// Nothing else provides it. net/http leaves this unlimited, so a provider
	// that goes slow is answered by every in-flight request opening its own
	// connection and holding it for the whole request budget — the service's
	// entire http.max_in_flight allowance spent waiting on one dependency, with
	// requests that never touch it shed for the capacity. Excess callers wait
	// here instead, bounded by the request budget they already carry.
	MaxConnsPerHost int
	// MaxIdleConnsPerHost is how many connections survive between bursts. Empty
	// follows MaxConnsPerHost, which is the right answer for a client pinned to
	// one authority.
	//
	// It has to be set. net/http's default is DefaultMaxIdleConnsPerHost, which
	// is 2 — correct for a general-purpose client spread across many hosts, and
	// badly wrong here: a burst of fifty calls opens fifty connections and keeps
	// two, so the next burst pays forty-eight TCP and TLS handshakes against the
	// callers' request budgets. That reads on a dashboard as the provider having
	// gotten slower.
	MaxIdleConnsPerHost int
	// Retry bounds repeat attempts, and is disabled by its zero value. A provider's
	// rolling deploy resets in-flight connections for a few seconds, and without a
	// policy here every service writes its own — usually fixed-delay, for every
	// method, with no attempt cap tied to the caller's remaining budget. See
	// retryTransport for the three rules that make a repeat safe.
	Retry RetryPolicy
	// Propagation is the immutable correlation-disclosure policy for this
	// target. Its zero value emits nothing remotely.
	Propagation PropagationPolicy
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
		otelhttp.WithPropagators(policyPropagator{policy: cfg.Propagation}),
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

func validateConfig(cfg Config) (*url.URL, error) {
	if !cfg.Propagation.valid() {
		return nil, errors.New("build outbound HTTP client: propagation policy is invalid")
	}
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
	if err := validateTarget(baseURL, cfg.TargetClass, privateHostSuffix(cfg.PrivateHostSuffix)); err != nil {
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
	// Required rather than defaulted, for the same reason every other bound here
	// is: the value that would look harmless is net/http's unlimited one, and a
	// missing bulkhead is invisible until the provider it was meant to contain
	// takes the service down with it.
	if cfg.MaxConnsPerHost <= 0 {
		return errors.New("build outbound HTTP client: max conns per host must be positive")
	}
	if cfg.MaxIdleConnsPerHost < 0 {
		return errors.New("build outbound HTTP client: max idle conns per host must be >= 0")
	}
	if cfg.MaxIdleConnsPerHost > cfg.MaxConnsPerHost {
		return errors.New("build outbound HTTP client: max idle conns per host must be <= max conns per host")
	}
	if cfg.Retry.MaxAttempts < 0 {
		return errors.New("build outbound HTTP client: retry max attempts must be >= 0")
	}
	if cfg.Retry.BaseDelay < 0 {
		return errors.New("build outbound HTTP client: retry base delay must be >= 0")
	}
	// A policy that names attempts without a delay would retry immediately, which
	// is the shape that turns one failure into a burst.
	if cfg.Retry.MaxAttempts > 1 && cfg.Retry.BaseDelay <= 0 {
		return errors.New("build outbound HTTP client: retry base delay is required when max attempts is > 1")
	}
	return nil
}

// privateHostSuffix normalizes the configured private DNS zone. An empty zone
// stays empty so validateTarget rejects it.
func privateHostSuffix(configured string) string {
	suffix := strings.ToLower(strings.TrimSpace(configured))
	if suffix == "" {
		return ""
	}
	if !strings.HasPrefix(suffix, ".") {
		suffix = "." + suffix
	}
	return strings.TrimSuffix(suffix, ".")
}

func validateTarget(baseURL *url.URL, targetClass TargetClass, requiredPrivateSuffix string) error {
	switch targetClass {
	case ExternalHTTPS:
		if !strings.EqualFold(baseURL.Scheme, "https") {
			return errors.New("build outbound HTTP client: external target requires HTTPS")
		}
		if address, parseErr := netip.ParseAddr(baseURL.Hostname()); parseErr == nil && isForbiddenExternalAddress(address) {
			return ErrTargetDenied
		}
	case PrivateHTTP:
		if requiredPrivateSuffix == "." || requiredPrivateSuffix == "" {
			return errors.New(
				"build outbound HTTP client: private target requires Config.PrivateHostSuffix, " +
					"the deployment platform's private DNS zone",
			)
		}
		if !strings.EqualFold(baseURL.Scheme, "http") {
			return errors.New("build outbound HTTP client: private target requires HTTP")
		}
		hostname := strings.ToLower(strings.TrimSuffix(baseURL.Hostname(), "."))
		if !strings.HasSuffix(hostname, requiredPrivateSuffix) {
			return fmt.Errorf(
				"build outbound HTTP client: private target requires a %s hostname",
				strings.TrimPrefix(requiredPrivateSuffix, "."),
			)
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
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
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

func enforceDialAddress(targetClass TargetClass, address string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return ErrTargetDenied
	}
	resolved, err := netip.ParseAddr(host)
	if err != nil {
		return ErrTargetDenied
	}
	switch targetClass {
	case ExternalHTTPS:
		if !isForbiddenExternalAddress(resolved) {
			return nil
		}
	case PrivateHTTP:
		if resolved.Unmap().IsPrivate() {
			return nil
		}
	}
	return ErrTargetDenied
}

func isForbiddenExternalAddress(address netip.Addr) bool {
	address = address.Unmap()
	if !address.IsValid() ||
		!address.IsGlobalUnicast() ||
		address.IsPrivate() ||
		address.IsLoopback() ||
		address.IsLinkLocalUnicast() ||
		address.IsMulticast() ||
		address.IsUnspecified() {
		return true
	}

	if address.Is4() {
		for _, prefix := range globallyReachableIPv4SpecialPrefixes {
			if prefix.Contains(address) {
				return false
			}
		}
		for _, prefix := range nonPublicIPv4Prefixes {
			if prefix.Contains(address) {
				return true
			}
		}
		return false
	}

	if publicNAT64Prefix.Contains(address) {
		bits := address.As16()
		return isForbiddenExternalAddress(netip.AddrFrom4([4]byte{
			bits[12], bits[13], bits[14], bits[15],
		}))
	}
	if !allocatedGlobalIPv6Prefix.Contains(address) {
		return true
	}
	for _, prefix := range globallyReachableIPv6SpecialPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	for _, prefix := range nonPublicIPv6Prefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}
