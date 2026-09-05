# Bounded Work And Backpressure

## Load When
Symptom: the diff launches a goroutine per item, adds a worker pool, `errgroup.SetLimit`, a semaphore, buffered job or result channels, an async send wrapper, a retry queue, or changes producer/consumer pressure.

## Decide
- Count both running work and waiting work: goroutines alive, items queued, retries held, result buffers, and the payloads all of them retain. A fixed worker count over a request-sized slice still holds the whole slice.
- Acquire the limiter before launching the goroutine, and release it inside. Acquiring inside bounds the critical section only; every item still gets a goroutine and keeps its payload alive while it waits.
- `errgroup.Group.Go` blocks at submission once `SetLimit` is reached, so the submitter is a blocking site too. Review where it blocks: under a lock, inside shutdown, or while holding responsibility for closing a channel, a full pool deadlocks even though the worker count is correct.
- The limit must not change while goroutines in the group are active; treat a dynamic `SetLimit` as a correctness defect rather than tuning.
- A buffered channel is a queue with a policy, not a liveness argument. State what happens when it is full — block with cancellation, drop with accounting, fail fast, or shed upstream — and whether producers can outpace consumers indefinitely.
- Spawning a goroutine so a send does not block converts backpressure into an unbounded goroutine backlog. If the send must not block, the bound moves, it does not disappear.
- Blocking with cancellation is often the correct local fix; dropping work is a policy, not a default. When the real question is overload behavior for the service, name the local unbounded-work defect and hand the shed/retry policy to `go-reliability`.

## Reject
- "It uses a semaphore, so only N run at once" where the acquire is inside the goroutine — true about execution, false about goroutines and memory.
- A bound proven only for workers, with pending items, retry lists, and result buffers uncounted.
- A submission path with no cancellation: a producer parked on `jobs <- item` leaks exactly like a worker parked on a result send.

## Prove
- Prove the bound by exceeding it: drive more items than the limit and fail on the first observed overflow of a counter the test owns.
- Prove the submitter: fill the pool, cancel, and assert the producer returns.
- Add `-race`, or `make test-race`, when the bound itself is tracked in shared state.
