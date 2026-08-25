package httpx

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

const rateLimitTestHeader = "X-Api-Key"

func TestRateLimitRejectsOverBudgetCallerWithRetryAfter(t *testing.T) {
	t.Parallel()

	// One request per second with no burst headroom past the first, so the second
	// request in the same instant is over budget.
	limiter := mustNewKeyedRateLimiter(t, 1, 1, 8)
	handler := RateLimit(limiter, HeaderRateLimitKey(rateLimitTestHeader), okHandler())

	first := doRateLimitedRequest(handler, "caller-a")
	if first.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want %d", first.Code, http.StatusOK)
	}

	second := doRateLimitedRequest(handler, "caller-a")
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}
	retryAfter, err := strconv.Atoi(second.Header().Get("Retry-After"))
	if err != nil || retryAfter < 1 {
		t.Fatalf("Retry-After = %q, want a positive whole number of seconds", second.Header().Get("Retry-After"))
	}
	if body := second.Body.String(); !strings.Contains(body, `"too_many_requests"`) {
		t.Fatalf("body = %q, want the too_many_requests problem", body)
	}
}

// TestRateLimitIsolatesCallers is the property the seam exists for: one caller
// over budget must not cost every other caller their requests, which is exactly
// what global shedding does.
func TestRateLimitIsolatesCallers(t *testing.T) {
	t.Parallel()

	limiter := mustNewKeyedRateLimiter(t, 1, 1, 8)
	handler := RateLimit(limiter, HeaderRateLimitKey(rateLimitTestHeader), okHandler())

	_ = doRateLimitedRequest(handler, "noisy")
	if got := doRateLimitedRequest(handler, "noisy"); got.Code != http.StatusTooManyRequests {
		t.Fatalf("noisy caller status = %d, want %d", got.Code, http.StatusTooManyRequests)
	}
	if got := doRateLimitedRequest(handler, "quiet"); got.Code != http.StatusOK {
		t.Fatalf("quiet caller status = %d, want %d", got.Code, http.StatusOK)
	}
}

// TestRateLimitLeavesUnkeyedRequestsAlone keeps a request the key function cannot
// attribute from being refused, which would turn an unset header into an outage.
func TestRateLimitLeavesUnkeyedRequestsAlone(t *testing.T) {
	t.Parallel()

	limiter := mustNewKeyedRateLimiter(t, 1, 1, 8)
	handler := RateLimit(limiter, HeaderRateLimitKey(rateLimitTestHeader), okHandler())

	for range 5 {
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/resource", nil))
		if resp.Code != http.StatusOK {
			t.Fatalf("unkeyed request status = %d, want %d", resp.Code, http.StatusOK)
		}
	}
}

// TestRateLimitExemptsHealthProbes keeps an orchestrator's own polling from
// evicting the instance it is polling.
func TestRateLimitExemptsHealthProbes(t *testing.T) {
	t.Parallel()

	limiter := mustNewKeyedRateLimiter(t, 1, 1, 8)
	handler := RateLimit(limiter, HeaderRateLimitKey(rateLimitTestHeader), okHandler())

	for range 10 {
		resp := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health/ready", nil)
		request.Header.Set(rateLimitTestHeader, "prober")
		handler.ServeHTTP(resp, request)
		if resp.Code != http.StatusOK {
			t.Fatalf("probe status = %d, want %d", resp.Code, http.StatusOK)
		}
	}
}

func TestRateLimitIsOmittedWithoutALimiter(t *testing.T) {
	t.Parallel()

	// Asserted by behavior rather than by identity: http.HandlerFunc is a func
	// type and therefore uncomparable.
	handler := RateLimit(nil, HeaderRateLimitKey(rateLimitTestHeader), okHandler())
	for attempt := range 5 {
		if got := doRateLimitedRequest(handler, "caller-a"); got.Code != http.StatusOK {
			t.Fatalf("request %d status = %d, want the middleware left out of the chain", attempt+1, got.Code)
		}
	}
}

func TestRateLimitPanicsForLimiterWithoutKey(t *testing.T) {
	t.Parallel()

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("RateLimit() panic = nil, want a limiter without a key to fail during construction")
		}
	}()

	RateLimit(mustNewKeyedRateLimiter(t, 1, 1, 8), nil, okHandler())
}

// TestHeaderRateLimitKeyDoesNotLeakTheCredential keeps the value that identifies
// a caller — usually their credential — out of the bucket key, and therefore out
// of any heap dump or debug log that key reaches.
func TestHeaderRateLimitKeyDoesNotLeakTheCredential(t *testing.T) {
	t.Parallel()

	const secret = "super-secret-api-key"
	key := HeaderRateLimitKey(rateLimitTestHeader)

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/resource", nil)
	request.Header.Set(rateLimitTestHeader, secret)

	got := key(request)
	if got == "" {
		t.Fatal("key is empty for a present header")
	}
	if strings.Contains(got, secret) {
		t.Fatalf("key %q contains the credential", got)
	}
	if key(httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/resource", nil)) != "" {
		t.Fatal("key for an absent header is not empty")
	}
}

// TestKeyedRateLimiterBoundsItsKeyMap is what separates this from the unbounded
// map[string]*rate.Limiter every service writes: keys come from callers, so an
// unbounded map is a memory leak with a rate limiter attached.
func TestKeyedRateLimiterBoundsItsKeyMap(t *testing.T) {
	t.Parallel()

	const maxKeys = 16
	limiter := mustNewKeyedRateLimiter(t, 1000, 1000, maxKeys)

	for i := range 10 * maxKeys {
		limiter.Allow(context.Background(), "caller-"+strconv.Itoa(i))
	}

	limiter.mu.Lock()
	live := len(limiter.current) + len(limiter.previous)
	limiter.mu.Unlock()

	if live > 2*maxKeys {
		t.Fatalf("tracked keys = %d, want at most two generations of %d", live, maxKeys)
	}
}

func TestKeyedRateLimiterBoundsPromotionsIntoAFullGeneration(t *testing.T) {
	t.Parallel()

	const maxKeys = 8
	limiter := mustNewKeyedRateLimiter(t, 1000, 1000, maxKeys)

	for i := range maxKeys {
		limiter.Allow(context.Background(), "old-"+strconv.Itoa(i))
	}
	for i := range maxKeys {
		limiter.Allow(context.Background(), "current-"+strconv.Itoa(i))
	}
	for i := range maxKeys {
		limiter.Allow(context.Background(), "old-"+strconv.Itoa(i))
	}

	limiter.mu.Lock()
	live := len(limiter.current) + len(limiter.previous)
	limiter.mu.Unlock()

	if live > 2*maxKeys {
		t.Fatalf("tracked keys after promotion = %d, want at most two generations of %d", live, maxKeys)
	}
}

// TestKeyedRateLimiterKeepsActiveCallersAcrossGenerations keeps a caller that is
// still sending from having its debt forgiven every time the map rolls over,
// which would make the limit unenforceable under key churn.
func TestKeyedRateLimiterKeepsActiveCallersAcrossGenerations(t *testing.T) {
	t.Parallel()

	const maxKeys = 4
	limiter := mustNewKeyedRateLimiter(t, 1, 1, maxKeys)

	if allowed, _ := limiter.Allow(context.Background(), "steady"); !allowed {
		t.Fatal("first request for the steady caller was refused")
	}
	// Churn enough distinct keys to roll the generation over twice.
	for i := range 3 * maxKeys {
		limiter.Allow(context.Background(), "churn-"+strconv.Itoa(i))
		if allowed, _ := limiter.Allow(context.Background(), "steady"); allowed {
			t.Fatalf("steady caller was admitted again after %d churned keys, want its bucket preserved", i+1)
		}
	}
}

func TestNewKeyedRateLimiterRejectsUnusableSettings(t *testing.T) {
	t.Parallel()

	// Rejected rather than answered with a nil limiter: a nil *KeyedRateLimiter in
	// a RateLimiter interface is a non-nil interface value, so the middleware
	// would install itself and panic on the first keyed request.
	if _, err := NewKeyedRateLimiter(0, 1, 8); err == nil {
		t.Fatal("NewKeyedRateLimiter(perSecond=0) error = nil, want a rejection")
	}
	if _, err := NewKeyedRateLimiter(math.NaN(), 1, 8); err == nil {
		t.Fatal("NewKeyedRateLimiter(perSecond=NaN) error = nil, want a rejection")
	}
	if _, err := NewKeyedRateLimiter(1, 0, 8); err == nil {
		t.Fatal("NewKeyedRateLimiter(burst=0) error = nil, want a rejection")
	}
	limiter, err := NewKeyedRateLimiter(1, 1, 0)
	if err != nil || limiter == nil {
		t.Fatalf("NewKeyedRateLimiter(maxKeys=0) = %v, %v; want the default bound", limiter, err)
	}
	if limiter.maxKeys != defaultRateLimitMaxKeys {
		t.Fatalf("maxKeys = %d, want the default %d", limiter.maxKeys, defaultRateLimitMaxKeys)
	}
}

func mustNewKeyedRateLimiter(tb testing.TB, perSecond float64, burst, maxKeys int) *KeyedRateLimiter {
	tb.Helper()

	limiter, err := NewKeyedRateLimiter(perSecond, burst, maxKeys)
	if err != nil {
		tb.Fatalf("NewKeyedRateLimiter() error = %v", err)
	}
	return limiter
}

func TestKeyedRateLimiterIsSafeUnderConcurrentKeys(t *testing.T) {
	t.Parallel()

	limiter := mustNewKeyedRateLimiter(t, 1000, 1000, 32)

	var waitGroup sync.WaitGroup
	for worker := range 8 {
		waitGroup.Go(func() {
			for i := range 200 {
				limiter.Allow(context.Background(), "w"+strconv.Itoa(worker)+"-"+strconv.Itoa(i%64))
			}
		})
	}
	waitGroup.Wait()
}

func TestKeyedRateLimiterConcurrentRejectionsDoNotConsumeBudget(t *testing.T) {
	previousMaxProcs := runtime.GOMAXPROCS(16)
	t.Cleanup(func() { runtime.GOMAXPROCS(previousMaxProcs) })

	synctest.Test(t, func(t *testing.T) {
		limiter := mustNewKeyedRateLimiter(t, 1, 1, 8)
		if allowed, _ := limiter.Allow(t.Context(), "caller"); !allowed {
			t.Fatal("first request was refused")
		}

		start := make(chan struct{})
		results := make(chan bool, 64)
		var waitGroup sync.WaitGroup
		for range cap(results) {
			waitGroup.Go(func() {
				<-start
				allowed, _ := limiter.Allow(t.Context(), "caller")
				results <- allowed
			})
		}
		close(start)
		waitGroup.Wait()
		close(results)
		for allowed := range results {
			if allowed {
				t.Fatal("concurrent over-budget request was admitted")
			}
		}

		if _, retryAfter := limiter.Allow(t.Context(), "caller"); retryAfter > time.Second {
			t.Fatalf("Retry-After after rejected requests = %s, want at most one token interval", retryAfter)
		}
		time.Sleep(time.Second)
		if allowed, _ := limiter.Allow(t.Context(), "caller"); !allowed {
			t.Fatal("rejected requests consumed budget")
		}
	})
}

func TestRouterRejectsARateLimiterWithoutAKey(t *testing.T) {
	t.Parallel()

	_, err := Harden(newTestServiceLogger(nil), telemetry.New(), HardenConfig{
		MaxBodyBytes:   1 << 10,
		RequestTimeout: time.Second,
		RateLimit:      mustNewKeyedRateLimiter(t, 1, 1, 8),
	}, okHandler())
	if err == nil {
		t.Fatal("Harden() error = nil, want a rate limiter without a key to be rejected")
	}
	if !strings.Contains(err.Error(), "rate limit key") {
		t.Fatalf("error = %q, want it to name the missing key", err.Error())
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func doRateLimitedRequest(handler http.Handler, caller string) *httptest.ResponseRecorder {
	resp := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/resource", nil)
	request.Header.Set(rateLimitTestHeader, caller)
	handler.ServeHTTP(resp, request)
	return resp
}
