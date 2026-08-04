# Shared State Publication And Locking

## Behavior Change Thesis
When loaded for visibility, publication, or lock-scope symptoms, this file makes the model check what the named edge actually covers — the whole payload, for its whole lifetime — instead of accepting an atomic store, a readiness flag, or a held mutex as protection for everything reachable from it.

## When To Load
Symptom: the diff publishes or reads shared fields, maps, slices, pointers, or interface values across goroutines; uses `atomic.Value`, `atomic.Pointer[T]`, `sync.Map`, or a readiness flag; or changes lock scope, `sync.Cond` use, or what runs while a lock is held.

## Decision Rubric
- Name the edge and its span. `Unlock` before `Lock`, a send before its receive, `close` before a receive that observes it, `Once.Do`, the return from a `WaitGroup.Go` function before the `Wait` it unblocks, and an atomic store before the load that observes it are edges. Goroutine start order, elapsed time, and "this only runs at startup" are not.
- An atomic store publishes the writes that preceded it and nothing after it. Publication is safe only when the value is fully built before the store and immutable afterward, or separately synchronized. A flag that gates a mutable payload publishes the payload's initial state and leaves every later mutation racing.
- `sync.Map.Range` is not a consistent snapshot: it visits no key twice, but a key stored or deleted during the call may be reflected from any point in the call. An invariant spanning several keys needs a map under one lock, or one immutable snapshot.
- Prefer one owner per invariant: one mutex, one owning goroutine, or one immutable snapshot swapped atomically. Per-field atomics under a multi-field invariant produce combinations no single writer ever published.
- Lock scope is a claim about what the invariant needs, not about what the function does. Calling out under a lock — a user callback, a channel send, an HTTP or database round trip — turns the mutex into a queue for every unrelated caller and lets re-entrant callers deadlock. This is the case worth spending review on, because no linter sees it.
- `sync.Cond` needs its predicate guarded by the associated locker and re-checked in a loop around `Wait`.
- `govet` in `make lint` already fails a sync value copied by value receiver, argument, or return (`copylocks`) and a `WaitGroup.Add` called from inside the goroutine it counts (`waitgroup`). Both are reported without help, so name the broken invariant rather than re-deriving the gate's finding.
- `RWMutex` over `Mutex` is a throughput claim. Route it to `go-performance` with a measurement rather than asserting it.

## Reject
- "Atomic, so it is safe" without saying which read the store covers — the useful finding names the non-atomic payload and the write that races it.
- "Single writer" where aliases to the written value reach readers with no edge between them.
- A visibility defect reported as "consider a mutex" — state the broken invariant and the observable failure, then the smallest ownership rule that closes it, which is often an immutable snapshot rather than a lock.

## Validation Shape
- `make test-race` covers the tree; the race detector reports races that actually executed, so a clean run refutes a race story and never proves absence.
- For a publication path, drive the writer and the reader through the overlap deliberately rather than repeating the test and relying on the scheduler.
- When the residual risk is a contract — "the snapshot must stay immutable after publication" — say so; no race run tests a rule the code no longer states.
