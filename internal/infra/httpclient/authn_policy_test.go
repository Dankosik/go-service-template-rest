package httpclient

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func TestAuthnExternalHTTPSPolicy(t *testing.T) {
	cfg := Config{
		DependencyName:         "oidc",
		BaseURL:                "https://issuer.example.com",
		TargetClass:            ExternalHTTPS,
		DisableInstrumentation: true,
		RequestTimeout:         5 * time.Second,
		ResponseHeaderTimeout:  5 * time.Second,
		MaxResponseHeaderBytes: 32 << 10,
		MaxResponseBodyBytes:   1 << 20,
		MaxConnsPerHost:        1,
		MaxIdleConnsPerHost:    1,
	}
	client, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New(authn egress) error = %v", err)
	}
	t.Cleanup(client.CloseIdleConnections)

	if _, ok := client.httpClient.Transport.(authorityTransport); !ok {
		t.Fatalf("authn transport = %T, want direct bounded authority transport", client.httpClient.Transport)
	}
	if _, ok := client.httpClient.Transport.(*otelhttp.Transport); ok {
		t.Fatal("authn egress unexpectedly enabled general outbound instrumentation")
	}
	if client.transport.Proxy != nil {
		t.Fatal("authn egress enabled an ambient proxy")
	}
	if got := client.httpClient.Timeout; got != 5*time.Second {
		t.Fatalf("request timeout = %s, want 5s", got)
	}
	if got := client.transport.ResponseHeaderTimeout; got != 5*time.Second {
		t.Fatalf("response header timeout = %s, want 5s", got)
	}
	if got := client.transport.MaxResponseHeaderBytes; got != 32<<10 {
		t.Fatalf("response header limit = %d, want 32768", got)
	}
	if got := client.transport.MaxConnsPerHost; got != 1 {
		t.Fatalf("connection limit = %d, want 1", got)
	}
	if got := client.transport.MaxIdleConnsPerHost; got != 1 {
		t.Fatalf("idle connection limit = %d, want 1", got)
	}
	if err := client.httpClient.CheckRedirect(&http.Request{}, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("redirect policy error = %v, want %v", err, http.ErrUseLastResponse)
	}

	ordinary := cfg
	ordinary.DisableInstrumentation = false
	ordinaryClient, err := New(ordinary, nil)
	if err != nil {
		t.Fatalf("New(ordinary egress) error = %v", err)
	}
	t.Cleanup(ordinaryClient.CloseIdleConnections)
	if _, ok := ordinaryClient.httpClient.Transport.(*otelhttp.Transport); !ok {
		t.Fatalf("ordinary transport = %T, want instrumented transport", ordinaryClient.httpClient.Transport)
	}
}
