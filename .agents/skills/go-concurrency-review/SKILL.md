---
name: go-concurrency-review
description: "Use when changed Go touches goroutines, shared state, channels, locks, atomics, WaitGroups, timers, worker bounds, cancellation unblock, or join protocols; Own concrete happens-before and lifecycle-mechanism correctness; Skip when the primary defect is service resilience policy, durable replay, or Go context API semantics."
---

# Go Concurrency Review

Load the [shared specialist contract](../specialist-contract.md) for common selection, scope, evidence, reference, return, and handoff mechanics; apply the domain-specific rules below.

## Trigger, Scope, And Boundary

Review changed goroutines, channels, mutexes, atomics, WaitGroups, `sync.Cond`, `errgroup`, worker pools, timers/tickers, shared state, backpressure, draining, and shutdown for merge-risk races, deadlocks, leaks, stalls, visibility defects, panics, and unbounded work.

Use approved lifecycle/shutdown contracts as governing evidence without suppressing code-visible findings. Stay review-only: do not redesign architecture, retry/degradation policy, DB/cache semantics, benchmark strategy, or test strategy when another owner holds the decisive question.

## Concurrency Invariants

1. Every shared-state read/write has a concrete happens-before edge or immutable ownership transfer; mixed synchronized and unsynchronized access is unsafe.
2. Every goroutine, worker, timer, ticker, and queue has an owner, bounded lifetime, stop signal, and join/drain or explicitly accepted abandonment semantics.
3. Channel send/receive/close ownership and progress policy are explicit; buffers do not substitute for cancellation, backpressure, or one closer.
4. Sync primitives are identity-bearing and never copied after use; lock/condition/atomic choices protect a named invariant rather than a folklore optimization.
5. Active and queued work are bounded; full-queue behavior, cancellation, early-return unblocking, and detached sender accumulation are deliberate.
6. Concurrent work observes its owned stop or cancellation signal at every blocking operation; shutdown is idempotent and promptly unblocks waits, sends, receives, timers, workers, and result paths before join.
7. Race, liveness, leak, timer, and shutdown proof matches the failure mode; sleeps and scheduler luck are not correctness evidence.

## Symptom-Driven Reference Selector

State how the selected reference changes the review judgment.

| Symptom | Load | Behavior change |
| --- | --- | --- |
| Visibility, readiness flags, atomics, immutable snapshots, or missing publication edge. | [happens-before-and-publication.md](references/happens-before-and-publication.md) | Require a real synchronization edge or immutable snapshot instead of goroutine-order intuition. |
| Fire-and-forget work, lost context, early-return leaks, `errgroup`, or shutdown join. | [goroutine-lifecycle-and-cancellation.md](references/goroutine-lifecycle-and-cancellation.md) | Require owner, stop, propagation, and join/abandonment semantics instead of vague “use context.” |
| Close ownership, send-on-closed, blocked send/receive, nil channels, `select default`, or buffer assumptions. | [channels-select-and-close-ownership.md](references/channels-select-and-close-ownership.md) | Assign one owner and progress/full-queue policy instead of trusting buffers or receiver close. |
| WaitGroup ordering/copy, lock scope, `sync.Cond`, `RWMutex`, or lock-free claim. | [sync-primitives-identity-and-locking.md](references/sync-primitives-identity-and-locking.md) | Review the protected invariant and identity semantics instead of filing style nits or defaulting to atomics. |
| Fan-out, pools, semaphores, `SetLimit`, queues, async send wrappers, or producer/consumer pressure. | [bounded-work-and-backpressure.md](references/bounded-work-and-backpressure.md) | Prove both execution width and queued work are bounded. |
| Timer/ticker reset/stop, `time.After` loops, sleep polling, `AfterFunc`, fake clocks, or shutdown timing. | [timers-tickers-and-shutdown.md](references/timers-tickers-and-shutdown.md) | Review timer ownership and prompt unblock semantics with current Go behavior. |
| Race/liveness/leak tests, deterministic coordination, `testing/synctest`, or residual proof gap. | [concurrency-review-validation.md](references/concurrency-review-validation.md) | Match proof to race, protocol, lifecycle, or timing failure instead of treating any green test as blanket safety. |

## Evidence And Domain Finding Rules

Inspect every changed launch, access, synchronization edge, close, cancellation, blocking operation, full-queue path, error return, and shutdown path. Demand an exact `file:line`, failed concurrency axis, broken invariant or missing happens-before assumption, concrete failure mode/blast radius, governing contract when present, and focused race/liveness/leak/shutdown command or evidence gap.

Specialist additions:

- start `Issue` with the concurrency axis when useful;
- `critical` examples include confirmed race, deadlock, send-on-closed, negative WaitGroup path, leaked significant work, or shutdown hang;
- use `No concurrency findings.` only when no merge-risk defect is supported, and still state residual evidence gaps.

## Success, Escalation, And Stop Conditions

Success means findings are merge-risk ordered, concurrency-specific, evidence-anchored, locally correctable or explicitly handed off, and proof recommendations match the defect class.

Escalate missing service lifecycle, bound, or shutdown policy to `go-reliability-spec`; durable workflow/reconciliation policy to `go-distributed-spec`; caller-visible blocking or async semantics to `go-api-contract-spec`; DB/cache ownership to `go-db-cache-spec`; and mechanism placement to `go-implementation-ownership-spec`. Stop rather than prescribe a local lock when the missing decision belongs to those owners.
