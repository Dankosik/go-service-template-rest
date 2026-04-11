# Happens-Before And Publication

Behavior Change Thesis: When loaded for shared-state visibility or publication symptoms, this file makes the model require a concrete happens-before edge or immutable snapshot publication instead of trusting goroutine order, a `single writer` story, or an atomic readiness flag.

## When To Load
Symptom: the diff reads or publishes shared fields, maps, slices, pointers, interface values, readiness flags, `atomic.Value`, or `atomic.Pointer[T]`, and the review question is whether another goroutine can safely observe the write.

## Decision Rubric
- Name the exact visibility edge before accepting the code: mutex unlock/lock, matched channel send/receive, channel close observed by receive, `sync.Once`, documented `sync.Map` behavior, or atomic operations that protect the whole published value.
- Treat an atomic flag that gates non-atomic fields as suspicious. Observing the atomic store can publish prior writes, but it does not protect later mutation or make mutable maps, slices, or multi-field invariants atomic.
- Accept atomic snapshot publication only when the stored value is fully built before publication and is immutable afterward or separately synchronized.
- Reject `single writer` arguments when aliases escape to readers without a visibility rule.
- If the invariant spans more than one field, prefer one mutex, one owner goroutine, or one immutable snapshot over per-field atomics.

## Imitate
```text
[high] [go-concurrency-review] cache/config.go:47
Issue:
Axis: Happens-Before And State Publication; `ready.Store(true)` publishes `cfg`, but readers load `ready` and then read `cfg.Routes` while reload can still mutate the map. The observed atomic store can publish prior writes, but it does not make later mutation of `cfg.Routes` safe or make this multi-field invariant atomic.
Impact:
Requests can observe a partially initialized or concurrently mutated config map, which is a merge-blocking data race and can panic under concurrent map access.
Suggested fix:
Publish a fully built immutable snapshot with `atomic.Value`, or guard `ready` and `cfg` behind the same mutex. Do not mutate the map after publication.
Reference:
Validate the shared publication path with `go test -race ./internal/cache -run TestConfigReload -count=100`.
```

Copy the shape: the finding identifies the non-atomic payload, the false synchronization assumption, the user-visible failure mode, and the smallest ownership rule that would make the publication reviewable.

## Reject
```text
[medium] cache/config.go:47
This uses an atomic ready flag. Maybe use a mutex to be safer.
```

Reject this shape: it is a style preference, not a race finding. It never says what data the flag supposedly publishes or why readers lack a visibility edge.

```text
No issue: writes happen in the reload goroutine before readers check `ready`.
```

Reject this shape: goroutine execution order is not synchronization. The review needs the edge that makes the reader observe the write.

## Agent Traps
- Do not say "atomic makes it safe" unless the observed atomic operation protects the whole publication being read and later mutation is impossible or separately synchronized.
- Do not treat goroutine exit, elapsed time, or "initialized during startup" as a visibility edge.
- Do not bury a real race behind "consider a mutex"; state the broken invariant and failure mode.
- Do not require a mutex when immutable snapshot ownership transfer is the smaller correction.

## Validation Shape
- Use `go test -race` for shared-memory publication and mixed atomic/non-atomic access.
- Add deterministic reload/read overlap when the bug depends on a narrow interleaving.
- Say when race evidence is necessary but not sufficient, such as when the remaining risk is stale visibility or immutable-snapshot contract drift that the test does not exercise.
