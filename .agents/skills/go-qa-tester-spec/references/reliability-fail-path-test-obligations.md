# Reliability Fail-Path Test Obligations

## When To Load
Load this when test strategy must cover timeouts, cancellation, retry budgets, retry/non-retry classification, idempotency under retry, backpressure, degradation, startup/shutdown, poison messages, or async recovery.

## Source Grounding
- Use approved reliability requirements for timeout durations, retry eligibility, backpressure policy, and degradation outcomes.
- Use Go context and race-detector docs to calibrate cancellation and concurrency proof, but do not design reliability behavior here.
- Use repo commands to name runnable evidence only after selecting the proof level.

## Selected/Rejected Level Examples
| Fail-path obligation | Selected level | Rejected level | Why |
| --- | --- | --- | --- |
| Pure retry classifier maps errors to retry/non-retry/poison | Unit | E2E | The policy can be proven deterministically without runtime orchestration. |
| DB query cancellation or transaction rollback on context deadline | Integration | Fake-only unit | The proof depends on `Context` propagation into real DB operations or transaction semantics. |
| Worker shutdown closes input, drains in-flight work, and joins goroutines | Targeted component test under race execution | Broad API smoke | The risky behavior is goroutine lifecycle and shared state, not HTTP success. |
| Retry after transport timeout must not duplicate a side effect | Contract plus integration if durable idempotency is storage-backed | Happy-path unit | The observable is duplicate suppression across the public or persistence boundary. |
| Dependency outage follows degraded-mode policy | Integration or component test with controlled dependency failure | Live e2e outage test | Controlled failure proves the exact degraded outcome without fragile environment dependence. |
| Poison async message is escalated and not retried forever | Integration or process-level component test | Unit-only classifier | The strategy must prove durable state, retry stopping, and escalation signal where those are owned. |

## Scenario Matrix Examples
| Reliability behavior | Required rows | Selected proof | Pass/fail observable |
| --- | --- | --- | --- |
| Timeout propagation | completes before deadline, dependency exceeds deadline, caller cancels request | Unit with fake clock/dependency or integration for DB/network boundary | context-derived error remains recognizable, no success side effect after cancellation |
| Retry budget | first attempt succeeds, transient failure then success, exhausted retries, non-retryable error | Unit for policy, integration when side effects are durable | attempt count, backoff budget if specified, final error class, no duplicate durable write |
| Backpressure or load shedding | under limit accepted, over limit rejected or queued as specified, recovery after pressure drops | Component or contract depending on boundary | explicit overload status/error, queue size/state, no unbounded goroutine or memory growth claim |
| Graceful shutdown | no in-flight work, in-flight completes before deadline, in-flight exceeds shutdown deadline | Component test under race when shared state exists | readiness/draining state, joined goroutines, flushed/abandoned work per policy |
| Async failure classes | retryable, non-retryable, poison, replay duplicate, stuck item reconciliation | Integration or process-level component | state transition, retry counter, DLQ/escalation, reconciliation marker |

## Pass/Fail Observables
- Cancellation proof preserves recognizable context/deadline semantics when the contract depends on it.
- Retry tests name retryable and non-retryable classes separately.
- Idempotency proof covers duplicate side-effect suppression, not just duplicate response shape.
- Shutdown proof names lifecycle state and completion/join signal; it must not rely on sleep luck.
- Race validation is required for shared-state or goroutine-lifecycle changes and must execute the risky path.
- Degradation proof states whether failure is fail-open, fail-closed, queued, skipped, or escalated.

## Exa Source Links
- [Canceling in-progress operations](https://go.dev/doc/database/cancel-operations)
- [Executing transactions](https://go.dev/doc/database/execute-transactions)
- [Data Race Detector](https://go.dev/doc/articles/race_detector.html)
- [Go security best practices](https://go.dev/doc/security/best-practices)
- [go command testing flags](https://pkg.go.dev/cmd/go#hdr-Testing_flags)

