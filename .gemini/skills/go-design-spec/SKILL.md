---
name: go-design-spec
description: "Design specification-integrity passes for Go services in a spec-first workflow. Use when a draft spec needs an integrated pre-coding pass to enforce simplicity, maintainability, and cross-artifact consistency across `15/20/30/40/50/55/60/70`. Skip when the task is a local code fix, endpoint/schema-only editing, implementation coding, code-review execution, or CI/container setup."
---

# Go Design Spec

## Purpose
Create a clear, reviewable design-integrity specification pass before implementation. Success means the spec package is coherent across artifacts, accidental complexity is controlled, and implementation can proceed without hidden design decisions.

## Scope And Boundaries
In scope:
- enforce design integrity across `15/20/30/40/50/55/60/70`
- identify and reduce accidental complexity (unnecessary layers, indirections, speculative abstractions)
- define maintainability-oriented constraints (locality of change, explicit seams, predictable impact radius)
- ensure implementation plan has no deferred system-level design decisions
- register design blockers and unresolved complexity risks with owners
- produce design decisions that are testable and reviewable in later phases

Out of scope:
- primary ownership of service decomposition and ownership boundaries
- primary ownership of endpoint-level API payload/status/error contracts
- primary ownership of physical SQL modeling, migration mechanics, and datastore selection
- primary ownership of cache key/TTL/invalidation policy and SQL access discipline
- primary ownership of security controls, observability SLI/SLO policy, and CI/CD container hardening
- primary ownership of performance budgets and benchmark protocol
- writing production code, writing tests, or performing code-review role tasks

## Hard Skills
### Design Integrity Core Instructions

#### Mission
- Keep the spec package internally coherent across `15/20/30/40/50/55/60/70` before coding starts.
- Convert ambiguity into explicit design decisions with bounded complexity and clear ownership.
- Ensure approved implementation steps can execute without hidden architecture/API/data/reliability decisions.

#### Default Posture
- Spec-first is mandatory: design decisions must be closed before coding and survive `Gate G2` `Spec Freeze`.
- Prefer the simplest explicit design that satisfies current requirements and preserves future change locality.
- Treat accidental complexity as a merge blocker in spec phase when it increases integration risk or impact radius.
- Prefer additive, compatibility-first evolution and explicit transition windows over big-bang replacement.
- Preserve specialist ownership: `go-design-spec` aligns and integrates decisions, it does not replace domain-specific `*-spec` ownership.

#### Spec-First Gate Integrity Competency
- Enforce `docs/spec-first-workflow.md` sequencing and gate intent:
  - Phase 1: architecture sanity-check and complexity baseline.
  - Phase 2+: integrated cross-artifact reconciliation pass.
- Require unresolved design uncertainty to be explicit in `80-open-questions.md` with owner and unblock condition.
- Require accepted design decisions to be captured in `90-signoff.md` with reopen conditions.
- Reject design outputs that defer system-level decisions into coding phase.

#### Complexity And Maintainability Competency
- Apply simplicity rules as design constraints:
  - avoid speculative abstractions and indirection layers without proven pressure;
  - avoid interface-per-struct and service-manager-factory chains without distinct responsibilities;
  - prefer explicit boundaries and control flow over hidden magic.
- Require each abstraction to justify:
  - what concrete complexity it removes now,
  - why simpler alternatives were rejected,
  - what change-impact radius it creates.
- Protect maintainability by design:
  - keep ownership and dependency direction explicit;
  - minimize cross-artifact coupling;
  - preserve predictable local change paths.

#### Boundary And Ownership Consistency Competency
- Use the four-axis boundary model (domain, data ownership, team ownership, transaction boundary) when boundary decisions are touched.
- Require explicit source-of-truth ownership for critical entities in design decisions.
- Reject shared-schema coupling and cross-service direct DB assumptions in design narratives.
- Surface distributed-monolith signals early: coordinated releases, chatty call graphs, hidden shared logic, cross-service ACID assumptions.

#### Sync/API Design Seam Competency
- Verify sync vs async choice before transport details:
  - sync only when immediate response semantics are required;
  - async when latency variability/fan-out/eventual consistency dominates.
- For sync seams, require explicit contracts for:
  - deadlines and fail-fast budget behavior;
  - retry classification and bounded retries;
  - idempotency policy for retry-unsafe operations;
  - deterministic error model and pagination behavior.
- Guard API design consistency:
  - resource-oriented semantics and stable status mapping;
  - no hidden action-RPC style contract drift;
  - explicit eventual-consistency disclosure when applicable.

#### Async/Distributed Consistency Seam Competency
- Require explicit event vs command intent and topology choice (pub/sub vs queue) by workflow semantics.
- Require outbox/inbox or equivalent atomic/dedup guarantees for side-effecting async flows.
- Enforce explicit saga/workflow state model when cross-service invariants are involved.
- Require explicit compensation or forward-recovery semantics per critical distributed step.
- Reject dual writes and implicit exactly-once assumptions as design defaults.

#### Data Evolution And Cache Integrity Competency
- Keep local transaction boundaries explicit and compatible with service-owned data boundaries.
- Require migration safety shape for behavior-changing data evolution:
  - `expand -> backfill/verify -> contract`;
  - mixed-version compatibility window;
  - rollback class and limitations.
- Require cache decisions to preserve correctness:
  - cache as accelerator, not source of truth by default;
  - explicit staleness contract, invalidation/fallback behavior, and tenant-safe key design;
  - no cache dependency without observable fail-open or approved fail-closed rationale.

#### Security And Abuse-Resistance Seam Competency
- Require boundary validation and strict decoding for untrusted input paths in design contracts.
- Require explicit trust-boundary, tenant-isolation, and fail-closed authorization expectations where behavior depends on identity.
- Require outbound call safety assumptions to be explicit (timeouts, SSRF posture, retry/idempotency safety).
- Reject design choices that implicitly depend on insecure defaults or secret leakage through logs/errors.

#### Observability And Delivery Integrity Competency
- Require observability as design contract:
  - trace/log/metric correlation fields remain end-to-end;
  - RED + saturation visibility exists for changed critical paths;
  - metric cardinality stays bounded.
- Require design changes to remain enforceable by CI delivery gates:
  - contract/codegen drift controls;
  - migration validation requirements;
  - security and compatibility checks for merge/release.
- Reject design proposals that depend on undocumented manual release behavior.

#### Reliability And Rollout Integrity Competency
- Require per-dependency failure contract for impacted critical dependencies:
  - timeout budget;
  - retry budget and non-retry classes;
  - fallback/degradation mode;
  - overload isolation/bulkhead expectation.
- Require bounded queue/concurrency and explicit overload behavior where async or high-load paths are affected.
- Require rollout/rollback safety assumptions for risky changes (progressive rollout and rollback authority).
- Reject "heroic operations" assumptions without explicit degradation and rollback design.

#### Evidence Threshold And Decision Quality Bar
- Every major `DES-###` decision must include:
  - at least two options and one explicit rejection reason;
  - explicit simplicity/flexibility/cost/risk trade-offs;
  - cross-domain impact summary for architecture/API/data/security/operability/reliability/testing;
  - affected artifacts and required status (`updated` or `no changes required`) with rationale.
- Decision quality is insufficient if rationale is tool preference or taste without workload/constraint evidence.
- If a conditional artifact is unchanged, require one sentence proving no contract drift and link the controlling `DES-###`.

#### Assumption And Uncertainty Discipline
- Mark missing critical facts as bounded `[assumption]` immediately.
- Resolve each assumption with source-backed validation in the same pass when possible.
- Promote unresolved critical assumptions to blockers in `80-open-questions.md` with owner and unblock condition.
- Never hide unresolved uncertainty in generic wording.

#### Review Blockers For This Skill
- Any hidden "decide later in coding" system-level design gap.
- Cross-artifact contradiction left unresolved across impacted specs.
- New abstraction or layer with no measurable simplification outcome.
- Design change that breaks ownership boundaries without explicit rationale and downstream impact handling.
- API/data/reliability/security seam behavior drift not reflected in impacted artifacts.
- Migration/cache/retry/degradation assumptions that are not rollout-safe or are non-observable.
- Missing owner, rejected option, or reopen condition for major `DES-###` decisions.

## Working Rules
1. Determine current `docs/spec-first-workflow.md` phase and target gate before drafting decisions.
2. Set phase-specific output targets:
   - Phase 1: perform architecture sanity-check and complexity baseline alignment.
   - Phase 2 and later: run integrated design pass on the full spec package and reconcile cross-artifact inconsistencies.
3. Load context using this skill's dynamic loading rules and stop when four design axes are source-backed: artifact consistency, complexity profile, cross-domain seam integrity, and implementation readiness.
4. Normalize the design problem: where complexity grows, where ownership/terminology diverges, where seam behavior drifts, and where change impact becomes unpredictable.
5. For each nontrivial design decision, compare at least two options and select one explicitly.
6. Assign decision ID (`DES-###`) and owner for each major design decision.
7. Record trade-offs, cross-domain impact (architecture/API/data/security/operability/reliability/testing), and required artifact status updates.
8. Preserve specialist ownership: express design constraints and integration decisions without replacing domain-specific decisions owned by other `*-spec` roles.
9. Mark missing critical facts as `[assumption]`; keep assumptions bounded and either validate in current pass or move to `80-open-questions.md` with owner and unblock condition.
10. If uncertainty blocks coherent design closure, record it in `80-open-questions.md` with concrete next step.
11. Keep design outputs integration-first: resolve contradictions between artifacts before introducing new abstractions.
12. Verify internal consistency and blocker status: no hidden design choices are deferred to coding and no active hard-skill review blockers remain untracked.

## Design Decision Protocol
For every major design decision, document:
1. decision ID (`DES-###`) and current phase
2. owner role
3. context and complexity symptom
4. options (minimum two)
5. selected option with rationale
6. at least one rejected option with explicit rejection reason
7. trade-offs (`simplicity`/`flexibility`/cost/risk/change-impact)
8. cross-domain seam impact (architecture/API/data/security/operability/reliability/testing)
9. evidence basis and measurable acceptance boundaries
10. affected artifacts with explicit status decision (`updated` or `no changes required`)
11. control measures and reopen conditions
12. linked open-question IDs (if any)

## Output Expectations
- Response format:
  - `Decision Register`: accepted `DES-###` decisions with rationale, trade-offs, evidence basis, and artifact status decision
  - `Artifact Update Matrix`: `20/60/80/90` and conditional `30/40/50/55/70` with `Status: updated|no changes required` and linked `DES-###`
  - `Assumptions`: active `[assumption]` items and resolution path
  - `Open Blockers`: unresolved design blockers for `80-open-questions.md` with owner and unblock condition
  - `Blocker Check`: explicit statement that no hard-skill review blockers remain active, or list them with owner/action
  - `Sign-Off Delta`: what must be appended to `90-signoff.md` in this pass
- Primary artifacts:
  - `20-architecture.md`:
    - design-integrity findings
    - simplification decisions
    - explicit complexity boundaries
  - `60-implementation-plan.md`:
    - complexity-safe sequencing
    - integration-risk reduction order
    - no hidden "decide later" design gaps
  - `80-open-questions.md`:
    - unresolved complexity/design blockers with owner
  - `90-signoff.md`:
    - accepted design decisions and reopen criteria
- Conditional alignment artifacts (update when impacted):
  - `30-api-contract.md`
  - `40-data-consistency-cache.md`
  - `50-security-observability-devops.md`
  - `55-reliability-and-resilience.md`
  - `70-test-plan.md`
- Conditional artifact status format for `30/40/50/55/70`:
  - include one explicit status: `Status: updated` or `Status: no changes required`
  - for `no changes required`, add one sentence justification with linked `DES-###`
  - for `updated`, list changed sections and linked `DES-###`
- Language: match user language when possible.
- Detail level: concrete and reviewable with explicit simplification choices and integration impacts.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when four design axes are covered with source-backed inputs: artifact consistency, complexity profile, cross-domain seam integrity, and implementation readiness.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Artifacts`, current phase subsection, and target gate criteria first
  - load additional sections only when unresolved design decisions require them
- `docs/project-structure-and-module-organization.md`
- `docs/llm/go-instructions/30-go-project-layout-and-modules.md`
- `docs/llm/architecture/10-service-boundaries-and-decomposition.md`

Load by trigger:
- Sync request-reply and boundary interaction design implications:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
- Event-driven or async workflow coupling/complexity implications:
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
- Cross-service consistency and saga complexity implications:
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- Degradation/startup-shutdown/rollback complexity implications:
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- API-level simplicity and behavioral consistency impact:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Data/cache coupling and evolution impact:
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Maintainability dispute or unclear simplification trade-off:
  - `docs/llm/go-instructions/70-go-review-checklist.md`
- Security/operability/delivery complexity impact:
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/delivery/10-ci-quality-gates.md`

Conflict resolution:
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prefer trigger-loaded documents over always-loaded documents.
- If conflict persists, preserve latest accepted decision in `90-signoff.md` and add reopen blocker in `80-open-questions.md`.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- Resolve each `[assumption]` by source validation in current pass or by promoting it to `80-open-questions.md` with owner and unblock condition.

## Definition Of Done
- Current phase and target gate are explicitly stated.
- Major design conflicts between impacted artifacts are resolved or explicitly tracked.
- Every major decision includes `DES-###`, owner, selected option, and at least one rejected option with reason.
- Every major decision includes explicit evidence basis, acceptance boundaries, and affected artifact status decision.
- `20/60/80/90` are synchronized and contain no hidden design deferrals.
- Impacted `30/40/50/55/70` artifacts have explicit status with linked `DES-###`.
- Critical complexity risks are reduced with explicit rationale or tracked in `80-open-questions.md` with owner and unblock condition.
- No active hard-skill review blockers remain unresolved outside `80-open-questions.md`.
- No system-level design uncertainty is silently carried into coding phase.

## Anti-Patterns
Treat each item as a blocker unless explicitly tracked and owned:
- deferring system-level decisions to coding phase ("decide later in implementation")
- keeping cross-artifact contradictions unresolved after design pass
- introducing abstractions/layers without measurable simplification outcome
- changing specialist-domain decisions without rationale and cross-domain impact trace
- weakening API/data/reliability/security seam contracts in the name of "simplification"
- closing a pass with unresolved critical assumptions not tracked in `80-open-questions.md`
