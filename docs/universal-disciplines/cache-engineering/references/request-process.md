# Request and In-Process Cache Mechanics

This file owns request- and process-local mechanics. The common authority, key, freshness, and failure invariants remain canonical in `SKILL.md`.

## Choose the boundary

| Boundary | Use when | Reject when |
| --- | --- | --- |
| Request memoization | One request/job repeats an equivalent deterministic read | Cross-request reuse is needed |
| In-process cache | Network-free hits matter and one process can own bounded memory | Instances must observe one shared cache state |
| Process-local single-flight only | Duplicate fills overlap but retained values do not pay for themselves | Duplicate work across replicas is the actual bottleneck |

Request memoization inherits the request's lifetime and isolation, so it usually needs no TTL or eviction. It still needs the complete semantic key if one request can carry multiple tenants, principals, locales, or versions.

Preserve the memoized function's exact result semantics. If a stable error or partial result is part of that request-scoped result, retain the same result/error pair rather than silently changing repeated-call behavior.

Use one focused test to prove both behaviors: repeated equal inputs compute once, and one response-varying input computes separately. Exercise the memoized lookup rather than only comparing key strings.

An in-process cache multiplies entries, fills, and invalidation consumers by process count. Include rolling deploys, autoscaling, forked workers, and process restarts in cold-start and origin-capacity calculations.

## Pick concurrency primitives

Prefer the runtime's established cache or a typed map guarded by one lock. Keep entry state, expiry, byte accounting, generation, and eviction metadata in the same synchronization domain. Release the lock before origin I/O.

Use a read/write lock only after a representative contention benchmark shows benefit. Use a concurrent map only for its documented access shapes; TTL/LRU and cross-entry byte limits commonly need additional coordination.

Single-flight suppresses overlapping work; it is not a value cache:

- use the complete cache key as the flight key;
- give the shared fill a deadline independent of an arbitrary leader request;
- bound each caller's wait independently;
- define whether the first error is shared and when a later call may retry;
- fence publication by generation because forgetting/cancelling coordination need not stop old work.

For Go specifically, `sync.Map` is optimized for write-once/read-many or disjoint-key patterns, and `Range` is not a consistent snapshot. `sync.Once` is process-lifetime initialization, not an expiring cache; a panic consumes `Once`. `OnceValue(s)` requires Go 1.21+, retains the first returned values, and therefore can make an error permanent. `golang.org/x/sync/singleflight` suppresses only overlapping calls; `Do` has no caller cancellation API, while abandoning `DoChan` does not cancel the fill. Verify the installed Go and module versions before relying on implementation details.

## Bound memory and expiry

Account by serialized or retained heap bytes, not entry count alone. Include keys, metadata, allocator overhead, and duplicated decoded objects. Set entry and total byte ceilings, an explicit admission rule, and an eviction policy suited to measured access skew.

Avoid unbounded background refresh goroutines and timer-per-entry designs unless measured scale proves them safe. Lazy expiry makes reads pay cleanup; periodic expiry adds scheduled work. Benchmark the chosen mechanism at the expected key count and churn. Use a monotonic duration source where the runtime supports it, while preserving origin generation time for externally meaningful age.

Compression trades CPU, allocation, and tail latency for bytes. Test representative value distributions; reject values over the contract's size/decode limit before allocation amplification.

## Deploy and observe

Process-local invalidation needs delivery to every live process or a version/generation check that makes missed delivery harmless. A rolling deploy temporarily runs multiple key/serializer versions. New processes begin cold, and a fleet restart can multiply fill pressure by replica count.

Observe per process and aggregate:

- entries and retained bytes;
- hit/miss/fill/stale/negative counts;
- eviction/expiry/decode errors;
- concurrent fills, shared waiters, wait time, and fill errors;
- process restart/cold-fill rate and resulting origin concurrency.

Test race detection where supported, plus update-during-fill, eviction during read, concurrent expiry, caller cancellation, restart/cold start, and per-process invalidation loss.

## Primary sources

Checked 2026-08-02:

- [Go `sync` package](https://pkg.go.dev/sync) and [Go memory model](https://go.dev/ref/mem)
- [Go `context` package](https://pkg.go.dev/context)
- [`golang.org/x/sync/singleflight` v0.22.0](https://pkg.go.dev/golang.org/x/sync@v0.22.0/singleflight)
- [Go 1.21 `sync` additions](https://go.dev/doc/go1.21#sync)
