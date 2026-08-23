package httpx

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/example/go-service-template-rest/internal/failure"
	"github.com/example/go-service-template-rest/internal/problem"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/sync/semaphore"
)

// shedRetryAfter is the Retry-After hint on a shed request. It is short on
// purpose: shedding means the server is momentarily past capacity, not down, and
// a long hint turns a brief spike into a client-side outage.
const shedRetryAfter = time.Second

const (
	serverMeterName           = "service.http.server"
	activeRequestsInstrument  = "http.server.active_requests"
	shedRequestsInstrument    = "http.server.shed_requests"
	requestInstrumentUnit     = "{request}"
	activeRequestsDescription = "Requests currently executing a handler, against the http.max_in_flight limit."
	shedRequestsDescription   = "Requests rejected without running a handler because the in-flight limit was reached."
)

// ServerLoad records what this transport's admission limiter did. Its zero
// value is safe for focused middleware tests and builds without telemetry.
type ServerLoad struct {
	active metric.Int64UpDownCounter
	shed   metric.Int64Counter
}

func newServerLoad(provider metric.MeterProvider) ServerLoad {
	meter := provider.Meter(serverMeterName)
	active, err := meter.Int64UpDownCounter(
		activeRequestsInstrument,
		metric.WithDescription(activeRequestsDescription),
		metric.WithUnit(requestInstrumentUnit),
	)
	if err != nil {
		otel.Handle(fmt.Errorf("create %s metric: %w", activeRequestsInstrument, err))
	}
	shed, err := meter.Int64Counter(
		shedRequestsInstrument,
		metric.WithDescription(shedRequestsDescription),
		metric.WithUnit(requestInstrumentUnit),
	)
	if err != nil {
		otel.Handle(fmt.Errorf("create %s metric: %w", shedRequestsInstrument, err))
	}
	return ServerLoad{active: active, shed: shed}
}

func (l ServerLoad) Admitted(ctx context.Context) func() {
	if l.active == nil {
		return func() {}
	}
	l.active.Add(ctx, 1)
	return func() { l.active.Add(ctx, -1) }
}

func (l ServerLoad) Shed(ctx context.Context) {
	if l.shed != nil {
		l.shed.Add(ctx, 1)
	}
}

// MaxInFlight bounds how many requests may execute a handler concurrently, and
// rejects the rest immediately.
//
// RequestTimeout is not backpressure: it bounds how long one request may run, not
// how many, so an arrival spike past a finite downstream resource queues instead
// of being rejected. Every queued request holds a goroutine, so latency rises to
// the full budget and the endpoint returns 504 across the board instead of
// serving the subset it has capacity for — and the instance then looks unhealthy
// and gets evicted mid-spike. Shedding at the door is what keeps admitted
// requests fast.
//
// load records what the limiter did. Without it the limit is unobservable: a shed
// request is one more 503 on a route that also answers 503 when the connection
// pool saturates, and nothing reports how close the service runs to the limit —
// so http.max_in_flight is set once from a guess and never revisited.
//
// Platform probe routes are exempt. Shedding a readiness probe would evict the
// instance for being busy, which is the opposite of what shedding is for.
func MaxInFlight(limit int, load ServerLoad, next http.Handler) http.Handler {
	if limit <= 0 {
		return next
	}

	sem := semaphore.NewWeighted(int64(limit))
	retryAfter := strconv.Itoa(int(shedRetryAfter.Seconds()))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isHealthProbeRequest(r) {
			next.ServeHTTP(w, r)
			return
		}
		// TryAcquire rather than Acquire: blocking here would rebuild the queue
		// this middleware exists to prevent, just one layer further out.
		if !sem.TryAcquire(1) {
			load.Shed(r.Context())
			w.Header().Set("Retry-After", retryAfter)
			writeProblem(w, r, problemResponse{
				code:   problem.CodeServiceUnavailable,
				detail: failure.AtCapacityDetail,
			})
			return
		}
		defer sem.Release(1)

		// Recorded around the handler rather than around the whole middleware, so
		// the gauge measures occupancy of the limit and not the shed path.
		released := load.Admitted(r.Context())
		defer released()

		next.ServeHTTP(w, r)
	})
}
