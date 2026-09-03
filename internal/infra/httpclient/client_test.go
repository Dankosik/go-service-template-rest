package httpclient

import (
	"compress/gzip"
	"context"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
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
		if _, err := NewExternalHTTPS(raw, testTransportLimits()); err == nil {
			t.Errorf("NewExternalHTTPS(%q) succeeded", raw)
		}
	}
	private, err := NewPrivateHTTPS("HTTPS://API.SERVICE.INTERNAL/v1", "service.internal", testTransportLimits())
	if err != nil {
		t.Fatalf("NewPrivateHTTPS() error = %v", err)
	}
	if got := private.BaseURL(); got != "https://api.service.internal/v1" {
		t.Fatalf("BaseURL() = %q", got)
	}
	private.CloseIdleConnections()
	if _, err := NewPrivateHTTPS("https://api.example.com", "service.internal", testTransportLimits()); err == nil {
		t.Fatal("private target outside suffix succeeded")
	}
	for _, suffix := range []string{"", "."} {
		if _, err := NewPrivateHTTPS("https://api.service.internal", suffix, testTransportLimits()); err == nil {
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
	client, err := NewExternalHTTPS("https://localhost:443", testTransportLimits())
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
	var baseCalls int
	base := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		baseCalls++
		received = request
		return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody}, nil
	})
	transport := propagationSanitizer{base: authorityTransport{base: base, scheme: "https", authority: "api.example.com"}}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/path", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Traceparent", "stale")
	request.Header.Set("Tracestate", "stale")
	request.Header.Set("Baggage", "secret=value")
	request.Header.Set("X-Request-ID", "stale")
	request.Header.Set("Accept-Encoding", "br")
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
	for _, name := range []string{"Accept-Encoding", "Traceparent", "Tracestate", "Baggage", "X-Request-ID"} {
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
	if baseCalls != 1 {
		t.Fatalf("base calls = %d, want 1", baseCalls)
	}

	overriddenHost := request.Clone(request.Context())
	overriddenHost.Host = "other.example.com"
	response, err = transport.RoundTrip(overriddenHost)
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	if !errors.Is(err, ErrTargetDenied) || baseCalls != 1 {
		t.Fatalf("overridden Host error = %v, base calls = %d", err, baseCalls)
	}

	other, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://other.example.com/path", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	response, err = transport.RoundTrip(other)
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	if !errors.Is(err, ErrTargetDenied) || baseCalls != 1 {
		t.Fatalf("alternate authority error = %v, base calls = %d", err, baseCalls)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

const (
	fixedHeaderTimeout  = 5 * time.Second
	fixedMaxHeaderBytes = 32 << 10
	fixedMaxBodyBytes   = 1 << 20
	fixedMaxInFlight    = 2
	headerLimitMargin   = 1024
	bodyCanary          = "body-must-not-reach-caller"
)

func testTransportLimits() TransportLimits {
	return TransportLimits{
		ResponseHeaderTimeout:  fixedHeaderTimeout,
		MaxResponseHeaderBytes: fixedMaxHeaderBytes,
		MaxInFlight:            fixedMaxInFlight,
		AbsoluteBodyBytes:      fixedMaxBodyBytes,
	}
}

func TestTransportLimitsAdmission(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	server := newPinnedTLSServer(t, false, func(response http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		response.WriteHeader(http.StatusNoContent)
	})
	valid := testTransportLimits()
	for _, limits := range []TransportLimits{
		{},
		{ResponseHeaderTimeout: fixedHeaderTimeout, MaxResponseHeaderBytes: fixedMaxHeaderBytes, MaxInFlight: 1},
		{ResponseHeaderTimeout: fixedHeaderTimeout, MaxResponseHeaderBytes: fixedMaxHeaderBytes, AbsoluteBodyBytes: 1},
		{ResponseHeaderTimeout: fixedHeaderTimeout, MaxInFlight: 1, AbsoluteBodyBytes: 1},
		{MaxResponseHeaderBytes: fixedMaxHeaderBytes, MaxInFlight: 1, AbsoluteBodyBytes: 1},
	} {
		if _, err := NewExternalHTTPS("https://example.com", limits); err == nil {
			t.Fatalf("NewExternalHTTPS(%+v) succeeded", limits)
		}
		if _, err := NewPrivateHTTPS("https://api.service.internal", "service.internal", limits); err == nil {
			t.Fatalf("NewPrivateHTTPS(%+v) succeeded", limits)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("invalid construction issued %d requests", calls.Load())
	}

	external, err := NewExternalHTTPS("https://example.com", valid)
	if err != nil {
		t.Fatalf("NewExternalHTTPS() error = %v", err)
	}
	private, err := NewPrivateHTTPS("https://api.service.internal", "service.internal", valid)
	if err != nil {
		t.Fatalf("NewPrivateHTTPS() error = %v", err)
	}
	t.Cleanup(external.CloseIdleConnections)
	t.Cleanup(private.CloseIdleConnections)

	transport := clientTransport(t, external)
	if transport.ResponseHeaderTimeout != fixedHeaderTimeout || transport.MaxResponseHeaderBytes != fixedMaxHeaderBytes {
		t.Fatal("constructor did not apply response-header limits")
	}
	for _, client := range []*Client{external, private} {
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

	pinClientToTLSServer(t, external, server)
	if body := doPinnedRequest(t, external, "/"); body != "" {
		t.Fatalf("under-limit body = %q", body)
	}
	if calls.Load() != 1 {
		t.Fatalf("under-limit calls = %d, want 1", calls.Load())
	}
}

func TestResponseHeaderTimeoutEnforced(t *testing.T) {
	const testHeaderTimeout = 250 * time.Millisecond

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
	limits := testTransportLimits()
	limits.ResponseHeaderTimeout = testHeaderTimeout

	controlClient, err := NewExternalHTTPS("https://example.com", limits)
	if err != nil {
		t.Fatalf("control constructor error = %v", err)
	}
	t.Cleanup(controlClient.CloseIdleConnections)
	pinClientToTLSServer(t, controlClient, control)
	if body := doPinnedRequest(t, controlClient, "/"); body != "" {
		t.Fatalf("under-time body = %q", body)
	}

	blockedClient, err := NewExternalHTTPS("https://example.com", limits)
	if err != nil {
		t.Fatalf("blocked constructor error = %v", err)
	}
	t.Cleanup(blockedClient.CloseIdleConnections)
	pinClientToTLSServer(t, blockedClient, blocked)
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, blockedClient.BaseURL()+"/", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	watchdog := time.NewTimer(5 * time.Second)
	defer watchdog.Stop()
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
	case <-watchdog.C:
		t.Fatal("server did not receive the withheld-header request")
	}
	select {
	case err := <-done:
		timeoutError, ok := errors.AsType[net.Error](err)
		if !ok || !timeoutError.Timeout() {
			t.Fatalf("withheld-header error = %v, want timeout", err)
		}
	case <-watchdog.C:
		t.Fatal("response-header timeout did not fire")
	}
	select {
	case <-canceled:
	case <-watchdog.C:
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
	client, err := NewExternalHTTPS("https://example.com", testTransportLimits())
	if err != nil {
		t.Fatalf("NewExternalHTTPS() error = %v", err)
	}
	t.Cleanup(client.CloseIdleConnections)
	pinClientToTLSServer(t, client, server)

	if body := doPinnedRequest(t, client, "/ok"); body != bodyCanary {
		t.Fatalf("under-limit body = %q", body)
	}
	if gotProto.Load() != wantProto {
		t.Fatalf("under-limit proto = %v, want %s", gotProto.Load(), wantProto)
	}
	if alpn, _ := gotALPN.Load().(string); alpn != wantALPN && (wantALPN != "http/1.1" || alpn != "") {
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
	if alpn, _ := gotALPN.Load().(string); alpn != wantALPN && (wantALPN != "http/1.1" || alpn != "") {
		t.Fatalf("over-limit ALPN = %q, want %s", alpn, wantALPN)
	}
}

func TestResponseBodyLimit(t *testing.T) {
	limits := testTransportLimits()
	limits.AbsoluteBodyBytes = 8
	server := newPinnedTLSServer(t, false, func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/content-length":
			response.Header().Set("Content-Length", "9")
			_, _ = io.WriteString(response, "123456789")
		case "/chunked":
			if !flushResponse(t, response) {
				return
			}
			_, _ = io.WriteString(response, "123456789")
		case "/compressed":
			response.Header().Set("Content-Encoding", "gzip")
			compressed := gzip.NewWriter(response)
			_, _ = io.WriteString(compressed, strings.Repeat("x", 9))
			_ = compressed.Close()
		}
	})
	client, err := NewExternalHTTPS("https://example.com", limits)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.CloseIdleConnections)
	pinClientToTLSServer(t, client, server)

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, client.BaseURL()+"/content-length", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	if response != nil || !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("content-length response = %#v, error = %v", response, err)
	}

	for _, path := range []string{"/chunked", "/compressed"} {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, client.BaseURL()+path, http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Do(request)
		if err != nil {
			t.Fatalf("Do(%s) error = %v", path, err)
		}
		body, readErr := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if len(body) != int(limits.AbsoluteBodyBytes) || !errors.Is(readErr, ErrResponseTooLarge) {
			t.Fatalf("ReadAll(%s) bytes = %d, error = %v", path, len(body), readErr)
		}
	}
}

func TestOperationPolicy(t *testing.T) {
	limits := testTransportLimits()
	server := newPinnedTLSServer(t, false, func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/body" {
			if !flushResponse(t, response) {
				return
			}
			_, _ = io.WriteString(response, "12")
			return
		}
		if !flushResponse(t, response) {
			return
		}
		<-request.Context().Done()
	})
	client, err := NewExternalHTTPS("https://example.com", limits)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.CloseIdleConnections)
	pinClientToTLSServer(t, client, server)

	for _, policy := range []OperationPolicy{
		{},
		{Timeout: time.Second},
		{MaxBodyBytes: 1},
		{Timeout: time.Second, MaxBodyBytes: limits.AbsoluteBodyBytes + 1},
	} {
		request, requestErr := http.NewRequestWithContext(t.Context(), http.MethodGet, client.BaseURL(), http.NoBody)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		response, err := client.DoWithPolicy(request, policy)
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		if err == nil {
			t.Fatalf("DoWithPolicy(%+v) succeeded", policy)
		}
	}

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, client.BaseURL()+"/body", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.DoWithPolicy(request, OperationPolicy{Timeout: time.Second, MaxBodyBytes: 1})
	if err != nil {
		t.Fatalf("DoWithPolicy(body) error = %v", err)
	}
	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if string(body) != "1" || !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("ReadAll(body) = %q, %v", body, err)
	}

	parentCtx, cancelParent := context.WithCancel(t.Context())
	cancelParent()
	request, err = http.NewRequestWithContext(parentCtx, http.MethodGet, client.BaseURL(), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	response, err = client.DoWithPolicy(request, OperationPolicy{Timeout: time.Second, MaxBodyBytes: 1})
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	if !errors.Is(err, context.Canceled) || errors.Is(err, ErrOperationTimeout) {
		t.Fatalf("parent-canceled error = %v", err)
	}

	request, err = http.NewRequestWithContext(t.Context(), http.MethodGet, client.BaseURL()+"/timeout", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	response, err = client.DoWithPolicy(request, OperationPolicy{Timeout: 100 * time.Millisecond, MaxBodyBytes: 1})
	if err != nil {
		t.Fatalf("DoWithPolicy() error = %v", err)
	}
	_, err = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if !errors.Is(err, ErrOperationTimeout) {
		t.Fatalf("ReadAll() error = %v, want ErrOperationTimeout", err)
	}
}

func TestConcurrencyAdmissionReleasesSlot(t *testing.T) {
	t.Run("cancellation", func(t *testing.T) {
		limits := testTransportLimits()
		limits.MaxInFlight = 1
		var calls atomic.Int32
		server := newPinnedTLSServer(t, false, func(response http.ResponseWriter, request *http.Request) {
			if calls.Add(1) == 1 {
				if !flushResponse(t, response) {
					return
				}
				<-request.Context().Done()
				return
			}
			response.WriteHeader(http.StatusNoContent)
		})
		client, err := NewExternalHTTPS("https://example.com", limits)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(client.CloseIdleConnections)
		pinClientToTLSServer(t, client, server)

		requestCtx, cancel := context.WithCancel(t.Context())
		request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, client.BaseURL(), http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = response.Body.Close() }()

		overflow, err := http.NewRequestWithContext(t.Context(), http.MethodGet, client.BaseURL(), http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		overflowResponse, err := client.Do(overflow)
		if overflowResponse != nil && overflowResponse.Body != nil {
			_ = overflowResponse.Body.Close()
		}
		if !errors.Is(err, ErrSaturated) {
			t.Fatalf("overflow error = %v, want ErrSaturated", err)
		}

		cancel()
		if _, err := io.ReadAll(response.Body); !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled body error = %v", err)
		}
		if body := doPinnedRequest(t, client, "/"); body != "" {
			t.Fatalf("recovered body = %q", body)
		}
	})

	t.Run("transport error", func(t *testing.T) {
		limits := testTransportLimits()
		limits.MaxInFlight = 1
		var calls atomic.Int32
		server := newPinnedTLSServer(t, false, func(response http.ResponseWriter, _ *http.Request) {
			if calls.Add(1) == 1 {
				hijacker, ok := response.(http.Hijacker)
				if !ok {
					t.Error("response writer cannot hijack the test connection")
					return
				}
				connection, _, err := hijacker.Hijack()
				if err != nil {
					t.Error(err)
					return
				}
				_ = connection.Close()
				return
			}
			response.WriteHeader(http.StatusNoContent)
		})
		client, err := NewExternalHTTPS("https://example.com", limits)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(client.CloseIdleConnections)
		pinClientToTLSServer(t, client, server)

		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, client.BaseURL(), http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Do(request)
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		if err == nil {
			t.Fatal("broken transport request succeeded")
		}
		if body := doPinnedRequest(t, client, "/"); body != "" {
			t.Fatalf("recovered body = %q", body)
		}
	})
}

func flushResponse(t *testing.T, response http.ResponseWriter) bool {
	t.Helper()
	flusher, ok := response.(http.Flusher)
	if !ok {
		t.Error("response writer cannot flush test headers")
		return false
	}
	flusher.Flush()
	return true
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
	defer func() { _ = response.Body.Close() }()
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
