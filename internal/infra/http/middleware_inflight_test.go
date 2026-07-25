package httpx

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/problem"
)

// TestMaxInFlightShedsPastLimitWithoutQueueing is the property the middleware
// exists for: excess load is refused immediately rather than queued until every
// request has spent its whole budget.
func TestMaxInFlightShedsPastLimitWithoutQueueing(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	entered := make(chan struct{}, 1)
	handler := MaxInFlight(1, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		entered <- struct{}{}
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))

	var admitted sync.WaitGroup
	admitted.Add(1)
	go func() {
		defer admitted.Done()
		doRequest(handler, http.MethodGet, "/work")
	}()

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("first request never reached the handler")
	}

	shed := make(chan *httptest.ResponseRecorder, 1)
	go func() { shed <- doRequest(handler, http.MethodGet, "/work") }()

	select {
	case resp := <-shed:
		if resp.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusServiceUnavailable)
		}
		if got := resp.Header().Get("Retry-After"); got != "1" {
			t.Fatalf("Retry-After = %q, want %q", got, "1")
		}
		assertProblemContentType(t, resp.Header())
		assertProblemCode(t, resp, problem.CodeServiceUnavailable)
	case <-time.After(2 * time.Second):
		t.Fatal("second request queued instead of being shed")
	}

	close(release)
	admitted.Wait()
}

// TestMaxInFlightReleasesCapacity keeps a shed burst from permanently consuming
// the limiter: a leaked permit would turn one spike into a dead instance.
func TestMaxInFlightReleasesCapacity(t *testing.T) {
	t.Parallel()

	handler := MaxInFlight(1, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for range 50 {
		resp := doRequest(handler, http.MethodGet, "/work")
		if resp.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusNoContent)
		}
	}
}

// TestMaxInFlightReleasesCapacityAfterPanic keeps a panicking handler from
// leaking its permit. Recover runs inside this middleware, so the deferred
// release has to survive the unwind.
func TestMaxInFlightReleasesCapacityAfterPanic(t *testing.T) {
	t.Parallel()

	handler := MaxInFlight(1, Recover(slog.New(slog.DiscardHandler), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})))

	for range 3 {
		resp := doRequest(handler, http.MethodGet, "/work")
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusInternalServerError)
		}
	}
}

// TestMaxInFlightExemptsHealthProbes keeps shedding from evicting an instance for
// being busy, which would invert the whole point of shedding.
func TestMaxInFlightExemptsHealthProbes(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	defer close(release)
	entered := make(chan struct{}, 1)

	var probes atomic.Int64
	handler := MaxInFlight(1, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isHealthProbeRequest(r) {
			probes.Add(1)
			w.WriteHeader(http.StatusOK)
			return
		}
		entered <- struct{}{}
		<-release
	}))

	go doRequest(handler, http.MethodGet, "/work")
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("saturating request never reached the handler")
	}

	for _, path := range healthProbeRoutePaths {
		resp := doRequest(handler, http.MethodGet, path)
		if resp.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d while saturated", path, resp.Code, http.StatusOK)
		}
	}
	if got := probes.Load(); got != int64(len(healthProbeRoutePaths)) {
		t.Fatalf("probe requests served = %d, want %d", got, len(healthProbeRoutePaths))
	}
}

// TestMaxInFlightOnlyExemptsProbeReads keeps the exemption from becoming a
// bypass: a write to a probe path is ordinary traffic and must be shed.
func TestMaxInFlightOnlyExemptsProbeReads(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	defer close(release)
	entered := make(chan struct{}, 1)
	handler := MaxInFlight(1, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		entered <- struct{}{}
		<-release
	}))

	go doRequest(handler, http.MethodGet, "/work")
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("saturating request never reached the handler")
	}

	resp := doRequest(handler, http.MethodPost, "/health/ready")
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusServiceUnavailable)
	}
}

func TestMaxInFlightDisabledWhenNotPositive(t *testing.T) {
	t.Parallel()

	for _, limit := range []int{0, -1} {
		inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTeapot)
		})
		handler := MaxInFlight(limit, inner)

		resp := doRequest(handler, http.MethodGet, "/work")
		if resp.Code != http.StatusTeapot {
			t.Fatalf("limit %d: status = %d, want %d", limit, resp.Code, http.StatusTeapot)
		}
	}
}

// TestShedResponseIsCorrelatedAndLogged pins the middleware's placement in the
// chain rather than its behavior in isolation.
//
// This composes the same order newRouter builds — correlation, access log,
// budget, shedding — because the shipped contract has no non-probe operation to
// saturate through the assembled router: every route it declares is a platform
// probe, and probes are exempt by design. Router-level acceptance of the setting
// is covered by the NewRouter validation table in openapi_contract_test.go.
func TestShedResponseIsCorrelatedAndLogged(t *testing.T) {
	t.Parallel()

	var logged bytes.Buffer
	log := newTestServiceLogger(&logged)

	release := make(chan struct{})
	defer close(release)
	entered := make(chan struct{}, 1)

	chain := RequestCorrelation(
		AccessLog(log, false,
			RequestTimeout(time.Minute,
				MaxInFlight(1, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
					entered <- struct{}{}
					<-release
				})),
			),
		),
	)

	go doRequest(chain, http.MethodGet, "/work")
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("saturating request never reached the handler")
	}

	resp := doRequest(chain, http.MethodGet, "/work")

	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusServiceUnavailable)
	}
	// A shed request is still a request an operator has to be able to find.
	if resp.Header().Get(requestIDHeader) == "" {
		t.Fatal("shed response carries no request id, want correlation applied outside shedding")
	}
	if got := logged.String(); !strings.Contains(got, `"status":503`) {
		t.Fatalf("access log = %q, want a 503 record", got)
	}
}
