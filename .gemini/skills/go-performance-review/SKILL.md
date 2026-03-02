---
name: go-performance-review
description: "Review Go code changes for performance risks in a spec-first workflow. Use when auditing diffs or pull requests for hot-path regressions, performance-budget conformance, latency/throughput/allocation/contention impact, and benchmark/profile/trace evidence quality. Skip when designing specifications, implementing features, or performing primary architecture/security/reliability/concurrency/DB-correctness reviews."
---

# Go Performance Review

## Purpose
Deliver domain-scoped, evidence-based code review findings for performance during Phase 4 review. Success means hot-path regressions and performance-budget mismatches are identified before `Gate G4`, with concrete fixes or explicit escalation.

## Scope And Boundaries
In scope:
- review changed code for performance-budget conformance against approved spec artifacts
- review hot-path regression risk in algorithmic cost, per-request work, allocations, and I/O behavior
- review contention and parallelism impact when it manifests as performance degradation
- review measurement evidence quality (benchmarks, profiles, traces) for reproducibility and relevance
- report actionable findings with exact `file:line`, impact, and minimal fix path
- escalate spec-level conflicts through `Spec Reopen`

Out of scope:
- redesigning architecture during code review
- primary ownership of idiomatic/style review, business-invariant review, concurrency-correctness review, DB/cache correctness review, reliability review, or security review
- editing spec artifacts during Phase 4
- blocking PRs on preference-only comments without performance-risk evidence

## Working Rules
1. Confirm the task is code review and identify changed scope (files, packages, execution paths).
2. Determine `feature-id` from review context, changed paths, or task metadata. If it cannot be identified, continue with bounded `[assumption]` and reduced certainty.
3. Load context using this skill's dynamic loading rules.
4. Evaluate performance in this order:
   - `Performance Budget Conformance`
   - `Hot-Path Regression Risk`
   - `Allocation And Memory Pressure`
   - `Contention And Parallelism Cost`
   - `I/O Efficiency Signals`
   - `Evidence Quality`
5. Record only evidence-backed findings and map each finding to a concrete approved obligation (`PERF-*` decision, budget clause, or explicit section in `20/60/70/90`).
6. If high-risk hot-path changes have no required evidence, record this as a finding.
7. Classify severity by merge risk (`critical/high/medium/low`) and provide the smallest safe corrective action.
8. Keep comments strictly in performance-review domain; hand off cross-domain risks to the corresponding reviewer role.
9. If code conflicts with approved performance intent or requires new performance decisions, create `Spec Reopen` in `reviews/<feature-id>/code-review-log.md`.
10. Do not edit spec files during code review.
11. If no findings exist, state this explicitly and include residual performance risks.

## Output Expectations
- Findings-first output ordered by severity.
- Match output language to the user language when practical.
- Use this exact finding format:

```text
[severity] [go-performance-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

- After findings, include:
  - `Handoffs`: cross-domain risks and owner skill.
  - `Spec Reopen`: `required` or `not required` with reason.
  - `Residual Risks`: non-blocking risks, missing evidence, or measurement gaps.
- Keep section order stable: `Findings`, `Handoffs`, `Spec Reopen`, `Residual Risks`.
- Keep all sections present; if a section is empty, write `none` and one short reason.
- If there are no findings, output `No performance findings.` and still include `Residual Risks`.

Severity guide:
- `critical`: proven severe hot-path regression or missing mandatory evidence for high-risk change; merge unsafe without fix/escalation.
- `high`: strong evidence of meaningful p95/p99/throughput/allocation regression risk.
- `medium`: bounded but notable performance weakness with limited impact radius.
- `low`: local optimization or clarity improvement with non-blocking impact.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading once all six review axes can be assessed with code evidence and approved spec references.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Phase 4`, `Reviewer Focus Matrix`, `Review Findings Format`, and `Gate G4` criteria first
- `docs/llm/go-instructions/70-go-review-checklist.md`
- `docs/llm/go-instructions/60-go-performance-and-profiling.md`
- review artifacts:
  - `specs/<feature-id>/20-architecture.md`
  - `specs/<feature-id>/60-implementation-plan.md`
  - `specs/<feature-id>/70-test-plan.md`
  - `specs/<feature-id>/90-signoff.md`
  - `reviews/<feature-id>/code-review-log.md` (if present)

Load by trigger:
- Concurrency-sensitive execution path changes:
  - `docs/llm/go-instructions/20-go-concurrency.md`
- Benchmark/test methodology or command expectations:
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
  - `docs/build-test-and-development-commands.md`
- Reliability/degradation/overload interaction:
  - `specs/<feature-id>/55-reliability-and-resilience.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- DB/cache bottleneck signals:
  - `specs/<feature-id>/40-data-consistency-cache.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/50-caching-strategy.md`
- API-visible latency/payload semantics:
  - `specs/<feature-id>/30-api-contract.md`
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`

Conflict resolution:
- Approved decisions in `specs/<feature-id>/90-signoff.md` override generic guidance unless `Spec Reopen` is raised.
- The more specific document is the decisive rule for that topic.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- If required spec artifacts are unavailable, mark `[assumption: missing-spec-artifacts]` and reduce certainty.
- Any unresolved assumption affecting merge safety must be surfaced in `Residual Risks` or escalated as `Spec Reopen`.

## Definition Of Done
- Review output stays within performance-review domain boundaries.
- Findings are evidence-backed and use exact `file:line` references.
- Every finding has impact, fix path, and spec reference.
- All `critical/high` findings are either resolved or clearly escalated.
- No spec-level mismatch remains implicit.
- If no findings, output explicitly states `No performance findings.` and includes residual risk note.

## Anti-Patterns
Use these preferred patterns to avoid anti-pattern drift:
- tie each finding to concrete performance risk and exact code location
- use measurable evidence (benchmark/profile/trace) before recommending blocking actions
- keep performance-domain ownership explicit and hand off deep cross-domain issues
- prefer the smallest safe correction that restores budget conformance
- explicitly report missing mandatory evidence for high-risk hot-path changes
