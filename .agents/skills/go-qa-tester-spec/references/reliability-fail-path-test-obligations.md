# Reliability Fail-Path Test Obligations

## Behavior Change Thesis
When loaded for symptom "the behavior involves timeouts, retries, cancellation, shutdown, degradation, or async failure", this file makes the model require deterministic fail-path proof instead of likely mistake "say to test retries/timeouts without failure classes or side-effect observables."

## When To Load
Load this when test strategy must cover timeout propagation, caller cancellation, retry budgets, retry/non-retry/poison classification, idempotency under retry, backpressure, degradation, startup/shutdown, or async recovery.

## Decision Rubric
- Cancellation proof must preserve recognizable deadline/cancellation semantics when the contract depends on them, and must prove no success side effect after cancellation.
- Retry proof must separate first-success, transient-then-success, exhausted retry, and non-retryable paths; add poison handling for async flows.
- Idempotency under retry must prove duplicate side-effect suppression, not just duplicate response shape.
- Shutdown proof must use deterministic lifecycle signals: drain, join, flush, abandon, readiness/draining state, or deadline outcome. Do not rely on sleep luck.
- Backpressure proof must name accepted, rejected, queued, shed, or degraded behavior plus bounded resource expectations if approved.
- Degradation proof must state fail-open, fail-closed, queued, skipped, stale-read, or escalated behavior from the approved reliability spec.
- Use integration only when real DB/network/cache/runtime behavior is the proof target; use component/unit proof when deterministic fakes can honestly drive the failure.

## Imitate
| Reliability Behavior | Required Rows | Selected Proof | Observable To Copy |
| --- | --- | --- | --- |
| Timeout propagation | completes before deadline; dependency exceeds deadline; caller cancels | unit with controlled dependency or integration for DB/network boundary | recognizable context-derived error, no success side effect after cancellation |
| Retry budget | first attempt succeeds; transient failure then success; exhausted retries; non-retryable error | unit for policy, integration when side effects are durable | attempt count, final error class, no duplicate durable write |
| Graceful shutdown | no in-flight work; in-flight completes before deadline; in-flight exceeds deadline | component scenario under `-race` when shared state exists | draining state, joined goroutines, flushed or abandoned work per policy |
| Poison async message | retryable; non-retryable; poison; replay duplicate; stuck item reconciliation | integration or process component | state transition, retry counter, DLQ/escalation, reconciliation marker |

## Reject
- "Test timeout" with no controlled slow dependency or cancellation trigger.
- "Test retry" without non-retryable and exhausted-retry rows.
- "Run race tests" without a scenario that executes worker lifecycle or shared state.
- "Shutdown should complete" without a join/drain/flush/abandon observable.
- "Degraded mode works" without naming the approved degraded outcome.

## Agent Traps
- Do not invent timeout durations, retry counts, poison policy, or degradation mode. Mark missing reliability semantics as blockers.
- Do not use live outage e2e as the primary proof when a controlled dependency failure can prove the behavior deterministically.
- Do not let idempotency proof stop at response equality; durable side effects are the usual bug.
- Do not claim cancellation propagation if the planned observable only checks "some error."

## Validation Shape
For each reliability behavior, name failure trigger, failure class, selected proof level, side-effect observable, and whether race or integration execution is required.
