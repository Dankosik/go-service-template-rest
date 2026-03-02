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

## Hard Skills
### Performance Review Core Instructions

#### Mission
- Protect merge safety from latency/throughput/allocation/contention regressions on changed hot paths.
- Enforce measurable performance-budget conformance defined in approved spec artifacts.
- Convert performance risk into minimal, testable corrective actions without architecture drift.

#### Default Posture
- Evidence first: no performance judgment without benchmark/profile/trace data or explicit mandatory-evidence gap.
- Review changed and directly impacted execution paths first; avoid broad speculative optimization advice.
- Prefer simpler code unless measured gains justify extra complexity.
- Treat unbounded work growth, unbounded queues, and implicit timeouts/retries as performance defects until disproven.
- Keep ownership strict: deep non-performance domains must be handed off to corresponding reviewer skills.

#### Spec-First Review Competency
- Enforce `docs/spec-first-workflow.md` Phase 4 constraints:
  - domain-scoped findings only;
  - exact `file:line` references;
  - practical fix path;
  - explicit `Spec Reopen` for spec-intent conflict.
- Treat unresolved `critical/high` performance findings as Gate G4 blockers.
- Never modify approved spec intent implicitly through review comments.

#### Performance Budget Conformance Competency
- Validate changed paths against approved `PERF-*` decisions and budget clauses in `20/60/70/90` artifacts.
- Flag missing budget clauses for high-risk hot-path changes as evidence/decision gaps.
- Require explicit checks for p95/p99 latency, throughput, and allocation impact when those dimensions are in scope.
- Treat "optimization by guesswork" as a review anti-pattern.

#### Hot-Path And Work-Amplification Competency
- Detect algorithmic cost regressions (`O(n)` -> `O(n*m)`, nested scans, avoidable sorting/re-serialization).
- Flag redundant per-request work:
  - repeated decode/encode/validation;
  - repeated dependency calls for the same data;
  - payload amplification without contract justification.
- Flag expensive operations moved into critical path without measurable benefit.

#### Benchmark, Profile, And Trace Evidence Competency
- Require benchmark evidence for localized performance claims on changed hot code.
- Require profile evidence (`cpu`, `heap`, `allocs`, `goroutine`, `block`, `mutex`) when bottleneck location is uncertain.
- Require trace evidence when scheduler behavior, blocking, wakeups, or tail-latency spikes are relevant.
- Treat microbenchmarks as insufficient for end-to-end claims unless accompanied by system-level evidence.
- Require reproducibility basics: realistic inputs, clear baseline/current comparison, and stable measurement setup.

#### Allocation And Memory Pressure Competency
- Flag allocation increases in hot loops only when evidence shows they matter.
- Prefer structural fixes (algorithm/data-flow/object lifetime) over micro-level syntax tricks.
- Treat premature pooling/reuse (`sync.Pool`, manual buffer reuse) as risky unless profiling proves benefit.
- Flag memory-pressure patterns that increase GC work or produce avoidable retention.

#### Contention, Concurrency, And Scheduler-Cost Competency
- Flag unbounded concurrency on hot paths (worker pools, fan-out, queue growth) as performance collapse risk.
- Require bounded concurrency and cancellation paths for potentially blocking operations.
- Flag lock contention patterns that likely increase tail latency.
- Keep performance ownership focused on cost/latency impact; hand off race/deadlock/lifecycle correctness depth to `go-concurrency-review`.

#### I/O, DB, And Cache Efficiency Competency
- DB path signals (from SQL-access defaults):
  - flag `N+1`, query-in-loop, deep hot-path `OFFSET`, and round-trip amplification;
  - flag missing per-call deadlines and pool-budget violations;
  - require query observability on critical paths and plan evidence for repeated slow queries.
- Cache path signals:
  - reject cache-driven optimization claims without measured bottleneck evidence;
  - require stampede protection, TTL+jitter policy, and explicit fail-open fallback for read acceleration paths;
  - flag cache designs that can overload origin during degradation.
- API-visible latency shape signals:
  - long/variable operations should use explicit async pattern (`202` + operation resource);
  - list endpoints must keep deterministic bounded pagination to avoid latency cliffs.

#### Reliability And Overload Performance Competency
- Flag missing explicit outbound deadlines/time budgets in performance-sensitive dependency chains.
- Flag retry behavior that can amplify load:
  - retry by default,
  - unbounded retries,
  - retrying non-transient failures.
- Flag missing backpressure/load-shedding controls where queueing risk is introduced.
- Ensure fallback/degradation behavior is explicit and observable when used as a performance safety valve.

#### Trigger-Driven Cross-Domain Signal Competency
- Concurrency-heavy changes:
  - perform performance-oriented contention/scheduling checks;
  - hand off race/deadlock/lifecycle correctness depth to `go-concurrency-review`.
- DB/cache-heavy changes:
  - perform query/round-trip/cache-cost checks;
  - hand off correctness/consistency depth to `go-db-cache-review`.
- Reliability/overload-sensitive changes:
  - evaluate timeout/retry/backpressure impact on latency and throughput;
  - hand off resilience-policy depth to `go-reliability-review`.
- API-shape/payload semantics changes:
  - evaluate latency and work-amplification impact;
  - hand off contract-depth decisions to API/design reviewers as needed.
- Test/measurement harness changes:
  - verify benchmark/profile methodology quality;
  - hand off full test-strategy completeness to `go-qa-review`.

#### Evidence Threshold And Severity Calibration Competency
- Every finding must include:
  - exact `file:line`;
  - measurable risk dimension (`latency`, `throughput`, `allocations`, `contention`, or `I/O`);
  - evidence type (`benchmark`, `profile`, `trace`, or missing-required-evidence);
  - smallest safe fix path;
  - verification command suggestion.
- Severity is merge-risk based, never preference based:
  - `critical`: proven severe regression or missing mandatory evidence on high-risk hot-path change;
  - `high`: strong evidence of meaningful p95/p99/throughput/allocation regression risk;
  - `medium`: notable but bounded performance weakness;
  - `low`: local non-blocking optimization opportunity.

#### Assumption And Uncertainty Discipline
- If key facts are missing, proceed with bounded `[assumption]` and reduced certainty.
- Any unresolved assumption that affects merge safety must appear in `Residual Risks` or escalate to `Spec Reopen`.
- Unknowns must be explicit, testable, and tied to a concrete evidence gap.

#### Review Blockers For This Skill
- High-risk hot-path change without required benchmark/profile/trace evidence.
- Proven or strongly evidenced breach of approved performance budget.
- Unbounded concurrency/queue/retry behavior introduced in hot paths.
- Obvious DB/cache round-trip amplification (`N+1`, stampede-prone cache miss path) in critical execution paths.
- Performance-sensitive dependency calls without explicit deadlines.
- Any correction path that conflicts with approved spec intent but lacks `Spec Reopen`.

## Working Rules
1. Confirm review scope: changed files, impacted packages, and execution paths.
2. Determine `feature-id` from review context, changed paths, or task metadata; if unavailable, proceed with bounded `[assumption]`.
3. Load context using this skill's dynamic-loading policy.
4. Apply `Hard Skills` defaults from this file; any deviation must be explicit in findings or residual risks.
5. Evaluate changed code in this order:
   - `Performance Budget Conformance`
   - `Hot-Path Regression Risk`
   - `Allocation And Memory Pressure`
   - `Contention And Parallelism Cost`
   - `I/O Efficiency Signals`
   - `Evidence Quality`
6. Record only evidence-backed findings and map each to a concrete approved obligation (`PERF-*` decision, budget clause, or explicit section in `20/60/70/90`).
7. If high-risk hot-path changes lack required evidence, record this as a finding.
8. Classify severity by merge risk (`critical/high/medium/low`) and provide the smallest safe correction path.
9. Keep comments strictly in performance ownership; hand off deep cross-domain issues.
10. If code conflicts with approved performance intent or requires new performance decisions, create `Spec Reopen` in `reviews/<feature-id>/code-review-log.md`.
11. Do not edit spec files during code review.
12. If no findings exist, state this explicitly and include residual performance risks.

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
  - `Validation commands`: minimal command set to verify proposed fixes.
- Keep section order stable:
  - `Findings`
  - `Handoffs`
  - `Spec Reopen`
  - `Residual Risks`
  - `Validation commands`
- Keep all sections present; if a section is empty, write `none` and one short reason.
- If there are no findings, output `No performance findings.` and still include `Residual Risks` and `Validation commands`.

Suggested validation command pool:
- `go test ./...`
- `go test -race ./...`
- `go test ./... -run '^$' -bench . -benchmem`
- `go test ./... -run '^$' -bench <BenchmarkName> -benchmem -count=5`
- `go test ./... -run <TargetedTest> -count=1`
- `go tool pprof <profile-file>`
- `go tool trace <trace-file>`
- repo-native wrappers when available:
  - `make test`
  - `make test-race`
  - `make test-cover`
  - `make lint`

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading once all six review axes are assessable with code evidence and approved references.

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
- If specificity is equal, prefer trigger-loaded documents over always-loaded documents.

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
