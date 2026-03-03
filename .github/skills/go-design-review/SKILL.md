---
name: go-design-review
description: "Review Go code changes for architecture and design integrity in a spec-first workflow. Use when reviewing implementation against approved specs and you need findings on architecture alignment, complexity control, and maintainability drift with spec-reopen escalation rules. Skip when planning specs before coding, writing code/tests, or performing deep domain reviews for performance, security, DB/cache, concurrency, or QA as primary focus."
---

# Go Design Review

## Purpose
Validate that code changes stay aligned with approved architecture and design decisions in spec-first workflow reviews. Success means actionable findings prevent architecture drift and unresolved spec conflicts before `Gate G4`.

## Scope And Boundaries
In scope:
- compare implementation with approved spec artifacts (`20/60/65` and relevant `15/30/40/50/55/70/90`)
- detect boundary violations, dependency-direction breaks, and hidden cross-layer coupling
- detect accidental complexity and maintainability regressions
- verify code does not introduce unapproved architecture-level decisions
- raise `Spec Reopen` when implementation requires spec-level changes
- produce domain-scoped findings with concrete `file:line`, impact, and fix

Out of scope:
- redesigning architecture from scratch without an explicit spec conflict
- editing any spec artifact during Phase 4 review
- deep primary-domain review for idiomatic/style, QA/testing, performance, concurrency, DB/cache, reliability, or security topics
- blocking changes by subjective preference without concrete design impact

## Hard Skills
### Design Review Core Instructions

#### Mission
- Protect approved architecture intent during Phase 4 by converting design drift into concrete, merge-actionable findings.
- Keep implementation aligned with approved boundaries, execution plan, and maintainability constraints before `Gate G4`.
- Detect cross-domain seam regressions (API/data/security/reliability/observability/delivery/testing) only when they create design-level drift or spec inconsistency.

#### Default Posture
- `20-architecture.md`, `60-implementation-plan.md`, `65-coder-detailed-plan.md`, and accepted decisions in `90-signoff.md` are the design source of truth.
- Review changed code and directly impacted paths first; do not start from broad repository cleanup.
- Keep findings evidence-backed, minimal, and correction-oriented.
- No spec edits in Phase 4; design/spec conflicts are escalated through `Spec Reopen`.
- Keep deep domain analysis owned by the corresponding `*-review` role; `go-design-review` evaluates system-level design integrity, not specialist depth.

#### Spec-First Review And Gate Competency
- Enforce workflow constraints from `docs/spec-first-workflow.md`:
  - domain-scoped review only;
  - concrete `file:line` evidence;
  - practical fixes, not abstract advice;
  - no subjective merge blocks.
- Enforce `Spec Freeze` behavior:
  - no hidden architecture/API/consistency/security/reliability decisions introduced in code;
  - any required design decision change triggers `Spec Reopen`.
- Treat unresolved `high/critical` design conflicts as `Gate G4` blockers.

#### Architecture Compliance Competency
- Validate implementation against approved boundaries, ownership, and dependency direction from `20-architecture.md`.
- Detect and flag:
  - hidden cross-layer coupling;
  - bypass of composition seams;
  - new undeclared dependencies that alter architecture shape;
  - implementation shortcuts that effectively redefine component ownership.
- Reject design drift masked as "local refactor" when it changes architectural behavior or responsibility boundaries.

#### Plan Conformance And Spec-Freeze Execution Competency
- Map implementation to approved task flow in `65-coder-detailed-plan.md` and validate preservation of strategic constraints from `60-implementation-plan.md`.
- Flag "decision later in code" behavior:
  - unresolved TODO-driven architecture choices;
  - ad hoc branching that contradicts signed-off execution sequence;
  - behavior changes lacking corresponding approved decision in `90`.
- Treat untracked divergence from approved plan semantics as design finding, even if tests pass.

#### Complexity Control Competency
- Identify accidental complexity that raises long-term change cost:
  - speculative abstractions without proven extension need;
  - unnecessary indirection layers;
  - duplicated responsibility across packages/components;
  - complex control flow that obscures lifecycle/ownership intent.
- Prefer the smallest correction that restores explicit, maintainable design over full redesign proposals.

#### Maintainability And Evolvability Competency
- Evaluate whether changed code keeps:
  - explicit ownership of behavior;
  - bounded impact radius for common future changes;
  - predictable extension path without hidden side effects.
- Flag design choices that increase operational/debug burden through unclear control flow, hidden dependency coupling, or unclear invariants at module seams.

#### API Contract Design-Seam Competency
- When API-touching code changes, verify no design-level contract drift against approved `30` decisions and API defaults:
  - method/status semantics remain consistent;
  - retry/idempotency/precondition semantics are not weakened;
  - async behavior uses explicit `202 + operation` semantics when required;
  - error model consistency is preserved.
- For cross-cutting API seams, flag design drift when contract obligations and middleware/runtime enforcement diverge (validation, limits, auth context, correlation, rate limit, async semantics).

#### Data/Consistency/Cache Design-Seam Competency
- When DB/cache/migration seams are touched, validate design integrity expectations:
  - service-owned data boundaries remain intact;
  - transaction boundaries stay explicit and local;
  - cache remains accelerator with explicit staleness/fallback semantics;
  - migration/evolution logic remains rollout-safe (`expand -> backfill -> contract`) where relevant.
- Raise design findings when implementation introduces coupling or lifecycle assumptions that conflict with approved consistency/evolution model.

#### Security Design-Seam Competency
- Validate that design-level trust-boundary controls remain intact:
  - boundary validation is not bypassed;
  - auth/tenant context remains fail-closed by design;
  - security-sensitive flows do not shift from explicit controls to implicit assumptions.
- Flag architectural/security drift (not low-level exploit analysis) when code structure weakens enforceability of secure defaults.

#### Observability Design-Seam Competency
- Validate observability as design contract on changed production paths:
  - correlation fields and context propagation remain end-to-end;
  - telemetry coverage exists across changed component seams;
  - metric cardinality discipline is preserved.
- Flag design regressions where operability becomes non-deterministic (for example, broken correlation chain or unbounded telemetry dimensions introduced by design choice).

#### Reliability And Degradation Design-Seam Competency
- Validate design-level resilience contracts when failure paths are touched:
  - explicit deadlines and propagation model are preserved;
  - bounded retry/backpressure behavior remains explicit;
  - degradation/fallback behavior is deliberate and observable;
  - startup/shutdown/probe semantics remain compatible with approved reliability model.
- Flag hidden reliability drift such as implicit infinite waits/retries, unbounded buffering, or unsafe rollout assumptions introduced through design changes.

#### Delivery And Quality-Gate Design-Seam Competency
- Treat delivery controls as design integrity constraints when behavior-impacting changes are made:
  - no contract/codegen/docs/migration drift introduced without corresponding updates;
  - merge-safety assumptions remain machine-verifiable through required gate evidence.
- Flag design-level process drift when implementation depends on undocumented or non-enforced CI/release assumptions.

#### Testability Design-Seam Competency
- Validate that changed behavior remains testable according to approved `70-test-plan.md` obligations:
  - nontrivial behavior has deterministic tests or explicit risk note;
  - concurrency-sensitive design changes are expected to include race-safety evidence;
  - contract-sensitive behavior has stable, reviewable verification surface.
- Record residual risk when design cannot be confidently validated from available test evidence.

#### Evidence Threshold And Severity Calibration Competency
- Every finding must include:
  - exact `file:line`;
  - explicit design impact;
  - smallest safe correction;
  - concrete spec reference.
- Severity is assigned by architecture integrity and merge risk, not by code style preference.
- "Looks cleaner" without explicit design risk is not a valid finding.

#### Assumption And Uncertainty Discipline
- If key artifacts are missing, continue with bounded `[assumption]` and state reduced certainty.
- Any unresolved assumption that can change merge safety must be surfaced in `Open Questions` or escalated via `Spec Reopen`.
- Do not hide uncertainty behind generic wording; make uncertainty explicit and testable.

#### Review Blockers For This Skill
- Boundary/ownership/dependency direction violation against approved architecture.
- Unapproved architecture-level decision introduced during implementation.
- Significant plan divergence with no signed-off spec decision.
- Design drift at API/data/security/reliability/observability seams that changes approved behavior semantics.
- Maintainability/complexity change that materially increases regression or future-change risk and is left unaddressed.
- Any spec-level conflict left without `Spec Reopen`.

## Working Rules
1. Confirm review unit from context (`single task` or `bounded task scope`) as a Phase 4 code review, then identify the target feature and diff scope.
2. Determine `feature-id` from task context, changed paths, or review metadata. If `feature-id` cannot be identified, continue with bounded `[assumption]` and state reduced certainty.
3. Load approved feature artifacts and review context using dynamic loading rules from this skill.
4. Review changed code first, then map each risky change to relevant approved spec decisions.
5. Evaluate exactly five design axes:
   - `Architecture Compliance`
   - `Plan Conformance`
   - `Complexity Control`
   - `Maintainability`
   - `Spec Consistency`
6. For each triggered seam (`API`, `Data/Cache`, `Security`, `Observability`, `Reliability`, `Delivery`, `Testing`), assess only architecture/design drift and maintainability impact; do not replace deep-domain reviewer ownership.
7. Report only evidence-backed findings where changed code and spec reference are both explicit.
8. Classify each finding severity (`critical/high/medium/low`) by architecture integrity and merge risk.
9. For every finding, provide the smallest safe correction and explicit spec reference.
10. If a required correction changes approved spec decisions, record a `Spec Reopen` entry in `reviews/<feature-id>/code-review-log.md`.
11. If a risk is valid but outside design-review ownership, record it in `Handoffs` to the owner reviewer skill.
12. If no findings exist, state that explicitly and include handoffs and residual design risks or verification gaps.

## Output Expectations
- Present findings first, sorted by severity (highest first).
- Match output language to the user language when possible.
- Use this exact format for each finding:

```text
[severity] [go-design-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

- After findings, include:
  - `Handoffs`: cross-domain risks that require owner reviewer follow-up and are not design findings by themselves.
  - `Open Questions`: unresolved design uncertainties only.
  - `Spec Reopen`: `required` or `not required` with reason.
  - `Residual Risks`: short list of remaining non-blocking design risks.
- If there are no findings, output `No design findings.` and still include `Handoffs` and `Residual Risks`.

Severity guide:
- `critical`: merge-blocking architecture or boundary violation, or change that invalidates approved design without reopen.
- `high`: significant architecture drift or complexity growth that materially raises regression or change cost risk.
- `medium`: maintainability issue that should be corrected but has bounded near-term risk.
- `low`: local design cleanup that improves clarity and future change safety.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when the five review axes and all triggered design seams can be assessed with concrete code evidence and at least one relevant approved source each.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Phase 4`, `Reviewer Focus Matrix`, `Review Findings Format`, and `Gate G4` sections first
- `docs/llm/go-instructions/70-go-review-checklist.md`
- review target artifacts:
  - `specs/<feature-id>/20-architecture.md`
  - `specs/<feature-id>/65-coder-detailed-plan.md`
  - `specs/<feature-id>/60-implementation-plan.md`
  - `specs/<feature-id>/90-signoff.md`
  - `reviews/<feature-id>/code-review-log.md` (if present)

Load by trigger:
- Invariant and acceptance behavior impact:
  - `specs/<feature-id>/15-domain-invariants-and-acceptance.md`
- API contract or cross-cutting API behavior impact:
  - `specs/<feature-id>/30-api-contract.md`
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Data/consistency/cache seam impact:
  - `specs/<feature-id>/40-data-consistency-cache.md`
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Security/observability/delivery control impact:
  - `specs/<feature-id>/50-security-observability-devops.md`
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/delivery/10-ci-quality-gates.md`
- Reliability/failure/degradation behavior impact:
  - `specs/<feature-id>/55-reliability-and-resilience.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Testability obligations impact:
  - `specs/<feature-id>/70-test-plan.md`
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`

Conflict resolution:
- Approved feature decisions in `specs/<feature-id>/90-signoff.md` override generic guidance unless `Spec Reopen` is initiated.
- The more specific document is the decisive rule for that topic.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- If `specs/<feature-id>` artifacts are missing, continue with available approved sources and mark `[assumption: missing-spec-artifacts]`.
- If seam-specific evidence is missing, mark `[assumption: missing-seam-evidence]` and reduce certainty for that seam.
- Any unresolved assumption that can affect merge safety must be recorded as an open question or `Spec Reopen` candidate.

## Definition Of Done
- Output contains only design-review-domain findings with concrete `file:line` references.
- Every finding has explicit impact and actionable fix.
- Every finding links to a concrete spec reference.
- Cross-domain risks outside design ownership are explicitly routed in `Handoffs`.
- No spec-level conflict remains unmarked.
- All `critical/high` design conflicts are either resolved or escalated through `Spec Reopen`.
- If no findings, output explicitly states `No design findings.` and includes handoffs (or `none`) and residual risks.

## Anti-Patterns
Use these preferred patterns to avoid anti-pattern drift:
- write findings as concrete architecture-impact statements, not abstract cleanliness advice
- keep recommendations aligned with approved spec decisions, or escalate through `Spec Reopen`
- keep comments in design-review domain and route deep domain checks to the corresponding reviewer
- include both `file:line` and `Spec reference` for every finding
- treat architecture alignment as a required gate even when tests are green
- do not file deep-domain findings as design findings unless architecture/maintainability drift is explicit
