---
name: go-distributed-architect-spec
description: "Design distributed-consistency-first specifications for Go services in a spec-first workflow. Use when planning or revising cross-service workflows before coding and you need explicit saga/orchestration-choreography decisions, invariant ownership, outbox/inbox idempotency contracts, compensation/forward-recovery policies, and reconciliation strategy. Skip when the task is a local code fix, endpoint-level API contract design, physical SQL schema/migration scripting, CI/container setup, or low-level resilience/observability tuning."
---

# Go Distributed Architect Spec

## Purpose
Create a clear, reviewable distributed-consistency specification package before implementation. Success means cross-service workflow ownership, consistency semantics, and failure-handling contracts are explicit, defensible, and directly translatable into implementation and tests.
Use `Hard Skills` as the normative baseline for decision quality and distributed-risk controls; use workflow sections below for execution sequence and artifact synchronization.

## Scope And Boundaries
In scope:
- decompose cross-service business flows into explicit steps, owners, and local transaction boundaries
- define invariant register with explicit split between `local_hard_invariant` and `cross_service_process_invariant`
- choose workflow style (orchestration vs choreography) and define valid boundaries and handoff rules
- define per-step contract: trigger, local transaction scope, idempotency key, timeout/retry class, success transition, compensation or forward recovery
- define pivot transaction and pre-pivot/post-pivot policies
- define outbox/inbox/dedup requirements and durable commit-before-ack ordering
- define replay/out-of-order/duplicate handling under at-least-once delivery
- define reconciliation ownership/cadence and read-model freshness constraints
- define race controls (single active workflow per business key, CAS/version transitions, serialization boundaries)

Out of scope:
- baseline service/module topology decisions outside distributed-consistency domain
- endpoint-level HTTP/JSON payload, status, and error contract design
- physical SQL schema design, DDL details, migration scripts, and backfill mechanics
- concrete cache runtime tuning (exact keys, TTL/jitter, invalidation implementation)
- detailed resilience tuning thresholds (bulkhead/circuit/load-shedding specifics)
- detailed observability schemas, SLI/SLO targets, and alert-threshold tuning
- full security-control catalogs and authn/authz hardening specifics
- CI/CD gate design and container/runtime hardening setup
- low-level test implementation details for specific test suites

## Hard Skills
### Distributed Consistency Core Instructions

#### Mission
- Produce distributed-consistency decisions that remain correct under partial failures, at-least-once delivery, and mixed-version rollout.
- Convert ambiguous cross-service behavior into explicit, reviewable workflow contracts before coding starts.
- Ensure every selected distributed option is implementable, testable, observable, and rollback-aware.

#### Default Posture
- Keep hard commit-time invariants inside one local transaction boundary whenever possible.
- Treat cross-service consistency as explicit eventual-consistency process design, never as implicit global ACID.
- Default to orchestration for business-critical or multi-step outcomes that need centralized operational policy.
- Treat at-least-once delivery as the default runtime assumption; exactly-once end-to-end is never assumed.
- Require outbox/inbox-idempotency plus reconciliation for side-effecting distributed flows by default.
- Use compatibility-first evolution (`expand -> migrate/backfill -> contract`) for distributed data and event contract changes.

#### Invariant Ownership And Consistency Contract Competency
- Require explicit source-of-truth owner per critical entity and per invariant.
- Classify every critical invariant as:
  - `local_hard_invariant` (must hold at local commit),
  - `cross_service_process_invariant` (allowed to converge over time).
- Require explicit per-flow consistency contract:
  - `consistency_type`,
  - `max_staleness`,
  - failure outcome policy (`retry`, `compensate`, `manual`),
  - reconciliation owner/cadence.
- Keep behavior local when compensation is unacceptable, user flow cannot tolerate intermediate states, or atomic invariant checks are required for most requests.
- Reject ownerless invariants and hidden enforcement assumptions ("some consumer will fix it").

#### Workflow State Model And Step Contract Competency
- Model each multi-step distributed flow as an explicit durable state machine.
- Require monotonic and version-checked transitions; reject implicit transition-by-log-only behavior.
- Require one active workflow instance per business key (`tenant + aggregate_id + flow_type`) via durable uniqueness or equivalent.
- Require deterministic timeout/stuck-flow behavior per step:
  - every step has explicit timeout,
  - expiry transitions are deterministic (`failed` or `compensating`),
  - no implicit unknown terminal state.
- Require each step contract to define:
  - trigger (`event` or `command`),
  - local transaction scope,
  - idempotency key source and dedup boundary,
  - timeout and retry class,
  - success transition,
  - compensation or explicit `no_compensation` with reason.

#### Orchestration vs Choreography Competency
- Default to orchestration when operational control of timeout/retry/compensation ordering is required.
- Require orchestration when any point is true:
  - more than two services participate in one business outcome,
  - auditable centralized workflow state is required,
  - one owner for retry/DLQ/reconciliation/SLO is required.
- Allow choreography only when all points are true:
  - reactions are independent and loosely coupled,
  - no central process-state outcome is required,
  - event cycles and duplicate reactions are explicitly prevented,
  - operational ownership is clear per consumer.
- Do not mix orchestration and choreography inside one flow without explicit boundary and owner handoff.
- Reject event-bus-as-hidden-sync-RPC designs.

#### Compensation, Pivot, And Forward-Recovery Competency
- Require explicit pivot transaction identification for every nontrivial saga.
- Enforce policy by phase:
  - pre-pivot steps must be compensable,
  - post-pivot steps must be idempotent, retryable, and forward-recoverable.
- Require compensation actions to be semantic inverse operations, idempotent, and precondition-guarded.
- Default compensation execution order to reverse order of completed compensable steps.
- If compensation is impossible:
  - mark step explicitly as non-compensable,
  - place pivot before it,
  - define forward-recovery path (bounded retries, manual queue, or alternate completion),
  - require operator runbook linkage.
- Reject flows with missing compensation/forward-recovery semantics.

#### Delivery Semantics, Idempotency, Dedup, And Commit-Ordering Competency
- Treat at-least-once delivery as default and require design to tolerate duplicates/out-of-order delivery.
- Require outbox-equivalent atomic linkage for state-change plus message emission; reject cross-system dual writes.
- Require consumer dedup/inbox for side-effecting handlers with durable uniqueness constraints.
- Default dedup key policy:
  - CloudEvents: `source + id`,
  - non-CloudEvents: `producer_service + message_id`.
- Default dedup retention: minimum 7 days or replay/redrive window, whichever is greater.
- Require external command idempotency policy for retry-unsafe operations:
  - key scope includes tenant/account and operation identity,
  - default TTL 24h,
  - same key + same payload => equivalent outcome,
  - same key + different payload => conflict.
- Enforce durable-state-first commit ordering:
  - persist side effects and dedup marker before ack/offset commit,
  - never ack before durable state transition.
- Require deterministic error classification for async handlers:
  - `retryable_transient`,
  - `non_retryable`,
  - `poison_payload`.
- Require bounded retries with jitter and explicit DLQ ownership/redrive policy; reject infinite retries.

#### Async Topology, Ordering, And Replay-Safety Competency
- Justify async usage first; do not choose broker/technology first.
- Classify message intent explicitly (`event` vs `command`) and bind ownership to that intent.
- Choose topology intentionally:
  - pub/sub for independent domain reactions,
  - queue for owned work distribution and bounded worker pools.
- Require explicit ordering boundary documentation; never assume global ordering across partitions/topics/queues.
- Require replay-safe consumer behavior:
  - deterministic processing for same input/version,
  - duplicate side effects prevented,
  - historical reprocessing can rebuild projections consistently.
- Require controlled replay/redrive with checkpointing and throughput guardrails.

#### Race-Control And Hidden-Invariant Prevention Competency
- Require serialization strategy for competing workflows mutating same aggregate.
- Require CAS/version checks for workflow and aggregate transitions.
- Reject hard write decisions based on stale projections/read models.
- Reject distributed locks as primary correctness mechanism.
- If a technical lock is unavoidable, require fencing-token semantics and explicit failure-mode analysis.
- Prevent parallel compensation for one workflow instance unless explicitly designed and proven safe.

#### Read-Model Freshness And Reconciliation Competency
- Treat read models as query optimization surfaces, not authoritative write-validation sources.
- Require freshness signals on projections (`updated_at`, lag, or equivalent).
- If freshness exceeds budget, write path must query owner service or fail fast by contract.
- Default staleness budgets:
  - critical financial/inventory status: target <= 10s, hard cap <= 60s,
  - standard user-facing status: target <= 60s, hard cap <= 15m.
- Require reconciliation for critical eventual-consistency flows:
  - critical: at least every 5 minutes,
  - non-critical: at least every 1 hour,
  - full-scope: at least daily.
- Require reconciliation jobs to be idempotent, resumable, watermark-based, and repair-oriented.
- Prefer repair commands/events over direct cross-service table writes.

#### Reliability And Degradation Interface Competency
- Require distributed step contracts to be compatible with dependency failure contracts:
  - timeout budget,
  - retry budget/class,
  - fallback mode,
  - containment model (bulkhead/circuit/degrade).
- Require explicit deadline propagation model for sync dependencies in distributed flows.
- Apply fail-fast rule when remaining inbound budget is insufficient for safe downstream call.
- Require bounded queue/concurrency assumptions and overload behavior for async stages.
- Require explicit degraded-mode behavior for critical flows where fallback is allowed.
- Reject hidden resilience semantics that change correctness without explicit contract update.

#### API Surface Interface Competency
- When distributed behavior is API-visible, require explicit contract alignment:
  - consistency disclosure (`strong` or `eventual`),
  - staleness expectations for eventual reads,
  - retry classification and idempotency behavior.
- Require `202 Accepted` plus operation-resource pattern for long-running distributed work.
- Require `Idempotency-Key` (or transport-equivalent field) for retry-unsafe operations that clients may retry.
- Require deterministic conflict semantics for idempotency-key payload mismatch.
- Reject fake immediate-success responses for operations that only queued work.

#### Schema Evolution And Migration Interface Competency
- Require compatibility windows across code/schema/event versions for distributed flows.
- Require phased rollout (`expand -> backfill/migrate -> contract`) when distributed contracts are affected.
- Require producer schema versioning and tolerant-reader behavior for consumer migration windows.
- Do not contract schema/event semantics while downstream consumers still depend on old shape.
- Require outbox/CDC reliability controls when migration changes side-effect publication path.

#### Observability And Diagnostics Interface Competency
- Require end-to-end correlation across producer, consumer, retries, DLQ, and reconciliation:
  - `trace_id`, `span_id`,
  - stable `correlation_id`,
  - `message_id`,
  - `attempt`,
  - flow/stage identity.
- Require mandatory async telemetry coverage:
  - processing latency histogram,
  - handler outcome counters,
  - retry and DLQ counters with bounded reason taxonomy,
  - backlog/lag/oldest-message-age signals.
- Require explicit observability for compensation and reconciliation outcomes.
- Require cardinality discipline:
  - no request/message/user IDs in metric labels,
  - high-cardinality IDs stay in logs/traces only.
- Require telemetry cost controls for distributed additions:
  - bounded labels,
  - stable histogram boundaries,
  - explicit sampling/retention assumptions.
- Require operationally safe diagnostics posture for distributed incidents:
  - separated liveness/readiness/startup semantics,
  - deterministic shutdown/drain sequence,
  - debug endpoints isolated from public ingress.

#### Evidence Threshold And Decision Quality Bar
- Every major distributed decision must include at least two options and one explicit rejection reason.
- Every selected option must include measurable acceptance boundaries, not only narrative rationale.
- Every selected option must include failure-path behavior (retry/compensate/forward-recover/manual).
- Every selected option must include cross-domain impact summary (architecture, API, data, reliability, observability, security).
- Every selected option must include reopen conditions tied to observable triggers.
- Minimum evidence by decision axis:
  - invariant ownership: source-of-truth map + invariant classification + enforcement point;
  - workflow contract: durable state model + step contracts + pivot policy;
  - delivery safety: outbox/inbox policy + commit-ordering + idempotency semantics;
  - freshness and repair: staleness budget + reconciliation cadence + repair mechanism;
  - trigger-domain impacts: explicit alignment notes for API/reliability/migration/observability when triggered.

#### Assumption And Uncertainty Discipline
- Mark unknown critical facts as `[assumption]` immediately.
- Keep assumptions bounded, testable, and decision-linked.
- Resolve assumptions in the same pass when source-backed validation is possible.
- Promote unresolved critical assumptions to `80-open-questions.md` with owner and unblock condition.

#### Review Blockers For This Skill
- Distributed recommendation without explicit trade-off analysis and rejected option.
- Missing invariant register, missing owner, or missing enforcement point for critical rules.
- Flow without explicit durable state model and step contracts.
- Pivot not identified for nontrivial saga, or pre/post-pivot policy undefined.
- Cross-system dual write used instead of outbox-equivalent atomic linkage.
- Missing idempotency/dedup policy for retry-unsafe or side-effecting handlers.
- Ack/offset commit before durable side effects.
- Ordering/replay assumptions left implicit or dependent on global-order fantasy.
- Reconciliation/freshness contract missing for eventual-consistency flows.
- Hard write decisions based on stale read-model snapshots.
- API-visible async/idempotency/consistency semantics changed without contract alignment.
- Async observability insufficient for retries/DLQ/lag/reconciliation diagnosis.
- Critical uncertainty deferred to coding instead of explicit blocker tracking.

## Working Rules
1. Determine current `docs/spec-first-workflow.md` phase and target gate before drafting decisions.
2. Set phase-specific output targets:
   - Phase 0: seed distributed assumptions/blockers in `80` and distributed constraints for `40`
   - Phase 1: define distributed constraints that must shape `20` and `60`
   - Phase 2 and later: maintain `40/80/90` and update impacted `20/30/55/70`
3. Apply `Hard Skills` defaults by default. Any deviation must be explicit, justified, and linked to decision ID (`DIST-###`) and reopen criteria.
4. Load context using this skill's dynamic loading rules and stop when four core distributed axes are source-backed (workflow ownership, invariant classification, delivery/idempotency semantics, reconciliation/freshness policy) and every triggered interface axis has at least one source-backed rule (API-visible semantics, reliability/degradation, migration/evolution, async observability).
5. Normalize each affected flow into explicit entities, owners, invariants, state transitions, failure classes, and operational ownership.
6. For each nontrivial distributed decision, compare at least two options and select one explicitly.
7. Assign decision ID (`DIST-###`) and owner for each major distributed decision.
8. Record trade-offs and cross-domain impact (architecture, API, data, reliability, observability, security).
9. Run a hard-skills gate for each `DIST-###` decision and verify no `Review Blockers For This Skill` remain unresolved.
10. Mark missing critical facts as `[assumption]`; keep assumptions bounded and either validate them in the current pass or move them to `80-open-questions.md` with owner and unblock condition.
11. Record any blocking uncertainty in `80-open-questions.md` with concrete next step.
12. Keep `40-data-consistency-cache.md` as primary artifact and synchronize distributed implications in impacted artifacts.
13. Verify internal consistency: no hidden global-atomicity assumptions and no critical distributed decisions deferred to coding.
14. Run final blocker check against `Hard Skills -> Review Blockers For This Skill` before closing a pass.

## Decision Heuristics
- keep behavior inside one local transaction boundary when an invariant must hold at commit time and compensation is unacceptable
- choose orchestration when one business outcome spans multiple services and needs explicit operational control of retries/timeouts/compensation
- choose choreography when reactions are independent and do not require centralized process state
- place non-compensable steps after the pivot and enforce idempotent forward recovery for post-pivot completion
- require one active workflow per business key and version-checked state transitions before approving the flow design

## Distributed Decision Protocol
For every major distributed decision, document:
1. decision ID (`DIST-###`) and current phase
2. owner role
3. context and problem
4. options (minimum two)
5. selected option with rationale
6. at least one rejected option with explicit rejection reason
7. trade-offs (gains and losses)
8. invariant and consistency impact
9. idempotency/dedup/outbox-inbox implications
10. compensation/forward-recovery and reconciliation implications
11. impact on architecture, API, data, reliability, observability, and security
12. reopen conditions, affected artifacts, and linked open-question IDs (if any)
13. evidence package by axis:
    - invariant ownership evidence
    - workflow/step-contract evidence
    - delivery/idempotency evidence
    - reconciliation/freshness evidence
    - trigger-domain interface evidence (if triggered)
14. hard-skills blocker-check status (`none` or linked blocker item)

## Output Expectations
- Response format:
  - `Decision Register`: accepted `DIST-###` decisions with rationale, trade-offs, evidence package summary, and blocker-check status
  - `Artifact Update Matrix`: `40/20/30/55/70` with `Status: updated|no changes required` and linked `DIST-###`
  - `Assumptions`: active `[assumption]` items and resolution path
  - `Open Blockers`: unresolved items for `80-open-questions.md` with owner and unblock condition
  - `Sign-Off Delta`: what must be appended to `90-signoff.md` in this pass
- Primary artifact:
  - `40-data-consistency-cache.md` with mandatory sections:
    - `Distributed Flow Inventory`
    - `Invariant Register And Ownership`
    - `Workflow State Models And Step Contracts`
    - `Consistency And Staleness Contracts`
    - `Outbox/Inbox/Dedup And Idempotency Policy`
    - `Compensation, Pivot, And Forward Recovery`
    - `Reconciliation Ownership And Cadence`
  - `40-data-consistency-cache.md` must include these checkable tables:
    - flow table: `flow_id`, `business_outcome`, `owner_service`, `interaction_style`, `consistency_type`, `max_staleness`, `reconciliation_owner`
    - step table: `flow_id`, `step_id`, `step_owner`, `trigger`, `local_tx_boundary`, `idempotency_key_source`, `timeout_budget`, `retry_class`, `success_transition`, `recovery_mode`
    - reconciliation table: `flow_id`, `job_owner`, `cadence`, `watermark_strategy`, `repair_action`, `escalation_path`
  - For each major `DIST-###` in `40-data-consistency-cache.md`, include a compact decision card with:
    - selected option and rejected option
    - failure-path policy (retry/compensate/forward-recover/manual)
    - evidence package summary
    - hard-skills blocker-check status
- Required core artifacts per pass:
  - `80-open-questions.md` with distributed blockers/uncertainties
  - `90-signoff.md` with accepted distributed decisions and reopen criteria
- Conditional alignment artifacts (update when impacted):
  - `20-architecture.md`
  - `30-api-contract.md`
  - `55-reliability-and-resilience.md`
  - `70-test-plan.md`
- Conditional artifact status format for `20/30/55/70`:
  - include one explicit status: `Status: updated` or `Status: no changes required`
  - for `no changes required`, add one sentence justification with linked `DIST-###`
  - for `updated`, list changed sections and linked `DIST-###`
- Language: match user language when possible.
- Detail level: concrete and reviewable with explicit owners, transitions, and failure-path semantics.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when four core distributed axes are covered with source-backed inputs (workflow ownership, invariant classification, delivery/idempotency semantics, reconciliation/freshness policy) and every triggered interface axis is covered (API-visible semantics, reliability/degradation, migration/evolution, async observability).

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Artifacts`, current phase subsection, and target gate criteria first
  - load additional sections only when unresolved decisions require them
- `docs/llm/architecture/30-event-driven-and-async-workflows.md`
- `docs/llm/architecture/40-distributed-consistency-and-sagas.md`

Load by trigger:
- Failure/degradation/system-evolution implications:
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Sync request-reply constraints and deadline coupling:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
- Schema evolution, migration sequencing, replay/reliability implications:
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
- API-visible idempotency/retry/async semantics:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Async observability coverage (lag/retry/dlq/reconciliation):
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`

Conflict resolution:
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prefer trigger-loaded documents over always-loaded documents.
- If conflict persists, preserve latest accepted decision in `90-signoff.md` and add reopen blocker in `80-open-questions.md`.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- Resolve each `[assumption]` by source validation in the current pass or by promoting it to `80-open-questions.md` with owner and unblock condition.

## Definition Of Done
- Current phase and target gate are explicitly stated.
- `40-data-consistency-cache.md` includes all mandatory sections and required tables from this skill.
- Every major `DIST-###` includes evidence package summary and blocker-check status.
- Every affected cross-service flow has explicit owner, state model, and transitions.
- Every critical invariant is classified with owner and enforcement point.
- Every workflow step defines trigger, transaction scope, idempotency key policy, timeout/retry class, and compensation or forward-recovery rule.
- Outbox/inbox/dedup and durable commit-before-ack ordering are defined for side-effecting flows.
- Replay/out-of-order/duplicate handling and reconciliation strategy are explicit and testable.
- Staleness/freshness contracts are defined where eventual consistency is used.
- Distributed blockers are closed or tracked in `80-open-questions.md` with owner and unblock condition.
- No active item from `Hard Skills -> Review Blockers For This Skill` remains unresolved.
- No hidden assumptions of global ACID semantics remain.

## Anti-Patterns
- assuming implicit global ACID behavior across services
- accepting dual-write (`db + publish`) without outbox-equivalent atomic linkage
- leaving workflow behavior without explicit state model and step contracts
- allowing retries without idempotency and dedup policy
- enforcing hard write invariants from stale projections/read models
- leaving compensation or forward recovery undefined for failure paths
- deferring critical distributed-consistency decisions to coding phase
