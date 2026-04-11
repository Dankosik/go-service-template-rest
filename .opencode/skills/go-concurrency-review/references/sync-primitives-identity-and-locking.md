# Sync Primitives Identity And Locking

Behavior Change Thesis: When loaded for `WaitGroup`, mutex, `Cond`, copied sync value, or lock-scope symptoms, this file makes the model treat synchronization objects as identity-bearing state and review the protected invariant instead of filing style nits or defaulting to `RWMutex` and atomics.

## When To Load
Symptom: the diff touches `sync.WaitGroup`, `WaitGroup.Go`, copied `Mutex`/`RWMutex`/`Cond`/`Once`/`Map`/`Pool` values, value receivers on structs with sync fields, lock scope, callbacks under lock, `sync.Cond`, recursive or upgrade-style `RWMutex` use, or local lock-free claims. If the main risk is atomic publication of shared data, load `happens-before-and-publication.md` instead.

## Decision Rubric
- A sync value has identity after first use. Copying it through value receivers, by-value helpers, map/slice movement, or struct returns can split the state that callers believe is shared.
- When `Add`/`Done` tracks a goroutine, `Add` must occur before that goroutine can call `Done` and before any possible `Wait` on a zero counter. If the counter is already non-zero, additional `Add` calls may be valid but still need an intentional lifecycle story.
- Lock scope must match the protected invariant. Blocking callbacks, channel sends, I/O, or user hooks under lock are findings when they can deadlock or stall unrelated callers.
- `sync.Cond` requires a predicate protected by the associated locker and `Wait` in a loop.
- `RWMutex` is not a default upgrade over `Mutex`; recursive reads, upgrades or downgrades, and reader blocking while a writer waits can make it worse.
- If a lock-free invariant cannot be stated in one sentence and validated with a concrete progress story, prefer a mutex or owner goroutine.

## Imitate
```text
[critical] [go-concurrency-review] poller/poller.go:24
Issue:
Axis: WaitGroups, Locks, And Atomics; `func (p Poller) Start()` copies `Poller`, including its `sync.WaitGroup`. The goroutine calls `Done` on the copied group while `Stop` waits on the original group.
Impact:
`Stop` can return before the worker exits or wait forever depending on which copy was incremented, so shutdown correctness is not deterministic.
Suggested fix:
Make `Start` a pointer receiver and ensure every `Add`, goroutine launch, `Done`, and `Wait` operates on the same `*Poller` instance. Consider `WaitGroup.Go` when the module's Go version supports it, or call `Add` before launching the goroutine.
Reference:
Validate with `go test -race ./internal/poller -run TestStopWaitsForWorker -count=100`.
```

Copy the shape: it frames the receiver choice as broken synchronization identity, not idiom.

## Reject
```text
[low] poller/poller.go:24
Use a pointer receiver here because it is more idiomatic with WaitGroups.
```

Reject this shape: it hides a possible shutdown correctness bug behind style language.

```go
mu.Lock()
defer mu.Unlock()
callbacks[id](msg)
```

Reject this as safe unless the callback contract is intentionally serialized under `mu`. User code under lock can re-enter, block, or call back into the same object and deadlock.

## Agent Traps
- Do not miss sync-value copies through value receiver methods; this is common in review diffs because the method body still "looks" locked.
- Do not recommend `RWMutex` without checking read dominance, upgrade/downgrade behavior, and writer progress.
- Do not suggest `WaitGroup.Go` unless the module's Go version supports it, and do not claim it fixes cancellation, panic handling, or downstream blocking by itself; its function must not panic, and it only changes launch/accounting shape.
- Do not conflate atomic publication defects with sync-object identity defects. Load the publication reference when the core problem is visibility of data behind atomics.

## Validation Shape
- For join behavior, assert `Stop` or `Wait` returns only after the worker exits.
- For copied sync values, run `go vet -copylocks` when available through the repo's normal vet target, plus a focused liveness or race test.
- For lock-scope changes, add a test where the callback blocks or re-enters and prove no deadlock.
- Good commands look like `go test -race ./internal/poller -run TestStopWaitsForWorker -count=100` and `go test ./internal/poller -run TestWaitGroupDoesNotHang -count=100 -timeout=5s`.
