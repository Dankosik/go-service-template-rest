---
name: go-data-architect-spec
description: "Design data-architecture-first specifications for Go services in a spec-first workflow. Use when planning or revising SQL/data modeling, consistency boundaries, datastore choice, schema evolution, migration rollout, and data reliability before coding. Skip when the task is a local code fix, endpoint-level API contract design, pure service decomposition work, CI/container setup, or low-level implementation tuning."
---

# Go Data Architect Spec

## Purpose
Create a clear, reviewable data specification package before implementation. Success means data ownership, consistency, evolution, and reliability decisions are explicit, defensible, and directly translatable into implementation and tests.
Use `Hard Skills` as the normative domain baseline for decision quality and risk controls; use workflow sections below for execution sequence and artifact synchronization.

## Scope And Boundaries
In scope:
- define service-owned data boundaries and schema ownership
- define OLTP data model shape (entities, relations, keys, constraints, indexes)
- define transaction boundaries and concurrency control expectations
- decide datastore class when needed (SQL OLTP default, NoSQL/columnar by justified exception)
- define schema evolution and migration safety strategy (expand/migrate/contract, compatibility window, rollback limits)
- define data reliability controls (verification, backup/restore expectations, retention/archival/PII deletion requirements)
- define implementation-facing data access constraints for Go code (query discipline, timeout/context expectations, pooling/batching boundaries)
- produce data deliverables that remove hidden "decide later" gaps

Out of scope:
- endpoint-level API contract design details
- service/module decomposition and ownership topology decisions outside data domain
- distributed orchestration implementation details as a primary concern
- runtime cache implementation details (exact keys, TTL/jitter tuning, invalidation mechanics)
- full security hardening catalog outside data-surface implications
- SLI/SLO targets and alert policy tuning
- CI/CD pipeline design and container runtime hardening
- low-level SQL implementation details and performance tuning in code

## Hard Skills
### Data Architecture Core Instructions

#### Mission
- Produce data decisions that remain correct under growth, partial failures, and mixed-version rollouts.
- Convert ambiguous requirements into explicit contracts for ownership, invariants, consistency, schema evolution, and recovery.
- Ensure every selected data option is implementable, testable, observable, and rollback-aware before coding starts.

#### Default Posture
- Default to SQL OLTP (`PostgreSQL`-compatible) as primary system-of-record for service business data.
- Default to one service-owned schema/database boundary and local ACID within that boundary.
- Default cross-service consistency model to eventual consistency with explicit outbox/idempotency/reconciliation.
- Default schema evolution model to compatibility-first rollout: `expand -> migrate/backfill -> contract`.
- Treat cache as an accelerator by default, not as source of truth.
- Treat new datastore engines (NoSQL/columnar) as exceptions that require workload-fit evidence and operational readiness proof.

#### Data Ownership And Boundary Competency
- Enforce explicit source-of-truth ownership per critical entity.
- Enforce service-private data boundaries:
  - no direct cross-service table access;
  - no cross-service foreign keys;
  - cross-service references by IDs only.
- Require cross-service read strategies to be explicit: API composition, replicated read models, or event-driven projections.
- Reject shared-schema coupling and service-per-table decomposition by default.
- Require ownership and transaction-boundary impact to be stated for every major model change.

#### SQL/OLTP Modeling Competency
- Start from normalized schema (`~3NF`) by default; allow denormalization only with explicit:
  - source of truth;
  - synchronization path;
  - staleness contract;
  - reconciliation owner.
- Treat constraints as contract, not optional hints:
  - PK on every table;
  - `UNIQUE` for business uniqueness;
  - `NOT NULL` by default for required fields;
  - row-local `CHECK` constraints where applicable.
- Keep referential integrity inside one service boundary; never model cross-service FK constraints.
- Build indexing policy from measured or explicitly known access patterns; avoid speculative index growth.
- Enforce deterministic pagination with stable ordering and unique tie-breaker; default operational lists to keyset pagination.
- Require explicit delete/history policy per dataset:
  - hard delete by default;
  - soft delete/temporal/audit only with restore/compliance/history requirement and lifecycle controls.
- For pooled multi-tenancy, require tenant-safe model (`tenant_id` scope + enforced centralized isolation controls such as RLS).

#### Consistency, Transactions, And Concurrency Competency
- Keep transaction boundaries local to one service datastore and short-lived.
- Reject cross-service global ACID assumptions for default designs.
- Default to optimistic concurrency for mutable entities:
  - version/CAS checks;
  - explicit conflict semantics on update races.
- Allow pessimistic locking only by proven contention/resource-serialization need.
- Treat stricter isolation levels as exception paths requiring retry design and tests for serialization/deadlock failures.
- Require retry policy to be bounded and idempotency-aware for write paths.

#### Go SQL Access Contract Competency
- Define implementation-facing DB access defaults for `Go` explicitly:
  - `sqlc` + explicit SQL as DAL contract baseline;
  - `pgx/v5 + pgxpool` for PostgreSQL-first path;
  - `database/sql + sqlc` only when portability is hard requirement.
- Keep SQL as source-of-truth artifacts; generated code is derivative and not manually edited.
- Require explicit transaction ownership at use-case boundary (`Begin -> defer Rollback -> Commit`).
- Require end-to-end context propagation and DB deadlines; reject context-less DB calls in production flows.
- Require explicit pool budgets and connection-capacity validation before scaling assumptions.
- Require prevention of `N+1` and chatty access patterns in hot paths.
- Require parameterized SQL values and allow-listed dynamic identifiers.
- Require critical-query observability contract:
  - stable query identity;
  - slow-query thresholds;
  - pool saturation visibility.

#### Datastore Class Selection Competency
- Keep SQL OLTP as default unless alternative engine is justified by explicit access-pattern evidence.
- Require an access-pattern catalog before selecting document/key-value/wide-column/time-series/columnar class.
- Require partition-key, skew, growth, and hot-partition mitigation plan before approving partitioned NoSQL designs.
- Require consistency model and conflict behavior to be explicit per critical operation.
- Require retention, deletion, archival, and downsampling semantics for non-OLTP engines before launch.
- Apply cache-first guardrail:
  - if bottleneck is repeated read amplification with safe cacheability, evaluate cache before new datastore class.
- Reject datastore adoption when operational readiness is absent (on-call, backup/restore, observability, runbooks).

#### Schema Evolution And Migration Safety Competency
- Enforce mixed-version compatibility contract across rollout.
- Require one controlled migration runner with immutable versioned migrations and separate migration/runtime roles.
- Require migration safety budgets (lock/statement/idle-in-transaction timeouts) at migration session level.
- Enforce phased rollout for behavior-changing schema updates:
  - `expand` (additive-compatible),
  - `migrate/backfill` (idempotent/resumable/throttled),
  - `contract` (destructive last, verification-gated).
- Require backfill control plane:
  - checkpoints/watermarks;
  - bounded retries;
  - abort thresholds for error rate, lag, lock waits.
- Require objective verification gates before contract:
  - row/aggregate parity;
  - deterministic sample diff;
  - domain invariant checks.
- Require rollback class declaration (`safe`, `conditional`, `restore-based`) and explicit irreversible-step notes.
- Reject cross-system dual writes for state-change + message emission; require outbox-equivalent atomic linkage.

#### Data Reliability, Lifecycle, And Recovery Competency
- Treat backups as insufficient without restore drills.
- Require explicit RPO/RTO by data class and tested restore runbooks.
- Require retention/archival ownership and deletion triggers per dataset.
- Require explicit PII data map and traceable deletion/anonymization workflow across primary and derived stores.
- Require explicit statement of backup/archive residual retention limitations when hard deletion is requested.
- Require disaster-recovery posture for destructive migration, regional outage, and corruption/backfill bug scenarios.
- Require downstream consumer/version alignment before schema contract that affects published data semantics.

#### Data vs Cache Responsibility Boundary Competency
- Keep authoritative data correctness contract in source-of-truth datastore decisions.
- Require explicit classification of cacheable vs strict-consistency reads.
- Require staleness and fallback class decisions at spec level when cache interaction affects data behavior.
- Keep runtime cache tuning details (exact keys, TTL tuning, invalidation mechanics) delegated to `go-db-cache-spec`.
- Reject cache proposals that introduce tenant leakage or make read availability depend on cache correctness by default.

#### API And Distributed Consistency Impact Competency
- When data decisions affect API behavior, require explicit contract updates for:
  - consistency model disclosure (`strong`/`eventual`);
  - staleness/freshness semantics;
  - async materialization patterns (`202` + operation resource) where applicable;
  - idempotency and concurrency semantics for retryable writes.
- For cross-service flows, require invariant ownership and explicit step contracts (idempotency, timeout, retry, compensation/forward recovery).
- Require durable ordering of side effects before async acknowledgements in consumer flows.
- Reject hidden invariant ownership and “eventual fix by consumer” assumptions.

#### Security And Tenant-Isolation Impact Competency
- Require parameterized SQL and allow-listed query-shape controls for any user-influenced filtering/sorting path.
- Require least-privilege role separation for migration runtime vs application runtime.
- Require tenant scope propagation and enforcement across DB access, cache boundaries, async payloads, and audit/operational trails.
- Require fail-closed behavior on tenant mismatch and object-level authorization dependencies before side effects.
- Require data-change plans to avoid secret/PII leakage in logs, errors, and telemetry.

#### Observability And Diagnostics Impact Competency
- Require telemetry contract for changed data paths:
  - DB operation latency/error metrics;
  - pool saturation metrics;
  - trace/log correlation fields.
- Require migration/backfill/reconciliation observability:
  - progress, lag, retry, failure, and invariant-violation visibility.
- Require async correlation continuity for data pipelines (`trace`, `correlation_id`, `attempt`, `message_id`).
- Enforce metric cardinality discipline; high-cardinality identifiers stay in logs/traces, not metric labels.
- Require telemetry changes to be cost-aware, bounded, and incident-operable.

#### Evidence Threshold And Decision Quality Bar
- Every major data decision must include at least two options and one explicit rejection reason.
- Every selected option must include measurable acceptance boundaries, not only narrative rationale.
- Every selected option must include compatibility class, rollout sequence, and rollback limitations.
- Every selected option must include cross-domain impact summary for architecture/API/security/operability.
- Every selected option must include reopen conditions tied to observable triggers.
- Minimum evidence by decision axis:
  - ownership/modeling:
    - source-of-truth map, invariants-to-constraints mapping, tenant model decision;
  - consistency/transactions:
    - transaction boundary map, conflict strategy, retry/idempotency class;
  - evolution/migrations:
    - phased rollout plan, backfill control strategy, verification queries and gates;
  - reliability/lifecycle:
    - rollback class, backup/restore proof strategy, retention/PII handling;
  - datastore-class exceptions:
    - access-pattern evidence, partition/hot-key strategy, operational readiness checklist.

#### Assumption And Uncertainty Discipline
- Mark unknown critical facts as `[assumption]` immediately.
- Keep assumptions bounded, testable, and decision-linked.
- Resolve assumptions within the same pass when source-backed validation is possible.
- Promote unresolved critical assumptions to `80-open-questions.md` with owner and unblock condition.

#### Review Blockers For This Skill
- Data recommendation without explicit trade-off analysis and rejected option.
- Data decision that defers core correctness/evolution choices into coding phase.
- Shared-schema or cross-service DB coupling introduced without explicit approved exception.
- Schema evolution plan without mixed-version compatibility contract.
- Migration plan without phased rollout, verification gates, or rollback limitations.
- Backfill plan without idempotency, checkpoints, throttling, and abort criteria.
- Datastore-class change without access-pattern evidence and operational readiness proof.
- Cache boundary blurred so correctness depends on cache behavior without explicit approved exception.
- Cross-system dual-write used as default consistency mechanism.
- Security/tenant/observability implications omitted for data decisions.
- Unresolved critical unknowns left implicit instead of explicit blockers.

## Working Rules
1. Determine current `docs/spec-first-workflow.md` phase and target gate before drafting decisions.
2. Set phase-specific output targets:
   - Phase 0: `80-open-questions.md` with data assumptions/blockers and their owners; add only the minimum data constraints needed to keep architecture drafting safe
   - Phase 1: explicit data constraints that shape `20-architecture.md` and rollout-safe data change sequencing in `60-implementation-plan.md`
   - Phase 2 and later: `40/80/90` plus impacted `20/30/50/55/60/70`
3. Apply `Hard Skills` defaults by default. Any deviation must be explicit, justified, and linked to decision ID (`DATA-###`) and reopen criteria.
4. Load context using this skill's dynamic loading rules and stop when four data axes are source-backed: ownership/modeling, consistency/transactions, evolution/migrations, and reliability controls.
5. Normalize the data problem: domain entities, invariants, consistency expectations, change constraints, and operational constraints.
6. For each nontrivial data decision, compare at least two options and select one explicitly.
7. Assign decision ID (`DATA-###`) and owner for each major data decision.
8. Record trade-offs and cross-domain impact (architecture, API, security, operability) for each selected decision.
9. Mark missing critical facts as `[assumption]`; keep assumptions bounded and either validate them in the current pass or convert them into blockers in `80-open-questions.md` with owner and unblock condition.
10. If uncertainty blocks decision quality or rollout safety, record it in `80-open-questions.md` with concrete next step.
11. Keep `40-data-consistency-cache.md` as primary artifact and maintain explicit boundary with cache-specific responsibilities.
12. Verify internal consistency: no contradictions between `40` and impacted `20/30/50/55/60/70/90`, and no hidden data decisions deferred to coding.
13. Run final blocker check against `Hard Skills -> Review Blockers For This Skill` before closing a pass.

## Data Decision Protocol
For every major data decision, document:
1. decision ID (`DATA-###`) and current phase
2. owner role
3. context and problem
4. options (minimum two)
5. selected option with rationale
6. at least one rejected option with explicit rejection reason
7. trade-offs (gains and losses)
8. compatibility impact (additive, behavior-change, breaking + migration window)
9. consistency and transaction semantics
10. migration/backfill/recovery strategy and rollback limitations
11. impact on architecture, API, security, and operability
12. reopen conditions, affected artifacts, and linked open-question IDs (if any)
13. evidence package by axis:
   - ownership/modeling evidence
   - consistency/transaction evidence
   - migration/evolution evidence
   - reliability/recovery evidence

## Output Expectations
- Phase-specific minimum output:
  - Phase 0:
    - `80-open-questions.md` with data blockers/unknowns, owner, and unblock condition
    - minimal data constraints for architecture safety captured in the current pass
  - Phase 1:
    - data-boundary and consistency constraints reflected in `20-architecture.md`
    - schema-change and migration-sequencing constraints reflected in `60-implementation-plan.md`
    - unresolved data blockers tracked in `80-open-questions.md`
  - Phase 2 and later:
    - full `40-data-consistency-cache.md`
    - synchronized `80-open-questions.md` and `90-signoff.md`
- Primary artifact:
  - `40-data-consistency-cache.md` containing:
    - `Data Ownership And Boundaries`
    - `Data Model And Constraints`
    - `Consistency And Transaction Rules`
    - `Datastore Choice Rationale`
    - `Schema Evolution And Migration Plan`
    - `Data Reliability And Verification Controls`
    - `Data vs Cache Responsibility Boundary`
  - For each major `DATA-###` in `40-data-consistency-cache.md`, include a compact decision card:
    - selected option and rejected option
    - compatibility class and rollout strategy
    - evidence summary and reopen conditions
- Required core artifacts per pass:
  - `80-open-questions.md` with data blockers/uncertainties
  - `90-signoff.md` with accepted data decisions and reopen criteria
- Conditional alignment artifacts (update when impacted by data decisions):
  - `20-architecture.md`
  - `30-api-contract.md`
  - `50-security-observability-devops.md`
  - `55-reliability-and-resilience.md`
  - `60-implementation-plan.md`
  - `70-test-plan.md`
- Conditional artifact status format for `20/30/50/55/60/70`:
  - include one explicit status: `Status: updated` or `Status: no changes required`
  - for `no changes required`, add one sentence justification with linked `DATA-###`
  - for `updated`, list changed sections and linked `DATA-###`
- Language: match user language when possible.
- Detail level: concrete and reviewable with explicit assumptions, trade-offs, and change safety constraints.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when the four data axes are covered with source-backed inputs: ownership/modeling, consistency/transactions, migration/evolution, reliability.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Artifacts`, current phase subsection, and target gate criteria first
  - load additional sections only if required for unresolved decisions
- `docs/llm/data/10-sql-modeling-and-oltp.md`
- `docs/llm/data/20-sql-access-from-go.md`
- `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`

Load by trigger:
- Datastore class choice or analytical/read-model introduction:
  - `docs/llm/data/30-nosql-and-columnar-decision-guide.md`
- Data/cache interaction or cache-boundary policy:
  - `docs/llm/data/50-caching-strategy.md`
- API consistency/idempotency implications:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Cross-service consistency implications:
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- Data-surface security implications:
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/security/20-authn-authz-and-service-identity.md`
- Data-change observability and diagnostics implications:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`

Conflict resolution:
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prefer trigger-loaded documents over always-loaded documents.
- If conflict persists, preserve latest accepted decision in `90-signoff.md` and add reopen blocker in `80-open-questions.md`.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- Resolve each `[assumption]` by source validation in current pass or by promoting it to `80-open-questions.md` with owner and unblock condition.

## Definition Of Done
- Current phase and target gate are explicitly stated.
- `40-data-consistency-cache.md` explicitly defines ownership, model, consistency, evolution, and reliability decisions.
- All major data decisions include `DATA-###`, owner, selected option, and at least one rejected option with reason.
- Schema changes include compatibility class, rollout sequence, and rollback limitations.
- Each major decision includes explicit evidence package by axis (ownership/modeling, consistency/transactions, migration/evolution, reliability/recovery).
- Data/cache boundary is explicit and non-overlapping.
- Data blockers are closed or tracked in `80-open-questions.md` with owner and unblock condition.
- Impacted `20/30/50/55/60/70` artifacts have explicit status with decision links and no contradictions.
- No active item from `Hard Skills -> Review Blockers For This Skill` remains unresolved.
- No hidden data decisions are deferred to coding.

## Anti-Patterns
- Data decisions made from API payload shape only, without domain invariants and ownership model.
- Shared-schema or cross-service DB coupling introduced as a default path.
- Critical schema changes planned without mixed-version compatibility contract.
- Destructive-first migrations (`DROP/RENAME`) before expand/backfill/verification gates.
- Backfill plans without idempotency, checkpoints, throttling, and abort thresholds.
- Cross-system dual writes used as default consistency mechanism instead of outbox-equivalent linkage.
- Datastore-class migration proposed without access-pattern evidence and operational readiness proof.
- Cache behavior treated as correctness authority without explicit approved exception.
- Security/tenant/observability implications omitted from data decisions.
- Critical unknowns left implicit instead of `[assumption]` validation or `80-open-questions.md` blockers.
