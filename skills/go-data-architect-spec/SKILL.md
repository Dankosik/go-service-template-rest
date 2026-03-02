---
name: go-data-architect-spec
description: "Design data-architecture-first specifications for Go services in a spec-first workflow. Use when planning or revising SQL/data modeling, consistency boundaries, datastore choice, schema evolution, migration rollout, and data reliability before coding. Skip when the task is a local code fix, endpoint-level API contract design, pure service decomposition work, CI/container setup, or low-level implementation tuning."
---

# Go Data Architect Spec

## Purpose
Create a clear, reviewable data specification package before implementation. Success means data ownership, consistency, evolution, and reliability decisions are explicit, defensible, and directly translatable into implementation and tests.

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

## Working Rules
1. Determine current `docs/spec-first-workflow.md` phase and target gate before drafting decisions.
2. Set phase-specific output targets:
   - Phase 0: `80-open-questions.md` with data assumptions/blockers and their owners; add only the minimum data constraints needed to keep architecture drafting safe
   - Phase 1: explicit data constraints that shape `20-architecture.md` and rollout-safe data change sequencing in `60-implementation-plan.md`
   - Phase 2 and later: `40/80/90` plus impacted `20/30/50/55/60/70`
3. Load context using this skill's dynamic loading rules and stop when four data axes are source-backed: ownership/modeling, consistency/transactions, evolution/migrations, and reliability controls.
4. Normalize the data problem: domain entities, invariants, consistency expectations, change constraints, and operational constraints.
5. For each nontrivial data decision, compare at least two options and select one explicitly.
6. Assign decision ID (`DATA-###`) and owner for each major data decision.
7. Record trade-offs and cross-domain impact (architecture, API, security, operability) for each selected decision.
8. Mark missing critical facts as `[assumption]`; keep assumptions bounded and either validate them in the current pass or convert them into blockers in `80-open-questions.md` with owner and unblock condition.
9. If uncertainty blocks decision quality or rollout safety, record it in `80-open-questions.md` with concrete next step.
10. Keep `40-data-consistency-cache.md` as primary artifact and maintain explicit boundary with cache-specific responsibilities.
11. Verify internal consistency: no contradictions between `40` and impacted `20/30/50/55/60/70/90`, and no hidden data decisions deferred to coding.

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
- Data/cache boundary is explicit and non-overlapping.
- Data blockers are closed or tracked in `80-open-questions.md` with owner and unblock condition.
- Impacted `20/30/50/55/60/70` artifacts have explicit status with decision links and no contradictions.
- No hidden data decisions are deferred to coding.

## Anti-Patterns
- prefer domain-invariant-led schema decisions over API-payload mirroring
- use expand/migrate/contract with explicit compatibility windows for risky schema changes
- plan dual-write/backfill with verification checkpoints and recovery path
- keep data ownership decisions separate from cache runtime tuning details
- justify NoSQL/columnar with explicit access-pattern evidence and operational fit
- resolve critical unknowns through `[assumption]` validation or explicit `80-open-questions.md` tracking before coding
