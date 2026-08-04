---
name: durable-background-jobs
description: "Lease-oriented discipline for durable background jobs. Use when designing, implementing, auditing, or diagnosing work that must survive request or worker failure: queues/workers and retries; visibility timeouts, heartbeats, stuck or poison jobs, and crash recovery; recurring jobs or scheduled billing; long backfills, checkpointing, cancellation, or drain. Route event/command delivery to reliable-messaging and multi-process business orchestration elsewhere."
---

# Durable Background Jobs

Use the **lease** as the lens for every job:

`identity -> durable acceptance -> claim/lease -> execute -> durable effect -> complete -> retry/recover -> prove`

A lease is expiring permission to attempt work, not proof that no other attempt exists. Design for overlapping attempts. Promise one durable business effect through idempotency, a unique ledger or conditional write, or reconciliation; never promise exactly-once execution.

## Choose the branch

Run only the branch and concerns the request needs. Record missing evidence as a gap instead of broadening the task.

- **Audit or diagnose:** trace the real producer-to-effect path and every caller of the shared transition code. Find the first illegal or ambiguous commit boundary, report the root cause, and propose the smallest fix plus a runnable proof. Keep code and production unchanged.
- **Design or plan:** define the in-scope contract and decisions. Supply SQL, pseudocode, matrices, and test cases only to the fidelity requested; label them proposed.
- **Build or fix:** patch the shared root cause locally and run the smallest proof that fails on the old behavior and passes on the new behavior.
- **Operate:** production enqueue, retry/requeue, schedule change, cancellation, purge, drain, backfill, or data repair requires explicit authorization for the exact action and targets. Preflight, execute only that action, then verify with fresh readback.

Hand event and command delivery to `reliable-messaging`; keep orchestration across several business processes outside this skill unless durable orchestration is itself requested. Hand PostgreSQL DDL and migration design to `postgres-schema-design`, and query, locking-load, index, or capacity work to `postgres-performance`. This skill retains job identity, transitions, leases, effects, and recovery.

**Complete when:** the branch, authority boundary, requested artifact, and excluded concerns are explicit.

## Decide whether a job should exist

Keep work synchronous when the caller needs the result, it reliably fits the request budget, and failure can be reported directly. Use a durable job only when work must outlive a request/process/deploy, or needs bounded retry, scheduling, throttling, checkpointing, operator recovery, or latency isolation.

Name the caller-visible acceptance point: completed now, durably accepted with a job ID, or rejected before acceptance. Backgrounding must not hide an error the caller still needs synchronously.

**Complete when:** the acceptance point, latency and durability need, and simpler synchronous alternative are decided.

## Pin the engine and identities

Identify the engine and version, topology, persistence owner, producer, workers, external effect owners, and time source. If the engine is unknown, state the assumed model rather than importing guarantees from one queue into another.

Read the matching execution reference before relying on implementation details:

- PostgreSQL or another database-backed queue: [references/database-backed.md](references/database-backed.md)
- visibility/acknowledgement broker such as SQS: [references/visibility-queue.md](references/visibility-queue.md)
- Temporal or another durable execution engine: [references/durable-engine.md](references/durable-engine.md)

Keep these identities distinct:

| Identity | Purpose |
| --- | --- |
| `job_id` | Stable logical job across attempts |
| producer deduplication key | One logical acceptance, with explicit scope and retention |
| `attempt_id` plus lease generation/token | One claim; rejects stale mutation |
| schedule occurrence ID | One intended civil or instant occurrence |
| business-effect key | One durable effect across overlapping attempts |

Version the payload envelope and any checkpoint. Record job type, tenant, correlation IDs, and immutable references. Define compatible worker versions, upgrade/routing behavior for queued payloads, and deduplication retention.

**Complete when:** every in-scope identity has a source, uniqueness scope, lifetime, and owner, and the cited engine contract covers delivery, claim, retry, scheduling, and recovery behavior actually used.

## Protect durable acceptance

Draw the producer commit boundary. Commit business state and the job row in one transaction when they share a database. Otherwise commit an outbox record with the business decision, publish it through `reliable-messaging`, and reconcile unpublished or unobserved records.

An enqueue after the business commit can lose work; an enqueue before it can create work for rolled-back state. Treat a timeout or ambiguous enqueue response as unknown. Retry with the same producer deduplication key and confirm acceptance from durable state.

**Complete when:** a crash or timeout between any two producer commits cannot silently lose the logical job or create an unbounded duplicate effect. The proof covers transaction rollback and, when enqueue acknowledgement can be lost, same-key retry plus durable readback.

## Define the lease and crash matrix

Use the smallest state set that expresses operationally different outcomes. Start here and omit unused states:

| From | Legal next states |
| --- | --- |
| `scheduled` | `queued`, `cancelled` |
| `queued` | `leased`, `cancelled` |
| `leased` | `succeeded`, `retry_wait`, `cancel_requested`, `terminal_failed` |
| `retry_wait` | `queued`, `cancelled`, `terminal_failed` |
| `cancel_requested` | `cancelled`, or `succeeded` when an already-committed effect wins |
| `terminal_failed` | `quarantined`, or audited manual retry |
| `quarantined` | audited manual retry |

`succeeded` means the effect and required result committed before completion/acknowledgement. Lease expiry routes through retry or reconciliation. Each transition is one atomic compare-and-set, transaction, or engine command.

For each lease define owner, attempt, generation/fencing token, acquisition, expiry, renewal cadence, server-side time source, maximum runtime, and the predicate for renewal, checkpoint, completion, and failure. A stale owner fails the conditional mutation. If the effect owner cannot check a fencing token, protect the effect with the business-effect key and reconcile ambiguity.

Size lease and heartbeat windows from measured runtime and pause/network tails. Renew with headroom and jitter. Once renewal is uncertain, stop starting irreversible effects; finish only already-started work whose idempotency or fencing contract remains safe.

Build a matrix for every reachable in-scope state and commit boundary: durable acceptance, claim, each durable effect/checkpoint, effect-before-complete, complete-response loss, lease expiry, and any requested retry, cancellation, schedule, or deploy boundary.

| State / crash point | Durable owner | Lease / commit fact | Next legal state | Disposition | Recovery | Signal | Executable test |
| --- | --- | --- | --- | --- | --- | --- | --- |

When the user asks for a matrix, copy these columns unchanged and fill every cell; merging columns makes missing transitions and tests hard to see. Include terminal/quarantine rows whenever retry or recovery is in scope. Mark impossible combinations and name the transaction or constraint that prevents them.

**Complete when:** every reachable in-scope row has one legal next state, recovery path, signal, and proof, and stale attempts cannot mutate current state.

## Make effects reentrant and failures bounded

Derive the business-effect key from stable business identity, not attempt identity. Prefer, in order:

1. one transaction that records the effect and idempotency result;
2. a unique effect ledger or conditional write that returns the prior result;
3. an external idempotency key with adequate retention;
4. reconciliation that distinguishes applied, unapplied, and unknown outcomes.

Treat `effect committed, completion unknown` as ambiguous completion. Consult the effect record or reconcile before replay, then converge duplicate attempts on the canonical result.

| Failure | Disposition |
| --- | --- |
| transient | Retry within attempt and elapsed-time/cost budgets |
| permanent | Terminal with a stable reason; retry after input or code changes |
| poison | Quarantine after a small bounded budget with payload version and evidence |
| operator-actionable | Pause or quarantine with owner, alert, runbook, and authorized recovery |

Set maximum attempts, elapsed retry window, per-tenant/global retry budget, capped exponential backoff, and jitter. Honor server retry hints within the budget. Retries release capacity and do not amplify an overloaded dependency. When retry or recovery is requested, classify all four failure rows or explicitly explain why a class is impossible for this job.

**Complete when:** duplicate or stale attempts cannot multiply the effect, and every failure reaches a bounded retry, cancellation, terminal, or quarantine disposition.

## Load operational branches only when needed

Read [references/operations.md](references/operations.md) when the request involves concurrency/fairness/backpressure, civil schedules or misfires, long-running checkpoints/cancellation, deploy/drain/versioning, retention/observability, or load/deploy proof. Apply only its in-scope sections.

## Prove the contract

Use the smallest executable proof at the real engine seam:

- deterministic tests for state transitions, retry timing, payload versions, and civil time when relevant;
- integration tests with multiple real workers and the real persistence/queue API;
- crash injection for every matrix row, especially effect-before-complete and lost completion/ack responses.

Fake time is for deterministic logic; keep one integration test for real claim/renewal timing. Record commands and results for local changes. For design or audit work, provide runnable cases and their required environment without claiming they ran.

When the user asks for **executable**, **runnable**, or **real** tests, a checklist is insufficient: include test source or a runnable harness, its setup command, and exact assertions. Run it when the required local engine exists; otherwise label it unrun and name the missing dependency.

Match the proof surface to the request. If the user asks for one focused check, give one executable scenario at the real engine seam that forces the relevant lease or acknowledgement transition; a handler mock or a final effect counter alone does not prove expiry, redelivery, or stale-token behavior. Do not attach the rest of the crash suite unless it can change that result.

**Complete when:** each in-scope invariant and matrix row maps to executable evidence. A build/fix includes fresh passing output; a design/audit labels unrun proof as proposed; an operation includes preflight and readback.

## Report

Lead with the verdict and include only relevant sections: execution contract, state/crash matrix, capacity/time, recovery/operations, proof, and authority/gaps. Keep facts separate from inference and label each artifact as proposed, implemented locally, tested, authorized live, or verified live.

For a narrow diagnosis, prefer `cause -> smallest safe contract -> one focused proof -> authority/gap`; do not restate the full job lifecycle or unrelated completion criteria.
