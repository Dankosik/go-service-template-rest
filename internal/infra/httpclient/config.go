package httpclient

import (
	"crypto/x509"
	"errors"
	"net/url"
	"strings"
	"time"
)

// Config defines the required safety bounds for one provider authority.
type Config struct {
	DependencyName string
	BaseURL        string
	TargetClass    TargetClass
	// OneAttempt sends each request on a new HTTP/1 connection without
	// transparent compression, so net/http cannot replay a request from its idle
	// pool or transform response bytes.
	OneAttempt bool
	// RootCAs is a caller-owned immutable trust snapshot. Nil preserves the
	// process system-root behavior used by existing callers.
	RootCAs *x509.CertPool
	// profile:credential-provider-http:start
	// DisableInstrumentation prevents fixed targets and credential-adjacent
	// request metadata from entering the general outbound trace path.
	DisableInstrumentation bool
	// profile:credential-provider-http:end
	// PrivateHostSuffix is the required hostname suffix for PrivateHTTP and
	// PrivateHTTPS targets. It is ignored for ExternalHTTPS.
	PrivateHostSuffix      string
	RequestTimeout         time.Duration
	ResponseHeaderTimeout  time.Duration
	MaxResponseHeaderBytes int64
	MaxResponseBodyBytes   int64
	// MaxConnsPerHost bounds concurrent connections to the fixed authority, and
	// is the bulkhead between one slow provider and the rest of the service.
	//
	// Nothing else provides it: net/http leaves this unlimited, so a slow
	// provider has every in-flight request open its own connection and hold it
	// for the whole request budget, spending the service's entire
	// http.max_in_flight allowance on one dependency.
	MaxConnsPerHost int
	// MaxIdleConnsPerHost is how many connections survive between bursts. Empty
	// follows MaxConnsPerHost, which is the right answer for a client pinned to
	// one authority.
	//
	// It has to be set: net/http's DefaultMaxIdleConnsPerHost is 2, so a burst of
	// fifty calls opens fifty connections and keeps two, and the next burst pays
	// forty-eight TCP and TLS handshakes against the callers' request budgets.
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

func validateConfig(cfg Config) (*url.URL, error) {
	if !cfg.Propagation.Valid() {
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
	return cfg.Retry.validate()
}
