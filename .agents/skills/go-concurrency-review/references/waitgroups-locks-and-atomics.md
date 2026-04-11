# WaitGroups, Locks, And Atomics Examples

## When To Load
Load this when a review touches `sync.WaitGroup`, `WaitGroup.Go`, copied sync values, `Mutex`, `RWMutex`, `Cond`, `sync.Map`, `sync/atomic`, atomic pointers, `atomic.Value`, or lock-free algorithms.

## Review Lens
Synchronization objects have identity. Copying them after first use, using value receivers on structs that contain them, or mixing atomic and non-atomic access usually breaks the reviewable ownership model.

## Bad Review Example
```text
[low] poller/poller.go:24
Use a pointer receiver here because it is more idiomatic with WaitGroups.
```

Why it fails: it frames a correctness bug as style and misses the broken join semantics.

## Good Review Example
```text
[critical] [go-concurrency-review] poller/poller.go:24
Issue:
Axis: WaitGroups, Locks, And Atomics; `func (p Poller) Start()` copies `Poller`, including its `sync.WaitGroup`. The goroutine calls `Done` on the copied group while `Stop` waits on the original group.
Impact:
`Stop` can return before the worker exits or wait forever depending on which copy was incremented, so shutdown correctness is not deterministic.
Suggested fix:
Make `Start` a pointer receiver and ensure every `Add`, goroutine launch, `Done`, and `Wait` operates on the same `*Poller` instance. Consider `WaitGroup.Go` where available, or call `Add` before launching the goroutine.
Reference:
`sync` docs say sync values including `WaitGroup` must not be copied after first use and positive `Add` when the counter is zero must happen before `Wait`; validate with `go test -race ./internal/poller -run TestStopWaitsForWorker -count=100`.
```

## Failure Mode
Write a finding when:
- `WaitGroup.Add` can run after goroutine launch or race with `Wait` while the counter is zero;
- a `WaitGroup`, `Mutex`, `RWMutex`, `Cond`, `sync.Map`, `atomic.Value`, or typed atomic is copied after first use;
- a callback, send, receive, I/O call, or user hook runs under a lock without a documented blocking contract;
- `Cond.Wait` is not in a predicate loop;
- `RWMutex` is used recursively, upgraded, downgraded, or chosen without a real read-dominant contention reason;
- atomics protect only one word but the code relies on a multi-field invariant.

## Smallest Safe Correction
Prefer corrections like:
- use pointer receivers for types containing sync or atomic fields;
- call `Add` before starting goroutines and `defer Done` inside each goroutine;
- use one mutex for the whole invariant rather than per-field atomic flags;
- copy-on-write before storing through `atomic.Value` or `atomic.Pointer[T]`;
- keep `Cond` predicates under the associated locker and wait in a loop;
- move blocking callbacks or channel sends out from under a lock, unless the lock intentionally serializes that blocking operation.

## Validation Evidence
Use validation that proves join and synchronization behavior:
```bash
go test -race ./internal/poller -run TestStopWaitsForWorker -count=100
go test ./internal/poller -run TestWaitGroupDoesNotHang -count=100 -timeout=5s
```

For lock-free claims, ask for a short invariant plus race evidence. If the invariant cannot be stated compactly, prefer a mutex or owner goroutine.

## Source Links From Exa
- [sync package docs](https://pkg.go.dev/sync)
- [sync/atomic package docs](https://pkg.go.dev/sync/atomic)
- [The Go Memory Model](https://go.dev/ref/mem)
- [Data Race Detector](https://go.dev/doc/articles/race_detector.html)

