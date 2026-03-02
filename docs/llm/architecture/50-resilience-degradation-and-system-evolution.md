# Resilience, degradation, and system evolution instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Designing or changing resilience behavior for sync/async service interactions
  - Defining timeout/retry/circuit-breaker/bulkhead/backpressure defaults
  - Defining graceful startup/shutdown and degraded-mode behavior
  - Planning rollout and migration strategies (canary, blue-green, strangler, feature flags)
  - Reviewing architecture for cascading-failure risk and rollout safety
- Do not load when: The task is a local implementation detail with no impact on failure handling, traffic control, or release strategy

## Purpose
- This document defines repository defaults for resilience controls and safe system evolution.
- The goal is predictable behavior under dependency failures, overload, and progressive delivery.
- The defaults below are mandatory unless an ADR explicitly approves an exception.

## Failure-domain and dependency classification protocol
Apply this protocol before choosing tools.

### 1) Classify dependency criticality
For each dependency, classify one of:
- `critical_fail_closed`: correctness/security/financial invariant depends on dependency (authz, payments, hard validation)
- `critical_fail_degraded`: service can continue in reduced mode for a bounded time (stale reads, deferred side effects)
- `optional_fail_open`: non-critical capability can be disabled without violating core contract (recommendations, enrichment, analytics)

### 2) Define per-dependency failure contract
For every dependency, document all fields:
- Timeout budget
- Retry class and retry budget
- Bulkhead limit and queue bound
- Fallback mode (`fail_closed`, `stale`, `defer_async`, `feature_off`, `fail_fast`)
- Circuit-breaker mode (`none`, `soft_retry_breaker`, `state_machine`)
- Observable degradation signal (`degradation_mode`, reason, start/end timestamps)

### 3) Enforce explicit owner and rollback authority
- Every critical dependency contract MUST declare owner team and on-call route.
- Every rollout MUST declare who can trigger rollback without additional approval.

## Timeout budgets and deadline propagation

### End-to-end budget defaults
- Interactive request default budget: 2500ms.
- Reserve 100ms for local response write and cleanup.
- If remaining budget before downstream call is less than 150ms, fail fast instead of calling dependency.

### Per-hop timeout defaults
- Read/query downstream call: 300ms.
- Write/command downstream call: 1000ms.
- Absolute per-hop cap: 2000ms.
- Outbound deadline formula:
  - `outbound_deadline = min(per-hop default, remaining_inbound_budget - 100ms)`

### Hard rules
- Every outbound call MUST have an explicit deadline.
- Infinite or implicit timeouts are prohibited.
- Deadline propagation from inbound context is mandatory.

## Retry budgets and jitter policy

### Retry eligibility
- Default: no retry.
- Retry is allowed only for retry-safe operations and transient failures.
- Retry-unsafe operations require idempotency key policy before retries are allowed.

### Retry defaults for interactive paths
- Max retry attempts: 1 retry (2 total attempts).
- Backoff: exponential with full jitter.
- Base delay: 50ms.
- Max delay: 250ms.

### Retry budget defaults
- Retry budget is mandatory per dependency.
- Default budget: extra retry attempts must not exceed 20% of primary attempts per dependency in a rolling 1-minute window.
- If retry budget is exhausted, disable retries and fail fast.
- Retry metrics MUST distinguish `initial_attempt` and `retry_attempt`.

### Never-retry conditions
- Validation and contract errors
- Authentication and authorization failures
- Business conflicts and not-found
- Caller cancellation

## Backpressure, load shedding, bulkheads, and circuit breaking

### Backpressure defaults
- Every inbound worker queue/channel MUST be bounded.
- Every dependency worker/concurrency lane MUST be bounded.
- If queue depth exceeds 80% of configured bound, enter degradation mode and shed optional work.

### Load shedding defaults
- At overload, prefer fast rejection over unbounded waiting.
- Default status for overload rejection:
  - `429` when local rate limit is exceeded
  - `503` when dependency/system capacity is exhausted
- Include `Retry-After` when recovery horizon is predictable.

### Bulkhead defaults
- Isolate concurrency per dependency using dedicated semaphore/pool.
- Default dependency concurrency limit: `min(64, 2*GOMAXPROCS)` per process.
- Do not share one global unbounded pool across all dependencies.
- For HTTP dependencies, set explicit per-host connection limits.

### Circuit-breaking defaults
- Default mode is `soft_retry_breaker` (retry budget + concurrency caps).
- State-machine circuit breaker is allowed only when repeated incidents show soft controls are insufficient.
- If state-machine breaker is enabled, default thresholds:
  - Open when failure rate is at least 50% in last 30s with at least 20 requests, or on 10 consecutive failures
  - Open-state cooldown: 30s
  - Half-open probe limit: 5 concurrent requests
- Circuit-breaker state transitions MUST be observable via metrics and logs.

## Graceful startup and shutdown

### Shutdown defaults
- On `SIGTERM`, mark readiness as not-ready immediately, then start draining.
- Stop accepting new traffic before closing dependencies.
- Drain timeout default: 20s.
- Kubernetes `terminationGracePeriodSeconds` MUST be greater than drain timeout + preStop budget (default minimum: 30s).
- Shutdown completion and timeout outcome MUST be logged.

### Startup defaults
- Use separate startup, readiness, and liveness checks.
- Liveness MUST not depend on external dependencies.
- Readiness MUST reflect only capabilities required for core traffic.
- Use startup probe when initialization can be slow.

### Probe anti-flap rule
- Do not flip readiness on single transient downstream failure.
- Apply short hysteresis (for example, 3 consecutive probe failures) before removing instance from traffic.

## Partial degradation and fallback policy

### Degradation mode model
Define explicit modes per service:
- `normal`: full functionality
- `degraded_optional_off`: optional features disabled, core flow preserved
- `degraded_read_only_or_stale`: writes or fresh reads restricted, bounded stale data allowed
- `emergency_fail_fast`: only critical minimal endpoints remain available

### Fallback decision rules
- `critical_fail_closed`: fail closed, no stale or synthetic fallback.
- `critical_fail_degraded`: allow bounded stale/deferred fallback only if invariants are preserved.
- `optional_fail_open`: disable capability and continue core response.

### Fallback defaults
- Stale-cache fallback default max staleness: 5 minutes unless stricter contract exists.
- Deferred processing fallback should return explicit async acknowledgment (`202`) with tracking ID.
- Every fallback activation MUST emit structured log + metric with dependency and mode.

### Dependency failure handling order
Use this order on dependency failure:
1. Apply timeout and retry policy within budget.
2. If still failing, apply circuit/bulkhead controls to contain blast radius.
3. Activate dependency-specific fallback mode.
4. If no safe fallback exists, fail fast with explicit error mapping.

## Evolution patterns and rollout safety

### Canary defaults
- Default canary traffic progression: 1% -> 5% -> 25% -> 50% -> 100%.
- Minimum soak per stage: 10 minutes for 1%/5%, 30 minutes for 25%/50%.
- Promotion is blocked if any rollout gate fails.

### Blue-green defaults
- Maintain two production-like environments with identical infra policies.
- Cutover must be reversible by routing switch.
- Rollback target after bad cutover: under 5 minutes.
- Do not run irreversible schema or data mutations before rollback window is closed.

### Strangler defaults
- Route at business-capability or endpoint boundary, not by shared-table hacks.
- Start with read paths, then write paths after parity evidence.
- Keep anti-corruption layer explicit while legacy and new paths coexist.
- Remove legacy path only after parity and incident-free observation window are met.

### Feature-flag defaults
- Feature flags MUST be managed through a vendor-agnostic contract (OpenFeature-compatible API).
- Risky features default to `off`.
- Every flag MUST have owner, expiry date, and rollback behavior.
- Release flags should be removed after stabilization (default target: within 2 releases).

## SLO/error-budget-aware rollout gates

### Error budget policy defaults
- Compliance window: rolling 28 days.
- If service is within budget, normal release process continues.
- If service exceeds error budget, freeze all non-P0/non-security changes until budget health is restored.
- If one incident consumes more than 20% of 28-day budget, postmortem and highest-priority reliability action item are mandatory.

### Burn-rate alert defaults
- Page on high-burn windows:
  - 1h/5m with burn rate 14.4
  - 6h/30m with burn rate 6
- Ticket on slow-burn window:
  - 3d/6h with burn rate 1
- Low-traffic guardrail is mandatory: do not fire burn-rate pages below minimum event volume threshold.

### Rollout gating rules
- Do not promote canary stage while page-level burn alert is active.
- Auto-rollback when two consecutive short windows fail promotion SLI thresholds.
- Rollout decision MUST include both service SLIs and dependency saturation signals.

## Anti-patterns
Treat each item as a review blocker unless an ADR explicitly accepts the risk.

- Unbounded retries without budget/jitter/time limit (retry storm)
- Missing explicit deadlines on inbound or outbound paths
- Shared unbounded worker/connection pools without bulkheads
- Unbounded in-memory queues instead of backpressure/load shedding
- Readiness tied directly to transient downstream failures causing fleet-wide flapping
- Liveness checks that depend on external systems, triggering restart storms
- Big-bang releases for high-risk changes without staged rollout or kill switch
- Canary without objective promotion gates and rollback automation
- Blue-green cutover with non-backward-compatible schema changes
- Feature flags without owner/expiry/cleanup plan
- Hidden fallback behavior that changes correctness semantics without contract update
- Degradation mode activation without observability signals

## MUST / SHOULD / NEVER

### MUST
- MUST define timeout, retry, bulkhead, and fallback policy per dependency.
- MUST enforce explicit deadline propagation for all outbound calls.
- MUST apply bounded retries with full jitter and retry budget.
- MUST bound queues/concurrency and implement overload shedding behavior.
- MUST implement graceful shutdown/startup with separate probes.
- MUST define explicit degradation modes and activation criteria.
- MUST use progressive delivery (canary or blue-green) for risky production changes.
- MUST enforce SLO/error-budget rollout gates and freeze policy.
- MUST keep feature flags owner-based, expiring, and rollback-capable.
- MUST make all resilience state transitions observable.

### SHOULD
- SHOULD prefer soft retry-breaking controls before introducing full circuit breaker state machines.
- SHOULD classify dependencies into fail-closed/fail-degraded/fail-open before coding fallback logic.
- SHOULD keep rollback operations reversible at routing/config layer.
- SHOULD test degradation paths in staging and game days.
- SHOULD remove temporary flags and migration bridges quickly after stabilization.

### NEVER
- NEVER rely on retries as primary recovery for non-idempotent operations.
- NEVER use infinite timeout, infinite retry, or unbounded buffering defaults.
- NEVER couple liveness to external dependency health.
- NEVER ship high-risk migration without rollback-safe plan.
- NEVER treat canary as safe if promotion gates are undefined.
- NEVER hide unsafe rollout risk behind manual heroics instead of explicit controls.

## Review checklist
Before approving resilience or evolution changes, verify:

- Dependency criticality classes are explicit and justified
- Per-dependency contract includes timeout/retry/bulkhead/fallback/circuit settings
- End-to-end and per-hop timeout budgets are documented and propagated
- Retry policy includes jitter, budget, and non-retry conditions
- Queue/concurrency bounds and overload behavior are explicit
- Bulkhead isolation exists for each critical dependency
- Circuit-breaker mode and thresholds are explicit if enabled
- Startup/readiness/liveness/shutdown lifecycle is correct and non-flapping
- Degradation modes are defined with observable activation/deactivation
- Canary/blue-green/strangler/feature-flag strategy is explicit where applicable
- Rollout gates include SLO burn-rate and error-budget policy checks
- Rollback is tested, fast, and operationally owned
- No blocked anti-patterns are introduced without approved ADR exception
