package oauth2clientcredentials

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/httpclient"
	"go.opentelemetry.io/otel/metric"
)

const (
	maxProviderHeaderBytes = 32 << 10
	maxProviderBodyBytes   = 64 << 10
	maxAccessTokenBytes    = 8 << 10
	maxTokenLifetime       = time.Hour
)

var bearerTokenPattern = regexp.MustCompile(`^[-A-Za-z0-9._~+/]+=*$`)

type requestDoer interface {
	Do(request *http.Request) (response *http.Response, err error)
}

type provider struct {
	config Config
	client requestDoer
	now    func() time.Time
}

type accessToken struct {
	value     string
	expiresAt time.Time
}

type providerCooldownError struct {
	err   error
	delay time.Duration
}

func (e *providerCooldownError) Error() string { return e.err.Error() }

func (e *providerCooldownError) Unwrap() error { return e.err }

func newTokenHTTPClient(cfg Config, meterProvider metric.MeterProvider) (*httpclient.Client, error) {
	validated, err := validateConfig(cfg)
	if err != nil {
		return nil, err
	}
	client, err := httpclient.New(tokenHTTPConfig(validated), meterProvider)
	if err != nil {
		return nil, failure(FailureInvalidConfiguration)
	}
	return client, nil
}

func tokenHTTPConfig(cfg Config) httpclient.Config {
	return httpclient.Config{
		DependencyName:         cfg.DependencyName,
		BaseURL:                cfg.TokenEndpoint,
		TargetClass:            cfg.TokenTargetClass,
		OneAttempt:             true,
		DisableInstrumentation: true,
		PrivateHostSuffix:      cfg.TokenPrivateHostSuffix,
		RequestTimeout:         cfg.AcquisitionTimeout,
		ResponseHeaderTimeout:  cfg.AcquisitionTimeout,
		MaxResponseHeaderBytes: maxProviderHeaderBytes,
		MaxResponseBodyBytes:   maxProviderBodyBytes,
		MaxConnsPerHost:        1,
		MaxIdleConnsPerHost:    1,
	}
}

func newProvider(cfg Config, client requestDoer, now func() time.Time) (*provider, error) {
	validated, err := validateConfig(cfg)
	if err != nil || client == nil || now == nil {
		return nil, failure(FailureInvalidConfiguration)
	}
	if tokenClient, ok := client.(*httpclient.Client); ok && tokenClient.BaseURL() != validated.TokenEndpoint {
		return nil, failure(FailureInvalidConfiguration)
	}
	return &provider{config: validated, client: client, now: now}, nil
}

func (p *provider) acquire(ctx context.Context) (accessToken, error) {
	form := url.Values{"grant_type": {"client_credentials"}}
	if len(p.config.Scopes) > 0 {
		form.Set("scope", strings.Join(p.config.Scopes, " "))
	}
	if p.config.Resource != "" {
		form.Set("resource", p.config.Resource)
	}
	if p.config.Audience != "" {
		form.Set("audience", p.config.Audience)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return accessToken{}, failure(FailureInvalidConfiguration)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(url.QueryEscape(p.config.ClientID), url.QueryEscape(p.config.ClientSecret))

	response, err := p.client.Do(request)
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return accessToken{}, classifyTransportFailure(ctx, err)
	}
	if response == nil {
		return accessToken{}, failure(FailureUnsupportedResponse)
	}
	if response.Body == nil {
		if response.StatusCode != http.StatusOK {
			return accessToken{}, providerResponseFailure(response, nil, p.now())
		}
		return accessToken{}, failure(FailureUnsupportedResponse)
	}
	defer func() { _ = response.Body.Close() }()
	receivedAt := p.now()
	body, err := readProviderBody(response.Body)
	if err != nil {
		if response.StatusCode != http.StatusOK {
			return accessToken{}, providerResponseFailure(response, nil, p.now())
		}
		return accessToken{}, failure(FailureUnsupportedResponse)
	}
	if response.StatusCode != http.StatusOK {
		return accessToken{}, providerResponseFailure(response, body, p.now())
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || !strings.EqualFold(mediaType, "application/json") {
		return accessToken{}, failure(FailureUnsupportedResponse)
	}
	token, err := parseTokenResponse(body, p.config.Scopes, receivedAt, p.now())
	if err != nil {
		return accessToken{}, failure(FailureUnsupportedResponse)
	}
	return token, nil
}

func providerResponseFailure(response *http.Response, body []byte, now time.Time) error {
	err := classifyProviderResponse(response.StatusCode, body)
	if response.StatusCode != http.StatusTooManyRequests && response.StatusCode != http.StatusServiceUnavailable {
		return err
	}
	delay, ok := httpclient.RetryAfter(response, now)
	if !ok {
		return err
	}
	return &providerCooldownError{err: err, delay: delay}
}

func providerCooldownDelay(err error) time.Duration {
	if cooldown, ok := errors.AsType[*providerCooldownError](err); ok {
		return cooldown.delay
	}
	return 0
}

func readProviderBody(body io.Reader) ([]byte, error) {
	limited := io.LimitReader(body, maxProviderBodyBytes+1)
	contents, err := io.ReadAll(limited)
	if err != nil || len(contents) > maxProviderBodyBytes {
		return nil, errors.New("provider response is invalid")
	}
	return contents, nil
}

func classifyTransportFailure(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return failure(FailureProviderTimeout)
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return failure(FailureProviderUnavailable)
	}
	if endpointTrustFailure(err) {
		return failure(FailureEndpointTrust)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return failure(FailureProviderTimeout)
	}
	return failure(FailureProviderUnavailable)
}

func endpointTrustFailure(err error) bool {
	if errors.Is(err, httpclient.ErrTargetDenied) {
		return true
	}
	var certificateVerification *tls.CertificateVerificationError
	var alert tls.AlertError
	var recordHeader tls.RecordHeaderError
	var unknownAuthority x509.UnknownAuthorityError
	var hostname x509.HostnameError
	var invalidCertificate x509.CertificateInvalidError
	return errors.As(err, &certificateVerification) || errors.As(err, &alert) || errors.As(err, &recordHeader) ||
		errors.As(err, &unknownAuthority) ||
		errors.As(err, &hostname) || errors.As(err, &invalidCertificate)
}

func classifyProviderResponse(status int, body []byte) error {
	switch {
	case status >= 300 && status < 400:
		return failure(FailureEndpointTrust)
	case status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || status >= 500 && status < 600:
		return failure(FailureProviderUnavailable)
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return failure(FailureClientRejected)
	case status >= 400 && status < 500:
		if providerErrorName(body) == "invalid_client" {
			return failure(FailureClientRejected)
		}
		return failure(FailureGrantRejected)
	default:
		return failure(FailureUnsupportedResponse)
	}
}

func providerErrorName(body []byte) string {
	object, err := decodeJSONObject(body)
	if err != nil {
		return ""
	}
	raw, ok := object["error"]
	if !ok {
		return ""
	}
	value, ok := decodeJSONString(raw)
	if !ok {
		return ""
	}
	return value
}

func parseTokenResponse(body []byte, requestedScopes []string, receivedAt, now time.Time) (accessToken, error) {
	object, err := decodeJSONObject(body)
	if err != nil {
		return accessToken{}, err
	}
	tokenValue, ok := requiredJSONString(object, "access_token")
	if !ok || len(tokenValue) > maxAccessTokenBytes || !validBearerToken(tokenValue) {
		return accessToken{}, errors.New("access token is invalid")
	}
	tokenType, ok := requiredJSONString(object, "token_type")
	if !ok || !strings.EqualFold(tokenType, "Bearer") {
		return accessToken{}, errors.New("token type is invalid")
	}
	expiresRaw, ok := object["expires_in"]
	if !ok {
		return accessToken{}, errors.New("expiry is missing")
	}
	expiresSeconds, err := parseExpiry(expiresRaw)
	if err != nil || expiresSeconds > int64(maxTokenLifetime/time.Second) {
		return accessToken{}, errors.New("expiry is invalid")
	}
	expiresAt := receivedAt.Add(time.Duration(expiresSeconds) * time.Second)
	if expiresAt.Sub(now) <= earlyExpiryMargin {
		return accessToken{}, errors.New("token is already unusable")
	}
	if rawScope, present := object["scope"]; present {
		scopeValue, valid := decodeJSONString(rawScope)
		if !valid || !sameScopeSet(scopeValue, requestedScopes) {
			return accessToken{}, errors.New("scope is invalid")
		}
	}
	if rawRefresh, present := object["refresh_token"]; present {
		refresh, valid := decodeJSONString(rawRefresh)
		if !valid || refresh != "" {
			return accessToken{}, errors.New("refresh token is unsupported")
		}
	}
	return accessToken{value: tokenValue, expiresAt: expiresAt}, nil
}

func decodeJSONObject(body []byte) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return nil, errors.New("response must be one object")
	}
	object := make(map[string]json.RawMessage)
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return nil, errors.New("object key is invalid")
		}
		name, ok := key.(string)
		if !ok {
			return nil, errors.New("object key is invalid")
		}
		if _, duplicate := object[name]; duplicate {
			return nil, errors.New("object key is duplicated")
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, errors.New("object value is invalid")
		}
		object[name] = value
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') {
		return nil, errors.New("object is incomplete")
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("response has trailing data")
	}
	return object, nil
}

func requiredJSONString(object map[string]json.RawMessage, name string) (string, bool) {
	raw, ok := object[name]
	if !ok {
		return "", false
	}
	return decodeJSONString(raw)
}

func decodeJSONString(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 || raw[0] != '"' {
		return "", false
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	return value, true
}

func parseExpiry(raw json.RawMessage) (int64, error) {
	var value int64
	if err := json.Unmarshal(raw, &value); err != nil || value < 1 {
		return 0, errors.New("expiry is invalid")
	}
	return value, nil
}

func validBearerToken(token string) bool {
	return bearerTokenPattern.MatchString(token)
}

func sameScopeSet(raw string, requested []string) bool {
	if raw == "" || strings.ContainsAny(raw, "\t\r\n") {
		return false
	}
	values := strings.Split(raw, " ")
	canonical, err := canonicalScopes(values)
	return err == nil && slices.Equal(canonical, requested)
}
