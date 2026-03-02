---
name: go-domain-invariant-spec
description: "Design domain-invariant-first specifications for Go services in a spec-first workflow. Use when planning or revising behavior before coding and you need explicit business invariants, state-transition rules, acceptance criteria, corner-case handling, and traceability into API/data/reliability/testing artifacts. Skip when the task is a local code fix, low-level implementation, endpoint schema-only design, SQL/migration mechanics, or CI/container setup."
---

# Go Domain Invariant Spec

## Purpose
Create a clear, reviewable domain-behavior specification package before implementation. Success means business invariants and acceptance behavior are explicit, testable, and directly translatable into implementation and test obligations.

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

## Working Rules
1. Determine current `docs/spec-first-workflow.md` phase and target gate before drafting decisions.
2. Set phase-specific output targets:
   - Phase 0: establish initial invariant register in `15` and record invariant unknowns in `80`.
   - Phase 1: refine invariants and acceptance criteria to verifiable form and align with baseline architecture constraints.
   - Phase 2 and later: run full invariant pass across spec package, maintain `15/80/90`, and update impacted artifacts.
3. Load context using this skill's dynamic loading rules and stop when four invariant axes are source-backed: invariant set, transition rules, violation semantics, and test traceability obligations.
4. Normalize domain behavior scope: entities, lifecycle states, command/event triggers, and failure semantics.
5. For each nontrivial domain decision, compare at least two options and select one explicitly.
6. Assign decision ID (`DOM-###`) and owner for each major invariant decision.
7. Record trade-offs and cross-domain impact (architecture, API, data, distributed consistency, reliability, security, testing).
8. Mark missing critical facts as `[assumption]`; keep assumptions bounded and either validate in current pass or move to `80-open-questions.md` with owner and unblock condition.
9. If uncertainty blocks invariant closure, record it in `80-open-questions.md` with concrete next step.
10. Keep `15-domain-invariants-and-acceptance.md` as primary artifact and synchronize impacted `30/40/55/60/70/90` sections.
11. Verify internal consistency: no contradictions between invariant definitions, acceptance criteria, and downstream artifacts.
12. Keep focus on domain expertise by making explicit behavior decisions, not generic process commentary.

## Invariant Decision Protocol
For every major invariant decision, document:
1. decision ID (`DOM-###`) and current phase
2. owner role
3. context and business problem
4. invariant statement in verifiable form
5. scope (entity/use-case/process/cross-service)
6. options (minimum two for nontrivial cases)
7. selected option with rationale
8. at least one rejected option with explicit rejection reason
9. transition constraints (allowed/forbidden transitions + preconditions/postconditions)
10. violation behavior (error semantics, compensation expectations when applicable)
11. cross-domain impact and trade-offs
12. test obligations, reopen conditions, affected artifacts, and linked open-question IDs (if any)

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
- Required core artifacts per pass:
  - `80-open-questions.md` with invariant blockers/unknowns
  - `90-signoff.md` with accepted invariant decisions and reopen criteria
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
- Critical state transitions include explicit allowed/forbidden rules with preconditions/postconditions.
- Acceptance criteria are behavior-level and testable without reinterpretation.
- Invariant-violation behavior is explicit and consistent with API/reliability expectations.
- Test obligations are synchronized with `70-test-plan.md` for critical invariants and corner cases.
- Invariant blockers are closed or tracked in `80-open-questions.md` with owner and unblock condition.
- Impacted `30/40/55/60/70` artifacts have explicit status with decision links and no contradictions.
- No hidden domain-behavior decisions are deferred to coding.

## Anti-Patterns
Use these preferred patterns to avoid anti-pattern drift:
- express each invariant as a verifiable rule with observable pass/fail outcomes
- cover both happy-path and forbidden/fail-path transitions in the same behavior frame
- keep domain decisions at behavior level and leave low-level implementation details to coding phase
- reference architecture/API/data/security decisions only through explicit domain rationale and impact
- track every critical invariant ambiguity in `80-open-questions.md` with owner and unblock condition
- close product-behavior decisions in spec artifacts before coding starts
