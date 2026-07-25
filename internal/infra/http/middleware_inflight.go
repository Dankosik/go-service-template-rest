package httpx

import (
	"net/http"
	"slices"
	"strconv"
	"time"

	"golang.org/x/sync/semaphore"
)

// shedRetryAfter is the Retry-After hint on a shed request. It is short on
// purpose: shedding means the server is momentarily past capacity, not down, and
// a long hint turns a brief spike into a client-side outage.
const shedRetryAfter = time.Second

// MaxInFlight bounds how many requests may execute a handler concurrently, and
// rejects the rest immediately.
//
// RequestTimeout is not backpressure. It bounds how long one request may run,
// not how many may run, so an arrival spike past the capacity of a finite
// downstream resource does not get rejected — it queues. Every queued request
// holds a goroutine and waits, so latency for requests that would have been
// served in milliseconds rises to the full budget and the endpoint returns 504
// across the board instead of serving the subset it has capacity for. Worse, the
// instance now looks unhealthy, so it gets evicted mid-spike.
//
// Shedding at the door is what keeps admitted requests fast: a request that
// cannot get capacity now is better told so in a millisecond than told so after
// the whole budget has been spent waiting.
//
// Platform probe routes are exempt. Shedding a readiness probe would evict the
// instance for being busy, which is the opposite of what shedding is for.
func MaxInFlight(limit int, next http.Handler) http.Handler {
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
			w.Header().Set("Retry-After", retryAfter)
			writeProblem(w, r, problemResponse{
				code:   problemCodeServiceUnavailable,
				detail: "server is at capacity",
			})
			return
		}
		defer sem.Release(1)

		next.ServeHTTP(w, r)
	})
}

// isHealthProbeRequest matches probe routes by raw path rather than by routed
// template, because this middleware runs before the generated router has matched
// anything. The path set is the same one AccessLog excludes.
func isHealthProbeRequest(r *http.Request) bool {
	if r == nil || r.Method != http.MethodGet {
		return false
	}
	return slices.Contains(healthProbeRoutePaths, r.URL.Path)
}
