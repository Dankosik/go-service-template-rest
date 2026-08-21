package httpclient

import (
	"errors"
	"net/http"
	"net/netip"
	"testing"
)

func TestFixedTargetPolicy(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{
		"", "http://example.com", "https://user@example.com", "https://example.com?query=1",
		"https://127.0.0.1", "https://[::1]",
	} {
		if _, err := NewExternalHTTPS(raw); err == nil {
			t.Errorf("NewExternalHTTPS(%q) succeeded", raw)
		}
	}
	private, err := NewPrivateHTTPS("HTTPS://API.SERVICE.INTERNAL/v1", "service.internal")
	if err != nil {
		t.Fatalf("NewPrivateHTTPS() error = %v", err)
	}
	if got := private.BaseURL(); got != "https://api.service.internal/v1" {
		t.Fatalf("BaseURL() = %q", got)
	}
	private.CloseIdleConnections()
	if _, err := NewPrivateHTTPS("https://api.example.com", "service.internal"); err == nil {
		t.Fatal("private target outside suffix succeeded")
	}
	for _, suffix := range []string{"", "."} {
		if _, err := NewPrivateHTTPS("https://api.service.internal", suffix); err == nil {
			t.Errorf("NewPrivateHTTPS() accepted suffix %q", suffix)
		}
	}
}

func TestDialAddressPolicy(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		policy  targetPolicy
		address string
		allowed bool
	}{
		{name: "external public", address: "8.8.8.8:443", allowed: true},
		{name: "external private", address: "10.0.0.1:443"},
		{name: "external loopback", address: "127.0.0.1:443"},
		{name: "private", policy: targetPolicy{privateSuffix: ".internal"}, address: "10.0.0.1:443", allowed: true},
		{name: "private public", policy: targetPolicy{privateSuffix: ".internal"}, address: "8.8.8.8:443"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := enforceDialAddress(test.policy, test.address)
			if (err == nil) != test.allowed {
				t.Fatalf("enforceDialAddress() error = %v, allowed = %t", err, test.allowed)
			}
		})
	}
	if !netip.MustParseAddr("10.0.0.1").IsPrivate() {
		t.Fatal("test private address is not private")
	}
}

func TestClientRejectsResolvedPrivateAddressAndRedirects(t *testing.T) {
	client, err := NewExternalHTTPS("https://localhost:443")
	if err != nil {
		t.Fatalf("NewExternalHTTPS() error = %v", err)
	}
	t.Cleanup(client.CloseIdleConnections)
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, client.BaseURL(), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	if !errors.Is(err, ErrTargetDenied) {
		t.Fatalf("Do() error = %v, want ErrTargetDenied", err)
	}
	if err := client.httpClient.CheckRedirect(request, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("CheckRedirect() error = %v", err)
	}
}

func TestAuthorityAndCorrelationPolicy(t *testing.T) {
	var received *http.Request
	base := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		received = request
		return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody}, nil
	})
	transport := propagationSanitizer{base: authorityTransport{base: base, scheme: "https", authority: "api.example.com"}}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/path", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Traceparent", "stale")
	request.Header.Set("Baggage", "secret=value")
	request.Header.Set("X-Request-ID", "stale")
	request.Header.Set("X-Provider", "retained")
	request.Trailer = http.Header{"Baggage": {"secret=trailer"}}
	response, err := transport.RoundTrip(request)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	_ = response.Body.Close()
	for _, name := range []string{"Traceparent", "Baggage", "X-Request-ID"} {
		if got := received.Header.Get(name); got != "" {
			t.Errorf("received %s = %q", name, got)
		}
	}
	if received.Trailer.Get("Baggage") != "" {
		t.Fatal("reserved trailer reached the provider")
	}
	if received.Header.Get("X-Provider") != "retained" || request.Header.Get("Traceparent") != "stale" {
		t.Fatal("sanitizer changed an allowed or original header")
	}

	other, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://other.example.com/path", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	response, err = transport.RoundTrip(other)
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	if !errors.Is(err, ErrTargetDenied) {
		t.Fatalf("alternate authority error = %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
