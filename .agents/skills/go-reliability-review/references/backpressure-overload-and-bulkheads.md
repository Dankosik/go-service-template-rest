# Backpressure, Overload, And Bulkheads

## When To Load
Load this reference when a Go diff touches fan-out, worker pools, semaphores, queues, buffered channels, rate limits, overload responses, circuit breakers, bulkheads, dependency-specific pools, or a path that can accumulate work faster than it processes it.

Keep findings local: ask for the changed path to bound resource use and reject or degrade predictably. Hand off throughput benchmarking to `go-performance-review`, goroutine coordination depth to `go-concurrency-review`, and architecture-wide overload policy to `go-reliability-spec` or `go-design-spec`.

## Review Smells
- A request loops over inputs and starts one goroutine per item with no cap.
- A new queue, slice backlog, or buffered channel has no maximum, no drop policy, or no producer backpressure.
- All dependencies share one worker pool, so a slow optional dependency can starve critical work.
- The code waits indefinitely for queue capacity instead of honoring `ctx.Done()`.
- Overload returns generic `500` or blocks until timeout instead of a deliberate overload signal.
- A circuit breaker is added without clear fallback, observability, or half-open limits.
- Queue-based load leveling is introduced, but consumers can autoscale until they overwhelm the target anyway.
- Bulkhead partitioning is based on convenience rather than the failing dependency or consumer class.

## Failure Modes
- A traffic spike creates unbounded goroutines, memory growth, and scheduler pressure.
- A noncritical dependency outage consumes the shared worker pool and breaks critical requests.
- A queue hides overload until latency is already outside caller deadlines.
- Rejections cost almost as much as successful work and still overload the backend.
- Half-open recovery floods a recovering dependency and reopens the failure.

## Review Examples

Bad: unbounded fan-out and no overload path.

```go
func (s *Service) Refresh(ctx context.Context, ids []string) error {
	errs := make(chan error, len(ids))
	for _, id := range ids {
		go func(id string) {
			errs <- s.remote.Refresh(ctx, id)
		}(id)
	}
	for range ids {
		if err := <-errs; err != nil {
			return err
		}
	}
	return nil
}
```

Review finding shape:

```text
[high] [go-reliability-review] internal/cache/refresh.go:31
Issue: The changed refresh path starts one remote call per id without a concurrency cap or overload response.
Impact: A large request or slow dependency can create a local fan-out storm, exhaust sockets/goroutines, and amplify downstream overload.
Suggested fix: Use the repository's bounded worker/semaphore pattern, honor ctx while acquiring capacity, and return a deliberate overload or partial result when capacity is unavailable.
Reference: Google SRE overload guidance and Azure Bulkhead pattern.
```

Good: use a local capacity guard and let the caller's context stop waiting.

```go
func (s *Service) RefreshOne(ctx context.Context, id string) error {
	select {
	case s.refreshSlots <- struct{}{}:
		defer func() { <-s.refreshSlots }()
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrOverloaded
	}

	return s.remote.Refresh(ctx, id)
}
```

This is a local review example, not a required implementation pattern. If the smallest fix still needs fan-out, goroutine joins, or shared limiter state, hand off that depth to `go-concurrency-review`.

## Smallest Safe Fix
- Cap fan-out with an existing worker pool, semaphore, or per-dependency limiter.
- Add a producer-side queue bound and explicit full-queue behavior: reject, shed, degrade, or block only within caller deadline.
- Split critical and noncritical dependency calls into separate pools or limiters when one can starve the other.
- Surface overload with a deliberate error/status that callers can handle without retrying blindly.
- Make circuit-breaker open and half-open states bounded and observable.
- If a queue is added, ensure the consumer rate cannot simply move overload to the downstream resource.

## Validation Commands
- `go test ./... -run 'Test.*(Overload|Backpressure|Bulkhead|QueueFull|RateLimit|Circuit)'`
- `go test ./... -run 'Test.*(Fanout|ConcurrencyLimit|WorkerPool)'`
- `go test -race ./...` for shared limiter or breaker state.
- `go test ./... -bench 'Benchmark.*(Overload|Fanout|Queue|Worker)' -run '^$'` when the repo already has benchmarks for the changed path.

Use deterministic tests that fill the queue or limiter and assert the call returns promptly.

## Exa Source Links
- Google SRE Handling Overload: https://sre.google/sre-book/handling-overload/
- Google SRE Addressing Cascading Failures: https://sre.google/sre-book/addressing-cascading-failures/
- Azure Bulkhead pattern: https://learn.microsoft.com/en-us/azure/architecture/patterns/bulkhead
- Azure Queue-Based Load Leveling pattern: https://learn.microsoft.com/en-us/azure/architecture/patterns/queue-based-load-leveling
- Azure Circuit Breaker pattern: https://learn.microsoft.com/en-us/azure/architecture/patterns/circuit-breaker
