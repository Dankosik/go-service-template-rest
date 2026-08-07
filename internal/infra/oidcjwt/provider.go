package oidcjwt

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

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
		return establishedTrust{}, errors.New("OIDC startup failed at JWKS")
	}
	keys, err := parseKeySet(jwksBytes, now())
	if err != nil {
		return establishedTrust{}, errors.New("OIDC startup failed at JWKS validation")
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
		return "", errors.New("OIDC startup failed at discovery")
	}
	var document discoveryDocument
	if err := strictUnmarshal(documentBytes, &document); err != nil ||
		document.Issuer != policy.issuer ||
		!validProviderURL(document.JWKSURI) {
		return "", errors.New("OIDC startup failed at discovery validation")
	}
	return document.JWKSURI, nil
}

// errProviderFetchFailed is the entire result of a failed provider request:
// every caller reads it as a boolean and none reads its text. A transport error,
// a non-200 status, and an oversized body are one answer here — the document was
// not obtained — and which of them it was may not travel any further.
//
// This comment owns that rule for the package. It is a redaction rule rather
// than an economy. A provider's error string and response body are exactly what
// must not reach a log or a returned error, so naming the failing step is the
// caller's to supply from what it already knows: bootstrapTrust and
// discoverJWKSURI name the startup stage, because an operator reads that instead
// of a running service, and fetchAndInstall answers with errRefreshFailed.
// TestErrorsAndLogsRedactCredentialAndProviderContent fails if this leaks, and
// docs/authentication.md publishes the same promise.
var errProviderFetchFailed = errors.New("provider document fetch failed")

// fetchDocument GETs one provider document under a bounded timeout. Its error is
// a boolean; errProviderFetchFailed owns why, and a failure mode added here
// wants that same value rather than a message of its own.
func fetchDocument(ctx context.Context, client requestClient, target string) ([]byte, error) {
	requestCtx, cancel := context.WithTimeout(ctx, ProviderTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, target, nil)
	if err != nil {
		return nil, errProviderFetchFailed
	}
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil, errProviderFetchFailed
	}
	defer func() {
		_ = response.Body.Close()
	}()
	if response.StatusCode != http.StatusOK {
		return nil, errProviderFetchFailed
	}
	body, err := io.ReadAll(response.Body)
	if err != nil || len(body) > MaxProviderBody {
		return nil, errProviderFetchFailed
	}
	return body, nil
}

// validProviderURL reports whether raw is a URL this package may fetch from. It
// owns that shape for the package, applied to the configured issuer by
// [NewPolicy] and to the JWKS URI the provider's own Discovery document names.
// Requiring absolute HTTPS with no user info, query, or fragment keeps a
// provider-supplied endpoint from carrying credentials or redirect parameters
// into a request this service makes on its own behalf.
//
// internal/config carries a second copy of this predicate, validAuthnIssuerURL,
// so a bad authn.issuer fails at configuration load rather than at authn startup.
// That copy is forced: the depguard rule config_no_runtime_owners stops
// internal/config from importing anything under internal/, so it cannot call this.
// It is written in this same shape, term for term and in the same order, so the
// two can be read against each other line by line.
// A constraint tightened here has to be tightened there too, or the two answers
// disagree about which issuer values a deployment may hold.
// TestPolicyRulesMatchConfigValidation makes that disagreement fail a build
// rather than a deployment, and owns what it can and cannot compare. This comment
// is the code-side owner of that arrangement; the other sites point here rather
// than restating it.
func validProviderURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	return err == nil &&
		parsed != nil &&
		parsed.IsAbs() &&
		strings.EqualFold(parsed.Scheme, "https") &&
		parsed.Host != "" &&
		parsed.Hostname() != "" &&
		parsed.Opaque == "" &&
		parsed.User == nil &&
		parsed.RawQuery == "" &&
		!parsed.ForceQuery &&
		parsed.Fragment == ""
}
