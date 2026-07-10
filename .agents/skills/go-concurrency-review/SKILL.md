---
name: go-concurrency-review
description: "Review Go code changes for goroutine lifecycle, cancellation, channel ownership, shared-state synchronization, `sync/atomic` correctness, bounded concurrency, timer/ticker hazards, and shutdown safety. Use whenever a Go review, PR, diff, flaky-test investigation, or bug hunt touches goroutines, channels, mutexes, atomics, WaitGroups, errgroup, worker pools, background loops, or shutdown behavior, even if the request is phrased as a generic code review."
---

# Go Concurrency Review

## Trigger, Scope, And Boundary

Review changed goroutines, channels, mutexes, atomics, WaitGroups, `sync.Cond`, `errgroup`, worker pools, timers/tickers, shared state, backpressure, draining, and shutdown for merge-risk races, deadlocks, leaks, stalls, visibility defects, panics, and unbounded work.

Use approved lifecycle/shutdown contracts as governing evidence without suppressing code-visible findings. Stay review-only: do not redesign architecture, retry/degradation policy, DB/cache semantics, benchmark strategy, or test strategy when another owner holds the decisive question.

## Concurrency Invariants

1. Every shared-state read/write has a concrete happens-before edge or immutable ownership transfer; mixed synchronized and unsynchronized access is unsafe.
2. Every goroutine, worker, timer, ticker, and queue has an owner, bounded lifetime, stop signal, and join/drain or explicitly accepted abandonment semantics.
3. Channel send/receive/close ownership and progress policy are explicit; buffers do not substitute for cancellation, backpressure, or one closer.
4. Sync primitives are identity-bearing and never copied after use; lock/condition/atomic choices protect a named invariant rather than a folklore optimization.
5. Active and queued work are bounded; full-queue behavior, cancellation, early-return unblocking, and detached sender accumulation are deliberate.
6. Request context reaches blocking work; shutdown is idempotent and promptly unblocks waits, sends, receives, timers, workers, and result paths.
7. Race, liveness, leak, timer, and shutdown proof matches the failure mode; sleeps and scheduler luck are not correctness evidence.

## Symptom-Driven Reference Selector

Load at most one reference by default and a second only for an independent pressure. State how it changes the review judgment.

| Symptom | Load | Behavior change |
| --- | --- | --- |
| Visibility, readiness flags, atomics, immutable snapshots, or missing publication edge. | [happens-before-and-publication.md](references/happens-before-and-publication.md) | Require a real synchronization edge or immutable snapshot instead of goroutine-order intuition. |
| Fire-and-forget work, lost context, early-return leaks, `errgroup`, or shutdown join. | [goroutine-lifecycle-and-cancellation.md](references/goroutine-lifecycle-and-cancellation.md) | Require owner, stop, propagation, and join/abandonment semantics instead of vague “use context.” |
| Close ownership, send-on-closed, blocked send/receive, nil channels, `select default`, or buffer assumptions. | [channels-select-and-close-ownership.md](references/channels-select-and-close-ownership.md) | Assign one owner and progress/full-queue policy instead of trusting buffers or receiver close. |
| WaitGroup ordering/copy, lock scope, `sync.Cond`, `RWMutex`, or lock-free claim. | [sync-primitives-identity-and-locking.md](references/sync-primitives-identity-and-locking.md) | Review the protected invariant and identity semantics instead of filing style nits or defaulting to atomics. |
| Fan-out, pools, semaphores, `SetLimit`, queues, async send wrappers, or producer/consumer pressure. | [bounded-work-and-backpressure.md](references/bounded-work-and-backpressure.md) | Prove both execution width and queued work are bounded. |
| Timer/ticker reset/stop, `time.After` loops, sleep polling, `AfterFunc`, fake clocks, or shutdown timing. | [timers-tickers-and-shutdown.md](references/timers-tickers-and-shutdown.md) | Review timer ownership and prompt unblock semantics with current Go behavior. |
| Race/liveness/leak tests, deterministic coordination, `testing/synctest`, or residual proof gap. | [concurrency-review-validation.md](references/concurrency-review-validation.md) | Match proof to race, protocol, lifecycle, or timing failure instead of treating any green test as blanket safety. |

## Evidence And Shared Finding Envelope

Inspect every changed launch, access, synchronization edge, close, cancellation, blocking operation, full-queue path, error return, and shutdown path. Demand an exact `file:line`, failed concurrency axis, broken invariant or missing happens-before assumption, concrete failure mode/blast radius, smallest safe correction, governing contract when present, and focused race/liveness/leak/shutdown command or evidence gap.

Use the [shared review finding envelope](../../../docs/subagent-contract.md#shared-review-finding-envelope). Specialist additions:

- start `Issue` with the concurrency axis when useful;
- `critical` examples include confirmed race, deadlock, send-on-closed, negative WaitGroup path, leaked significant work, or shutdown hang;
- use `No concurrency findings.` only when no merge-risk defect is supported, and still state residual evidence gaps.

## Success, Escalation, And Stop Conditions

Success means findings are merge-risk ordered, concurrency-specific, evidence-anchored, locally correctable or explicitly handed off, and proof recommendations match the defect class.

Escalate changes to concurrency model/bounds/shutdown policy to reliability design, durable workflow/reconciliation to distributed design, caller-visible blocking/async semantics to API/chi design, DB/cache ownership to DB/cache design, and unsafe package ownership to integrated design. Stop rather than prescribe a local lock when the missing decision belongs to those owners.
