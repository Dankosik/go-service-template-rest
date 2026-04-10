---
name: go-reliability-spec
description: "Design reliability requirements for Go services: timeouts, deadlines, retry budgets, overload handling, degradation, lifecycle behavior, recovery, and resilience verification."
---

# Go Reliability Spec

## Purpose
Define or review reliability behavior so failure handling, timeout policy, retry policy, overload response, degradation, and shutdown behavior are explicit, bounded, and testable.

## Specialist Stance
- Treat reliability as explicit failure contracts, budgets, backpressure, and lifecycle behavior.
- Prefer bounded retries, deadlines, concurrency limits, degradation rules, and rollback-safe defaults over optimistic recovery.
- Make caller-visible and operator-visible failure semantics testable before implementation.
- Hand off service decomposition, API modeling, data/cache mechanics, and security policy when reliability is only a dependent seam.

## Scope
- define per-dependency failure contracts and criticality classes
- define timeout and deadline policy, including propagation and fail-fast behavior
- define retry eligibility, retry budgets, jitter policy, and never-retry categories
- define overload containment, bulkheads, bounded queues, and rejection semantics
- define degradation, fallback, startup, readiness, liveness, and shutdown behavior
- define rollout, rollback, and recovery expectations for risky changes
- define verification obligations for resilience and failure behavior

## Boundaries
Do not:
- take primary ownership of service decomposition, API resource modeling, physical schema design, or cache-key mechanics
- prescribe low-level middleware, worker, queue, or shutdown-hook code as the main output
- let observability, security, or delivery mechanics displace the core reliability contract unless they directly affect failure behavior
- treat broad platform policy as the main domain when the task is local failure behavior or resilience design

## Core Defaults
- Classify dependency criticality before choosing resilience mechanisms.
- Treat explicit deadlines, bounded retries, and bounded concurrency as mandatory defaults.
- Prefer simpler controls first: deadline, retry budget, bulkhead, and shedding before more complex control loops.
- Keep reliability behavior compatible with mixed-version deployments and partial rollout.
- Missing critical reliability facts are blockers, not details to improvise during implementation.

## Expertise

### Dependency Criticality And Failure Contracts
- Classify each dependency as exactly one:
  - `critical_fail_closed`
  - `critical_fail_degraded`
  - `optional_fail_open`
- For each dependency, define:
  - timeout/deadline budget
  - retry class and retry budget
  - bulkhead limit and queue bound
  - fallback mode
  - circuit mode
  - observable degradation signal
  - owner or operational accountability
- Ownerless or ambiguous failure contracts are not sign-off quality.

### Timeout And Deadline Design
- Default interactive end-to-end budget: `2500ms`.
- Reserve `100ms` for response write and cleanup when decomposing downstream budgets.
- Fail fast when remaining inbound budget is less than `150ms`.
- Default per-hop timeouts:
  - read/query: `300ms`
  - write/command: `1000ms`
  - absolute cap: `2000ms`
- Default outbound deadline formula:
  - `min(per-hop default, remaining inbound budget - 100ms)`
- Every outbound call should have an explicit deadline; infinite timeout is unacceptable.
- Deadline propagation from inbound context is mandatory.

### Retry Budget And Jitter
- Default retry policy is no retry.
- Retries are allowed only for transient failures on retry-safe operations.
- Retry-unsafe operations need an explicit idempotency contract before retries are acceptable.
- Default interactive retry policy:
  - max attempts: `2` total
  - backoff: exponential with full jitter
  - base delay: `50ms`
  - max delay: `250ms`
- Default retry budget:
  - extra retry attempts must stay within `20%` of primary attempts in a rolling `1m` window
  - when budget is exhausted, disable retries and fail fast
- Never retry:
  - validation failures
  - authentication or authorization failures
  - not-found or business conflicts
  - caller cancellation

### Overload, Backpressure, And Bulkheads
- Every inbound queue or channel must be bounded.
- Every worker lane and dependency concurrency lane must be bounded.
- If queue depth exceeds `80%` of its configured bound, enter degradation and shed optional work.
- Prefer fast rejection over unbounded waiting.
- Keep rejection semantics explicit:
  - `429` for policy throttling
  - `503` for capacity or dependency exhaustion
- Use `Retry-After` when the recovery horizon is predictable.
- Isolate dependencies with bulkheads; do not share one global unbounded pool.
- Default dependency concurrency limit per process:
  - `min(64, 2*GOMAXPROCS)`

### Circuit Breaking And Containment
- Default mode is a soft breaker: retry budget, bulkhead, and shedding.
- State-machine circuit breakers are justified only when simpler controls are not enough.
- If a state-machine breaker is used, make thresholds explicit:
  - open at failure rate `>= 50%` in `30s` with at least `20` requests, or after `10` consecutive failures
  - open cooldown: `30s`
  - half-open probe concurrency: `5`
- Circuit state transitions must be visible in logs and metrics.

### Startup, Readiness, Liveness, And Shutdown
- Split probe responsibilities:
  - `/livez` for restart decision only
  - `/readyz` for traffic admission only
  - `/startupz` for startup completion only
- Liveness must not depend on external dependencies.
- Readiness should represent only capabilities required for core traffic.
- Add anti-flap hysteresis for readiness, such as `3` consecutive failures.
- On shutdown:
  1. set draining flag
  2. fail readiness
  3. stop new traffic or work
  4. drain in-flight work
  5. flush telemetry providers
  6. exit before hard kill
- Default drain timeout: `20s`.
- `terminationGracePeriodSeconds` should exceed drain timeout plus preStop budget, with a practical default minimum of `30s`.
- Long-lived or hijacked connections need an explicit shutdown policy.

### Degradation And Fallback
- Use an explicit degradation mode model:
  - `normal`
  - `degraded_optional_off`
  - `degraded_read_only_or_stale`
  - `emergency_fail_fast`
- Fallback must match criticality class:
  - `critical_fail_closed`: fail closed
  - `critical_fail_degraded`: bounded stale or deferred fallback only if invariants stay intact
  - `optional_fail_open`: disable optional capability and continue core flow
- Default stale fallback max staleness: `5m` unless a stricter contract exists.
- Deferred fallback should use explicit async acknowledgment with a tracking reference.
- Every fallback activation and deactivation should emit a structured signal.
- Dependency failure handling should follow a clear order:
  1. timeout + retry within budget
  2. containment controls
  3. dependency-specific fallback
  4. explicit fail-fast when no safe fallback exists

### API-Visible Reliability Semantics
- Make retry classification explicit for every endpoint or externally visible action.
- Retry-unsafe operations that may be retried by clients should use explicit idempotency behavior.
- Make overload behavior explicit, including `429`, `503`, and `Retry-After`.
- Use explicit async acknowledgment for long-running or variable-latency side effects.
- Do not return fake synchronous success for queued or unfinished work.
- Keep boundary validation and input size limits strong enough to help overload containment.

### Async And Distributed Reliability
- Do not add synchronous hops by default when the flow can be safely asynchronous.
- For async or state-changing flows:
  - use outbox or equivalent atomic linkage between state change and publish
  - use inbox or dedup storage for side-effecting consumers
  - acknowledge or commit offsets only after durable side effects
- Default async retry policy:
  - total attempts: `8`
  - exponential backoff with jitter
  - base: `1s`
  - factor: `2`
  - cap: `5m`
  - never infinite retries
- Non-retryable or poison messages must go to a DLQ with diagnostic context.
- Keep cross-service process reliability explicit:
  - invariant ownership
  - state model
  - compensation or forward-recovery per step
  - reconciliation ownership and cadence where convergence matters
- Do not use 2PC or cross-system dual writes as the default strategy.

### Observability, Recovery, And Release Safety
- Reliability behavior must be visible through low-cardinality logs, metrics, and traces.
- Required runtime visibility includes:
  - timeout and retry budget behavior
  - overload, shedding, and bulkhead state
  - degradation mode transitions
  - rollback and recovery triggers
- If SLI/SLO-based control is used, make `good/total` semantics explicit and keep budget math consistent over a rolling window.
- Burn-rate paging should include event-floor guards for low-traffic services.
- Risky changes need explicit staged rollout checkpoints and rollback triggers.
- Backup strategy is valid only when restore behavior is tested; backup-only claims are insufficient.

### Data Evolution And Recovery
- Schema and data changes that affect reliability should follow:
  - `expand -> migrate/backfill -> contract`
- Preserve mixed-version compatibility until removal of old behavior is demonstrably safe.
- Backfills should be idempotent, resumable, throttled, and have explicit kill criteria.
- Destructive steps require an explicit rollback class:
  - `safe`
  - `conditional`
  - `restore-based`
- Do not contract schema while downstream consumers still depend on old semantics.

## Decision Quality Bar
Major reliability recommendations should make the following explicit:
- the failure scenario and affected invariant
- dependency criticality
- at least two viable options when the decision is nontrivial
- selected timeout, retry, bulkhead, fallback, lifecycle, and recovery values
- verification signals and runtime evidence
- cross-domain impact and reopen conditions

Narrative reliability claims without explicit contract values are incomplete.

## Deliverable Shape
Return reliability work in a compact, reviewable form:
- `Failure Contracts`
- `Timeout, Retry, And Bulkhead Policy`
- `Degradation And Lifecycle Behavior`
- `Recovery And Rollback Expectations`
- `Verification Obligations`
- `Assumptions And Residual Risks`

## Escalate When
Escalate if:
- critical dependencies do not have explicit failure contracts
- outbound calls rely on implicit or infinite timeouts
- retry policy lacks bounded attempts, jitter, or a retry budget
- queue depth or concurrency is unbounded
- degradation or fallback behavior lacks entry, exit, or recovery criteria
- startup, readiness, liveness, or shutdown behavior is contradictory or undefined
- recovery or rollback assumptions materially affect safety but remain unclear
- reliability-visible behavior changes at the API boundary without an explicit contract
