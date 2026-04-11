# Happens-Before And Publication Examples

## When To Load
Load this when a review touches shared state visibility, readiness flags, publication of pointers, maps, slices, interface values, `atomic.Value`, or any claim that one goroutine "must see" another goroutine's writes.

## Review Lens
Ask which concrete happens-before edge makes the read observe the write: channel send/receive, channel close/receive, mutex unlock/lock, `sync.Once`, documented `sync.Map` behavior, or atomic operations on the same atomic variable. If the answer is "goroutine order", "it happens fast", or "the flag is atomic so the struct is safe", write a finding.

## Bad Review Example
```text
[medium] cache/config.go:47
This uses an atomic ready flag. Maybe use a mutex if you want to be safer.
```

Why it fails: it does not name the missing visibility edge, the non-atomic data being published, the failure mode, or a validation command.

## Good Review Example
```text
[high] [go-concurrency-review] cache/config.go:47
Issue:
Axis: Happens-Before And State Publication; `ready.Store(true)` publishes `cfg` but readers load `ready` and then read `cfg` without synchronizing access to the pointed-to map. The atomic flag only orders observations of that flag; it does not make later mutation of `cfg.Routes` safe or make this multi-field invariant atomic.
Impact:
Requests can observe a partially initialized or concurrently mutated config map, which is a merge-blocking data race and can panic under concurrent map access.
Suggested fix:
Publish an immutable snapshot with `atomic.Value` after fully constructing it, or guard `ready` and `cfg` behind the same mutex. Do not mutate the published map after `Store`.
Reference:
Go memory model visibility rules; `sync/atomic` package guidance for atomic operations; validate with `go test -race ./internal/cache -run TestConfigReload`.
```

## Failure Mode
The reviewer should call out a data race or stale/partial publication when:
- a non-atomic field, map, slice, string, interface, or pointer is read after checking an atomic flag but the field itself is not protected;
- a pointer is atomically swapped but the pointed-to object remains mutable without a separate synchronization rule;
- a goroutine writes state and exits, and another goroutine assumes the exit itself synchronizes the write;
- a buffered channel is treated as if it provided the same ordering as an unbuffered rendezvous outside the documented memory-model rules.

## Smallest Safe Correction
Prefer the smallest correction that creates one clear ownership rule:
- guard the whole invariant with one mutex;
- use channel ownership transfer and stop touching the value after send;
- publish an immutable snapshot through `atomic.Value` or `atomic.Pointer[T]`;
- replace double-checked locking with `sync.Once` or a locked fast path;
- keep atomic and non-atomic access to the same variable from mixing.

## Validation Evidence
Use validation that exercises the concurrent path:
```bash
go test -race ./internal/cache -run TestConfigReload
go test -count=100 ./internal/cache -run TestConfigReload
```

Say explicitly when `-race` is necessary but not sufficient, for example if the bug is stale visibility rather than a reliably triggered conflicting access.

## Source Links From Exa
- [The Go Memory Model](https://go.dev/ref/mem)
- [sync package docs](https://pkg.go.dev/sync)
- [sync/atomic package docs](https://pkg.go.dev/sync/atomic)
- [Data Race Detector](https://go.dev/doc/articles/race_detector.html)

