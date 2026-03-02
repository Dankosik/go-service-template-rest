---
name: go-language-simplifier-review
description: "Review Go code changes for language-level simplification in a spec-first workflow. Use when auditing diffs or pull requests for lower cognitive complexity, clearer naming, simpler control flow, and easier maintenance without changing approved architecture or API contracts. Skip when designing specs, implementing features, or running deep performance/security/concurrency/DB/reliability reviews as the primary focus."
---

# Go Language Simplifier Review

## Purpose
Deliver domain-scoped code review findings that reduce cognitive load and improve code clarity during Phase 4 review. Success means changed code is easier to read, reason about, and safely modify, while approved behavior and spec intent remain intact.

## Scope And Boundaries
In scope:
- review changed Go code for simplification opportunities in structure, naming, and local reasoning
- detect avoidable cognitive complexity (deep nesting, mixed abstraction levels, noisy indirection)
- detect unclear naming and intent masking that raise maintenance risk
- review call-site/signature clarity, package-boundary simplicity, and error/context readability when they increase cognitive load
- recommend local, behavior-preserving refactors that improve readability
- report findings with concrete impact and minimal fix path
- escalate spec mismatches through `Spec Reopen` instead of redesigning in review

Out of scope:
- redesigning approved architecture
- changing API contracts or business behavior during review
- primary-domain idiomatic correctness review, design integrity review, or invariant validation
- deep primary-domain audits for performance, concurrency, DB/cache, reliability, or security
- primary correctness ownership for error contracts, concurrency safety, security controls, or data/reliability guarantees (handoff domains)
- preference-only comments without maintainability impact
- rewriting code only to satisfy subjective style preferences without proven readability or maintenance impact

## Hard Skills
### Language Simplification Review Core Instructions

#### Mission
- Protect merge safety by reducing cognitive complexity that can cause misreads, risky edits, and regression-prone maintenance.
- Keep review output aligned with Phase 4 constraints and `Gate G4` readiness criteria.
- Convert readability risks into minimal, concrete fixes that preserve approved design and behavior.

#### Default Posture
- Review changed and directly impacted code first; avoid repository-wide cleanup suggestions.
- Prefer explicit, local, low-indirection code over clever abstractions that hide intent.
- Treat simplification as behavior-preserving by default; if behavior impact is uncertain, escalate.
- Use objective readability and maintenance signals, not personal style taste.
- Keep strict domain ownership and hand off deep non-simplification analysis to the proper reviewer skill.

#### Spec-First Review Competency
- Enforce `docs/spec-first-workflow.md` Phase 4 constraints:
  - domain-scoped findings only;
  - exact `file:line` references;
  - practical fix path;
  - explicit `Spec Reopen` for spec-intent conflicts.
- Treat open `critical/high` simplification findings as `Gate G4` blockers when they materially threaten safe change.
- Never introduce implicit architecture, contract, or domain changes through simplification advice.

#### Readability-First With Semantic Safety Competency
- Simplification must preserve approved behavior semantics, invariants, and external contract expectations.
- Flag "cleaner-looking" rewrites that hide side effects, reorder failure handling, or blur domain intent.
- Prefer minimal transformations that improve readability while keeping runtime behavior stable.
- If safe simplification requires architecture/contract change, require `Spec Reopen` instead of ad hoc redesign.

#### Control-Flow Simplification Competency
- Enforce clear happy path with guard clauses and early returns.
- Flag unnecessary `else` after `return`, deeply nested condition chains, and branching pyramids.
- Flag functions that mix orchestration, transformation, and IO concerns without clear boundaries.
- Prefer one abstraction level per function where practical.
- Recommend extracting cohesive steps only when extraction improves local reasoning and does not create trivial wrapper noise.

#### Cognitive Complexity Reduction Competency
- Identify paths where reader must keep too much transient state in memory to understand behavior.
- Flag mixed abstraction levels in one block (high-level intent + low-level mechanics interleaving).
- Flag multi-hop helper chains that force excessive file jumping for simple reasoning.
- Flag unnecessary passthrough layers and ceremony that obscure core behavior.
- Prefer direct, explicit code paths when abstractions do not remove meaningful duplication.

#### Naming And Intent Exposure Competency
- Require names that reveal domain intent, not just implementation mechanics.
- Flag ambiguous abbreviations, overloaded terms, and inconsistent vocabulary in one feature area.
- Require boolean names that read as facts/questions (`isReady`, `hasNext`, `enabled`).
- Ensure parameter and return naming makes call-site intent obvious.
- Prefer comments that explain "why" or constraints; reject comments that restate obvious code.

#### API And Call-Site Clarity Competency
- Review function/method signatures for readability at call sites.
- Flag signatures that add cognitive burden via unclear parameter roles or overloaded flag combinations.
- Prefer simple, stable, unsurprising APIs over configurable-but-opaque surfaces.
- For exported symbols in touched scope, require public naming and documentation clarity aligned with Go conventions.
- Treat accidental export growth or unclear exported semantics as readability and maintenance risk.

#### Package And Boundary Simplicity Competency
- Enforce focused package responsibilities and predictable import direction.
- Flag junk-drawer package patterns (`util`, `utils`, `common`, `helpers`, `misc`) that hide intent.
- Preserve explicit wiring in composition root (`cmd/<service>/main.go`) over hidden control flow.
- Prefer minimal exported surface and correct `internal/` usage for private implementation.
- Treat boundary-spanning helper indirection as complexity debt unless justified by repeated, stable use.

#### Error And Context Readability Competency
- Error paths must remain explicit and diagnosable, not hidden behind logs-only side effects.
- Ensure error wrapping/context adds clarity without noisy over-wrapping.
- Require `errors.Is`/`errors.As` style checks where needed; reject string-based error matching.
- Keep cancellation/deadline semantics readable and visible in control flow.
- Flag error-handling shapes that bury happy path or make failure behavior hard to trace.

#### Test Readability And Validation Competency
- For touched tests, prefer readability and diagnostic clarity over heavy helper indirection.
- Flag test structures that hide scenario intent or make failures hard to interpret.
- Suggest table/subtest structure only when it improves readability and maintenance.
- Require validation-command guidance that matches repository workflow when simplification changes behavior-critical paths.

#### Trigger-Driven Cross-Domain Signal Competency
- When concurrency constructs are touched:
  - perform simplification sanity check on lifecycle readability;
  - hand off deep race/deadlock/leak analysis to `go-concurrency-review`.
- When exported/public API surface is touched:
  - enforce naming/call-site clarity and docs baseline;
  - hand off deep compatibility semantics to `go-idiomatic-review` or `go-design-review` as needed.
- When tests or quality gates are touched:
  - enforce test readability and maintainability baseline;
  - hand off full coverage/traceability decisions to `go-qa-review`.
- When simplification intersects with security/performance/reliability/DB risks:
  - keep only readability-owned findings;
  - hand off primary-domain analysis to owner reviewer skill.

#### Evidence Threshold And Severity Calibration Competency
- Every finding must include:
  - exact `file:line`;
  - concrete readability/simplification defect;
  - impact on maintenance safety or change risk;
  - smallest safe fix path;
  - verification command suggestion when applicable.
- Severity is assigned by merge risk from cognitive complexity, not by subjective style preference:
  - `critical/high`: complexity likely to cause misinterpretation or unsafe modification of important behavior;
  - `medium`: meaningful readability debt with bounded short-term risk;
  - `low`: local clarity/consistency improvement.

#### Assumption And Uncertainty Discipline
- If facts are missing, proceed with bounded `[assumption]` and reduced certainty.
- Any unresolved assumption affecting merge safety must be surfaced in `Residual Risks` or escalated via `Spec Reopen`.
- Unknowns must be explicit and testable; avoid vague phrasing.

#### Review Blockers For This Skill
- Control flow complexity that materially obscures critical behavior paths.
- Naming ambiguity in changed critical paths that can cause incorrect maintenance decisions.
- Incidental abstraction/indirection that hides ownership, intent, or side effects.
- Required simplification path conflicts with approved spec intent but no `Spec Reopen` is raised.
- Missing evidence or unclear fix path for `critical/high` readability risks.

## Working Rules
1. Confirm the task is a code review and identify changed scope.
2. Determine `feature-id` from task context, changed paths, or review metadata. If unknown, proceed with bounded `[assumption]` and reduced certainty.
3. Load review context using this skill's dynamic loading rules.
4. Apply `Hard Skills` defaults from this file; any deviation must be explicit in findings or residual risks.
5. Evaluate changed code across five simplification axes:
   - `Control-Flow Simplicity`
   - `Cognitive Complexity`
   - `Naming Clarity`
   - `Intent Exposure`
   - `Local Simplification Opportunities`
6. Apply trigger competencies where present:
   - `API And Call-Site Clarity`
   - `Package And Boundary Simplicity`
   - `Error And Context Readability`
   - `Test Readability And Validation`
7. Apply this simplification order when drafting fixes:
   - make intent obvious on first read;
   - flatten control flow with clear guard clauses and early returns;
   - keep one abstraction level per function;
   - improve names to be domain-revealing;
   - remove low-value wrappers and pass-through indirection.
8. Record only evidence-backed, actionable findings with exact `file:line`.
9. Prefer minimal safe simplification over broad rewrites.
10. If a suggested fix requires spec or architecture change, create `Spec Reopen` in `reviews/<feature-id>/code-review-log.md`.
11. Keep comments strictly in simplification domain and hand off cross-domain primary issues.
12. If no findings exist, state this explicitly and include residual readability risks and verification gaps.

## Output Expectations
- Findings-first output ordered by severity: `critical`, `high`, `medium`, `low`.
- Match output language to user language when practical.
- Use this exact format for each finding:

```text
[severity] [go-language-simplifier-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

- After findings, include:
  - `Handoffs`: cross-domain issues and owner review skill.
  - `Spec Reopen`: `required` or `not required` with reason.
  - `Residual Risks`: non-blocking readability/maintainability risks and assumption notes.
  - `Validation commands`: minimal command set to verify proposed fixes.
- Keep section order stable:
  - `Findings`
  - `Handoffs`
  - `Spec Reopen`
  - `Residual Risks`
  - `Validation commands`
- Keep each section present; if empty, write `none` and one short reason.
- If there are no findings, output `No simplification findings.` and still include `Residual Risks` and `Validation commands`.

Severity guide:
- `critical`: complexity blocks safe modification and creates high risk of mis-changing critical behavior.
- `high`: substantial cognitive load or naming ambiguity with material maintenance risk.
- `medium`: clear readability debt that should be fixed with bounded short-term risk.
- `low`: local simplification that improves consistency and readability.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading once all simplification axes can be evaluated with code evidence and approved references.

Always load:
- `docs/spec-first-workflow.md`:
  - read `Core Principles`, `Phase 4`, `Reviewer Focus Matrix`, `Review Findings Format`, and `Gate G4`
- `docs/llm/go-instructions/70-go-review-checklist.md`
- `docs/llm/go-instructions/30-go-project-layout-and-modules.md`
- `docs/project-structure-and-module-organization.md`
- review artifact if present:
  - `reviews/<feature-id>/code-review-log.md`

Load by trigger:
- Error-flow readability or context propagation complexity:
  - `docs/llm/go-instructions/10-go-errors-and-context.md`
- Test readability and validation expectations:
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
  - `docs/build-test-and-development-commands.md`
- Exported identifiers, public API naming, or call-site contract readability:
  - `docs/llm/go-instructions/50-go-public-api-and-docs.md`
- Concurrency constructs increase cognitive complexity:
  - `docs/llm/go-instructions/20-go-concurrency.md`
- If simplification concern may conflict with approved design:
  - `specs/<feature-id>/20-architecture.md`
  - `specs/<feature-id>/60-implementation-plan.md`
  - `specs/<feature-id>/90-signoff.md`

Conflict resolution:
- Approved decisions in `specs/<feature-id>/90-signoff.md` override generic guidance unless `Spec Reopen` is raised.
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prefer trigger-loaded documents over always-loaded documents.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- If required review artifacts are missing, mark `[assumption: missing-review-artifacts]` and reduce certainty.
- Any unresolved assumption affecting merge safety must be surfaced in `Residual Risks` or escalated as `Spec Reopen`.

## Definition Of Done
- Review output stays within simplification domain boundaries.
- Findings are concrete, evidence-backed, and anchored to exact `file:line`.
- Every finding includes impact, minimal fix path, and relevant spec reference.
- `Validation commands` section is present and scoped to changed risk surface.
- Cross-domain issues are explicitly handed off.
- Spec-level conflicts are not implicit; they are escalated through `Spec Reopen`.
- If no findings, output explicitly states `No simplification findings.` and includes residual-risk notes.

## Anti-Patterns
- style-policing comments without concrete readability or maintenance impact
- architecture redesign proposals disguised as simplification advice
- vague suggestions without exact code location and minimal fix path
- replacing explicit behavior with abstractions that are harder to reason about
- taking ownership of other review domains instead of handoff
- hiding uncertainty instead of explicit `[assumption]` and residual-risk annotation
