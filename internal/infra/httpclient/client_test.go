package httpclient

import (
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"
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
	if response == nil || response.Body == nil || received == nil {
		t.Fatal("RoundTrip() returned an incomplete response")
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

const (
	fixedHeaderTimeout  = 5 * time.Second
	fixedMaxHeaderBytes = 32 << 10
	headerLimitMargin   = 1024
	bodyCanary          = "body-must-not-reach-caller"
)

func TestResponseLimitsAdmissionAndCompatibility(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	server := newPinnedTLSServer(t, false, func(response http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		response.WriteHeader(http.StatusNoContent)
	})
	valid := ResponseLimits{ResponseHeaderTimeout: fixedHeaderTimeout, MaxResponseHeaderBytes: fixedMaxHeaderBytes}
	for _, limits := range []ResponseLimits{
		{},
		{ResponseHeaderTimeout: fixedHeaderTimeout},
		{MaxResponseHeaderBytes: fixedMaxHeaderBytes},
		{ResponseHeaderTimeout: -time.Second, MaxResponseHeaderBytes: fixedMaxHeaderBytes},
		{ResponseHeaderTimeout: fixedHeaderTimeout, MaxResponseHeaderBytes: -1},
	} {
		if _, err := NewExternalHTTPSWithLimits("https://example.com", limits); err == nil {
			t.Fatalf("NewExternalHTTPSWithLimits(%+v) succeeded", limits)
		}
		if _, err := NewPrivateHTTPSWithLimits("https://api.service.internal", "service.internal", limits); err == nil {
			t.Fatalf("NewPrivateHTTPSWithLimits(%+v) succeeded", limits)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("invalid construction issued %d requests", calls.Load())
	}

	oldExternal, err := NewExternalHTTPS("https://example.com")
	if err != nil {
		t.Fatalf("NewExternalHTTPS() error = %v", err)
	}
	limitedExternal, err := NewExternalHTTPSWithLimits("https://example.com", valid)
	if err != nil {
		t.Fatalf("NewExternalHTTPSWithLimits() error = %v", err)
	}
	oldPrivate, err := NewPrivateHTTPS("https://api.service.internal", "service.internal")
	if err != nil {
		t.Fatalf("NewPrivateHTTPS() error = %v", err)
	}
	limitedPrivate, err := NewPrivateHTTPSWithLimits("https://api.service.internal", "service.internal", valid)
	if err != nil {
		t.Fatalf("NewPrivateHTTPSWithLimits() error = %v", err)
	}
	t.Cleanup(oldExternal.CloseIdleConnections)
	t.Cleanup(limitedExternal.CloseIdleConnections)
	t.Cleanup(oldPrivate.CloseIdleConnections)
	t.Cleanup(limitedPrivate.CloseIdleConnections)

	oldTransport := clientTransport(t, oldExternal)
	if oldTransport.ResponseHeaderTimeout != 0 || oldTransport.MaxResponseHeaderBytes != 0 {
		t.Fatal("existing constructor applied response limits")
	}
	for _, client := range []*Client{oldExternal, limitedExternal, oldPrivate, limitedPrivate} {
		if clientTransport(t, client).Proxy != nil {
			t.Fatal("transport proxy is not disabled")
		}
		if err := client.httpClient.CheckRedirect(nil, nil); !errors.Is(err, http.ErrUseLastResponse) {
			t.Fatalf("CheckRedirect() error = %v", err)
		}
		denied, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://other.example.com/", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Do(denied)
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		if !errors.Is(err, ErrTargetDenied) {
			t.Fatalf("alternate authority error = %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("denied requests reached the server: %d", calls.Load())
	}

	pinClientToTLSServer(t, oldExternal, server)
	pinClientToTLSServer(t, limitedExternal, server)
	for _, client := range []*Client{oldExternal, limitedExternal} {
		if body := doPinnedRequest(t, client, "/"); body != "" {
			t.Fatalf("under-limit body = %q", body)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("under-limit calls = %d, want 2", calls.Load())
	}
}

func TestResponseHeaderTimeoutEnforced(t *testing.T) {
	received := make(chan struct{})
	canceled := make(chan struct{})
	blocked := newPinnedTLSServer(t, false, func(_ http.ResponseWriter, request *http.Request) {
		close(received)
		<-request.Context().Done()
		close(canceled)
	})
	control := newPinnedTLSServer(t, false, func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	})
	limits := ResponseLimits{ResponseHeaderTimeout: fixedHeaderTimeout, MaxResponseHeaderBytes: fixedMaxHeaderBytes}

	controlClient, err := NewExternalHTTPSWithLimits("https://example.com", limits)
	if err != nil {
		t.Fatalf("control constructor error = %v", err)
	}
	t.Cleanup(controlClient.CloseIdleConnections)
	pinClientToTLSServer(t, controlClient, control)
	if body := doPinnedRequest(t, controlClient, "/"); body != "" {
		t.Fatalf("under-time body = %q", body)
	}

	blockedClient, err := NewExternalHTTPSWithLimits("https://example.com", limits)
	if err != nil {
		t.Fatalf("blocked constructor error = %v", err)
	}
	t.Cleanup(blockedClient.CloseIdleConnections)
	pinClientToTLSServer(t, blockedClient, blocked)
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, blockedClient.BaseURL()+"/", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		response, requestErr := blockedClient.Do(request)
		if response != nil && response.Body != nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			done <- errors.New("withheld-header request delivered a body")
			return
		}
		done <- requestErr
	}()
	select {
	case <-received:
	case <-time.After(7 * time.Second):
		t.Fatal("server did not receive the withheld-header request")
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("withheld-header request succeeded")
		}
	case <-time.After(7 * time.Second):
		t.Fatal("response-header timeout did not fire")
	}
	select {
	case <-canceled:
	case <-time.After(7 * time.Second):
		t.Fatal("server did not observe request cancellation")
	}
}

func TestResponseHeaderLimitEnforced(t *testing.T) {
	t.Parallel()
	t.Run("HTTP/1.1", func(t *testing.T) {
		t.Parallel()
		assertResponseHeaderLimit(t, false, "HTTP/1.1", "http/1.1")
	})
	t.Run("HTTP/2", func(t *testing.T) {
		t.Parallel()
		assertResponseHeaderLimit(t, true, "HTTP/2.0", "h2")
	})
}

func assertResponseHeaderLimit(t *testing.T, enableHTTP2 bool, wantProto, wantALPN string) {
	t.Helper()
	var gotProto, gotALPN atomic.Value
	server := newPinnedTLSServer(t, enableHTTP2, func(response http.ResponseWriter, request *http.Request) {
		gotProto.Store(request.Proto)
		if request.TLS != nil {
			gotALPN.Store(request.TLS.NegotiatedProtocol)
		}
		if request.URL.Path == "/over" {
			response.Header().Set("X-Pad", overLimitHeaderValue(t))
		}
		_, _ = io.WriteString(response, bodyCanary)
	})
	client, err := NewExternalHTTPSWithLimits("https://example.com", ResponseLimits{
		ResponseHeaderTimeout:  fixedHeaderTimeout,
		MaxResponseHeaderBytes: fixedMaxHeaderBytes,
	})
	if err != nil {
		t.Fatalf("NewExternalHTTPSWithLimits() error = %v", err)
	}
	t.Cleanup(client.CloseIdleConnections)
	pinClientToTLSServer(t, client, server)

	if body := doPinnedRequest(t, client, "/ok"); body != bodyCanary {
		t.Fatalf("under-limit body = %q", body)
	}
	if gotProto.Load() != wantProto {
		t.Fatalf("under-limit proto = %v, want %s", gotProto.Load(), wantProto)
	}
	if alpn, _ := gotALPN.Load().(string); alpn != wantALPN && !(wantALPN == "http/1.1" && alpn == "") {
		t.Fatalf("under-limit ALPN = %q, want %s", alpn, wantALPN)
	}

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, client.BaseURL()+"/over", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if response != nil && response.Body != nil {
		body, _ := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if string(body) == bodyCanary {
			t.Fatal("over-limit response delivered the body")
		}
	}
	if err == nil {
		t.Fatal("over-limit request succeeded")
	}
	if gotProto.Load() != wantProto {
		t.Fatalf("over-limit proto = %v, want %s", gotProto.Load(), wantProto)
	}
	if alpn, _ := gotALPN.Load().(string); alpn != wantALPN && !(wantALPN == "http/1.1" && alpn == "") {
		t.Fatalf("over-limit ALPN = %q, want %s", alpn, wantALPN)
	}
}

func newPinnedTLSServer(t *testing.T, enableHTTP2 bool, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewUnstartedServer(handler)
	server.EnableHTTP2 = enableHTTP2
	server.StartTLS()
	t.Cleanup(server.Close)
	return server
}

func pinClientToTLSServer(t *testing.T, client *Client, server *httptest.Server) {
	t.Helper()
	source, ok := server.Client().Transport.(*http.Transport)
	if !ok {
		t.Fatal("httptest client transport has unexpected type")
	}
	transport := clientTransport(t, client)
	transport.TLSClientConfig = source.TLSClientConfig.Clone()
	transport.DialContext = source.DialContext
	transport.ForceAttemptHTTP2 = source.ForceAttemptHTTP2
}

func clientTransport(t *testing.T, client *Client) *http.Transport {
	t.Helper()
	sanitizer, ok := client.httpClient.Transport.(propagationSanitizer)
	if !ok {
		t.Fatal("client transport is not a propagationSanitizer")
	}
	authority, ok := sanitizer.base.(authorityTransport)
	if !ok {
		t.Fatal("sanitizer base is not an authorityTransport")
	}
	transport, ok := authority.base.(*http.Transport)
	if !ok {
		t.Fatal("authority base is not an *http.Transport")
	}
	return transport
}

func doPinnedRequest(t *testing.T, client *Client, path string) string {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, client.BaseURL()+path, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do(%s) error = %v", path, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("ReadAll(%s) error = %v", path, err)
	}
	return string(body)
}

func overLimitHeaderValue(t *testing.T) string {
	t.Helper()
	pad := make([]byte, fixedMaxHeaderBytes+headerLimitMargin)
	if _, err := rand.Read(pad); err != nil {
		t.Fatal(err)
	}
	// High-entropy bytes as a header-safe token so HTTP/2 cannot shrink them.
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for i, b := range pad {
		pad[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(pad)
}
