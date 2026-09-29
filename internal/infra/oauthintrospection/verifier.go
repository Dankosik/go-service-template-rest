package oauthintrospection

import (
	"context"
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

// Fixed provider-work bounds. Changing one is a code-reviewed trust decision;
// shared token size and clock-skew bounds live in bearerauthn.
const (
	providerTimeout        = 5 * time.Second
	maxProviderHeaderBytes = 32 << 10
	maxProviderBodyBytes   = 1 << 20
	maxProviderInFlight    = 32
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
	limits := httpclient.TransportLimits{
		ResponseHeaderTimeout:  providerTimeout,
		MaxResponseHeaderBytes: maxProviderHeaderBytes,
		MaxInFlight:            maxProviderInFlight,
		AbsoluteBodyBytes:      maxProviderBodyBytes,
	}
	switch policy.targetClass {
	case authntrust.TargetClassExternalHTTPS:
		client, err := httpclient.NewExternalHTTPS(policy.endpoint, limits)
		if err != nil {
			return nil, fmt.Errorf("build introspection client: %w", bearerauthn.NewError(bearerauthn.KindUnavailable))
		}
		return client, nil
	case authntrust.TargetClassPrivateHTTPS:
		client, err := httpclient.NewPrivateHTTPS(policy.endpoint, policy.privateSuffix, limits)
		if err != nil {
			return nil, fmt.Errorf("build introspection client: %w", bearerauthn.NewError(bearerauthn.KindUnavailable))
		}
		return client, nil
	default:
		return nil, fmt.Errorf("build introspection client: %w", bearerauthn.NewError(bearerauthn.KindUnavailable))
	}
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
	attemptCtx, cancel := context.WithTimeout(ctx, providerTimeout)
	defer cancel()
	request, err := v.newIntrospectionRequest(attemptCtx, token)
	if err != nil {
		return bearerauthn.Result{}, classifyContextOrUnavailable(ctx)
	}

	response, err := v.client.Do(request)
	if err != nil {
		return bearerauthn.Result{}, classifyContextOrUnavailable(ctx)
	}
	if response != nil && response.Body != nil {
		defer func() { _ = response.Body.Close() }()
	}
	body, ok := readBoundedBody(response)
	if !ok {
		return bearerauthn.Result{}, classifyContextOrUnavailable(ctx)
	}
	if response.StatusCode != http.StatusOK || !jsonMediaType(response.Header.Get("Content-Type")) {
		return bearerauthn.Result{}, bearerauthn.VerificationFailure(bearerauthn.KindUnavailable)
	}
	return admitResponse(body, v.policy, v.now())
}

func (v *Verifier) newIntrospectionRequest(ctx context.Context, token string) (*http.Request, error) {
	form := url.Values{}
	form.Set("token", token)
	form.Set("token_type_hint", "access_token")
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, v.policy.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build introspection request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	// client_secret_basic form-encodes both values before Basic encoding
	// (RFC 6749 section 2.3.1).
	request.SetBasicAuth(url.QueryEscape(v.policy.clientID), url.QueryEscape(v.policy.clientSecret))
	return request, nil
}

// readBoundedBody reports false for a missing, unreadable, or oversized body.
func readBoundedBody(response *http.Response) ([]byte, bool) {
	if response == nil || response.Body == nil {
		return nil, false
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxProviderBodyBytes+1))
	if err != nil || len(body) > maxProviderBodyBytes {
		return nil, false
	}
	return body, true
}

func jsonMediaType(value string) bool {
	media, _, err := mime.ParseMediaType(value)
	return err == nil && strings.EqualFold(media, "application/json")
}

func classifyContextOrUnavailable(ctx context.Context) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("verify access token: %w", ctxErr)
	}
	return bearerauthn.VerificationFailure(bearerauthn.KindUnavailable)
}
