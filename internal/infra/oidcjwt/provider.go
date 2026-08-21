package oidcjwt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/example/go-service-template-rest/internal/authntrust"
	"github.com/example/go-service-template-rest/internal/infra/httpclient"
	"go.opentelemetry.io/otel/metric"
)

type discoveryDocument struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`
}

func discoverJWKSURI(ctx context.Context, policy Policy, provider metric.MeterProvider) (string, error) {
	issuerURL, err := url.Parse(policy.issuer)
	if err != nil || issuerURL == nil {
		return "", errors.New("OIDC startup failed at issuer validation")
	}
	authority := (&url.URL{Scheme: issuerURL.Scheme, Host: issuerURL.Host}).String()
	client, err := httpclient.New(providerHTTPConfig(authority), provider)
	if err != nil {
		return "", errors.New("OIDC startup failed at discovery client")
	}
	defer client.CloseIdleConnections()

	discoveryURL := *issuerURL
	discoveryURL.Path = strings.TrimSuffix(discoveryURL.Path, "/") + "/.well-known/openid-configuration"
	body, err := fetchDocument(ctx, client, discoveryURL.String())
	if err != nil {
		return "", fmt.Errorf("OIDC startup failed at discovery: %w", err)
	}
	return validateDiscovery(body, policy)
}

func validateDiscovery(body []byte, policy Policy) (string, error) {
	var document discoveryDocument
	if err := json.Unmarshal(body, &document); err != nil ||
		document.Issuer != policy.issuer ||
		!authntrust.ValidJWKSURL(document.JWKSURI) {
		return "", errors.New("OIDC startup failed at discovery validation")
	}
	return document.JWKSURI, nil
}

func newJWKSClient(jwksURI string, provider metric.MeterProvider) (*http.Client, func(), error) {
	parsed, err := url.Parse(jwksURI)
	if err != nil || parsed == nil {
		return nil, nil, errors.New("OIDC startup failed at JWKS URL validation")
	}
	authority := (&url.URL{Scheme: parsed.Scheme, Host: parsed.Host}).String()
	bounded, err := httpclient.New(providerHTTPConfig(authority), provider)
	if err != nil {
		return nil, nil, errors.New("OIDC startup failed at JWKS client")
	}
	client := &http.Client{
		Transport: jwksRoundTripper{client: bounded},
		Timeout:   ProviderTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return client, bounded.CloseIdleConnections, nil
}

type jwksRoundTripper struct {
	client *httpclient.Client
}

func (t jwksRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.client.Do(request)
	if err != nil {
		return response, fmt.Errorf("send JWKS request: %w", err)
	}
	return response, nil
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

type requestClient interface {
	Do(request *http.Request) (*http.Response, error)
}

func fetchDocument(ctx context.Context, client requestClient, target string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, http.NoBody)
	if err != nil {
		return nil, errors.New("provider request is invalid")
	}
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("provider request canceled: %w", ctxErr)
		}
		return nil, errors.New("provider request failed")
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("provider returned an unsuccessful status")
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("provider response canceled: %w", ctxErr)
		}
		return nil, errors.New("provider response failed")
	}
	return body, nil
}
