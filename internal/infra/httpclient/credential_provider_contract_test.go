package httpclient

// profile:credential-provider-http:start

import (
	"testing"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func TestCredentialProviderDisablesGeneralInstrumentation(t *testing.T) {
	t.Parallel()
	cfg := validExternalConfig()
	cfg.DependencyName = "credential-provider"
	cfg.BaseURL = "https://issuer.example.com"
	cfg.DisableInstrumentation = true
	client, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New(credential-provider egress) error = %v", err)
	}
	t.Cleanup(client.CloseIdleConnections)

	inner := client.httpClient.Transport
	// profile:outbound-auth-http:start
	authorized, ok := inner.(attemptAuthorizationTransport)
	if !ok {
		t.Fatalf("credential-provider transport = %T, want direct attempt-authorization transport", inner)
	}
	inner = authorized.base
	// profile:outbound-auth-http:end
	if _, ok := inner.(authorityTransport); !ok {
		t.Fatalf("credential-provider inner transport = %T, want bounded authority transport", inner)
	}
	if _, ok := client.httpClient.Transport.(*otelhttp.Transport); ok {
		t.Fatal("credential-provider egress unexpectedly enabled general outbound instrumentation")
	}
	ordinary := cfg
	ordinary.DisableInstrumentation = false
	ordinaryClient, err := New(ordinary, nil)
	if err != nil {
		t.Fatalf("New(ordinary egress) error = %v", err)
	}
	t.Cleanup(ordinaryClient.CloseIdleConnections)
	sanitized, ok := ordinaryClient.httpClient.Transport.(propagationSanitizer)
	if !ok {
		t.Fatalf("ordinary transport = %T, want propagation sanitizer", ordinaryClient.httpClient.Transport)
	}
	if _, ok := sanitized.base.(*otelhttp.Transport); !ok {
		t.Fatalf("ordinary inner transport = %T, want instrumented transport", sanitized.base)
	}
}

// profile:credential-provider-http:end
