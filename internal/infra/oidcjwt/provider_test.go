package oidcjwt

import (
	"testing"

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
