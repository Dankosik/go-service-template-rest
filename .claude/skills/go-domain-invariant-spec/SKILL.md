---
name: go-domain-invariant-spec
description: "Design domain-invariant-first specifications for Go services in a spec-first workflow. Use when planning or revising behavior before coding and you need explicit business invariants, state-transition rules, acceptance criteria, corner-case handling, and traceability into API/data/reliability/testing artifacts. Skip when the task is a local code fix, low-level implementation, endpoint schema-only design, SQL/migration mechanics, or CI/container setup."
---

# Go Domain Invariant Spec

## Purpose
Create a clear, reviewable domain-behavior specification package before implementation. Success means business invariants and acceptance behavior are explicit, testable, and directly translatable into implementation and test obligations.
Use `Hard Skills` as the normative domain baseline for decision quality and risk controls; use workflow sections below for execution sequence and artifact synchronization.

## Scope And Boundaries
In scope:
- define business invariants in verifiable form (what must always hold and what must never happen)
- define state-transition constraints (`allowed`, `forbidden`, preconditions, postconditions)
- define acceptance behavior for happy-path, fail-path, and domain corner cases
- define invariant-preservation expectations for sync and async paths
- define expected behavior when an invariant is violated (reject/fail/compensate semantics)
- define traceability from invariants to impacted `30/40/55/60/70/80/90` artifacts
- produce invariant deliverables that remove hidden "decide later" domain gaps

Out of scope:
- service decomposition and ownership topology as a primary domain
- full endpoint/resource contract design as a primary domain
- physical SQL schema design, DDL details, and migration mechanics as a primary domain
- cache runtime tuning details (exact keys, TTL/jitter, invalidation mechanics)
- reliability control-plane design as a primary domain
- SLI/SLO and alert policy design as a primary domain
- security control-catalog design as a primary domain
- writing production code or test code
- code-review responsibilities of `go-domain-invariant-review`

## Hard Skills
### Domain Invariant Core Instructions

#### Mission
- Produce domain decisions that remain correct under retries, duplicates, out-of-order delivery, partial failures, and mixed-version rollouts.
- Convert ambiguous product behavior into explicit invariant contracts before coding starts.
- Ensure every selected invariant decision is testable, observable, and traceable across `15/30/40/55/60/70/80/90`.

#### Default Posture
- Start from business invariants and lifecycle semantics first; map to API/data/reliability artifacts second.
- Keep one explicit invariant register with owner and enforcement point for every critical business rule.
- Classify invariants by enforcement shape:
  - `local_hard_invariant` (must hold in one local transaction boundary),
  - `cross_service_process_invariant` (preserved by saga/workflow + idempotency + reconciliation).
- Use explicit state-machine modeling for non-trivial lifecycles; avoid implicit event-sequence reasoning.
- Treat invariant violations as explicit outcomes (reject, compensate, forward-recover, manual intervention), never as "best effort."
- Default to compatibility-first invariant evolution; avoid breaking behavior shifts during mixed-version rollouts.

#### Invariant Modeling And Ownership Competency
- Define each critical invariant as one verifiable rule with:
  - `name`,
  - `owner_service`,
  - `type`,
  - `enforcement_point`,
  - observable pass/fail condition.
- Require explicit source-of-truth ownership per invariant-related entity.
- Require identity/tenant/authorization constraints to be represented as domain invariants when they affect correctness.
- Require a clear enforcement boundary for each invariant:
  - API boundary validation/precondition,
  - domain/use-case transition guard,
  - persistence constraint (`UNIQUE`, `CHECK`, FK-in-boundary),
  - workflow/saga step contract,
  - reconciliation control.
- Reject ownerless invariants and "eventual fix by some consumer" assumptions.
- Reject invariant definitions that are descriptive only and not falsifiable.

#### State Transition And Workflow Competency
- Model domain lifecycles as explicit states and transitions, not prose narratives.
- For each transition, define:
  - trigger (`command`, `event`, timeout, operator action),
  - preconditions,
  - postconditions,
  - allowed and forbidden next states.
- Require monotonic, version-checked transitions for workflow/saga state.
- Enforce one active workflow instance per business key when concurrent flows can violate invariants.
- Define timeout and stuck-flow behavior explicitly; no implicit "unknown" state.
- If compensation is impossible, mark pivot placement explicitly and require forward-recovery semantics for post-pivot steps.
- Never infer domain state transitions from logs or side effects alone.

#### Acceptance Criteria And Corner-Case Competency
- Define acceptance criteria as observable behavior contracts, not internal implementation hints.
- Cover all required behavior classes per critical invariant:
  - happy path,
  - forbidden path,
  - fail path,
  - corner/edge conditions.
- Include duplicate/replay/out-of-order behavior when async processing is involved.
- Include idempotency conflict behavior where retries can happen:
  - same key + same payload => equivalent outcome,
  - same key + different payload => explicit conflict.
- For eventually consistent read paths, require explicit freshness/staleness semantics and boundaries.
- For long-running side effects, require explicit async acknowledgement semantics (`202` + operation state), not fake completion.

#### Invariant Violation Semantics Competency
- Map each invariant violation to one deterministic behavior class:
  - immediate reject (`400`/`422`/`409`/`412`),
  - authorization/tenant deny (`403`),
  - deferred/async processing with explicit status resource,
  - compensation/forward-recovery flow,
  - manual intervention queue.
- Keep one stable external error model per API surface (`application/problem+json` for HTTP).
- Never return success status for failed invariant checks.
- Never mask cancellation/timeouts as business-domain success or unrelated business errors.
- Keep violation handling fail-closed for authz/tenant/object-ownership invariants.

#### API And Sync Contract Impact Competency
- When invariants affect external behavior, require contract updates for:
  - method semantics and status codes,
  - retry class and idempotency requirements,
  - optimistic concurrency/precondition policy (`If-Match`, `412`, `428` where applicable),
  - consistency disclosure (`strong` vs `eventual`).
- Require deterministic list/query behavior where invariant decisions depend on ordering.
- Require boundary validation/normalization/input-limit policy to execute before domain use-case logic.
- Ensure principal and tenant context requirements are explicit in contract and runtime expectations.
- Keep endpoint behavior stable within major API version unless explicitly approved.

#### Async, Distributed Consistency, And Saga Competency
- For state changes that emit messages, require outbox-equivalent atomic linkage.
- Require consumer idempotency and durable dedup/inbox for side-effecting handlers.
- Require deterministic retry classification (`retryable`, `non_retryable`, `poison`) with bounded jittered retries and DLQ ownership.
- Require explicit ordering assumptions and replay-safety rules; never assume global ordering.
- Require explicit saga/workflow step contracts (trigger, local transaction scope, timeout, retry class, compensation/forward recovery).
- Require reconciliation ownership, cadence, and repair behavior for critical eventual-consistency invariants.
- Reject dual writes, hidden invariant ownership, and "exactly once by default" assumptions.

#### Data And Persistence Alignment Competency
- Require DB constraints for DB-enforceable invariants (PK, `UNIQUE`, `NOT NULL`, row-level `CHECK`) instead of app-only checks.
- Keep transaction boundaries local to one service-owned datastore.
- Default to optimistic conflict handling for concurrently mutable entities.
- Require explicit migration compatibility strategy for invariant-preserving evolution:
  - `Expand -> Migrate/Backfill -> Contract`.
- Require objective invariant verification gates before destructive contract steps.
- Require rollback class and limitations to be explicit when invariant-related schema/data semantics change.
- Keep read-model freshness contracts explicit when write invariants depend on projected data.

#### Cache And Freshness Contract Competency
- Treat cache as acceleration layer by default, not invariant authority.
- Define staleness contract before using cached data in invariant-sensitive decisions.
- Require tenant/scope/version-safe key design whenever cache affects domain behavior visibility.
- Require fail-open fallback for read acceleration caches by default, unless fail-closed is explicitly justified.
- Never rely on exact TTL expiry moment for correctness-critical invariants.
- Require stampede protection and bounded cache timeouts when hot-path acceptance semantics depend on cache fallback.

#### Security, Identity, And Tenant-Isolation Invariant Competency
- Require explicit `AuthContext` requirements when identity influences domain invariants.
- Require `tenant_id` from verified identity or trusted signed internal credential only.
- Enforce default deny for cross-tenant access unless explicitly approved in contract.
- Require object-level authorization checks before side effects on resource-by-ID paths.
- Never trust arbitrary caller-supplied identity headers as source of truth.
- Never propagate raw bearer tokens through async payloads.
- Ensure invariants that depend on identity propagation include sync and async propagation rules.

#### Reliability, Degradation, And Rollout Impact Competency
- Require dependency failure mode classification where invariant outcomes depend on downstream systems:
  - `critical_fail_closed`,
  - `critical_fail_degraded`,
  - `optional_fail_open`.
- Require explicit deadline/timeout budget assumptions for invariant-sensitive operations.
- Require bounded retries with jitter and retry budget; never use unbounded retries for invariant-preserving flows.
- Require overload/backpressure behavior to preserve invariant safety under stress.
- Require explicit degradation modes and their invariant impact (what remains guaranteed, what is degraded).
- Require rollout/rollback safety notes when invariant behavior changes can affect production correctness.
- Require observability of invariant-related degradation and recovery transitions.

#### Testing And Traceability Competency
- Every critical invariant must map to explicit tests in `70-test-plan.md`:
  - positive/happy-path,
  - negative/forbidden-path,
  - corner-case coverage.
- Require traceability links from each `DOM-###` decision to concrete test obligations.
- Require contract tests where invariant behavior crosses API or async message boundaries.
- Require replay/idempotency tests for async invariants and duplicate handling.
- Require deterministic tests for timeout/cancellation/fail-path semantics affecting invariant outcomes.
- Keep quality baseline explicit for behavior changes:
  - `go test ./...`,
  - `go vet ./...`,
  - `go test -race ./...` when concurrency paths are affected.

#### Evidence Threshold And Decision Quality Bar
- Every major invariant decision must include at least two options and one explicit rejection reason.
- Every selected option must include measurable acceptance boundaries, not only narrative intent.
- Every selected option must include violation semantics and operational handling path.
- Every selected option must include cross-domain impact summary for API/data/distributed/reliability/security/testing.
- Every selected option must include rollout compatibility and rollback limitations when behavior changes.
- Every selected option must include reopen conditions linked to observable triggers.
- Minimum evidence by axis:
  - invariant set:
    - owner map + enforcement points + falsifiable rule statements;
  - transitions:
    - explicit state model + pre/postconditions + forbidden transitions;
  - violation semantics:
    - deterministic mapping of failure class to external/internal behavior;
  - test traceability:
    - `DOM-###` to `70-test-plan.md` mapping for critical and corner scenarios.

#### Assumption And Uncertainty Discipline
- Mark unknown critical facts as `[assumption]` immediately.
- Keep assumptions bounded, testable, and decision-linked.
- Resolve assumptions in the same pass when source-backed validation is possible.
- Promote unresolved critical assumptions to `80-open-questions.md` with owner and unblock condition.

#### Review Blockers For This Skill
- Critical invariant without explicit owner, enforcement point, or falsifiable pass/fail condition.
- State-transition design without explicit forbidden transitions and pre/postconditions.
- Invariant violation behavior unspecified or inconsistent across API/async/reliability artifacts.
- Retry/idempotency semantics missing where duplicated execution is possible.
- Cross-service invariant declared without outbox/idempotency/reconciliation strategy.
- Invariant-sensitive behavior delegated to stale read models without freshness contract.
- Identity/tenant/object-ownership invariant implied but not explicitly enforced.
- Migration or rollout plan can break invariant semantics during mixed-version deployments.
- Test traceability for critical invariant paths missing in `70-test-plan.md`.
- Critical unknowns left implicit instead of tracked in `80-open-questions.md`.

## Working Rules
1. Determine current `docs/spec-first-workflow.md` phase and target gate before drafting decisions.
2. Set phase-specific output targets:
   - Phase 0: establish initial invariant register in `15` and record invariant unknowns in `80`.
   - Phase 1: refine invariants and acceptance criteria to verifiable form and align with baseline architecture constraints.
   - Phase 2 and later: run full invariant pass across spec package, maintain `15/80/90`, and update impacted artifacts.
3. Apply `Hard Skills` defaults by default. Any deviation must be explicit, justified, and linked to decision ID (`DOM-###`) and reopen criteria.
4. Load context using this skill's dynamic loading rules and stop when four invariant axes are source-backed: invariant set, transition rules, violation semantics, and test traceability obligations.
5. Normalize domain behavior scope: entities, lifecycle states, command/event triggers, identity/tenant constraints, and failure semantics.
6. For each nontrivial domain decision, compare at least two options and select one explicitly.
7. Assign decision ID (`DOM-###`) and owner for each major invariant decision.
8. Record trade-offs and cross-domain impact (architecture, API, data, distributed consistency, reliability, security, testing).
9. Mark missing critical facts as `[assumption]`; keep assumptions bounded and either validate in current pass or move to `80-open-questions.md` with owner and unblock condition.
10. If uncertainty blocks invariant closure, record it in `80-open-questions.md` with concrete next step.
11. Keep `15-domain-invariants-and-acceptance.md` as primary artifact and synchronize impacted `30/40/55/60/70/90` sections.
12. Verify internal consistency: no contradictions between invariant definitions, acceptance criteria, and downstream artifacts.
13. Keep focus on domain expertise by making explicit behavior decisions, not generic process commentary.
14. Run final blocker check against `Hard Skills -> Review Blockers For This Skill` before closing a pass.

## Decision Classification
Treat a decision as major invariant when it changes at least one of:
- invariant ownership or source-of-truth boundary
- invariant classification (`local_hard_invariant` vs `cross_service_process_invariant`)
- state-machine transitions, transition guards, or terminal outcomes
- invariant-violation behavior (reject/deny/compensate/forward-recover/async/manual)
- idempotency, replay, ordering, or duplicate-handling semantics
- consistency/freshness promises that affect invariant safety
- identity/tenant/object-ownership rules that affect correctness

## Invariant Decision Protocol
For every major invariant decision, document:
1. decision ID (`DOM-###`) and current phase
2. owner role and source-of-truth owner service
3. context and business problem
4. invariant statement in verifiable form (pass/fail observable rule)
5. invariant class (`local_hard_invariant` or `cross_service_process_invariant`)
6. scope (entity/use-case/process/cross-service)
7. enforcement points (API/domain/persistence/workflow/reconciliation)
8. options (minimum two for nontrivial cases)
9. selected option with rationale
10. at least one rejected option with explicit rejection reason
11. transition constraints (allowed/forbidden transitions + preconditions/postconditions)
12. violation behavior (error semantics, deny/reject/compensate/forward-recover/manual path)
13. duplicate/replay/ordering/idempotency behavior where applicable
14. cross-domain impact and trade-offs (API/data/distributed/reliability/security/testing)
15. rollout compatibility and rollback limitations when behavior changes
16. test obligations, reopen conditions, affected artifacts, and linked open-question IDs (if any)

## Output Expectations
- Primary artifact:
  - `15-domain-invariants-and-acceptance.md` with mandatory sections:
    - `Domain Terms And Scope`
    - `Invariant Register`
    - `State Transition Rules`
    - `Acceptance Criteria`
    - `Corner Cases And Edge Conditions`
    - `Invariant Violation Semantics`
    - `Traceability To Related Artifacts`
  - `Invariant Register` minimum fields:
    - `DOM-###`
    - invariant class (`local_hard_invariant` or `cross_service_process_invariant`)
    - owner role and owner service
    - enforcement point(s)
    - source-of-truth entity/store
    - verifiable pass/fail rule
  - `State Transition Rules` minimum structure:
    - explicit states and allowed transitions
    - forbidden transitions
    - transition preconditions and postconditions
    - timeout/stuck-state handling policy
  - `Invariant Violation Semantics` minimum structure:
    - deterministic mapping `violation -> response/status/outcome`
    - sync vs async behavior differences
    - compensation/forward-recovery/manual escalation policy
  - `Traceability To Related Artifacts` minimum mapping:
    - `DOM-### -> 30/40/55/60/70` impacted sections
    - `DOM-### -> 70` test obligations
- Required core artifacts per pass:
  - `80-open-questions.md` with invariant blockers/unknowns
  - `90-signoff.md` with accepted invariant decisions, reopen criteria, and explicit note for every approved deviation from `Hard Skills`
- Conditional alignment artifacts (update when impacted):
  - `30-api-contract.md`
  - `40-data-consistency-cache.md`
  - `55-reliability-and-resilience.md`
  - `60-implementation-plan.md`
  - `70-test-plan.md`
- Conditional artifact status format for `30/40/55/60/70`:
  - include one explicit status: `Status: updated` or `Status: no changes required`
  - for `no changes required`, add one sentence justification with linked `DOM-###`
  - for `updated`, list changed sections and linked `DOM-###`
- Language: match user language when possible.
- Detail level: concrete and reviewable with explicit state behavior and acceptance boundaries.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when four invariant axes are covered with source-backed inputs: invariant set, transition rules, violation semantics, test traceability obligations.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Artifacts`, current phase subsection, and target gate criteria first
  - load additional sections only if unresolved invariant decisions require them

Load by trigger:
- API-visible behavior and acceptance semantics:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Sync/async orchestration and cross-service consistency implications:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Data model, migration, and cache consistency implications:
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Error semantics and cancellation/timeout behavior affecting invariant outcomes:
  - `docs/llm/go-instructions/10-go-errors-and-context.md`
- Test traceability and coverage obligations:
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
- Identity/tenant/object-ownership invariants:
  - `docs/llm/security/20-authn-authz-and-service-identity.md`

Conflict resolution:
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prefer trigger-loaded documents over always-loaded documents.
- If conflict persists, preserve latest accepted decision in `90-signoff.md` and add reopen blocker in `80-open-questions.md`.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- Resolve each `[assumption]` by source validation in current pass or by promoting it to `80-open-questions.md` with owner and unblock condition.

## Definition Of Done
- Current phase and target gate are explicitly stated.
- `15-domain-invariants-and-acceptance.md` contains all mandatory sections from this skill.
- All major invariant decisions include `DOM-###`, owner, selected option, and at least one rejected option with reason.
- All critical invariants have explicit class, owner service, source-of-truth, and enforcement point.
- Critical state transitions include explicit allowed/forbidden rules with preconditions/postconditions.
- Acceptance criteria are behavior-level and testable without reinterpretation.
- Invariant-violation behavior is explicit and consistent with API/reliability expectations.
- Duplicate/replay/idempotency behavior is explicit wherever repeated execution is possible.
- Cross-service invariants include explicit outbox/idempotency/reconciliation stance or explicit non-applicability rationale.
- Test obligations are synchronized with `70-test-plan.md` for critical invariants and corner cases.
- Invariant blockers are closed or tracked in `80-open-questions.md` with owner and unblock condition.
- Impacted `30/40/55/60/70` artifacts have explicit status with decision links and no contradictions.
- Behavior-changing invariant updates include mixed-version compatibility and rollback notes.
- No hidden domain-behavior decisions are deferred to coding.
- No active item from `Hard Skills -> Review Blockers For This Skill` remains unresolved.

## Anti-Patterns
- ownerless invariants or invariants without source-of-truth and enforcement point
- state transitions defined only as prose with no explicit forbidden paths
- invariant violations that return success or ambiguous outcomes
- retry/idempotency semantics omitted where repeated execution can happen
- cross-service invariants defined without outbox/inbox/dedup/reconciliation controls
- write correctness based on stale projection/read-model data without freshness contract
- hidden authz/tenant assumptions instead of explicit invariant rules
- migration/rollout plan that can silently break invariants during mixed-version deployment
- critical invariant decisions deferred to coding or "implementation detail"
- unresolved critical assumptions left outside `80-open-questions.md`
