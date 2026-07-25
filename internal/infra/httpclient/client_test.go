package httpclient

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

func TestNewRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "missing dependency", mutate: func(cfg *Config) { cfg.DependencyName = "" }},
		{name: "invalid URL", mutate: func(cfg *Config) { cfg.BaseURL = "://invalid" }},
		{name: "relative URL", mutate: func(cfg *Config) { cfg.BaseURL = "/provider" }},
		{name: "user info", mutate: func(cfg *Config) { cfg.BaseURL = "https://user@example.com" }},
		{name: "query", mutate: func(cfg *Config) { cfg.BaseURL = "https://example.com?token=secret" }},
		{name: "fragment", mutate: func(cfg *Config) { cfg.BaseURL = "https://example.com#fragment" }},
		{name: "external HTTP", mutate: func(cfg *Config) { cfg.BaseURL = "http://example.com" }},
		{name: "external private literal", mutate: func(cfg *Config) { cfg.BaseURL = "https://127.0.0.1" }},
		{name: "private HTTPS", mutate: func(cfg *Config) {
			cfg.BaseURL = "https://api.railway.internal"
			cfg.TargetClass = PrivateHTTP
			cfg.PrivateHostSuffix = "railway.internal"
		}},
		{name: "private public host", mutate: func(cfg *Config) {
			cfg.BaseURL = "http://example.com"
			cfg.TargetClass = PrivateHTTP
			cfg.PrivateHostSuffix = "railway.internal"
		}},
		// No platform default: a private target names its own DNS zone or is
		// refused, so the package cannot silently assume one platform's zone.
		{name: "private suffix is required", mutate: func(cfg *Config) {
			cfg.BaseURL = "http://api.railway.internal"
			cfg.TargetClass = PrivateHTTP
			cfg.PrivateHostSuffix = ""
		}},
		{name: "private host outside configured suffix", mutate: func(cfg *Config) {
			cfg.BaseURL = "http://api.railway.internal"
			cfg.TargetClass = PrivateHTTP
			cfg.PrivateHostSuffix = "svc.cluster.local"
		}},
		{name: "private suffix cannot be dot only", mutate: func(cfg *Config) {
			cfg.BaseURL = "http://api.railway.internal"
			cfg.TargetClass = PrivateHTTP
			cfg.PrivateHostSuffix = "."
		}},
		{name: "invalid target class", mutate: func(cfg *Config) { cfg.TargetClass = 99 }},
		{name: "external ignores private suffix", mutate: func(cfg *Config) {
			cfg.BaseURL = "http://provider.example"
			cfg.PrivateHostSuffix = "provider.example"
		}},
		{name: "missing request timeout", mutate: func(cfg *Config) { cfg.RequestTimeout = 0 }},
		{name: "missing header timeout", mutate: func(cfg *Config) { cfg.ResponseHeaderTimeout = 0 }},
		{name: "missing header limit", mutate: func(cfg *Config) { cfg.MaxResponseHeaderBytes = 0 }},
		{name: "missing body limit", mutate: func(cfg *Config) { cfg.MaxResponseBodyBytes = 0 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := validExternalConfig()
			tt.mutate(&cfg)
			_, err := New(cfg, nil)
			if err == nil {
				t.Fatal("New() error = nil, want non-nil")
			}
			if strings.Contains(err.Error(), "token=secret") {
				t.Fatalf("New() error leaks query secret: %q", err)
			}
		})
	}
}

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
	if _, ok := client.httpClient.Transport.(*otelhttp.Transport); !ok {
		t.Fatalf("http client transport = %T, want *otelhttp.Transport", client.httpClient.Transport)
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

func TestAuthorityTransportAllowsOnlyConfiguredAuthority(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64
	transport := authorityTransport{
		base: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return responseWithBody("ok"), nil
		}),
		scheme:    "https",
		authority: "api.example.com",
	}

	allowed, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.example.com/v1/items", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := transport.RoundTrip(allowed)
	if err != nil {
		t.Fatalf("allowed RoundTrip() error = %v", err)
	}
	_ = response.Body.Close()

	for _, target := range []string{
		"http://api.example.com/v1/items",
		"https://other.example.com/v1/items",
		"https://user@api.example.com/v1/items",
	} {
		request, requestErr := http.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		response, roundTripErr := transport.RoundTrip(request)
		if response != nil {
			_ = response.Body.Close()
		}
		if !errors.Is(roundTripErr, ErrTargetDenied) {
			t.Fatalf("RoundTrip(%q) error = %v, want %v", target, roundTripErr, ErrTargetDenied)
		}
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("base transport calls = %d, want 1", got)
	}
}

func TestResponseLimitTransportBoundsDecodedBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      func(t *testing.T) io.ReadCloser
		limit     int64
		want      string
		wantLarge bool
	}{
		{
			name:  "exact limit",
			body:  func(*testing.T) io.ReadCloser { return io.NopCloser(strings.NewReader("hello")) },
			limit: 5,
			want:  "hello",
		},
		{
			name:      "plain overflow",
			body:      func(*testing.T) io.ReadCloser { return io.NopCloser(strings.NewReader("hello")) },
			limit:     4,
			want:      "hell",
			wantLarge: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var closed atomic.Bool
			transport := responseLimitTransport{
				base: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     make(http.Header),
						Body: &trackedBody{
							ReadCloser: tt.body(t),
							closed:     &closed,
						},
					}, nil
				}),
				limit: tt.limit,
			}

			request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com", nil)
			if err != nil {
				t.Fatal(err)
			}
			response, err := transport.RoundTrip(request)
			if err != nil {
				t.Fatalf("RoundTrip() error = %v", err)
			}
			body, readErr := io.ReadAll(response.Body)
			if got := string(body); got != tt.want {
				t.Fatalf("body = %q, want %q", got, tt.want)
			}
			var tooLarge *ResponseTooLargeError
			if got := errors.As(readErr, &tooLarge); got != tt.wantLarge {
				t.Fatalf("ResponseTooLargeError present = %t, want %t; error = %v", got, tt.wantLarge, readErr)
			}
			if tooLarge != nil && tooLarge.Limit != tt.limit {
				t.Fatalf("ResponseTooLargeError.Limit = %d, want %d", tooLarge.Limit, tt.limit)
			}
			if tt.wantLarge && !closed.Load() {
				t.Fatal("overflow did not close the underlying response body")
			}
			if err := response.Body.Close(); err != nil {
				t.Fatalf("Body.Close() error = %v", err)
			}
			if !closed.Load() {
				t.Fatal("underlying response body was not closed")
			}
		})
	}
}

func TestPublicDialAddressGate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		address string
		allowed bool
	}{
		{address: "8.8.8.8:443", allowed: true},
		{address: "[2606:4700:4700::1111]:443", allowed: true},
		{address: "10.0.0.1:443"},
		{address: "127.0.0.1:443"},
		{address: "169.254.169.254:80"},
		{address: "224.0.0.1:443"},
		{address: "255.255.255.255:443"},
		{address: "[::]:443"},
		{address: "example.com:443"},
		{address: "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.address, func(t *testing.T) {
			t.Parallel()

			err := enforcePublicDialAddress(context.Background(), "tcp", tt.address, nil)
			if tt.allowed && err != nil {
				t.Fatalf("enforcePublicDialAddress() error = %v, want nil", err)
			}
			if !tt.allowed && !errors.Is(err, ErrTargetDenied) {
				t.Fatalf("enforcePublicDialAddress() error = %v, want %v", err, ErrTargetDenied)
			}
		})
	}
}

func TestExternalClientRejectsPrivateDNSResolutionBeforeConnect(t *testing.T) {
	t.Parallel()

	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	cfg := validExternalConfig()
	cfg.BaseURL = "https://localhost:" + port
	client, err := New(cfg, metricnoop.NewMeterProvider())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, client.BaseURL(), nil)
	if err != nil {
		t.Fatal(err)
	}
	response, requestErr := client.Do(request)
	if response != nil {
		_ = response.Body.Close()
	}
	if !errors.Is(requestErr, ErrTargetDenied) {
		t.Fatalf("Do() error = %v, want %v", requestErr, ErrTargetDenied)
	}

	tcpListener, ok := listener.(*net.TCPListener)
	if !ok {
		t.Fatalf("listener = %T, want *net.TCPListener", listener)
	}
	if err := tcpListener.SetDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if connection, acceptErr := listener.Accept(); acceptErr == nil {
		_ = connection.Close()
		t.Fatal("private target accepted a connection")
	}
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
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.BaseURL()+"/gzip", nil)
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
	var tooLarge *ResponseTooLargeError
	if !errors.As(readErr, &tooLarge) {
		t.Fatalf("decoded body error = %v, want ResponseTooLargeError", readErr)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("gzip Body.Close() error = %v", err)
	}

	if got, _ := traceparent.Load().(string); got == "" {
		t.Fatal("traceparent header is empty")
	}

	redirectRequest, err := http.NewRequestWithContext(context.Background(), http.MethodGet, client.BaseURL()+"/redirect", nil)
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

func validExternalConfig() Config {
	return Config{
		DependencyName:         "provider",
		BaseURL:                "https://example.com",
		TargetClass:            ExternalHTTPS,
		RequestTimeout:         time.Second,
		ResponseHeaderTimeout:  500 * time.Millisecond,
		MaxResponseHeaderBytes: 32 << 10,
		MaxResponseBodyBytes:   1 << 20,
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type trackedBody struct {
	io.ReadCloser
	closed *atomic.Bool
}

func (b *trackedBody) Close() error {
	b.closed.Store(true)
	if err := b.ReadCloser.Close(); err != nil {
		return fmt.Errorf("close tracked response body: %w", err)
	}
	return nil
}

func responseWithBody(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestNewAcceptsCustomPrivateHostSuffix(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name    string
		suffix  string
		baseURL string
	}{
		{name: "kubernetes cluster zone", suffix: "svc.cluster.local", baseURL: "http://billing.default.svc.cluster.local"},
		{name: "leading dot accepted", suffix: ".internal", baseURL: "http://billing.internal"},
		{name: "railway zone", suffix: "railway.internal", baseURL: "http://billing.railway.internal"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := validExternalConfig()
			cfg.TargetClass = PrivateHTTP
			cfg.PrivateHostSuffix = tt.suffix
			cfg.BaseURL = tt.baseURL

			client, err := New(cfg, metricnoop.NewMeterProvider())
			if err != nil {
				t.Fatalf("New() error = %v, want nil", err)
			}
			t.Cleanup(client.CloseIdleConnections)
		})
	}
}
