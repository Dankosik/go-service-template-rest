# Reference Selector

State how the selected reference changes the review judgment.

| Symptom | Load | Behavior change |
| --- | --- | --- |
| Visibility, readiness flags, atomics, immutable snapshots, or missing publication edge. | [happens-before-and-publication.md](happens-before-and-publication.md) | Require a real synchronization edge or immutable snapshot instead of goroutine-order intuition. |
| Fire-and-forget work, lost context, early-return leaks, `errgroup`, or shutdown join. | [goroutine-lifecycle-and-cancellation.md](goroutine-lifecycle-and-cancellation.md) | Require owner, stop, propagation, and join/abandonment semantics instead of vague “use context.” |
| Close ownership, send-on-closed, blocked send/receive, nil channels, `select default`, or buffer assumptions. | [channels-select-and-close-ownership.md](channels-select-and-close-ownership.md) | Assign one owner and progress/full-queue policy instead of trusting buffers or receiver close. |
| WaitGroup ordering/copy, lock scope, `sync.Cond`, `RWMutex`, or lock-free claim. | [sync-primitives-identity-and-locking.md](sync-primitives-identity-and-locking.md) | Review the protected invariant and identity semantics instead of filing style nits or defaulting to atomics. |
| Fan-out, pools, semaphores, `SetLimit`, queues, async send wrappers, or producer/consumer pressure. | [bounded-work-and-backpressure.md](bounded-work-and-backpressure.md) | Prove both execution width and queued work are bounded. |
| Timer/ticker reset/stop, `time.After` loops, sleep polling, `AfterFunc`, fake clocks, or shutdown timing. | [timers-tickers-and-shutdown.md](timers-tickers-and-shutdown.md) | Review timer ownership and prompt unblock semantics with current Go behavior. |
| Race/liveness/leak tests, deterministic coordination, `testing/synctest`, or residual proof gap. | [concurrency-review-validation.md](concurrency-review-validation.md) | Match proof to race, protocol, lifecycle, or timing failure instead of treating any green test as blanket safety. |
