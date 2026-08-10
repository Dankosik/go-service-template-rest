package oidcjwt

// Proof for provider.go: startup, which either establishes complete trust or
// fails closed. What a running verifier does with that trust is in
// verifier_test.go.

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/httpclient"
)

func TestAuthnExternalHTTPSPolicyConstruction(t *testing.T) {
	cfg := providerHTTPConfig("https://issuer.example.com")
	if cfg.DependencyName != "oidc" ||
		cfg.BaseURL != "https://issuer.example.com" ||
		cfg.TargetClass != httpclient.ExternalHTTPS ||
		!cfg.DisableInstrumentation ||
		cfg.RequestTimeout != ProviderTimeout ||
		cfg.ResponseHeaderTimeout != ProviderTimeout ||
		cfg.MaxResponseHeaderBytes != providerHeaderLimit ||
		cfg.MaxResponseBodyBytes != MaxProviderBody ||
		cfg.MaxConnsPerHost != 1 ||
		cfg.MaxIdleConnsPerHost != 1 {
		t.Fatalf("provider HTTP config = %+v, want the fixed OIDC egress policy", cfg)
	}
}

func TestInitialTrustOutageFailsClosed(t *testing.T) {
	transportErr := errors.New("poison outage")
	client := &scriptedClient{responses: []scriptedResponse{{err: transportErr}}}
	_, err := buildTestVerifier(t, testVerifierOptions{now: time.Now, client: client})
	if err == nil || strings.Contains(err.Error(), "poison") {
		t.Fatalf("newVerifier() error = %v, want sanitized startup failure", err)
	}
	if errors.Is(err, transportErr) {
		t.Fatalf("newVerifier() error = %v, want transport details to remain opaque", err)
	}
	if reason := providerFailureReason(err); reason != "transport" {
		t.Fatalf("startup provider failure reason = %q, want transport", reason)
	}
}

func TestProviderFetchFailureCategories(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		target string
		client requestClient
		want   string
	}{
		{
			name:   "request",
			target: "\x7f",
			client: &scriptedClient{},
			want:   "request",
		},
		{
			name:   "transport",
			target: testJWKSURI,
			client: &scriptedClient{responses: []scriptedResponse{{err: errors.New("poison transport")}}},
			want:   "transport",
		},
		{
			name:   "canceled",
			target: testJWKSURI,
			client: &scriptedClient{responses: []scriptedResponse{{err: context.Canceled}}},
			want:   "canceled",
		},
		{
			name:   "timeout",
			target: testJWKSURI,
			client: &scriptedClient{responses: []scriptedResponse{{err: context.DeadlineExceeded}}},
			want:   "timeout",
		},
		{
			name:   "body canceled",
			target: testJWKSURI,
			client: bodyErrorClient{err: errors.Join(errors.New("poison body"), context.Canceled)},
			want:   "canceled",
		},
		{
			name:   "body timeout",
			target: testJWKSURI,
			client: bodyErrorClient{err: errors.Join(errors.New("poison body"), context.DeadlineExceeded)},
			want:   "timeout",
		},
		{
			name:   "client status",
			target: testJWKSURI,
			client: &scriptedClient{responses: []scriptedResponse{{status: http.StatusBadRequest}}},
			want:   "status_4xx",
		},
		{
			name:   "rate limited status",
			target: testJWKSURI,
			client: &scriptedClient{responses: []scriptedResponse{{status: http.StatusTooManyRequests}}},
			want:   "rate_limited",
		},
		{
			name:   "server status",
			target: testJWKSURI,
			client: &scriptedClient{responses: []scriptedResponse{{status: http.StatusBadGateway}}},
			want:   "status_5xx",
		},
		{
			name:   "other status",
			target: testJWKSURI,
			client: &scriptedClient{responses: []scriptedResponse{{status: http.StatusNotModified}}},
			want:   "status",
		},
		{
			name:   "oversize",
			target: testJWKSURI,
			client: &scriptedClient{responses: []scriptedResponse{{
				status: http.StatusOK,
				body:   []byte(strings.Repeat("x", MaxProviderBody+1)),
			}}},
			want: "oversize",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := fetchDocument(t.Context(), testCase.client, testCase.target)
			if err == nil {
				t.Fatal("fetchDocument() error = nil")
			}
			if strings.Contains(err.Error(), "poison") {
				t.Fatalf("fetchDocument() error = %v, want provider details redacted", err)
			}
			if reason := providerFailureReason(err); reason != testCase.want {
				t.Fatalf("provider failure reason = %q, want %q; error = %v", reason, testCase.want, err)
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("fetchDocument() error = %v, want provider-owned context cause opaque", err)
			}
		})
	}
}

type bodyErrorClient struct {
	err error
}

func (c bodyErrorClient) Do(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(iotest.ErrReader(c.err)),
	}, nil
}

func TestInitialTrustAcceptsDiscoveredJWKSQuery(t *testing.T) {
	const jwksURI = "https://keys.example.net/jwks?tenant=main"
	key := loadTestRSAKey(t, testSigningKey)
	client := &scriptedClient{responses: []scriptedResponse{
		{
			status: http.StatusOK,
			body:   []byte(`{"issuer":"https://issuer.example.com","jwks_uri":"` + jwksURI + `"}`),
		},
		{status: http.StatusOK, body: jwksDocument(t, key, "key-1")},
	}}
	verifier, err := buildTestVerifier(t, testVerifierOptions{client: client})
	if err != nil {
		t.Fatalf("newVerifier() error = %v, want discovered JWKS query accepted", err)
	}
	t.Cleanup(verifier.Close)
	if verifier.jwksURI != jwksURI {
		t.Fatalf("JWKS URI = %q, want %q", verifier.jwksURI, jwksURI)
	}
}

func TestInitialTrustHonorsCancellationAndDeadline(t *testing.T) {
	for _, testCase := range []struct {
		name string
		ctx  func() (context.Context, context.CancelFunc)
		want error
	}{
		{
			name: "canceled",
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
			want: context.Canceled,
		},
		{
			name: "deadline",
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.Background(), time.Unix(1, 0))
			},
			want: context.DeadlineExceeded,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, cancel := testCase.ctx()
			defer cancel()
			client := &scriptedClient{responses: []scriptedResponse{{status: http.StatusOK, body: []byte(`{}`)}}}
			_, err := buildTestVerifier(t, testVerifierOptions{
				bootstrapCtx: func() context.Context { return ctx },
				now:          time.Now,
				client:       client,
			})
			if err == nil || strings.Contains(err.Error(), testIssuer) {
				t.Fatalf("newVerifier() error = %v, want sanitized fail-closed cancellation", err)
			}
			if !errors.Is(err, testCase.want) {
				t.Fatalf("newVerifier() error = %v, want wrapped %v", err, testCase.want)
			}
		})
	}
}
