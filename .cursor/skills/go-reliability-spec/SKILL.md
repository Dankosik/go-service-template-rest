---
name: go-reliability-spec
description: "Design reliability-first specifications for Go services in a spec-first workflow. Use when planning or revising timeout/deadline, retry budget, backpressure, degradation, startup/shutdown, and rollout/rollback safety behavior before coding and you need explicit failure contracts and resilience acceptance criteria. Skip when the task is a local code fix, endpoint-only payload/schema design, SQL schema-only modeling, CI/container setup, or low-level implementation of middleware/worker/runtime code."
---

# Go Reliability Spec

## Purpose
Create a clear, reviewable reliability specification package before implementation. Success means failure behavior and resilience controls are explicit, defensible, and directly translatable into implementation and test work.
Use `Hard Skills` as the normative domain baseline for decision quality and reliability-risk control; use workflow sections below for execution sequence and artifact synchronization.

## Scope And Boundaries
In scope:
- define per-dependency failure contracts and criticality classes (`critical_fail_closed`, `critical_fail_degraded`, `optional_fail_open`)
- define timeout and deadline policy (end-to-end budget, per-hop caps, propagation rules, fail-fast thresholds)
- define retry eligibility, retry budgets, jitter policy, and never-retry categories
- define overload containment policy (bounded queues, bulkheads, load shedding, rejection semantics)
- define circuit-breaking policy and containment escalation rules
- define graceful lifecycle policy (startup/readiness/liveness responsibilities and shutdown draining semantics)
- define degradation/fallback mode model, activation and recovery criteria
- define rollout and rollback reliability gates for risky changes
- define reliability acceptance obligations for `70-test-plan.md`
- synchronize reliability implications across affected spec artifacts
- produce reliability deliverables that remove hidden "decide later" gaps

Out of scope:
- primary ownership of service decomposition and ownership topology
- endpoint-level API payload and error-schema design beyond reliability semantics
- primary ownership of distributed workflow topology and saga decomposition
- primary ownership of SQL ownership/DDL/migration implementation mechanics
- primary ownership of cache topology, keying, and invalidation strategy
- primary ownership of SLI/SLO target governance and alert routing
- primary ownership of secure-coding controls and threat catalog
- primary ownership of CI/CD implementation mechanics and container hardening details
- implementation-level coding of middleware, retry wrappers, worker pools, or shutdown hooks before spec sign-off

## Hard Skills
### Reliability Spec Core Instructions

#### Mission
- Convert failure behavior into enforceable pre-coding contracts for timeout/deadline, retry, containment, degradation, lifecycle, and rollout safety.
- Protect `Gate G2` readiness by eliminating hidden "decide in implementation" reliability gaps.
- Preserve core invariants under dependency failure, overload, and mixed-version rollout.

#### Default Posture
- Classify dependency criticality before selecting resilience mechanisms.
- Treat explicit deadlines, bounded retries, and bounded concurrency as mandatory defaults.
- Prefer simpler controls first (`deadline + retry budget + bulkhead + shedding`) before complex control loops.
- Keep reliability policy compatible with rolling/canary deployment where old and new code coexist.
- Treat missing critical reliability facts as blockers until bounded as `[assumption]` with owner and unblock path.

#### Spec-First Workflow Competency
- Enforce phase-aware behavior from `docs/spec-first-workflow.md`; close reliability decisions in spec phase, not coding phase.
- Keep `55-reliability-and-resilience.md` as primary reliability artifact.
- Synchronize reliability implications in `20/30/40/50/60/70/80/90` when affected.
- Require `REL-###` linkage for every material reliability decision and affected artifact section.
- Treat unresolved timeout/retry/degradation/rollback semantics as blocker conditions for `Gate G2`.

#### Dependency Criticality And Failure-Contract Competency
- Classify each dependency as exactly one:
  - `critical_fail_closed`
  - `critical_fail_degraded`
  - `optional_fail_open`
- For each dependency, require explicit contract fields:
  - timeout/deadline budget
  - retry class and retry budget
  - bulkhead limit and queue bound
  - fallback mode (`fail_closed`, `stale`, `defer_async`, `feature_off`, `fail_fast`)
  - circuit mode (`none`, `soft_retry_breaker`, `state_machine`)
  - observable degradation signal and owner
- Every critical dependency contract must include owner team, on-call route, and rollback authority.
- Ownerless or ambiguous failure contracts are not sign-off ready.

#### Timeout And Deadline Competency
- Default interactive end-to-end budget: `2500ms`.
- Reserve `100ms` for response write/cleanup in downstream budget decomposition.
- Fail fast when remaining inbound budget is less than `150ms`.
- Default per-hop timeouts:
  - read/query: `300ms`
  - write/command: `1000ms`
  - absolute cap: `2000ms`
- Enforce outbound deadline formula:
  - `outbound_deadline = min(per-hop default, remaining_inbound_budget - 100ms)`
- Every outbound call must have explicit deadline; implicit/infinite timeout is prohibited.
- Deadline propagation from inbound context is mandatory.

#### Retry Budget And Jitter Competency
- Default retry policy is no retry.
- Retries are allowed only for transient failures on retry-safe operations.
- Retry-unsafe operations require explicit idempotency contract before retries are allowed.
- Default interactive retry policy:
  - max attempts: `2` total (`1` retry)
  - backoff: exponential, full jitter
  - base delay: `50ms`
  - max delay: `250ms`
- Mandatory retry budget per dependency:
  - extra retry attempts must be `<= 20%` of primary attempts in rolling `1m`
  - when budget is exhausted, retries are disabled and flow fails fast
- Never retry:
  - validation/contract failures
  - authn/authz failures
  - not-found/business conflict
  - caller cancellation
- Require observability split between `initial_attempt` and `retry_attempt`.

#### Overload, Backpressure, And Bulkhead Competency
- Every inbound queue/channel must be bounded.
- Every dependency worker lane/concurrency lane must be bounded.
- If queue depth exceeds `80%` of configured bound, trigger degradation and shed optional work.
- Overload response must prefer fast rejection over unbounded waiting.
- Rejection semantics must distinguish:
  - `429` for policy/rate-limit throttling
  - `503` for dependency/system capacity exhaustion
- Use `Retry-After` when recovery horizon is predictable.
- Enforce per-dependency bulkhead isolation; do not share one global unbounded pool.
- Default dependency concurrency limit per process:
  - `min(64, 2*GOMAXPROCS)`

#### Circuit-Breaking And Containment Competency
- Default mode is `soft_retry_breaker` (retry budget + bulkhead + shedding).
- State-machine circuit breaker is allowed only with incident evidence that soft controls are insufficient.
- If state-machine breaker is used, require explicit thresholds:
  - open at failure rate `>= 50%` in `30s` with at least `20` requests, or `10` consecutive failures
  - open cooldown: `30s`
  - half-open probe concurrency: `5`
- Circuit state transitions must be observable in logs and metrics.

#### Startup, Readiness, Liveness, And Shutdown Competency
- Split probe semantics:
  - `/livez`: restart decision only
  - `/readyz`: traffic admission only
  - `/startupz`: startup completion only
- Liveness must not depend on external dependencies.
- Readiness should represent only capabilities required for core traffic.
- Enforce anti-flap hysteresis for readiness (for example, `3` consecutive failures).
- On shutdown (`SIGTERM`/`SIGINT`) enforce deterministic order:
  1. set draining flag
  2. fail readiness immediately
  3. stop new traffic/work
  4. drain in-flight work
  5. flush telemetry providers
  6. exit before hard kill
- Default drain timeout: `20s`.
- `terminationGracePeriodSeconds` must exceed drain timeout plus preStop budget (default minimum `30s`).
- Long-lived/hijacked connections must have explicit shutdown behavior.

#### Degradation And Fallback Competency
- Define explicit degradation mode model:
  - `normal`
  - `degraded_optional_off`
  - `degraded_read_only_or_stale`
  - `emergency_fail_fast`
- Fallback decisions must follow criticality class:
  - `critical_fail_closed`: fail closed
  - `critical_fail_degraded`: bounded stale/deferred fallback only if invariants are preserved
  - `optional_fail_open`: disable optional capability and continue core flow
- Default stale fallback max staleness: `5m` unless stricter contract exists.
- Deferred fallback should use explicit async acknowledgement (`202`) with tracking ID.
- Every fallback activation/deactivation must emit structured signal with dependency and mode.
- Enforce dependency failure handling order:
  1. timeout + retry within budget
  2. containment controls (bulkhead/circuit/shedding)
  3. dependency-specific fallback
  4. explicit fail-fast when no safe fallback exists

#### API And Cross-Cutting Reliability Semantics Competency
- Reliability-visible behavior must be explicit in contract artifacts:
  - retry classification per endpoint
  - idempotency requirements for retry-unsafe operations
  - overload semantics (`429`/`503`, `Retry-After`)
  - async acknowledgement (`202` + operation resource) for long-running flows
- Default idempotency policy for retried retry-unsafe operations:
  - key required (`Idempotency-Key` or equivalent)
  - dedup TTL: `24h`
  - scope includes tenant/account + operation + route/method
  - same key + same payload => equivalent outcome
  - same key + different payload => conflict (`409`/`ABORTED`)
- Do not return fake sync success for queued/unfinished side effects.
- Enforce strict boundary validation and input size limits for overload protection.

#### Sync, Async, And Distributed Workflow Reliability Competency
- Do not add synchronous hops by default when workflow can be async.
- For async/state-changing flows:
  - outbox-equivalent atomic linkage is mandatory for state-change + publish
  - inbox/dedup store is mandatory for side-effecting consumers
  - ack/offset commit happens only after durable side effects
- Async retry defaults:
  - bounded exponential backoff with jitter
  - default processing attempts: `8` total (`1` initial + `7` retries)
  - base `1s`, factor `2`, cap `5m`
  - no infinite retries
- Non-retryable/poison messages must go to DLQ with diagnostic context.
- Enforce distributed workflow reliability hygiene:
  - explicit invariant ownership
  - explicit workflow state model
  - compensation or forward-recovery contract per step
  - reconciliation ownership/cadence for eventual-consistency critical flows
- Do not use 2PC or cross-system dual writes as default consistency strategy.

#### Observability, SLO, And Budget-Gate Competency
- Reliability policy must be observable via low-cardinality logs/metrics/traces.
- Required runtime visibility includes:
  - timeout/retry budget behavior
  - overload/shedding/bulkhead state
  - degradation mode transitions
  - rollback and rollout gate outcomes
- Use SLI/SLO defaults with explicit `good/total` semantics and 28-day budget tracking.
- Burn-rate defaults for release/degradation gates:
  - page: `1h/5m` at `14.4`
  - page: `6h/30m` at `6`
  - ticket: `3d/6h` at `1`
- Burn-rate paging must include event-floor guards for low-traffic services.
- Reliability gates must consume both service SLI state and dependency saturation signals.

#### Delivery And Quality-Gate Competency
- Translate reliability decisions into executable obligations in `70-test-plan.md` and CI/release gates when relevant.
- Reliability-sensitive plans must include repository-native validation path:
  - `make test`
  - `make test-race` for concurrency-affected paths
  - `make test-integration` for dependency/degradation paths
- If API contract/migrations are impacted, include compatibility and migration validation gates.
- Risky changes must include staged rollout checkpoints and explicit rollback trigger authority in `60-implementation-plan.md`.

#### Data Evolution And Recovery Competency
- Schema/data changes that affect reliability must follow phased rollout:
  - `Expand -> Migrate/Backfill -> Contract`
- Keep schema/application mixed-version compatibility until contract phase gates pass.
- Backfills must be idempotent, resumable, throttled, and kill-criteria bounded.
- Contract/destructive steps require explicit rollback class (`safe`, `conditional`, `restore-based`) and limitations.
- Backup strategy is valid only with restore-drill evidence; backup-only claims are insufficient.
- Do not contract schema while downstream consumers still depend on old semantics.

#### Evidence Threshold Competency
- Every major reliability decision (`REL-###`) must include:
  1. decision ID, phase, and owner
  2. context and failure scenario
  3. dependency criticality and invariant impact
  4. minimum two options
  5. selected option and at least one rejected option with reason
  6. explicit contract values (timeout/retry/bulkhead/fallback/lifecycle/rollout)
  7. verification obligations (tests + required runtime signals)
  8. cross-domain impact (architecture/API/data/security/observability/delivery/performance)
  9. reopen conditions and linked blockers
- Narrative reliability claims without explicit numeric/default contract values are invalid.

#### Assumption And Uncertainty Discipline
- Mark unknown critical facts as `[assumption]` immediately.
- Keep assumptions bounded, testable, and linked to concrete decision IDs.
- Resolve assumptions in current pass when source-backed validation is possible.
- Promote unresolved critical assumptions to `80-open-questions.md` with owner and unblock condition.
- Never hide uncertainty in generic wording or defer critical reliability decisions to coding.

#### Review Blockers For This Skill
- Missing criticality class or per-dependency failure contract for changed critical dependencies.
- Missing explicit outbound deadlines or reliance on infinite/implicit timeout defaults.
- Retry policy without bounded attempts, jitter, retry budget, and never-retry class.
- Unbounded queue/concurrency or missing overload rejection semantics.
- Degradation/fallback policy without entry/exit criteria and observable state transitions.
- Startup/readiness/liveness/shutdown policy missing or contradictory.
- Rollout/rollback policy without explicit gates, triggers, and authority.
- Reliability-visible API semantics changed without contract-level updates.
- Critical reliability uncertainty deferred to coding instead of blocker tracking.

## Working Rules
1. Determine current `docs/spec-first-workflow.md` phase and target gate before drafting decisions.
2. Set phase-specific output targets:
   - Phase 0: establish reliability baseline in `55-reliability-and-resilience.md` and seed blockers in `80-open-questions.md`
   - Phase 1: define architecture-shaping reliability constraints for `20-architecture.md` and sequencing constraints for `60-implementation-plan.md`
   - Phase 2 and later: maintain `55/80/90` and update impacted `20/30/40/50/60/70` as needed
3. Apply `Hard Skills` defaults by default. Any deviation must be explicit, justified, and linked to decision ID (`REL-###`) plus reopen criteria.
4. Load context using this skill's dynamic loading rules and stop when five reliability axes are source-backed: dependency criticality, timeout/retry contract, overload containment, degradation lifecycle, rollout/rollback safety.
5. Classify each critical dependency first, then define explicit contract fields: timeout, retry class/budget, bulkhead bound, fallback mode, circuit mode, observability trigger, and owner/rollback authority.
6. For each nontrivial reliability decision, compare at least two options and select one explicitly.
7. Assign decision ID (`REL-###`) and owner for each major reliability decision.
8. Record trade-offs and cross-domain impact (architecture, API, data/cache, security, observability, delivery, performance).
9. Mark missing critical facts as `[assumption]`; keep assumptions bounded and either validate in the current pass or move them to `80-open-questions.md` with owner and unblock condition.
10. If uncertainty blocks a safe reliability decision, record it in `80-open-questions.md` with concrete next step.
11. Keep `55-reliability-and-resilience.md` as primary artifact and synchronize reliability implications in affected artifacts.
12. Verify internal consistency: no contradictory timeout/retry/degradation policy and no critical reliability decisions deferred to coding.
13. Run final blocker check against `Hard Skills -> Review Blockers For This Skill` before closing a pass.

## Reliability Decision Protocol
For every major reliability decision, document:
1. decision ID (`REL-###`) and current phase
2. owner role
3. context and failure scenario
4. dependency criticality class and invariant impact
5. options (minimum two)
6. selected option with rationale
7. at least one rejected option with explicit rejection reason
8. contract details:
   - timeout/deadline budget and propagation
   - retry eligibility, attempts, budget, and jitter
   - queue bounds, bulkhead isolation, and shedding behavior
   - circuit mode and thresholds (if state-machine is used)
   - fallback/degradation mode entry and exit criteria
   - observability trigger for reliability state transitions
   - startup/readiness/liveness/shutdown behavior
   - rollout promotion/rollback triggers and authority
9. verification obligations (tests and required signals)
10. cross-domain impact and affected artifacts
11. reopen conditions and linked open-question IDs (if any)

## Output Expectations
- Response format:
  - `Decision Register`: accepted `REL-###` decisions with rationale and trade-offs
  - `Artifact Update Matrix`: required updates for `55/80/90` and status for impacted `20/30/40/50/60/70`
  - `Assumptions`: active `[assumption]` items and resolution path
  - `Open Blockers`: unresolved reliability items for `80-open-questions.md` with owner and unblock condition
  - `Sign-Off Delta`: what must be appended to `90-signoff.md` in this pass
- Primary artifact:
  - `55-reliability-and-resilience.md` with mandatory reliability sections:
    - `Dependency Criticality And Failure Contracts`
    - `Timeout, Deadline, And Retry Policy`
    - `Backpressure, Bulkheads, And Overload Response`
    - `Circuit-Breaking And Containment Policy`
    - `Degradation Modes And Fallback Policy`
    - `Startup, Readiness, Liveness, And Shutdown`
    - `Rollout, Rollback, And Reliability Gates`
- Required core artifacts per pass:
  - `80-open-questions.md` with reliability blockers/uncertainties
  - `90-signoff.md` with accepted reliability decisions and reopen criteria
- Conditional alignment artifacts (update when impacted):
  - `20-architecture.md`
  - `30-api-contract.md`
  - `40-data-consistency-cache.md`
  - `50-security-observability-devops.md`
  - `60-implementation-plan.md`
  - `70-test-plan.md`
- Conditional artifact status format for `20/30/40/50/60/70`:
  - include one explicit status: `Status: updated` or `Status: no changes required`
  - for `no changes required`, add one sentence justification with linked `REL-###`
  - for `updated`, list changed sections and linked `REL-###`
- Language: match user language when possible.
- Detail level: concrete and reviewable with explicit reliability policy semantics and verification criteria.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when five reliability axes are covered with source-backed inputs: dependency criticality, timeout/retry policy, overload containment, degradation lifecycle, rollout/rollback safety.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Artifacts`, current phase subsection, and target gate criteria first
  - load additional sections only when unresolved reliability decisions require them
- `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`

Load by trigger:
- Error wrapping, cancellation semantics, and context deadline behavior:
  - `docs/llm/go-instructions/10-go-errors-and-context.md`
- Goroutine lifecycle, bounded queues/channels, worker pools, and shutdown coordination:
  - `docs/llm/go-instructions/20-go-concurrency.md`
- API-visible reliability semantics (`429`/`503`, `Retry-After`, idempotency/retry, `202` fallback):
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Sync/async and distributed workflow reliability implications:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- Observability and budget-aware release implications:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
  - `docs/llm/delivery/10-ci-quality-gates.md`
- Data evolution/reconciliation implications:
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`

Conflict resolution:
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prefer trigger-loaded documents over always-loaded documents.
- If conflict persists, preserve latest accepted decision in `90-signoff.md` and add reopen blocker in `80-open-questions.md`.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- Resolve each `[assumption]` by source validation in the current pass or by promoting it to `80-open-questions.md` with owner and unblock condition.

## Definition Of Done
- Current phase and target gate are explicitly stated.
- `55-reliability-and-resilience.md` contains all mandatory reliability sections from this skill.
- Every major reliability decision includes `REL-###`, owner, selected option, and at least one rejected option with reason.
- Every critical dependency has explicit timeout/retry/bulkhead/fallback/circuit contract, observability trigger, and owner/rollback authority.
- Overload, degradation, and shutdown behavior is explicit and testable.
- Startup/readiness/liveness semantics are explicit and anti-flap by policy.
- Rollout and rollback reliability gates are explicit with trigger and authority semantics.
- Every `[assumption]` is either source-validated in the current pass or tracked in `80-open-questions.md` with owner and unblock condition.
- Reliability blockers are closed or tracked in `80-open-questions.md` with owner and unblock condition.
- Impacted `20/30/40/50/60/70` artifacts have explicit status with decision links and no contradictions.
- No active item from `Hard Skills -> Review Blockers For This Skill` remains unresolved.
- No hidden reliability decisions are deferred to coding.

## Anti-Patterns
Treat each item as a blocker unless an approved exception is explicitly recorded:
- implicit or infinite timeout defaults on outbound calls
- retries without explicit eligibility, bounded attempts, jitter, and retry budget
- unbounded queues/concurrency or shared unbounded dependency pools
- overload handling without deterministic rejection semantics (`429`/`503` and `Retry-After` policy)
- degradation/fallback behavior without explicit entry/exit/recovery criteria
- readiness/liveness configuration that causes probe flapping or restart storms
- rollout without explicit promotion gates, rollback triggers, and rollback authority
- unresolved critical reliability uncertainty deferred to coding instead of tracked blocker
