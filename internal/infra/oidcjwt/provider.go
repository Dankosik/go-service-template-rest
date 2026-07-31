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

func bootstrapTrust(
	ctx context.Context,
	policy Policy,
	factory clientFactory,
	now func() time.Time,
	log *slog.Logger,
) (*keySet, string, providerClient, error) {
	issuerURL, _ := url.Parse(policy.issuer)
	authority := &url.URL{Scheme: issuerURL.Scheme, Host: issuerURL.Host}
	discoveryClient, err := factory(authority.String())
	if err != nil {
		return nil, "", providerClient{}, errors.New("OIDC startup failed at discovery client")
	}
	defer discoveryClient.close()

	discoveryURL := *issuerURL
	discoveryURL.Path = strings.TrimSuffix(discoveryURL.Path, "/") + "/.well-known/openid-configuration"
	documentBytes, err := fetchDocument(ctx, discoveryClient.request, discoveryURL.String())
	if err != nil {
		return nil, "", providerClient{}, errors.New("OIDC startup failed at discovery")
	}
	var document discoveryDocument
	if err := strictUnmarshal(documentBytes, &document); err != nil ||
		document.Issuer != policy.issuer ||
		!validProviderURL(document.JWKSURI) {
		return nil, "", providerClient{}, errors.New("OIDC startup failed at discovery validation")
	}

	jwksURL, _ := url.Parse(document.JWKSURI)
	jwksAuthority := (&url.URL{Scheme: jwksURL.Scheme, Host: jwksURL.Host}).String()
	jwksClient, err := factory(jwksAuthority)
	if err != nil {
		return nil, "", providerClient{}, errors.New("OIDC startup failed at JWKS client")
	}
	jwksBytes, err := fetchDocument(ctx, jwksClient.request, document.JWKSURI)
	if err != nil {
		jwksClient.close()
		return nil, "", providerClient{}, errors.New("OIDC startup failed at JWKS")
	}
	keys, err := parseKeySet(jwksBytes, now())
	if err != nil {
		jwksClient.close()
		return nil, "", providerClient{}, errors.New("OIDC startup failed at JWKS validation")
	}
	if log != nil {
		log.InfoContext(ctx, "authn_trust_initialized", "component", "authn", "result", "success")
	}
	return keys, document.JWKSURI, jwksClient, nil
}

func fetchDocument(ctx context.Context, client requestClient, target string) ([]byte, error) {
	requestCtx, cancel := context.WithTimeout(ctx, ProviderTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, target, nil)
	if err != nil {
		return nil, errors.New("build provider request")
	}
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.New("provider request failed")
	}
	defer func() {
		_ = response.Body.Close()
	}()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("provider response status is not successful")
	}
	body, err := io.ReadAll(response.Body)
	if err != nil || len(body) > MaxProviderBody {
		return nil, errors.New("provider response body is invalid")
	}
	return body, nil
}

func validProviderURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	return err == nil &&
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
