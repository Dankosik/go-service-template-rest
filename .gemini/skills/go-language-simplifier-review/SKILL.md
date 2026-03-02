---
name: go-language-simplifier-review
description: "Review Go code changes for language-level simplification in a spec-first workflow. Use when auditing diffs or pull requests for lower cognitive complexity, clearer naming, simpler control flow, and easier maintenance without changing approved architecture or API contracts. Skip when designing specs, implementing features, or running deep performance/security/concurrency/DB/reliability reviews as the primary focus."
---

# Go Language Simplifier Review

## Purpose
Deliver domain-scoped code review findings that reduce cognitive load and improve code clarity during Phase 4 review. Success means changed code is easier to read, reason about, and safely modify, while remaining aligned with approved spec intent.

## Scope And Boundaries
In scope:
- review changed Go code for simplification opportunities in structure and naming
- detect avoidable cognitive complexity (deep nesting, mixed abstraction levels, noisy indirection)
- detect unclear naming that obscures intent
- recommend local refactors that improve readability without changing behavior
- report findings with concrete impact and minimal fix path
- escalate spec mismatches through `Spec Reopen` instead of redesigning in review

Out of scope:
- redesigning approved architecture
- changing API contracts or business behavior during review
- primary-domain idiomatic correctness review, design integrity review, or invariant validation
- deep primary-domain audits for performance, concurrency, DB/cache, reliability, or security
- preference-only comments without maintainability impact

## Working Rules
1. Confirm the task is a code review and identify changed scope.
2. Determine `feature-id` from task context, changed paths, or review metadata. If unknown, proceed with bounded `[assumption]` and reduced certainty.
3. Load review context with this skill's dynamic loading rules.
4. Evaluate changed code across five simplification axes:
   - `Control-Flow Simplicity`
   - `Cognitive Complexity`
   - `Naming Clarity`
   - `Intent Exposure`
   - `Local Simplification Opportunities`
5. Apply this simplification order when drafting fixes:
   - make intent obvious on first read
   - flatten control flow with clear guard clauses and early returns
   - keep one abstraction level per function
   - improve names to be domain-revealing, not mechanism-only
   - remove low-value wrappers and pass-through indirection
6. Record only evidence-backed, actionable findings with exact `file:line`.
7. Prefer minimal safe simplification over broad rewrites.
8. If a suggested fix requires spec or architecture change, create `Spec Reopen` in `reviews/<feature-id>/code-review-log.md`.
9. Keep comments strictly in simplification domain and hand off cross-domain issues to the appropriate reviewer role.
10. If no findings exist, state this explicitly and include residual readability risks.

## Output Expectations
- Present findings first, ordered by severity: `critical`, `high`, `medium`, `low`.
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
  - `Residual Risks`: non-blocking readability/maintainability risks.
- Keep section order stable:
  - `Findings`
  - `Handoffs`
  - `Spec Reopen`
  - `Residual Risks`
- Keep each section present even when empty:
  - if empty, write `none` and one short reason.
- If there are no findings, output `No simplification findings.` and still include `Residual Risks`.

Severity guide:
- `critical`: complexity blocks safe change and creates high risk of mis-modification of critical behavior.
- `high`: substantial cognitive load or naming ambiguity with material maintenance risk.
- `medium`: clear readability debt that should be fixed but has bounded short-term risk.
- `low`: local simplification that improves consistency and readability.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading once all simplification axes can be evaluated with code evidence and approved spec references.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Phase 4`, `Reviewer Focus Matrix`, `Review Findings Format`, and `Gate G4` first
- `docs/llm/go-instructions/70-go-review-checklist.md`
- `docs/llm/go-instructions/30-go-project-layout-and-modules.md`
- `docs/project-structure-and-module-organization.md`
- review artifact if present:
  - `reviews/<feature-id>/code-review-log.md`

Load by trigger:
- Error-flow readability or context propagation complexity:
  - `docs/llm/go-instructions/10-go-errors-and-context.md`
- Test readability and test-complexity findings:
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
- Exported identifiers or public API naming concerns:
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

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- If required spec artifacts are unavailable, mark `[assumption: missing-spec-artifacts]` and lower certainty.
- Any unresolved assumption affecting merge safety must be surfaced in `Residual Risks` or escalated as `Spec Reopen`.

## Definition Of Done
- Review output stays within simplification domain boundaries.
- Findings are concrete, evidence-backed, and anchored to exact `file:line`.
- Every finding includes impact and minimal fix path.
- Cross-domain issues are explicitly handed off.
- Spec-level conflicts are not implicit; they are escalated through `Spec Reopen`.
- If no findings, output explicitly states `No simplification findings.` and includes residual risk notes.

## Anti-Patterns
Use these preferred patterns to avoid anti-pattern drift:
- tie each finding to concrete readability or maintenance impact
- recommend the smallest change that reduces cognitive load
- keep ownership boundaries clear and route cross-domain issues via handoff
- provide file-anchored fix paths for every nontrivial finding
- preserve approved spec intent and escalate through `Spec Reopen` when needed
