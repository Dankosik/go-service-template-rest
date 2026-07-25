package httpclient

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const (
	retryTestBaseDelay   = time.Millisecond
	retryTestMaxAttempts = 3
)

// newRetryTestClient builds a client whose fixed target is a private-zone hostname
// and whose dialer reaches the test server, which is how the rest of this package
// exercises a live request through the real bounded transport chain. The retry
// decorator is inside that chain rather than around a bare RoundTripper, so these
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
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, client.BaseURL(), nil)
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

// TestRetryRepeatsKeyedUnsafeMethod is the other half: a caller that attached an
// Idempotency-Key has already made the repeat safe, and the server's own middleware
// will replay rather than re-run.
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

// TestRetryStopsAtMaxAttempts keeps the cap honest: without it a persistently
// failing dependency is retried for as long as the caller's budget allows.
func TestRetryStopsAtMaxAttempts(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	client := newRetryTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}), RetryPolicy{MaxAttempts: retryTestMaxAttempts, BaseDelay: retryTestBaseDelay})
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, client.BaseURL(), nil)
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
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.BaseURL(), nil)
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

func TestRetryHonorsRetryAfterSeconds(t *testing.T) {
	t.Parallel()

	if got, ok := retryAfter(&http.Response{Header: http.Header{"Retry-After": []string{"3"}}}); !ok || got != 3*time.Second {
		t.Fatalf("retryAfter(3) = (%s, %t), want (3s, true)", got, ok)
	}
	// The HTTP-date form is deliberately ignored: honoring it means trusting the
	// client's clock against the server's, and skew turns a one-second hint into a
	// stall measured in minutes.
	if _, ok := retryAfter(&http.Response{
		Header: http.Header{"Retry-After": []string{"Wed, 21 Oct 2026 07:28:00 GMT"}},
	}); ok {
		t.Fatal("retryAfter(HTTP-date) reported a delay, want it ignored")
	}
	for _, raw := range []string{"", "-1", "soon"} {
		if _, ok := retryAfter(&http.Response{Header: http.Header{"Retry-After": []string{raw}}}); ok {
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
