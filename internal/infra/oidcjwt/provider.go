package oidcjwt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/authntrust"
	"github.com/example/go-service-template-rest/internal/infra/httpclient"
	"go.opentelemetry.io/otel/metric"
)

type requestClient interface {
	Do(request *http.Request) (*http.Response, error)
}

type providerClient struct {
	request requestClient
	close   func()
}

type clientFactory func(string) (providerClient, error)

type discoveryDocument struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`
}

func productionClientFactory(provider metric.MeterProvider) clientFactory {
	return func(baseURL string) (providerClient, error) {
		client, err := httpclient.New(providerHTTPConfig(baseURL), provider)
		if err != nil {
			return providerClient{}, errors.New("build OIDC provider client")
		}
		return providerClient{
			request: client,
			close:   client.CloseIdleConnections,
		}, nil
	}
}

func providerHTTPConfig(baseURL string) httpclient.Config {
	return httpclient.Config{
		DependencyName:         "oidc",
		BaseURL:                baseURL,
		TargetClass:            httpclient.ExternalHTTPS,
		DisableInstrumentation: true,
		RequestTimeout:         ProviderTimeout,
		ResponseHeaderTimeout:  ProviderTimeout,
		MaxResponseHeaderBytes: providerHeaderLimit,
		MaxResponseBodyBytes:   MaxProviderBody,
		MaxConnsPerHost:        1,
		MaxIdleConnsPerHost:    1,
	}
}

// establishedTrust is what a successful startup hands the Verifier: the first
// completely validated key set, the endpoint to refresh it from, and the client
// that owns the connection to that endpoint. They travel as one value because the
// Verifier cannot use any of them without the other two.
type establishedTrust struct {
	keys    *keySet
	jwksURI string
	client  providerClient
}

func bootstrapTrust(
	ctx context.Context,
	policy Policy,
	factory clientFactory,
	now func() time.Time,
	log *slog.Logger,
) (establishedTrust, error) {
	jwksURI, err := discoverJWKSURI(ctx, policy, factory)
	if err != nil {
		return establishedTrust{}, err
	}

	jwksURL, err := url.Parse(jwksURI)
	if err != nil || jwksURL == nil {
		return establishedTrust{}, errors.New("OIDC startup failed at JWKS URL validation")
	}
	jwksAuthority := (&url.URL{Scheme: jwksURL.Scheme, Host: jwksURL.Host}).String()
	jwksClient, err := factory(jwksAuthority)
	if err != nil {
		return establishedTrust{}, errors.New("OIDC startup failed at JWKS client")
	}
	// The JWKS client is the one thing this function creates and does not own on
	// success, so its release is deferred behind the handover rather than repeated
	// at each remaining failure. A step added below is then covered by default;
	// closing by hand would make a forgotten close a leaked connection pool on a
	// path only a provider outage reaches.
	handedOver := false
	defer func() {
		if !handedOver {
			jwksClient.close()
		}
	}()

	jwksBytes, err := fetchDocument(ctx, jwksClient.request, jwksURI)
	if err != nil {
		return establishedTrust{}, fmt.Errorf("OIDC startup failed at JWKS: %w", err)
	}
	keys, err := parseKeySet(jwksBytes, now())
	if err != nil {
		return establishedTrust{}, fmt.Errorf(
			"OIDC startup failed at JWKS validation: %w",
			errProviderInvalidDocument,
		)
	}
	log.InfoContext(ctx, "authn_trust_initialized", "component", "authn", "result", "success")
	handedOver = true
	return establishedTrust{keys: keys, jwksURI: jwksURI, client: jwksClient}, nil
}

// discoverJWKSURI reads the issuer's Discovery document and returns the JWKS
// endpoint it names.
//
// It owns its provider client outright: that client exists for this one request,
// so an ordinary defer releases it on every path. bootstrapTrust's client is the
// opposite case — it survives a successful return — and keeping the two in
// separate functions is what stops a reader from having to work out which
// discipline the step they are adding falls under.
func discoverJWKSURI(ctx context.Context, policy Policy, factory clientFactory) (string, error) {
	issuerURL, err := url.Parse(policy.issuer)
	if err != nil || issuerURL == nil {
		return "", errors.New("OIDC startup failed at issuer validation")
	}
	authority := &url.URL{Scheme: issuerURL.Scheme, Host: issuerURL.Host}
	discoveryClient, err := factory(authority.String())
	if err != nil {
		return "", errors.New("OIDC startup failed at discovery client")
	}
	defer discoveryClient.close()

	discoveryURL := *issuerURL
	discoveryURL.Path = strings.TrimSuffix(discoveryURL.Path, "/") + "/.well-known/openid-configuration"
	documentBytes, err := fetchDocument(ctx, discoveryClient.request, discoveryURL.String())
	if err != nil {
		return "", fmt.Errorf("OIDC startup failed at discovery: %w", err)
	}
	var document discoveryDocument
	// The JWKS URI is provider-supplied and this service is about to fetch it.
	// Query parameters are allowed because they may identify the provider's exact
	// endpoint; authority remains pinned separately by the outbound client.
	if err := strictUnmarshal(documentBytes, &document); err != nil ||
		document.Issuer != policy.issuer ||
		!authntrust.ValidJWKSURL(document.JWKSURI) {
		return "", fmt.Errorf(
			"OIDC startup failed at discovery validation: %w",
			errProviderInvalidDocument,
		)
	}
	return document.JWKSURI, nil
}

// providerError is the closed, sanitized outcome of a provider document
// attempt. It preserves the failing phase or bounded HTTP status class for
// startup errors and refresh metrics without carrying a URL, exact status code,
// response body, or transport error text.
type providerError string

const (
	errProviderRequest         providerError = "request"
	errProviderCanceled        providerError = "canceled"
	errProviderTimeout         providerError = "timeout"
	errProviderTransport       providerError = "transport"
	errProviderStatus          providerError = "status"
	errProviderStatus4xx       providerError = "status_4xx"
	errProviderRateLimited     providerError = "rate_limited"
	errProviderStatus5xx       providerError = "status_5xx"
	errProviderBody            providerError = "body"
	errProviderOversize        providerError = "oversize"
	errProviderInvalidDocument providerError = "invalid_document"
	errProviderPanic           providerError = "panic"
)

func (f providerError) Error() string {
	return "provider document failure: " + string(f)
}

func providerFailureReason(err error) string {
	failure, ok := errors.AsType[providerError](err)
	if !ok {
		return "unknown"
	}
	return string(failure)
}

func classifyProviderError(ctx context.Context, err error, fallback providerError) error {
	// Only the parent context's cause belongs to our caller. The request also has
	// ProviderTimeout, and publishing that internal deadline through errors.Is
	// makes transports report a provider outage as if the caller's budget expired.
	if ctxErr := ctx.Err(); ctxErr != nil {
		switch {
		case errors.Is(ctxErr, context.Canceled):
			return fmt.Errorf("%w: %w", errProviderCanceled, ctxErr)
		case errors.Is(ctxErr, context.DeadlineExceeded):
			return fmt.Errorf("%w: %w", errProviderTimeout, ctxErr)
		}
	}
	if errors.Is(err, context.Canceled) {
		return errProviderCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errProviderTimeout
	}
	return fallback
}

// fetchDocument GETs one provider document under a bounded timeout. Its error is
// one of the closed providerError phases above.
func fetchDocument(ctx context.Context, client requestClient, target string) ([]byte, error) {
	requestCtx, cancel := context.WithTimeout(ctx, ProviderTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, target, http.NoBody)
	if err != nil {
		return nil, errProviderRequest
	}
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil, classifyProviderError(ctx, err, errProviderTransport)
	}
	defer func() {
		_ = response.Body.Close()
	}()
	if response.StatusCode != http.StatusOK {
		switch {
		case response.StatusCode == http.StatusTooManyRequests:
			return nil, errProviderRateLimited
		case response.StatusCode >= http.StatusBadRequest && response.StatusCode < http.StatusInternalServerError:
			return nil, errProviderStatus4xx
		case response.StatusCode >= http.StatusInternalServerError && response.StatusCode < 600:
			return nil, errProviderStatus5xx
		default:
			return nil, errProviderStatus
		}
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		if _, ok := errors.AsType[*httpclient.ResponseTooLargeError](err); ok {
			return nil, errProviderOversize
		}
		return nil, classifyProviderError(ctx, err, errProviderBody)
	}
	if len(body) > MaxProviderBody {
		return nil, errProviderOversize
	}
	return body, nil
}
