# Go concurrency instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - The code uses goroutines, channels, mutexes, condition variables, wait groups, errgroup, worker pools, fan-out/fan-in, or pipelines
  - The task involves cancellation, shutdown, backpressure, or race avoidance
- Do not load when: The code is single-threaded and the task does not involve concurrency or synchronization

## Concurrency design principles

- Do not add concurrency unless it has a clear benefit.
- Prefer designs that make ownership and lifetime obvious.
- Prefer communication and ownership transfer when that model is natural.
- Use shared-memory locking only when it is the simpler and clearer solution.
- Make goroutine lifetime explicit from the start.

## Goroutine lifetime rules

- Never start a goroutine without knowing how it will stop.
- Every goroutine should finish because:
  - its work is complete,
  - its input channel is closed,
  - its context is canceled,
  - or its parent component is shutting down.
- Avoid fire-and-forget goroutines in production code unless their lifetime is intentionally tied to process lifetime and failure is irrelevant.
- On shutdown paths, ensure goroutines can unblock from sends, receives, and waits.

## Cancellation and error propagation

- Use `context.Context` to coordinate cancellation across concurrent work.
- For related goroutines where one failure should cancel the group, prefer `errgroup.WithContext`.
- Use `errgroup.SetLimit` or another explicit mechanism to bound concurrency.
- If only waiting is needed and no error propagation or cancellation is required, `sync.WaitGroup` may be enough.
- Do not build elaborate custom goroutine orchestration when `errgroup` already matches the need.

## Channel rules

- Use channels for data flow, work distribution, signaling, or ownership transfer.
- The sending side should usually own channel closure.
- Receivers should generally not close channels they did not create and send on.
- Do not close a channel from multiple goroutines.
- Only close a channel to signal that no more values will be sent.
- Buffered channels are fine when they model bounded queues or decouple producer and consumer rates, but choose buffer sizes deliberately.

## Select and timeout rules

- Use `select` to combine channel operations with `ctx.Done()` or timeout behavior.
- Ensure sends and receives that may block have a cancellation path where appropriate.
- Prefer context deadlines and timeouts over ad hoc timer logic when the whole operation has a natural context.
- Avoid busy loops and repeated polling when channel or context signals can express the same behavior.

## Shared state rules

- If multiple goroutines mutate shared state, protect it with synchronization or confine the state to one goroutine.
- Keep critical sections small.
- Prefer `sync.Mutex` by default; use `sync.RWMutex` only when measurement or access patterns justify it.
- Do not read and write maps concurrently without synchronization.
- Be deliberate about pointer sharing and mutable aliasing.

## Pipeline and worker-pool guidance

- In pipelines, ensure upstream stages can stop when downstream fails or exits early.
- Avoid goroutine leaks caused by blocked sends to abandoned channels.
- In worker pools, decide who owns queue closure, result collection, and error handling before writing code.
- Use backpressure intentionally instead of letting goroutines accumulate unbounded work.

## Race-safety guidance

- Assume code is unsafe until synchronization makes the happens-before relationship obvious.
- Run `go test -race` for concurrent code.
- Do not rely on "it usually works" timing.
- If correctness depends on scheduling luck, the design is wrong.

## Common anti-patterns to avoid

- Launching goroutines in loops without a shutdown plan
- Blocking forever on send or receive after cancellation
- Closing a channel from the receiver side
- Multiple goroutines racing to close the same channel
- Building unbounded worker pools
- Using channels for everything when a mutex would be simpler
- Using a mutex everywhere when ownership via a channel would make the design clearer
- Ignoring errors from worker goroutines
- Forgetting to stop timers or release resources in long-running concurrent code

## What good output looks like

- Goroutine lifetimes are easy to reason about.
- Cancellation propagates correctly.
- Error handling is integrated with concurrency instead of bolted on later.
- Channels, locks, and contexts each have a clear purpose.
- The design remains readable to a Go reviewer.

## Checklist

Before finalizing, verify that:
- Every goroutine has a completion or cancellation path.
- Blocking operations can unblock on shutdown where needed.
- Channel ownership and closure are unambiguous.
- Related goroutines use `errgroup` when error propagation and cancellation matter.
- Shared state has obvious synchronization.
- The code is suitable for `go test -race`.
