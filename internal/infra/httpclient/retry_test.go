package httpclient

import (
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

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"go.opentelemetry.io/otel/trace"
)

const (
	retryTestBaseDelay   = time.Millisecond
	retryTestMaxAttempts = 3
)

// newRetryTestClient builds a client whose fixed target is a private-zone hostname
// and whose dialer reaches the test server, which is how the rest of this package
// exercises a live request through the real bounded transport chain. The retry
// decorator wraps that chain rather than a bare RoundTripper, so these
// tests also cover its interaction with the authority and response-size bounds.
func newRetryTestClient(t *testing.T, handler http.Handler, policy RetryPolicy) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatalf("split test server address: %v", err)
	}

	cfg := validExternalConfig()
	cfg.BaseURL = "http://provider.railway.internal:" + port
	cfg.TargetClass = PrivateHTTP
	cfg.PrivateHostSuffix = "railway.internal"
	cfg.Retry = policy

	client, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}
	var dialer net.Dialer
	client.transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, server.Listener.Addr().String())
	}
	return client
}

func TestRetryRepeatsRetryableStatus(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	client := newRetryTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}), RetryPolicy{MaxAttempts: retryTestMaxAttempts, BaseDelay: retryTestBaseDelay})
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, client.BaseURL(), http.NoBody)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if got := attempts.Load(); got != 2 {
		t.Fatalf("attempts = %d, want 2", got)
	}
}

// TestRetryLeavesUnsafeMethodsAlone is the rule that keeps a retry from creating a
// second resource: a POST the server did process is indistinguishable to the client
// from one that never arrived, and only the caller knows whether repeating it is
// safe.
func TestRetryLeavesUnsafeMethodsAlone(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	client := newRetryTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}), RetryPolicy{MaxAttempts: retryTestMaxAttempts, BaseDelay: retryTestBaseDelay})
	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		client.BaseURL(),
		strings.NewReader(`{}`),
	)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })

	if got := attempts.Load(); got != 1 {
		t.Fatalf("attempts = %d, want exactly 1 for an unsafe method without an idempotency key", got)
	}
}

// TestRetryRepeatsKeyedUnsafeMethod is the other half: attaching an
// Idempotency-Key is the caller's explicit assertion that the upstream operation
// persists and scopes that key.
func TestRetryRepeatsKeyedUnsafeMethod(t *testing.T) {
	t.Parallel()

	var (
		attempts atomic.Int32
		bodies   = make(chan string, retryTestMaxAttempts)
	)
	client := newRetryTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		bodies <- string(body)
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}), RetryPolicy{MaxAttempts: retryTestMaxAttempts, BaseDelay: retryTestBaseDelay})
	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		client.BaseURL(),
		strings.NewReader(`{"amount":1}`),
	)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	request.Header.Set(idempotencyKeyHeader, "retry-key")

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusCreated)
	}
	if got := attempts.Load(); got != 2 {
		t.Fatalf("attempts = %d, want 2", got)
	}
	// The rewound body is what makes the second attempt equivalent to the first. A
	// retry that sent an empty body would still get a 201 and would be silently wrong.
	close(bodies)
	for body := range bodies {
		if body != `{"amount":1}` {
			t.Fatalf("attempt body = %q, want the original payload", body)
		}
	}
}

func TestClientRejectsBlankIdempotencyKeyBeforeSending(t *testing.T) {
	t.Parallel()

	for _, key := range []string{"", " \t"} {
		t.Run(fmt.Sprintf("%q", key), func(t *testing.T) {
			t.Parallel()

			var attempts atomic.Int32
			client := newRetryTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				attempts.Add(1)
				w.WriteHeader(http.StatusCreated)
			}), RetryPolicy{})
			request, err := http.NewRequestWithContext(
				context.Background(),
				http.MethodPost,
				client.BaseURL(),
				strings.NewReader(`{}`),
			)
			if err != nil {
				t.Fatalf("build request: %v", err)
			}
			request.Header[idempotencyKeyHeader] = []string{key}

			response, err := client.Do(request)

			if err == nil {
				if response != nil && response.Body != nil {
					_ = response.Body.Close()
				}
				t.Fatal("Do() error = nil, want blank idempotency key rejected")
			}
			if response != nil {
				t.Fatalf("Do() response = %#v, want nil", response)
			}
			if got := attempts.Load(); got != 0 {
				t.Fatalf("attempts = %d, want 0", got)
			}
		})
	}
}

// TestRetryStopsAtMaxAttempts keeps the cap honest: without it a persistently
// failing dependency is retried for as long as the caller's budget allows.
func TestRetryStopsAtMaxAttempts(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	client := newRetryTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}), RetryPolicy{MaxAttempts: retryTestMaxAttempts, BaseDelay: retryTestBaseDelay})
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, client.BaseURL(), http.NoBody)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })

	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want the last failure passed through", response.StatusCode)
	}
	if got := attempts.Load(); got != retryTestMaxAttempts {
		t.Fatalf("attempts = %d, want %d", got, retryTestMaxAttempts)
	}
}

func TestRetryCancellationNeverReturnsADrainedResponse(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://provider.example", http.NoBody)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	transport := retryTransport{
		base: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
				Header:     make(http.Header),
				Body: cancelReadCloser{
					cancel: cancel,
				},
			}, nil
		}),
		policy: RetryPolicy{MaxAttempts: 2, BaseDelay: time.Hour},
	}

	response, err := transport.RoundTrip(request)
	if response != nil {
		if response.Body != nil {
			_ = response.Body.Close()
		}
		t.Fatalf("RoundTrip() response = %#v, want nil after its body was drained", response)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RoundTrip() error = %v, want context.Canceled", err)
	}
}

func TestRetryRecordsOneClientSpanPerAttempt(t *testing.T) {
	recorder := telemetrytest.InstallSpanRecorder(t)

	var attempts atomic.Int32
	client := newRetryTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}), RetryPolicy{MaxAttempts: 2, BaseDelay: retryTestBaseDelay})
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, client.BaseURL(), http.NoBody)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do() error = %v, want nil", err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()

	var clientSpans int
	for _, span := range recorder.Ended() {
		if span.SpanKind() == trace.SpanKindClient {
			clientSpans++
		}
	}
	if clientSpans != 2 {
		t.Fatalf("client spans = %d, want one for each of 2 attempts", clientSpans)
	}
}

// TestRetryDoesNotOutliveTheRequestBudget is the rule that keeps a retry from
// holding a handler goroutine, and its in-flight slot, past the deadline of the
// request that asked for it.
func TestRetryDoesNotOutliveTheRequestBudget(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	// The base delay is far larger than the caller's remaining budget, so no retry
	// can fit and the first result must be returned as-is.
	client := newRetryTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}), RetryPolicy{MaxAttempts: retryTestMaxAttempts, BaseDelay: time.Minute})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.BaseURL(), http.NoBody)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	started := time.Now()
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })

	if elapsed := time.Since(started); elapsed > 30*time.Second {
		t.Fatalf("Do() elapsed = %s, want it to give up rather than wait out the backoff", elapsed)
	}
	if got := attempts.Load(); got != 1 {
		t.Fatalf("attempts = %d, want 1 when no retry fits the remaining budget", got)
	}
}

func TestRetryHonorsRetryAfter(t *testing.T) {
	t.Parallel()

	now := time.Date(1994, time.November, 6, 8, 49, 37, 0, time.UTC)
	if got, ok := RetryAfter(&http.Response{Header: http.Header{"Retry-After": []string{"3"}}}, now); !ok || got != 3*time.Second {
		t.Fatalf("retryAfter(3) = (%s, %t), want (3s, true)", got, ok)
	}
	for _, tt := range []struct {
		name string
		raw  string
		want time.Duration
	}{
		{name: "IMF-fixdate", raw: "Sun, 06 Nov 1994 08:49:40 GMT", want: 3 * time.Second},
		{name: "RFC850", raw: "Sunday, 06-Nov-94 08:49:41 GMT", want: 4 * time.Second},
		{name: "asctime", raw: "Sun Nov  6 08:49:42 1994", want: 5 * time.Second},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := RetryAfter(&http.Response{
				Header: http.Header{"Retry-After": []string{tt.raw}},
			}, now); !ok || got != tt.want {
				t.Fatalf("retryAfter(%q) = (%s, %t), want (%s, true)", tt.raw, got, ok, tt.want)
			}
		})
	}

	response := &http.Response{Header: http.Header{
		"Date":        []string{"Wed, 21 Oct 2026 07:27:58 GMT"},
		"Retry-After": []string{"Wed, 21 Oct 2026 07:28:00 GMT"},
	}}
	if got, ok := RetryAfter(response, time.Date(2026, time.October, 21, 8, 28, 0, 0, time.UTC)); !ok || got != 2*time.Second {
		t.Fatalf("retryAfter() with server Date = (%s, %t), want (2s, true)", got, ok)
	}

	for _, raw := range []string{"", "-1", "soon", "Sun, 06 Nov 1994 08:49:36 GMT", "9223372036854775807"} {
		if _, ok := RetryAfter(&http.Response{Header: http.Header{"Retry-After": []string{raw}}}, now); ok {
			t.Fatalf("retryAfter(%q) reported a delay, want it ignored", raw)
		}
	}
}

func TestRetryableResultClassification(t *testing.T) {
	t.Parallel()

	for status, want := range map[int]bool{
		http.StatusOK:                  false,
		http.StatusBadRequest:          false,
		http.StatusInternalServerError: false,
		http.StatusTooManyRequests:     true,
		http.StatusBadGateway:          true,
		http.StatusServiceUnavailable:  true,
		http.StatusGatewayTimeout:      true,
	} {
		if got := retryableResult(&http.Response{StatusCode: status}, nil); got != want {
			t.Fatalf("retryableResult(%d) = %t, want %t", status, got, want)
		}
	}

	// A denied target and an oversized response fail identically forever, so
	// repeating them only wastes the caller's budget.
	if retryableResult(nil, ErrTargetDenied) {
		t.Fatal("retryableResult(ErrTargetDenied) = true, want false")
	}
	if retryableResult(nil, &ResponseTooLargeError{Limit: 1}) {
		t.Fatal("retryableResult(ResponseTooLargeError) = true, want false")
	}
	// profile:outbound-auth-http:start
	if retryableResult(nil, fmt.Errorf("wrapped attempt authorization: %w", &attemptAuthorizationError{err: errors.New("closed failure")})) {
		t.Fatal("retryableResult(attemptAuthorizationError) = true, want false")
	}
	// profile:outbound-auth-http:end
}

func TestNewRejectsIncoherentRetryPolicy(t *testing.T) {
	t.Parallel()

	base := validExternalConfig()

	for name, policy := range map[string]RetryPolicy{
		"attempts without delay": {MaxAttempts: 3},
		"negative attempts":      {MaxAttempts: -1, BaseDelay: time.Second},
		"negative delay":         {MaxAttempts: 2, BaseDelay: -time.Second},
	} {
		cfg := base
		cfg.Retry = policy
		if _, err := New(cfg, nil); err == nil {
			t.Fatalf("New(%s) error = nil, want rejection", name)
		}
	}

	// The zero value is the shipped default and must build.
	if _, err := New(base, nil); err != nil {
		t.Fatalf("New(no retry policy) error = %v, want nil", err)
	}
}

type cancelReadCloser struct {
	cancel context.CancelFunc
}

func (c cancelReadCloser) Read([]byte) (int, error) {
	c.cancel()
	return 0, io.EOF
}

func (cancelReadCloser) Close() error { return nil }
