---
name: go-concurrency-review
description: "Review Go code changes for concurrency correctness in a spec-first workflow. Use when auditing pull requests or diffs with goroutines, channels, mutexes, worker pools, or shutdown/cancellation behavior to find race/deadlock/leak and bounded-concurrency risks with spec-reopen escalation. Skip when designing specs, implementing code, or performing primary-domain business, API, DB/cache, security, reliability, or broad style review."
---

# Go Concurrency Review

## Purpose
Deliver domain-scoped code review findings for concurrent behavior during Phase 4 review. Success means changed concurrent paths have safe goroutine lifecycle, cancellation/shutdown behavior, synchronized shared state, and bounded concurrency, with spec mismatches escalated before `Gate G4`.

## Scope And Boundaries
In scope:
- review concurrent code paths that use goroutines, channels, mutexes, wait groups, `errgroup`, worker pools, pipelines, fan-out/fan-in
- verify goroutine lifecycle and termination paths are explicit and safe
- verify cancellation and deadline propagation through concurrent operations
- verify channel ownership, close semantics, and blocking behavior
- verify shared mutable state synchronization and race-risk controls
- verify bounded concurrency and backpressure behavior
- verify shutdown behavior can unblock waits/sends/receives
- verify concurrent error propagation is not lost
- produce actionable findings with exact `file:line`, impact, and fix
- escalate spec mismatches through `Spec Reopen`

Out of scope:
- endpoint business meaning and product acceptance semantics as primary domain
- full idiomatic/style review outside concurrency concerns
- primary performance proof and benchmarking ownership
- primary DB/query/cache correctness ownership
- primary reliability policy ownership outside concurrent control-flow defects
- primary security review ownership
- full test-strategy ownership outside concurrency-specific coverage gaps
- editing spec artifacts in Phase 4

## Working Rules
1. Confirm the task is code review and determine changed scope.
2. Map changed code to one or more concurrency axes. If no axis applies, return `No concurrency findings.` with `Residual Risks: none; no concurrency surface detected in changed scope.`.
3. Determine `feature-id` from review context, changed paths, or task metadata. If it cannot be identified, continue with bounded `[assumption]` and reduced certainty.
4. Load review context using this skill's dynamic loading rules.
5. Review only changed and directly impacted concurrent paths first; avoid broad repository scanning.
6. Evaluate eight concurrency axes for changed scope:
   - `Goroutine Lifecycle Safety`
   - `Cancellation And Deadline Semantics`
   - `Channel Ownership And Closure`
   - `Shared State Synchronization`
   - `Bounded Concurrency And Backpressure`
   - `Deadlock And Shutdown Safety`
   - `Concurrency Error Propagation`
   - `Concurrency Verification Readiness`
7. Record only evidence-backed findings with concrete code location and specific obligation reference (prefer `RLY-*`/`TST-*`/decision IDs when present).
8. Classify severity by merge risk (`critical/high/medium/low`).
9. Provide the smallest safe corrective action for each finding.
10. If significant concurrent behavior changed, record verification status (`go test -race` or equivalent evidence) in `Residual Risks` when evidence is missing.
11. If safe resolution requires changing approved spec intent, create `Spec Reopen` in `reviews/<feature-id>/code-review-log.md`.
12. Keep comments strictly in concurrency-review domain and hand off cross-domain risks to the corresponding reviewer role.
13. If no findings exist, state this explicitly and include residual concurrency risks.

## Output Expectations
- Findings-first output ordered by severity.
- Match output language to the user language when practical.
- Use this exact finding format:

```text
[severity] [go-concurrency-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

- After findings, include:
  - `Handoffs`: cross-domain risks and owner skill.
  - `Spec Reopen`: `required` or `not required` with reason.
  - `Residual Risks`: non-blocking concurrency risks or verification gaps.
- Start each `Issue` value with axis context: `Axis: <one of the eight axes>; ...`.
- Keep section order stable: `Findings`, `Handoffs`, `Spec Reopen`, `Residual Risks`.
- Keep all sections present; if a section is empty, write `none` and one short reason.
- If there are no findings, output `No concurrency findings.` and still include `Residual Risks`.

Severity guide:
- `critical`: confirmed risk of deadlock, goroutine leak, data race on critical path, or shutdown hang that blocks safe merge.
- `high`: high-probability race/deadlock/leak or unbounded concurrency in significant path.
- `medium`: localized concurrency risk with bounded blast radius.
- `low`: local robustness/readability improvements that reduce concurrency risk.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading once all eight concurrency review axes can be evaluated with code evidence and approved spec references.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Phase 4`, `Reviewer Focus Matrix`, `Review Findings Format`, and `Gate G4` criteria first
- `docs/llm/go-instructions/70-go-review-checklist.md`
- `docs/llm/go-instructions/20-go-concurrency.md`
- review artifacts:
  - `specs/<feature-id>/90-signoff.md` (if present)
  - `reviews/<feature-id>/code-review-log.md` (if present)

Load by trigger:
- Concurrency behavior tied to reliability contracts:
  - `specs/<feature-id>/55-reliability-and-resilience.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Required test obligations for concurrent paths:
  - `specs/<feature-id>/70-test-plan.md`
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
  - `docs/build-test-and-development-commands.md`
- Architecture-level ownership or lifecycle boundary ambiguity:
  - `specs/<feature-id>/20-architecture.md`
  - `specs/<feature-id>/60-implementation-plan.md`
- Async workflow semantics affecting channel/pipeline design:
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
- Data/cache interactions in concurrent flows:
  - `specs/<feature-id>/40-data-consistency-cache.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/50-caching-strategy.md`
- Security implications caused by concurrent control flow:
  - `docs/llm/security/10-secure-coding.md`

Conflict resolution:
- Approved decisions in `specs/<feature-id>/90-signoff.md` override generic guidance unless `Spec Reopen` is raised.
- The more specific document is the decisive rule for that topic.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- If required spec artifacts are unavailable, mark `[assumption: missing-spec-artifacts]` and reduce certainty.
- Any unresolved assumption affecting merge safety must be surfaced in `Residual Risks` or escalated as `Spec Reopen`.
- If concurrency-verification evidence is missing for significant concurrent changes, mark `[assumption: missing-race-evidence]` and surface it in `Residual Risks`.

## Definition Of Done
- Concurrency review output stays within concurrency-review domain boundaries.
- Every finding is mapped to one explicit concurrency axis.
- Findings are evidence-backed and use exact `file:line` references.
- Every finding has impact, fix path, and spec reference.
- All `critical/high` concurrency findings are either resolved or clearly escalated.
- No spec-level mismatch remains implicit.
- If no findings, output explicitly states `No concurrency findings.` and includes residual risk note.

## Anti-Patterns
Use these preferred patterns to avoid anti-pattern drift:
- prioritize deterministic lifecycle and cancellation guarantees over ad hoc timing assumptions
- prefer explicit ownership and bounded concurrency over implicit growth and hidden blocking
- tie every finding to a concrete failure mode and merge risk
- keep concurrency-domain ownership explicit and hand off deep cross-domain issues
- escalate unresolved spec-impacting risks through `Spec Reopen`
