# Bounded Work And Backpressure

Behavior Change Thesis: When loaded for worker-pool, fan-out, queue, semaphore, or detached-sender symptoms, this file makes the model prove both execution width and queued work are bounded instead of accepting a worker pool, buffered channel, or semaphore-looking code as automatically safe.

## When To Load
Symptom: the diff launches goroutines per item, adds worker pools, `errgroup.SetLimit`, semaphore limiting, buffered job/result channels, async send wrappers, retry queues, or producer/consumer backpressure behavior.

## Decision Rubric
- Count both active work and waiting work. A fixed worker count with a request-sized input slice, retry list, or detached sender goroutine can still retain unbounded work relative to production inputs.
- Acquire semaphores before launching goroutines. Acquiring inside the goroutine limits only the critical section, not goroutine creation.
- `errgroup.SetLimit` can bound running functions, but only if all work is launched through that group and blocked submitters have a cancellation story, usually because active workers observe `ctx` and return promptly.
- Do not modify an `errgroup` limit while goroutines are active; treat dynamic limit changes as a correctness or panic risk, not a tuning tweak.
- Buffered channels need an explicit full-queue policy: block with cancellation, drop with accounting, fail fast, or shed upstream.
- Do not hide backpressure by spawning a goroutine just to avoid a blocking send; that creates an unbounded goroutine queue under slow consumers.
- If the correct policy is overload behavior rather than local concurrency mechanics, record a handoff to reliability review while still flagging the local unbounded-work defect.

## Imitate
```text
[high] [go-concurrency-review] importer/run.go:74
Issue:
Axis: Bounded Concurrency And Backpressure; the loop starts one goroutine per job and acquires `sem` inside the goroutine. Under a large import, all jobs still allocate goroutines and retain their payloads while waiting for the semaphore, so the semaphore does not bound goroutine or memory growth.
Impact:
A large tenant import can exhaust memory or delay shutdown even though only `limit` jobs run the critical section at once.
Suggested fix:
Acquire the semaphore before launching the goroutine and release it in the goroutine, or use `errgroup.SetLimit`/a fixed worker pool where submission observes `ctx.Done()`. Make the full-queue behavior explicit if producers can outpace workers.
Reference:
Validate with `go test -race ./internal/importer -run TestImportFanoutBounded -count=100` and a cancellation test that proves blocked submitters return.
```

Copy the shape: it counts the resource that is actually unbounded and fixes the submission boundary, not just the worker body.

## Reject
```text
No issue: the code uses a semaphore, so only 10 jobs run at once.
```

Reject this shape when the semaphore is acquired after goroutine launch; the running critical section is bounded, but goroutine count and retained queued work are not.

```go
go func() {
    results <- result
}()
```

Reject this as a backpressure fix unless the goroutine is itself bounded and canceled. It often turns one blocked send into an unbounded goroutine backlog.

## Agent Traps
- Do not count only workers; count pending jobs, result buffers, retry buffers, and detached goroutines too.
- Do not require dropping work by default. Blocking with cancellation may be the correct local fix when the caller already owns backpressure.
- Do not miss cancellation while submitting to a bounded pool; a producer blocked on `jobs <- item` can leak just like a worker blocked on result send.
- Do not collapse overload policy into concurrency review. Name the local defect, then hand off global shed/drop/retry policy when needed.

## Validation Shape
- Add tests that exceed the limit and prove active workers never exceed it.
- Add cancellation tests that block submission or result delivery, cancel, then assert the producer and workers return.
- Use race evidence when bounded counters or shared queues are touched concurrently.
- Good commands look like `go test -race ./internal/importer -run TestImportFanoutBounded -count=100` and `go test ./internal/importer -run TestCancelUnblocksSubmitters -count=100 -timeout=5s`.
