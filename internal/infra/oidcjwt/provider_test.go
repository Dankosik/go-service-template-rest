package oidcjwt

// Proof for provider.go: startup, which either establishes complete trust or
// fails closed. What a running verifier does with that trust is in
// verifier_test.go.

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
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
	client := &scriptedClient{responses: []scriptedResponse{{err: errors.New("poison outage")}}}
	_, err := buildTestVerifier(t, testVerifierOptions{now: time.Now, client: client})
	if err == nil || strings.Contains(err.Error(), "poison") {
		t.Fatalf("newVerifier() error = %v, want sanitized startup failure", err)
	}
}

func TestInitialTrustHonorsCancellationAndDeadline(t *testing.T) {
	for _, testCase := range []struct {
		name string
		ctx  func() (context.Context, context.CancelFunc)
	}{
		{
			name: "canceled",
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
		},
		{
			name: "deadline",
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.Background(), time.Unix(1, 0))
			},
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
		})
	}
}
