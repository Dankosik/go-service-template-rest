package oauthintrospection

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/example/go-service-template-rest/internal/authntrust"
	"github.com/example/go-service-template-rest/internal/infra/bearerauthn"
	"github.com/example/go-service-template-rest/internal/infra/httpclient"
)

var _ bearerauthn.Verifier = (*Verifier)(nil)

const (
	ProviderTimeout        = 5 * time.Second
	MaxResponseHeaderBytes = 32 << 10
	MaxProviderBody        = 1 << 20
)

type providerClient interface {
	Do(request *http.Request) (*http.Response, error)
	CloseIdleConnections()
}

// Verifier owns one fixed-authority introspection client and no background work.
type Verifier struct {
	policy    Policy
	client    providerClient
	now       func() time.Time
	closeOnce sync.Once
}

// New builds the bounded provider client without performing provider I/O.
func New(policy Policy) (*Verifier, error) {
	client, err := newProviderClient(policy)
	if err != nil {
		return nil, err
	}
	return newVerifier(policy, client, time.Now), nil
}

func newVerifier(policy Policy, client providerClient, now func() time.Time) *Verifier {
	if now == nil {
		now = time.Now
	}
	return &Verifier{policy: policy, client: client, now: now}
}

func newProviderClient(policy Policy) (*httpclient.Client, error) {
	limits := httpclient.ResponseLimits{
		ResponseHeaderTimeout:  ProviderTimeout,
		MaxResponseHeaderBytes: MaxResponseHeaderBytes,
	}
	var (
		client *httpclient.Client
		err    error
	)
	switch policy.targetClass {
	case authntrust.TargetClassExternalHTTPS:
		client, err = httpclient.NewExternalHTTPSWithLimits(policy.endpoint, limits)
	case authntrust.TargetClassPrivateHTTPS:
		client, err = httpclient.NewPrivateHTTPSWithLimits(policy.endpoint, policy.privateSuffix, limits)
	default:
		return nil, fmt.Errorf("build introspection client: %w", failure(bearerauthn.KindUnavailable))
	}
	if err != nil {
		return nil, fmt.Errorf("build introspection client: %w", failure(bearerauthn.KindUnavailable))
	}
	return client, nil
}

// Close releases idle provider connections. It is idempotent.
func (v *Verifier) Close() {
	if v == nil || v.client == nil {
		return
	}
	v.closeOnce.Do(v.client.CloseIdleConnections)
}

// Verify implements bearerauthn.Verifier for one already-parsed opaque bearer.
func (v *Verifier) Verify(ctx context.Context, token string) (bearerauthn.Result, error) {
	request, err := v.newIntrospectionRequest(ctx, token)
	if err != nil {
		return bearerauthn.Result{}, classifyProviderError(ctx, err)
	}
	attemptCtx, cancel := context.WithTimeout(ctx, ProviderTimeout)
	defer cancel()
	request = request.WithContext(attemptCtx)

	response, err := v.client.Do(request)
	if err != nil {
		return bearerauthn.Result{}, classifyProviderError(ctx, err)
	}
	if response != nil && response.Body != nil {
		defer func() { _ = response.Body.Close() }()
	}
	body, err := readBoundedBody(response)
	if err != nil {
		return bearerauthn.Result{}, classifyProviderError(ctx, err)
	}
	if response.StatusCode != http.StatusOK || !jsonMediaType(response.Header.Get("Content-Type")) {
		return bearerauthn.Result{}, failure(bearerauthn.KindUnavailable)
	}
	return admitResponse(body, v.policy, v.now())
}

func (v *Verifier) newIntrospectionRequest(ctx context.Context, token string) (*http.Request, error) {
	form := url.Values{}
	form.Set("token", token)
	form.Set("token_type_hint", "access_token")
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, v.policy.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build introspection request: %w", failure(bearerauthn.KindUnavailable))
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", oauthBasicHeader(v.policy.clientID, v.policy.clientSecret))
	return request, nil
}

func oauthBasicHeader(clientID, clientSecret string) string {
	user := url.QueryEscape(clientID)
	password := url.QueryEscape(clientSecret)
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
}

func readBoundedBody(response *http.Response) ([]byte, error) {
	if response == nil || response.Body == nil {
		return nil, failure(bearerauthn.KindUnavailable)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, MaxProviderBody+1))
	if err != nil {
		return nil, fmt.Errorf("read introspection response: %w", failure(bearerauthn.KindUnavailable))
	}
	if len(body) > MaxProviderBody {
		return nil, failure(bearerauthn.KindUnavailable)
	}
	return body, nil
}

func jsonMediaType(value string) bool {
	media, _, err := mime.ParseMediaType(value)
	return err == nil && strings.EqualFold(media, "application/json")
}

func classifyProviderError(ctx context.Context, _ error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("verify access token: %w", ctxErr)
	}
	return failure(bearerauthn.KindUnavailable)
}
