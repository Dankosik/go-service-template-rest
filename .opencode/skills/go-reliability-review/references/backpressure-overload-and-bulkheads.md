# Backpressure, Overload, And Bulkheads

## Behavior Change Thesis
When loaded for symptom `fan-out, queue, limiter, worker pool, rate limit, circuit breaker, or bulkhead changed`, this file makes the model ask how work is bounded, rejected, or isolated instead of likely mistake `accept more goroutines, larger queues, or wait-until-timeout behavior as capacity handling`.

## When To Load
Load when a Go diff touches fan-out, worker pools, semaphores, queues, buffered channels, rate limits, overload responses, circuit breakers, bulkheads, dependency-specific pools, or a path that can accumulate work faster than it processes it.

Keep findings local: ask for the changed path to bound resource use and reject or degrade predictably. Hand off throughput benchmarking to `go-performance-review`, goroutine coordination depth to `go-concurrency-review`, and architecture-wide overload policy to `go-reliability-spec` or `go-design-spec`.

## Decision Rubric
- A request loops over inputs and starts one goroutine per item with no cap.
- A new queue, slice backlog, or buffered channel has no maximum, no drop policy, or no producer backpressure.
- All dependencies share one worker pool, so a slow optional dependency can starve critical work.
- The code waits indefinitely for queue capacity instead of honoring `ctx.Done()`.
- Overload returns generic `500` or blocks until timeout instead of a deliberate overload signal.
- A circuit breaker is added without clear fallback, observability, or half-open limits.
- Queue-based load leveling is introduced, but consumers can autoscale until they overwhelm the target anyway.
- Bulkhead partitioning is based on convenience rather than the failing dependency or consumer class.

## Imitate

Bad finding shape to copy: the defect is local amplification, not "could be optimized."

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

```text
[high] [go-reliability-review] internal/cache/refresh.go:31
Issue: The changed refresh path starts one remote call per id without a concurrency cap or overload response.
Impact: A large request or slow dependency can create a local fan-out storm, exhaust sockets/goroutines, and amplify downstream overload.
Suggested fix: Use the repository's bounded worker/semaphore pattern, honor ctx while acquiring capacity, and return a deliberate overload or partial result when capacity is unavailable.
Reference: Google SRE overload guidance and Azure Bulkhead pattern.
```

Good correction shape: use a local capacity guard and let the caller's context stop waiting.

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

Copy the review move: suggest the repo's existing worker/semaphore/limiter pattern, not a fresh concurrency framework.

## Reject

```go
for _, item := range items {
	go s.callRemote(ctx, item)
}
```

Reject because request size can create unbounded concurrent downstream work and local goroutine pressure.

```go
jobs <- job
```

Reject when the send can block forever or until the caller's whole deadline is burned. Require bounded enqueue, `ctx.Done()`, or an explicit full-queue path.

```go
var shared = make(chan func(), 1000) // used by billing, search, and recommendations
```

Reject when unrelated dependency classes can starve each other; the bulkhead should usually follow the failing resource or caller class.

## Agent Traps
- Do not equate a buffered channel with backpressure unless the full-buffer behavior is explicit.
- Do not assume a queue helps reliability; it can hide overload until latency is already unrecoverable.
- Do not recommend larger pools or buffers without capacity evidence; that often moves the failure downstream.
- Do not duplicate concurrency ownership. If the fix depends on goroutine join, channel close, or shared limiter correctness, hand off to `go-concurrency-review`.
- Do not treat circuit breakers as a complete fix unless open, half-open, fallback, and observability behavior are bounded.

## Validation Shape
- `go test ./... -run 'Test.*(Overload|Backpressure|Bulkhead|QueueFull|RateLimit|Circuit)'`
- `go test ./... -run 'Test.*(Fanout|ConcurrencyLimit|WorkerPool)'`
- `go test -race ./...` for shared limiter or breaker state.
- `go test ./... -bench 'Benchmark.*(Overload|Fanout|Queue|Worker)' -run '^$'` when the repo already has benchmarks for the changed path.

Use deterministic tests that fill the queue or limiter and assert the call returns promptly.
