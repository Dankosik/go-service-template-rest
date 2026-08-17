package httpclient

import (
	"compress/gzip"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

func TestNewBuildsBoundedInstrumentedClient(t *testing.T) {
	t.Parallel()

	cfg := validExternalConfig()
	cfg.BaseURL = "HTTPS://EXAMPLE.COM/provider/"

	client, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if got, want := client.BaseURL(), "https://example.com/provider/"; got != want {
		t.Fatalf("BaseURL() = %q, want %q", got, want)
	}
	if got := client.httpClient.Timeout; got != cfg.RequestTimeout {
		t.Fatalf("http client timeout = %s, want %s", got, cfg.RequestTimeout)
	}
	sanitizer, ok := client.httpClient.Transport.(propagationSanitizer)
	if !ok {
		t.Fatalf("http client transport = %T, want propagationSanitizer", client.httpClient.Transport)
	}
	if _, ok := sanitizer.base.(*otelhttp.Transport); !ok {
		t.Fatalf("sanitizer base = %T, want *otelhttp.Transport", sanitizer.base)
	}
	if client.transport.Proxy != nil {
		t.Fatal("base transport proxy is enabled")
	}
	if got := client.transport.ResponseHeaderTimeout; got != cfg.ResponseHeaderTimeout {
		t.Fatalf("ResponseHeaderTimeout = %s, want %s", got, cfg.ResponseHeaderTimeout)
	}
	if got := client.transport.MaxResponseHeaderBytes; got != cfg.MaxResponseHeaderBytes {
		t.Fatalf("MaxResponseHeaderBytes = %d, want %d", got, cfg.MaxResponseHeaderBytes)
	}
	if err := client.httpClient.CheckRedirect(&http.Request{}, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("CheckRedirect() error = %v, want %v", err, http.ErrUseLastResponse)
	}

	client.CloseIdleConnections()
}

func TestClientEnforcesDecodedLimitPropagatesTraceAndRejectsRedirect(t *testing.T) {
	t.Parallel()

	var traceparent atomic.Value
	var redirected atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/gzip":
			traceparent.Store(request.Header.Get("Traceparent"))
			response.Header().Set("Content-Encoding", "gzip")
			writer := gzip.NewWriter(response)
			_, _ = writer.Write([]byte("decoded response"))
			_ = writer.Close()
		case "/redirect":
			http.Redirect(response, request, "/final", http.StatusTemporaryRedirect)
		case "/final":
			redirected.Add(1)
			response.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(response, request)
		}
	}))
	t.Cleanup(server.Close)

	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	cfg := validExternalConfig()
	cfg.BaseURL = "http://api.railway.internal:" + port
	cfg.TargetClass = PrivateHTTP
	cfg.PrivateHostSuffix = "railway.internal"
	cfg.MaxResponseBodyBytes = 7
	cfg.Propagation = PropagationTraceContext
	client, err := New(cfg, metricnoop.NewMeterProvider())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	var dialer net.Dialer
	client.transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, server.Listener.Addr().String())
	}

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1},
		SpanID:     trace.SpanID{2},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanContext)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.BaseURL()+"/gzip", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do(gzip) error = %v", err)
	}
	body, readErr := io.ReadAll(response.Body)
	if got, want := string(body), "decoded"; got != want {
		t.Fatalf("decoded body = %q, want %q", got, want)
	}
	if _, ok := errors.AsType[*ResponseTooLargeError](readErr); !ok {
		t.Fatalf("decoded body error = %v, want ResponseTooLargeError", readErr)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("gzip Body.Close() error = %v", err)
	}

	if got, _ := traceparent.Load().(string); got == "" {
		t.Fatal("traceparent header is empty")
	}

	redirectRequest, err := http.NewRequestWithContext(context.Background(), http.MethodGet, client.BaseURL()+"/redirect", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	redirectResponse, err := client.Do(redirectRequest)
	if err != nil {
		t.Fatalf("Do(redirect) error = %v", err)
	}
	if got := redirectResponse.StatusCode; got != http.StatusTemporaryRedirect {
		t.Fatalf("redirect status = %d, want %d", got, http.StatusTemporaryRedirect)
	}
	if err := redirectResponse.Body.Close(); err != nil {
		t.Fatalf("redirect Body.Close() error = %v", err)
	}
	if got := redirected.Load(); got != 0 {
		t.Fatalf("redirect target calls = %d, want 0", got)
	}
}

func TestOneAttemptTransportDoesNotReplayOrTransform(t *testing.T) {
	t.Parallel()

	t.Run("does not replay", func(t *testing.T) {
		t.Parallel()

		var connections atomic.Int64
		var requests atomic.Int64
		server := httptest.NewUnstartedServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			switch requests.Add(1) {
			case 1:
				response.WriteHeader(http.StatusNoContent)
			case 2:
				hijacker, ok := response.(http.Hijacker)
				if !ok {
					t.Error("response writer does not implement http.Hijacker")
					return
				}
				conn, _, err := hijacker.Hijack()
				if err != nil {
					t.Errorf("Hijack() error = %v", err)
					return
				}
				_ = conn.Close()
			default:
				t.Errorf("server received request %d, want at most 2", requests.Load())
			}
		}))
		server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
			if state == http.StateNew {
				connections.Add(1)
			}
		}
		server.Start()
		t.Cleanup(server.Close)

		client := newOneAttemptTestClient(t, server.URL)
		for _, path := range []string{"/warm", "/drop"} {
			request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, client.BaseURL()+path, http.NoBody)
			if err != nil {
				t.Fatal(err)
			}
			response, err := client.Do(request)
			if path == "/drop" {
				if err == nil {
					t.Fatal("Do(drop) error = nil, want connection failure")
				}
				continue
			}
			if err != nil {
				t.Fatalf("Do(warm) error = %v", err)
			}
			_ = response.Body.Close()
		}
		if got := connections.Load(); got != 2 {
			t.Fatalf("opened connections = %d, want 2", got)
		}
		if got := requests.Load(); got != 2 {
			t.Fatalf("server received %d requests, want 2", got)
		}
	})

	t.Run("preserves wire bytes", func(t *testing.T) {
		t.Parallel()

		const wireBody = "\x1f\x8b\x08\x00raw gzip bytes"
		server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if request.ProtoMajor != 1 {
				t.Errorf("HTTP major version = %d, want 1", request.ProtoMajor)
			}
			if got := request.Header.Get("Accept-Encoding"); got != "" {
				t.Errorf("Accept-Encoding = %q, want empty", got)
			}
			response.Header().Set("Content-Encoding", "gzip")
			_, _ = response.Write([]byte(wireBody))
		}))
		t.Cleanup(server.Close)

		client := newOneAttemptTestClient(t, server.URL)
		request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, client.BaseURL(), http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Do(request)
		if err != nil {
			t.Fatalf("Do() error = %v", err)
		}
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		if err := response.Body.Close(); err != nil {
			t.Fatalf("Body.Close() error = %v", err)
		}
		if got := string(body); got != wireBody {
			t.Fatalf("wire body = %q, want %q", got, wireBody)
		}
	})
}

func TestOneAttemptTransportUsesRequestDeadlineAndExplicitRoots(t *testing.T) {
	t.Parallel()
	pki := newGeneratedClientPKI(t)
	server, requests := newRootTestServer(t, pki.certificate(t, "provider.example"))
	roots := x509.NewCertPool()
	roots.AddCert(pki.root)

	deadline := time.Now().Add(time.Second)
	client := newRootTestClient(t, roots, server, deadline)
	ctx, cancel := context.WithDeadline(t.Context(), deadline)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.BaseURL(), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("response body close: %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("server requests = %d, want 1", requests.Load())
	}

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for blocked TLS peer: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr == nil {
			accepted <- conn
		}
	}()

	blockedDeadline := time.Now().Add(time.Minute)
	client.transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		got, ok := ctx.Value(requestDeadlineKey{}).(time.Time)
		if !ok || !got.Equal(blockedDeadline) {
			return nil, errors.New("dial did not retain the request deadline")
		}
		var dialer net.Dialer
		return dialer.DialContext(ctx, network, listener.Addr().String())
	}
	blockedCtx, stopBlocked := context.WithDeadline(t.Context(), blockedDeadline)
	blockedRequest, err := http.NewRequestWithContext(blockedCtx, http.MethodGet, client.BaseURL(), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		response, requestErr := client.Do(blockedRequest)
		if response != nil {
			requestErr = errors.Join(requestErr, response.Body.Close())
		}
		result <- requestErr
	}()
	peer := <-accepted
	t.Cleanup(func() { _ = peer.Close() })
	stopBlocked()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("blocked TLS Do() error = %v, want context canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("blocked TLS handshake ignored request cancellation")
	}
}

func TestTransportUsesCallerRootCAsWithoutAmbientFallback(t *testing.T) {

	configured := newGeneratedClientPKI(t)
	ambient := newGeneratedClientPKI(t)
	roots := x509.NewCertPool()
	roots.AddCert(configured.root)
	snapshot := roots.Clone()
	ambientFile := t.TempDir() + "/ambient.pem"
	if err := os.WriteFile(ambientFile, ambient.rootPEM, 0o600); err != nil {
		t.Fatalf("write ambient root: %v", err)
	}
	t.Setenv("SSL_CERT_FILE", ambientFile)
	t.Setenv("SSL_CERT_DIR", t.TempDir())

	t.Run("configured root succeeds", func(t *testing.T) {
		t.Parallel()
		server, requests := newRootTestServer(t, configured.certificate(t, "provider.example"))
		client := newRootTestClient(t, roots, server, time.Time{})
		response, err := client.Do(mustRootTestRequest(t, client))
		if err != nil {
			t.Fatalf("Do() error = %v", err)
		}
		if err := response.Body.Close(); err != nil {
			t.Fatalf("response body close: %v", err)
		}
		if client.transport.TLSClientConfig == nil || client.transport.TLSClientConfig.RootCAs != roots || client.transport.TLSClientConfig.ServerName != "provider.example" || client.transport.TLSClientConfig.InsecureSkipVerify {
			t.Fatalf("TLS config = %#v", client.transport.TLSClientConfig)
		}
		if requests.Load() != 1 {
			t.Fatalf("server requests = %d, want 1", requests.Load())
		}
	})

	t.Run("alternate root is denied", func(t *testing.T) {
		t.Parallel()
		server, requests := newRootTestServer(t, ambient.certificate(t, "provider.example"))
		client := newRootTestClient(t, roots, server, time.Time{})
		response, err := client.Do(mustRootTestRequest(t, client))
		if response != nil {
			if closeErr := response.Body.Close(); closeErr != nil {
				t.Fatalf("response body close: %v", closeErr)
			}
		}
		if err == nil {
			t.Fatal("Do() error = nil for alternate root")
		}
		if requests.Load() != 0 {
			t.Fatalf("server requests = %d, want 0", requests.Load())
		}
	})

	t.Run("wrong hostname is denied", func(t *testing.T) {
		t.Parallel()
		server, requests := newRootTestServer(t, configured.certificate(t, "other.internal"))
		client := newRootTestClient(t, roots, server, time.Time{})
		response, err := client.Do(mustRootTestRequest(t, client))
		if response != nil {
			if closeErr := response.Body.Close(); closeErr != nil {
				t.Fatalf("response body close: %v", closeErr)
			}
		}
		if err == nil {
			t.Fatal("Do() error = nil for wrong hostname")
		}
		if requests.Load() != 0 {
			t.Fatalf("server requests = %d, want 0", requests.Load())
		}
	})

	t.Run("nil preserves existing system-root mode", func(t *testing.T) {
		t.Parallel()
		cfg := validExternalConfig()
		client, err := New(cfg, nil)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		t.Cleanup(client.CloseIdleConnections)
		if client.transport.TLSClientConfig != nil {
			t.Fatalf("nil-root TLS config = %#v, want nil", client.transport.TLSClientConfig)
		}
	})

	if !roots.Equal(snapshot) {
		t.Fatal("caller root pool was mutated")
	}
}

func newRootTestServer(t *testing.T, certificate tls.Certificate) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	requests := new(atomic.Int64)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		response.WriteHeader(http.StatusNoContent)
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12}
	server.StartTLS()
	t.Cleanup(server.Close)
	return server, requests
}

func newRootTestClient(t *testing.T, roots *x509.CertPool, server *httptest.Server, deadline time.Time) *Client {
	t.Helper()
	cfg := validExternalConfig()
	cfg.OneAttempt = true
	cfg.TargetClass = ExternalHTTPS
	cfg.BaseURL = "https://provider.example"
	cfg.RootCAs = roots
	client, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(client.CloseIdleConnections)
	client.transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		if !deadline.IsZero() {
			got, ok := ctx.Value(requestDeadlineKey{}).(time.Time)
			if !ok || !got.Equal(deadline) {
				t.Fatalf("dial deadline value = %s, present=%t; want request deadline %s", got, ok, deadline)
			}
		}
		var dialer net.Dialer
		return dialer.DialContext(ctx, network, server.Listener.Addr().String())
	}
	return client
}

func mustRootTestRequest(t *testing.T, client *Client) *http.Request {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	t.Cleanup(cancel)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.BaseURL(), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func newOneAttemptTestClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	cfg := validExternalConfig()
	cfg.OneAttempt = true
	cfg.TargetClass = PrivateHTTP
	cfg.PrivateHostSuffix = ".internal"
	cfg.BaseURL = "http://provider.internal"
	client, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(client.CloseIdleConnections)
	client.transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		var dialer net.Dialer
		return dialer.DialContext(ctx, network, strings.TrimPrefix(serverURL, "http://"))
	}
	return client
}

// TestClientReusesConnectionsAcrossBursts covers the net/http default this
// package exists to override.
//
// A transport cloned from http.DefaultTransport keeps DefaultMaxIdleConnsPerHost
// idle connections, which is 2 — right for a client spread across many hosts and
// wrong for one pinned to a single authority: each burst kept two, so the next
// paid a TCP and TLS handshake for every connection beyond the second, inside the
// request budget of the caller that triggered it.
func TestClientReusesConnectionsAcrossBursts(t *testing.T) {
	t.Parallel()

	const (
		burstSize  = 6
		burstCount = 3
	)

	var opened atomic.Int64
	// Unstarted, because ConnState has to be installed before the serve loop
	// reads it; httptest.NewServer would already be accepting.
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			opened.Add(1)
		}
	}
	server.Start()
	t.Cleanup(server.Close)

	// httptest binds loopback, which the external public-address gate refuses.
	// The private class is the one that permits a plaintext internal host, and
	// the dialer below is what points that name at the test server.
	cfg := validExternalConfig()
	cfg.TargetClass = PrivateHTTP
	cfg.PrivateHostSuffix = ".internal"
	cfg.BaseURL = "http://provider.internal"
	client, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}
	client.transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		var dialer net.Dialer
		return dialer.DialContext(ctx, network, strings.TrimPrefix(server.URL, "http://"))
	}

	for range burstCount {
		var wg sync.WaitGroup
		for range burstSize {
			wg.Go(func() {
				request, reqErr := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://provider.internal/", http.NoBody)
				if reqErr != nil {
					return
				}
				response, doErr := client.Do(request)
				if doErr != nil {
					return
				}
				_, _ = io.Copy(io.Discard, response.Body)
				_ = response.Body.Close()
			})
		}
		wg.Wait()
	}

	// The first burst has to dial; the ones after it must not. Allowing the
	// burst size plus slack keeps the test about pooling rather than about how
	// many goroutines happened to race into the first burst.
	if got := opened.Load(); got > burstSize+1 {
		t.Fatalf("opened %d connections across %d bursts of %d, want the pool reused after the first burst",
			got, burstCount, burstSize)
	}
}

// TestRetryReusesTheConnectionItAbandons covers the other half of the same cost.
//
// drainResponse used to close the abandoned response without reading it, which
// net/http reports to its read loop as a body that never reached EOF — so the
// connection was destroyed instead of pooled and every retry dialed again,
// against a dependency that had just answered 503.
func TestRetryReusesTheConnectionItAbandons(t *testing.T) {
	t.Parallel()

	var opened atomic.Int64
	var attempts atomic.Int64
	// Unstarted for the same reason as above: ConnState must be set before the
	// serve loop reads it.
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"try again"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			opened.Add(1)
		}
	}
	server.Start()
	t.Cleanup(server.Close)

	cfg := validExternalConfig()
	cfg.TargetClass = PrivateHTTP
	cfg.PrivateHostSuffix = ".internal"
	cfg.BaseURL = "http://provider.internal"
	cfg.Retry = RetryPolicy{MaxAttempts: 2, BaseDelay: time.Millisecond}
	client, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}
	client.transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		var dialer net.Dialer
		return dialer.DialContext(ctx, network, strings.TrimPrefix(server.URL, "http://"))
	}

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://provider.internal/", http.NoBody)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v, want nil", err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d after the retry", response.StatusCode, http.StatusOK)
	}
	if attempts.Load() != 2 {
		t.Fatalf("server saw %d attempts, want 2", attempts.Load())
	}
	if got := opened.Load(); got != 1 {
		t.Fatalf("opened %d connections for one retried request, want 1: the abandoned response must be drained, not just closed", got)
	}
}
