package httpx

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/example/go-service-template-rest/internal/waittest"

	"github.com/example/go-service-template-rest/internal/problem"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type failingMeterProvider struct{ metric.MeterProvider }

func (failingMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter { //nolint:ireturn // Test provider seam.
	return failingMeter{}
}

type failingMeter struct{ metric.Meter }

func (failingMeter) Int64UpDownCounter(string, ...metric.Int64UpDownCounterOption) (metric.Int64UpDownCounter, error) { //nolint:ireturn // Test failure seam.
	return nil, errors.New("instrument failed")
}

func (failingMeter) Int64Counter(string, ...metric.Int64CounterOption) (metric.Int64Counter, error) { //nolint:ireturn // Test failure seam.
	return nil, errors.New("instrument failed")
}

func TestServerLoadPublishesAdmissionMetrics(t *testing.T) {
	t.Parallel()

	reader, provider := telemetrytest.NewManualMeterProvider(t)
	load := newServerLoad(provider)
	release := load.Admitted(t.Context())
	load.Shed(t.Context())

	if got := telemetrytest.Int64SumValue(t, reader, activeRequestsInstrument); got != 1 {
		t.Fatalf("%s = %d, want 1", activeRequestsInstrument, got)
	}
	if got := telemetrytest.Int64SumValue(t, reader, shedRequestsInstrument); got != 1 {
		t.Fatalf("%s = %d, want 1", shedRequestsInstrument, got)
	}
	release()
	if got := telemetrytest.Int64SumValue(t, reader, activeRequestsInstrument); got != 0 {
		t.Fatalf("%s after release = %d, want 0", activeRequestsInstrument, got)
	}
}

func TestServerLoadReportsInstrumentFailures(t *testing.T) {
	previous := otel.GetErrorHandler()
	t.Cleanup(func() { otel.SetErrorHandler(previous) })
	var reported atomic.Int32
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(error) { reported.Add(1) }))

	load := newServerLoad(failingMeterProvider{})
	load.Admitted(t.Context())()
	load.Shed(t.Context())
	if got := reported.Load(); got != 2 {
		t.Fatalf("reported instrument failures = %d, want 2", got)
	}
}

// TestMaxInFlightShedsPastLimitWithoutQueueing is the property the middleware
// exists for: excess load is refused immediately rather than queued until every
// request has spent its whole budget.
func TestMaxInFlightShedsPastLimitWithoutQueueing(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	entered := make(chan struct{}, 1)
	handler := MaxInFlight(1, ServerLoad{}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		entered <- struct{}{}
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))

	var requests sync.WaitGroup
	t.Cleanup(func() {
		close(release)
		requests.Wait()
	})
	requests.Go(func() {
		doRequest(handler, http.MethodGet, "/work")
	})

	waittest.ReceiveSignal(t, entered, 2*time.Second, "first request to reach the handler")

	shed := make(chan *httptest.ResponseRecorder, 1)
	requests.Go(func() { shed <- doRequest(handler, http.MethodGet, "/work") })

	resp := waittest.Receive(t, shed, 2*time.Second, "second request to be shed")
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusServiceUnavailable)
	}
	if got := resp.Header().Get("Retry-After"); got != "1" {
		t.Fatalf("Retry-After = %q, want %q", got, "1")
	}
	assertProblemContentType(t, resp.Header())
	assertProblemCode(t, resp, problem.CodeServiceUnavailable)
}

// TestMaxInFlightReleasesCapacity keeps a shed burst from permanently consuming
// the limiter: a leaked permit would turn one spike into a dead instance.
func TestMaxInFlightReleasesCapacity(t *testing.T) {
	t.Parallel()

	handler := MaxInFlight(1, ServerLoad{}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	handler := MaxInFlight(1, ServerLoad{}, Recover(slog.New(slog.DiscardHandler), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
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
	entered := make(chan struct{}, 1)
	var requests sync.WaitGroup
	t.Cleanup(func() {
		close(release)
		requests.Wait()
	})

	var probes atomic.Int64
	handler := MaxInFlight(1, ServerLoad{}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isHealthProbeRequest(r) {
			probes.Add(1)
			w.WriteHeader(http.StatusOK)
			return
		}
		entered <- struct{}{}
		<-release
	}))

	requests.Go(func() { doRequest(handler, http.MethodGet, "/work") })
	waittest.ReceiveSignal(t, entered, 2*time.Second, "saturating request to reach the handler")

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
	entered := make(chan struct{}, 1)
	var requests sync.WaitGroup
	t.Cleanup(func() {
		close(release)
		requests.Wait()
	})
	handler := MaxInFlight(1, ServerLoad{}, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		entered <- struct{}{}
		<-release
	}))

	requests.Go(func() { doRequest(handler, http.MethodGet, "/work") })
	waittest.ReceiveSignal(t, entered, 2*time.Second, "saturating request to reach the handler")

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
		handler := MaxInFlight(limit, ServerLoad{}, inner)

		resp := doRequest(handler, http.MethodGet, "/work")
		if resp.Code != http.StatusTeapot {
			t.Fatalf("limit %d: status = %d, want %d", limit, resp.Code, http.StatusTeapot)
		}
	}
}

// TestShedResponseIsCorrelatedAndLogged pins Harden's real middleware order:
// correlation and access logging must remain outside shedding.
func TestShedResponseIsCorrelatedAndLogged(t *testing.T) {
	t.Parallel()

	var logged bytes.Buffer
	log := newTestServiceLogger(&logged)

	release := make(chan struct{})
	entered := make(chan struct{}, 1)
	var requests sync.WaitGroup
	t.Cleanup(func() {
		close(release)
		requests.Wait()
	})

	chain, err := Harden(log, telemetry.New(), HardenConfig{
		MaxBodyBytes:   1,
		RequestTimeout: time.Minute,
		MaxInFlight:    1,
	}, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		entered <- struct{}{}
		<-release
	}))
	if err != nil {
		t.Fatalf("Harden() error = %v", err)
	}

	requests.Go(func() { doRequest(chain, http.MethodGet, "/work") })
	waittest.ReceiveSignal(t, entered, 2*time.Second, "saturating request to reach the handler")

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
