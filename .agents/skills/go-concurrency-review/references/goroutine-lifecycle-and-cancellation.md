# Goroutine Lifecycle And Cancellation

Behavior Change Thesis: When loaded for goroutine, context, or early-return symptoms, this file makes the model require an owner, stop signal, and join or accepted abandonment story instead of giving the vague advice to "use context" or assuming `errgroup` cancels work by itself.

## When To Load
Symptom: the diff starts goroutines, uses `context.Context`, `errgroup`, worker loops, background watchers, pipelines, request cancellation, shutdown joins, or early returns that can abandon upstream senders.

## Decision Rubric
- For every goroutine, identify the owner, stop trigger, and join path. If it is intentionally process-lifetime, the code or surrounding contract should say so.
- Context only helps when every blocking operation in the goroutine observes the derived context.
- `errgroup.WithContext` is not enough by itself; workers must pass the derived context into downstream calls and select blocking sends/receives against it.
- Early return from one pipeline stage must unblock upstream senders or drain intentionally.
- Creating a `CancelFunc` creates a lifecycle obligation; call it on every path unless the parent cancellation is the documented owner.
- Handler/request code that replaces the caller context with `context.Background()` or `context.TODO()` usually severs cancellation and deadline ownership.

## Imitate
```text
[high] [go-concurrency-review] workers/loader.go:88
Issue:
Axis: Goroutine Lifecycle And Cancellation; `Start` launches a worker using `context.Background()` instead of the caller's request context, and the loop sends on `results` without a `<-ctx.Done()` arm. If the caller returns after the first error, this goroutine can remain blocked forever on the send.
Impact:
Each canceled request can retain a worker goroutine and the payload reachable from its stack, and shutdown can wait indefinitely under error fan-out.
Suggested fix:
Derive a child context from the caller's `ctx`, pass that context into all blocking downstream calls, and select the result send against `ctx.Done()`. Join the worker through the existing `WaitGroup` or `errgroup`.
Reference:
Validate with `go test -race ./internal/workers -run TestLoaderCancelsWorkers -count=100` plus a liveness check that `Stop` or `Wait` returns after cancellation.
```

Copy the shape: it names the lost owner, the exact blocking operation, the leak mechanism, and the smallest cancellation plus join correction.

## Reject
```text
[medium] workers/loader.go:88
This goroutine might leak. Consider using context.
```

Reject this shape: it does not prove where the goroutine blocks, which context is lost, which shutdown contract breaks, or how to validate the fix.

```text
No issue: `errgroup.WithContext` is used, so siblings stop on error.
```

Reject this shape unless each sibling actually observes the derived context while blocked or doing downstream work.

## Agent Traps
- Do not equate "goroutine exits when channel closes" with proof that the channel is always closed.
- Do not miss sender leaks when reviewing receiver-side early returns.
- Do not accept process-lifetime goroutines in request-scoped code without a clear ownership boundary.
- Do not recommend a context parameter without also routing it into sends, receives, waits, sleeps, retries, and I/O.

## Validation Shape
- Prefer deterministic cancellation tests that park work at the risky blocking point, cancel, then assert `Wait`, `Stop`, or the handler returns.
- Use race evidence for shared state touched by the goroutine, but add a liveness test for leaks and shutdown hangs.
- Good commands look like `go test -race ./internal/workers -run TestLoaderCancelsWorkers -count=100` and `go test ./internal/workers -run TestLoaderStopReturns -count=100 -timeout=5s`.
