---
name: concurrency-control
description: "Interleaving-first concurrency control for shared durable state. Use when designing, building, auditing, or diagnosing lost updates, double effects from concurrent requests, check-then-act races, duplicate submissions, optimistic vs pessimistic locking, version columns and ETags, unique-constraint arbitration, isolation anomalies like write skew, advisory and distributed locks, fencing tokens, leader election, or singleton work. Route job claim/lease design to durable-background-jobs, PostgreSQL lock contention as a bottleneck to postgres-performance, and constraint modeling to postgres-schema-design."
---

# Interleaving-First Concurrency Control

Judge every mechanism against a concrete **interleaving**:

`shared state -> writers -> breaking schedule -> weakest mechanism -> conflict path -> failure windows -> prove`

A race is not "two things at once"; it is a specific schedule of reads and writes that violates a named invariant. Design against that schedule, prefer the weakest mechanism that closes it, and assume every "exclusive" holder can stall and resume after losing exclusivity.

## Choose the branch

Run only the branch the request needs. Record missing evidence as a gap instead of broadening the task.

- **Audit or diagnose:** enumerate every writer of the corrupted state, construct the breaking schedule that reproduces the observed damage, and propose the smallest closing mechanism plus a forced-interleaving proof. Keep code and production unchanged.
- **Design or plan:** map state, writers, and schedules, choose mechanisms and conflict dispositions to the fidelity requested; label artifacts proposed.
- **Build or fix:** apply the smallest mechanism change and a proof that fails on the old code under the forced schedule.
- **Operate:** releasing stuck locks, forcing leadership changes, or repairing raced state in production requires explicit authorization for the exact action and targets. Preflight, execute only that action, verify with fresh readback.

Hand work-queue claim and lease design to `durable-background-jobs`; PostgreSQL lock waits as a load or latency bottleneck to `postgres-performance`; declaring constraints as schema invariants to `postgres-schema-design`; per-key ordered delivery to `reliable-messaging`; idempotency of external provider calls to `external-api-integration`; cross-service consistency topology to `distributed-system-design`. In-process memory races on shared variables belong to the language runtime's race tooling, not here. This skill retains the interleaving analysis, mechanism choice, conflict disposition, fencing, and proof for shared durable state.

**Complete when:** the branch, authority boundary, requested artifact, and excluded concerns are explicit.

## Name the state and every writer

Define the contested unit: a row, document, or key — or an invariant spanning several rows, such as a balance with its entries or "at most one active". Then enumerate the writers: concurrent user requests, retries of the same request, other endpoints touching the same rows, background jobs, consumers, other services, operators — and the other replicas of this same service. The most-missed writers are your own horizontal scale and your own retries.

Reads that feed decisions are part of the write path: a check is only as fresh as its snapshot.

**Complete when:** the state unit, its invariant, and a complete writer list with triggers exist.

## Construct the breaking schedule

For each invariant, write the smallest two-actor schedule that violates it:

| Race | Schedule | Typical damage |
| --- | --- | --- |
| Lost update | A reads, B reads, A writes, B writes | the later write silently discards the earlier; drifting counters, negative stock |
| Check-then-act | A checks, B checks, A acts, B acts | duplicate booking, signup, payment |
| Write skew | A and B each check a predicate, write disjoint rows | cross-row invariant broken |
| Stale holder | lock expires during a pause; old holder resumes | two writers with "exclusive" access |

If no schedule breaks the invariant — a single writer, or commutative and idempotent effects — record that and stop: a mechanism without a schedule is cost without benefit.

**Complete when:** each claimed race has a concrete schedule, or the invariant is shown interleaving-safe.

## Pick the weakest sufficient mechanism

Work down the ladder; stop at the first rung that closes the schedule:

1. **One atomic conditional statement.** `UPDATE ... SET stock = stock - 1 WHERE id = $1 AND stock > 0` with the affected row count checked; conditional writes. No window exists at all.
2. **Unique or exclusion constraint plus a handled conflict.** The database arbitrates the race; the conflict error path is part of the design, not an exception.
3. **Optimistic concurrency.** A version column or ETag compare-and-set; losers take the conflict path below. Fits low contention and stateless callers.
4. **Pessimistic row locks.** `SELECT ... FOR UPDATE` with one global acquisition order, minimal scope, and a hold time that never spans user interaction or an external call. Fits hot rows where optimistic retries would thrash.
5. **Serializable isolation with retry on serialization failure.** When the invariant spans rows that no single conditional statement or constraint can guard (write skew).
6. **Advisory or application lock.** When the contested resource has no row of its own.
7. **Distributed lock, honestly a lease.** Only when writers share no transactional arbiter. A distributed lock is expiring advice, not mutual exclusion: a paused holder can resume after expiry. Every protected effect therefore checks a monotonic **fencing token**, or the effect itself is idempotent with reconciliation. If the effect owner cannot check the token, the lock only reduces contention — correctness must come from idempotency.

Name the isolation level actually in force and the anomaly it admits: under read committed, two read-modify-write transactions still lose updates; snapshot levels change which schedules survive. The mechanism must close the schedule at the real level, not the imagined one.

**Complete when:** the chosen rung provably closes the schedule at the real isolation level, and each rejected weaker rung has a stated reason.

## Design the conflict path

The loser of every arbitrated race needs a defined disposition: retry against fresh state within attempt and time budgets with jitter; surface the conflict to the caller (409, version mismatch) with enough context to re-decide; merge when the domain allows it; or serialize through a single writer per key (route ordered delivery to `reliable-messaging`, queued work to `durable-background-jobs`).

Conflict retries recompute the decision from fresh reads — replaying the stale intent re-creates the race. Retries that include external effects stay idempotent under the same operation identity (`external-api-integration`). Under sustained contention, watch for starvation and livelock: a hot key where optimistic losers always retry is an undesigned queue.

**Complete when:** every conflict has a bounded disposition, and retry composition cannot multiply effects or act on stale intent.

## Fence at-most-one activity

Leader election, singleton workers, and distributed cron never achieve at-most-one-active; they achieve at-most-one *most of the time*, with overlap during pauses, partitions, and failover. Design for the overlap with rung 7's discipline: the leadership term is the lease and its monotonic generation is the fencing token, so every durable effect writes conditionally on the current generation and a resumed stale leader is rejected at the effect. Give each scheduled occurrence its own identity with a unique claim — one row per occurrence — instead of hoping only one instance fires; hand recurring-job mechanics to `durable-background-jobs`.

**Complete when:** the overlap window is stated, and every effect from the "single" actor is fenced by generation or idempotent under a stable effect key.

## Enumerate failure windows

| Window | Consequence | Guard |
| --- | --- | --- |
| Crash while holding a lock | contested state frozen | expiry or lease plus recovery to a legal state |
| Pause past expiry, then resume | two holders | fencing token checked at the effect |
| Lock or arbiter unavailable | no writer proceeds | explicit fail-closed or degrade decision, alarmed |
| Deadlock | mutual wait | one global acquisition order; timeout and victim retry |
| Hot-key conflict storm | retry amplification | capped jittered retries; per-key serialization above the store |

Fill only the rows the chosen mechanism can reach, and give each a signal an operator can see. Route sizing of PostgreSQL lock waits and deadlock frequency to `postgres-performance`.

**Complete when:** every reachable window has a guard and an observable signal.

## Prove under forced interleavings

A race test that merely runs many concurrent workers proves little — force the schedule:

- hold two writers at the read or check point with a barrier or injected pause, release together, assert the invariant and exactly one winner; the test fails when the mechanism is removed;
- exercise the conflict path deliberately: the constraint violation is handled, the optimistic loser retries once against fresh state, a single effect is asserted;
- for fencing: acquire, pause the holder past expiry, let a new holder proceed, resume the old one, assert its write is rejected;
- for deadlocks: acquire in both orders under load, assert the timeout and victim policy instead of a hang;
- one stress run with N writers asserting the invariant (balance never negative, exactly one active row) at the real store — races are store-specific, and in-memory fakes do not arbitrate like the real engine.

**Complete when:** removing the mechanism makes a deterministic proof fail, and the invariant survives a real-store stress run.

## Report

Lead with the verdict: the breaking schedule, the chosen mechanism and why weaker rungs fail, and residual windows with guards. For a diagnosis, prefer `schedule -> smallest mechanism -> conflict path -> one forced-interleaving proof`; do not restate the whole ladder. Separate facts from inference; label artifacts proposed, implemented locally, tested, or verified.
