# Goroutine Lifecycle And Cancellation Examples

## When To Load
Load this when a review touches goroutine ownership, `context.Context`, `errgroup`, worker loops, background watchers, pipeline stages, request cancellation, shutdown joins, or early returns that can abandon senders.

## Review Lens
Every goroutine needs an owner, a stop signal, and either join semantics or a clearly accepted process-lifetime reason. Cancellation only helps if the blocking operations observe it.

## Bad Review Example
```text
[medium] workers/loader.go:88
This goroutine might leak. Consider using context.
```

Why it fails: it does not show which send or receive blocks forever, which context is lost, what contract is violated, or how to validate the fix.

## Good Review Example
```text
[high] [go-concurrency-review] workers/loader.go:88
Issue:
Axis: Goroutine Lifecycle And Cancellation; `Start` launches a worker using `context.Background()` instead of the caller's request context, and the loop sends on `results` without a `<-ctx.Done()` arm. If the caller returns after the first error, this goroutine can remain blocked forever on the send.
Impact:
Each canceled request can retain a worker goroutine and the payload reachable from its stack, which makes shutdown wait indefinitely under error fan-out.
Suggested fix:
Derive a child context from the caller's `ctx`, pass that context into all blocking downstream calls, and select the result send against `ctx.Done()`. Join the worker through the existing `WaitGroup` or `errgroup`.
Reference:
Official context docs and Go pipeline cancellation guidance; validate with `go test -race ./internal/workers -run TestLoaderCancelsWorkers -count=100`.
```

## Failure Mode
Write a finding when:
- a request path replaces a caller context with `context.Background()` or `context.TODO()`;
- `errgroup.WithContext` is used but workers do not pass the derived context into blocking calls;
- a pipeline stage can return early without unblocking upstream senders;
- a `CancelFunc` is not called on every control-flow path and the derived context or timer can remain live until its parent is canceled;
- `Close`, `Stop`, or handler return does not prove that started goroutines have exited.

## Smallest Safe Correction
Prefer corrections like:
- thread the existing `ctx` into the goroutine and every blocking operation;
- `defer cancel()` as soon as a derived context is created;
- add a `<-ctx.Done()` case around sends, receives, waits, and retry sleeps;
- use `errgroup.WithContext` plus workers that actually observe the derived context;
- close a shared done channel from the owner to broadcast cancellation to all pipeline stages;
- record explicit process-lifetime ownership when a goroutine is intentionally not joined.

## Validation Evidence
Use deterministic cancellation and liveness checks:
```bash
go test -race ./internal/workers -run TestLoaderCancelsWorkers -count=100
go test ./internal/workers -run TestLoaderStopReturns -count=100 -timeout=5s
```

A race-clean test does not prove there is no goroutine leak. Prefer a test that cancels before all sends complete and asserts `Stop` or `Wait` returns.

## Source Links From Exa
- [context package docs](https://pkg.go.dev/context)
- [Go Concurrency Patterns: Context](https://go.dev/blog/context)
- [Go Concurrency Patterns: Pipelines and cancellation](https://go.dev/blog/pipelines)
- [The Go Memory Model](https://go.dev/ref/mem)

