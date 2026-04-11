---
name: go-concurrency-review
description: "Review Go code changes for goroutine lifecycle, cancellation, channel ownership, shared-state synchronization, `sync/atomic` correctness, bounded concurrency, timer/ticker hazards, and shutdown safety. Use whenever a Go review, PR, diff, flaky-test investigation, or bug hunt touches goroutines, channels, mutexes, atomics, WaitGroups, errgroup, worker pools, background loops, or shutdown behavior, even if the request is phrased as a generic code review."
---

# Go Concurrency Review

## Purpose
Protect changed concurrent paths from merge-risk race, deadlock, goroutine leak, send or receive stall, memory-visibility bug, timer leak, shutdown hang, and unbounded-work defects.

## Specialist Stance
- Demand concrete happens-before, ownership, cancellation, and shutdown evidence instead of scheduler intuition.
- Treat every goroutine, channel, timer, worker, and shared state path as needing an owner and exit story.
- Prefer simple synchronization and bounded work over clever lock-free or timing-based fixes.
- Hand off broader workflow, reliability, performance, or DB/cache design when concurrency is only a symptom surface.

## Scope
- review goroutines, channels, mutexes, `sync.RWMutex`, `sync.WaitGroup`, `sync.Cond`, `sync/atomic`, `errgroup`, worker pools, pipelines, fan-out or fan-in paths
- review goroutine lifecycle, stop ownership, and explicit termination behavior
- review cancellation, deadline propagation, and blocking-operation escape paths
- review channel ownership, close semantics, queue bounds, and send or receive behavior
- review synchronization and publication safety for shared mutable state
- review bounded concurrency, backpressure, and queue growth behavior
- review timer, ticker, and sleep-based coordination hazards
- review concurrent error propagation, draining, and shutdown safety
- review race and liveness evidence for significant concurrent changes

## Boundaries
Do not:
- turn concurrency review into broad style cleanup or architecture redesign
- take primary ownership of benchmark proof, DB/cache correctness, or resilience policy unless concurrency is the direct root cause
- accept timing luck, sleep-based reasoning, or scheduler luck as proof of correctness
- hide uncertain concurrent behavior behind vague `seems safe` language
- overprescribe lock-free or `RWMutex` solutions when a simpler mutex or ownership transfer is safer

## Core Defaults
- Prove concurrency safety through concrete synchronization edges, not through intuition about goroutine order.
- Every goroutine, timer, ticker, worker, and queue needs explicit ownership and a stop or drain story.
- Mixed synchronized and unsynchronized access to the same state is a bug until proven otherwise.
- Prefer bounded work, explicit backpressure, and idempotent shutdown over eventual drains or polling luck.
- If an approved spec, plan, or contract exists, use it as governing evidence for lifecycle and shutdown expectations without suppressing local findings.
- Prefer the smallest safe correction that restores deterministic concurrent behavior.

## Reference Selection
Keep this file focused on the review workflow. References are compact rubrics and example banks, not exhaustive checklists or documentation dumps. Load at most one reference by default; load a second only when the diff clearly spans independent decision pressures, such as a channel close race plus weak validation evidence.

Choose by symptom and behavior change:

| Symptom in the diff | Load | Behavior change |
| --- | --- | --- |
| shared state visibility, unsafe readiness flags, mixed atomic/non-atomic access, `atomic.Value`, immutable snapshots, or missing visibility edges | `references/happens-before-and-publication.md` | makes the review require a concrete happens-before edge or immutable snapshot instead of trusting goroutine order, `single writer`, or an atomic readiness flag |
| fire-and-forget goroutines, context propagation, early return leaks, pipeline abandonment, `errgroup` cancellation, or shutdown joins | `references/goroutine-lifecycle-and-cancellation.md` | makes the review require owner, stop signal, and join or accepted abandonment semantics instead of vague "use context" advice |
| channel close ownership, send-on-closed risk, blocked sends or receives, `select` default spin, nil-channel gating, or fragile buffer assumptions | `references/channels-select-and-close-ownership.md` | makes the review assign one channel owner and explicit progress/full-queue policy instead of trusting receiver close, buffers, or `default` branches |
| `WaitGroup` ordering, copied sync values, lock scope, `sync.Cond` predicates, `RWMutex` misuse, or local lock-free claims | `references/sync-primitives-identity-and-locking.md` | makes the review treat sync primitives as identity-bearing state and review the protected invariant instead of filing style nits or defaulting to `RWMutex`/atomics |
| per-item goroutine fan-out, worker pools, semaphores, `errgroup.SetLimit`, buffered job/result queues, async send wrappers, or producer/consumer backpressure | `references/bounded-work-and-backpressure.md` | makes the review prove active work and queued work are both bounded instead of accepting worker-pool or semaphore-shaped code as safe |
| timer/ticker reset or stop behavior, `time.After` loops, sleep polling, `AfterFunc` completion, fake-clock tests, or shutdown timing | `references/timers-tickers-and-shutdown.md` | makes the review focus on timer ownership and prompt unblock semantics instead of stale timer-leak folklore or sleep-as-synchronization |
| evidence quality, `go test -race`, leak or liveness tests, deterministic coordination, `testing/synctest`, or residual risk wording | `references/concurrency-review-validation.md` | makes the review match proof to the failure mode instead of treating "tests passed" or race-clean output as blanket validation |

When you load a reference, translate the example into the current diff's concrete `file:line`, failure mode, smallest safe correction, and validation command. Do not paste generic examples as final review output.

## Expertise

### Happens-Before And State Publication
- Require a concrete happens-before edge for shared state: channel send or receive, close observation, mutex unlock or lock, `WaitGroup` or `errgroup` completion, or atomic operations on the same variable.
- Flag mixed atomic and non-atomic access to the same variable.
- Do not accept an atomic flag as proof that separately stored fields, slices, maps, or pointers are safely published unless later mutation is impossible or separately synchronized.
- Treat `single writer` claims as incomplete if aliases escape or readers have no visibility guarantee.
- When state spans more than one field or invariant, prefer a mutex or ownership transfer over ad hoc atomics.

### Goroutine Lifecycle And Ownership
- Every started goroutine needs an owner, a stop signal, and join or abandonment semantics.
- Flag fire-and-forget goroutines unless process-lifetime ownership and failure irrelevance are explicit.
- Verify downstream early exit cannot strand upstream senders or worker goroutines.
- Background watchers, retries, and select loops need a bounded exit path on cancellation or channel close.
- Goroutines created per item must still prove bounded width or explicit drop or backpressure behavior.

### Context, Cancellation, And `errgroup` Semantics
- Require the derived context to reach all blocking downstream calls and sibling workers.
- Flag request-path replacement of request context with `context.Background()` or `context.TODO()`.
- `errgroup.WithContext` only helps if workers actually observe the derived context; otherwise leaked work remains.
- Distinguish fail-fast `errgroup` semantics from collect-all semantics; returning the first error may still require explicit result draining or cleanup.
- Spawning new work after group-context cancellation is usually a lifecycle or rollback defect.

### Channel Ownership, `select` Behavior, And Blocking
- Make close ownership explicit; most channels should have one closer and one clearly defined control point.
- Flag send-on-closed risk, multiple closers, and receiver-side close unless the contract explicitly makes the receiver the owner.
- Treat buffered channels as bounded queues with explicit full-queue policy: block, drop, fail, or shed.
- Flag blocked sends or receives that have no cancellation, close, or bounded escape path.
- `nil` channels should only appear as intentional select gating; accidental nil paths block forever.
- `select { default: ... }` inside a loop often means busy-spin, starvation, or hidden loss of backpressure.

### WaitGroups, Locks, `sync.Cond`, And Copy Safety
- `WaitGroup.Add` must happen before launch and before any possible `Wait`; `Add` or `Done` imbalance is merge-risk.
- Flag copying of structs containing `sync.WaitGroup`, `sync.Mutex`, `sync.RWMutex`, or `sync.Cond` after first use, including value receivers and by-value helper calls.
- Keep lock scope clear; flag callbacks, channel sends, or blocking I/O under lock unless the lock is intentionally protecting that blocking contract.
- Prefer `sync.Mutex` over `sync.RWMutex` unless read dominance and contention behavior are justified.
- `sync.Cond` requires predicate-in-a-loop reasoning; signal or broadcast must correspond to a state change that waiters can actually observe.

### Atomics And Lock-Free Claims
- Use `sync/atomic` for single-word state, counters, or immutable snapshot publication, not for multi-field invariants.
- Flag CAS or spin loops with no backoff, no cancellation, or no progress guarantee.
- `atomic.Value` and atomic pointer publication require type consistency and immutable-or-separately-synchronized pointed-to data.
- On portability-sensitive code, be careful with 64-bit atomic alignment assumptions on 32-bit targets.
- If the reviewer cannot explain the invariant in one sentence, the code is probably not safely lock-free.

### Timers, Tickers, And Time-Based Coordination
- `time.After` in hot or long-lived loops creates timer churn and often hides cancellation or reset semantics; account for Go version differences before calling it a timer leak.
- `Ticker` must be `Stop`ped on all exit paths.
- `Timer.Stop` or `Reset` flows need correct stop or drain coordination when another goroutine may already observe the tick.
- Sleep-based polling is not an acceptable substitute for a real signal or bounded retry strategy.

### Bounded Concurrency And Backpressure
- Flag unbounded goroutine fan-out, unbounded worker pools, and queue growth without explicit limits.
- Use `errgroup.SetLimit`, semaphores, or fixed worker pools when concurrency width must stay bounded.
- Boundedness must cover both execution width and queued work; a bounded worker pool with an unbounded submission queue is still unbounded.
- Detached sender goroutines against slow or abandoned consumers deserve their own finding when they can accumulate independently.

### Shutdown, Draining, And Async Workers
- Shutdown should be idempotent and should unblock sends, receives, waits, timers, and worker loops.
- `Close` or `Stop` must define whether it drains in-flight work, cancels it, or hands it off; silent ambiguity is a bug.
- Verify result channels, ack paths, or worker completion signals cannot deadlock during shutdown.
- For async consumers, require ack or commit only after the relevant local side effect is durable when applicable.
- Separate local merge blockers from broader lifecycle-policy questions; do not hide visible code defects behind architecture language.

### Tests And Validation Evidence
- Significant concurrency changes should carry race evidence, deterministic coordination, or an explicit evidence gap.
- `go test -race` is useful but not sufficient for pure protocol deadlocks or shutdown hangs; say when race-clean code can still be wrong.
- Prefer gates, fake clocks, leak detection, or explicit completion signals over `time.Sleep`.
- Sleep-based tests are weak evidence unless they only supplement stronger coordination assertions.
- Missing race or liveness evidence on meaningful concurrency changes should become a finding or residual risk, not a shrug.

### Cross-Domain Handoffs
- Hand off retry, overload, and degradation policy depth to `go-reliability-review`.
- Hand off DB/query/cache contract defects to `go-db-cache-review`.
- Hand off benchmark, pprof, or lock-contention proof to `go-performance-review`.
- Hand off test-strategy depth to `go-qa-review`.
- Hand off broader structural drift to `go-design-review`.

## Finding Quality Bar
Each finding should include:
- exact `file:line`
- the failed concurrency axis
- the broken invariant or missing happens-before assumption
- the concrete failure mode and blast radius
- the smallest safe correction
- a validation command when useful
- the governing spec, plan, or contract reference when one exists
- whether the issue is local code drift or needs design escalation

Severity is merge-risk based:
- `critical`: confirmed race, deadlock, send-on-closed, leaked background work, negative `WaitGroup` path, or shutdown hang in a significant path
- `high`: high-probability concurrency defect or unbounded-work risk with meaningful blast radius
- `medium`: bounded but important concurrency weakness or evidence gap on a risky path
- `low`: local hardening or clarity improvement

## Deliverable Shape
Return review output in this order:
- `Findings`
- `Handoffs`
- `Design Escalations`
- `Residual Risks`
- `Validation Commands`

If there are no findings, say `No concurrency findings.` and still note any residual risks or evidence gaps.

Use this format for each finding:

```text
[severity] [go-concurrency-review] [file:line]
Issue:
Impact:
Suggested fix:
Reference:
```

In `Issue`, start with the axis context, for example `Axis: Happens-Before And State Publication; ...`.

## Escalate When
Escalate when:
- the safe correction changes the concurrency model, bounded-work policy, or shutdown contract (`go-reliability-spec`)
- the fix depends on a new async workflow, durable coordination model, or reconciliation design (`go-distributed-architect-spec`)
- the issue reveals a caller-visible contract change around blocking, async, or lifecycle semantics (`api-contract-designer-spec` or `go-chi-spec`)
- correctness depends on new DB/cache ownership or cache-coalescing contract (`go-db-cache-spec`)
- the current package or ownership boundaries make local concurrency repair unsafe (`go-design-spec` or `go-architect-spec`)
