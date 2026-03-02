---
name: go-reliability-review
description: "Review Go code changes for reliability and resilience correctness in a spec-first workflow. Use when auditing diffs or pull requests for timeout/deadline propagation, retry budget and jitter policy, backpressure and overload behavior, graceful startup/shutdown, degradation modes, and rollout/rollback safety against approved reliability contracts. Skip when designing specifications, implementing features, or performing primary architecture/security/performance/concurrency/DB-correctness reviews."
---

# Go Reliability Review

## Purpose
Deliver domain-scoped code review findings for reliability and resilience during Phase 4 review. Success means failure-path behavior remains aligned with approved reliability contracts, critical outage risks are surfaced before `Gate G4`, and spec mismatches are escalated explicitly.

## Scope And Boundaries
In scope:
- review changed code against approved reliability contracts in `specs/<feature-id>/55-reliability-and-resilience.md`
- review timeout/deadline propagation and fail-fast behavior in changed critical paths
- review retry eligibility, bounded retry budget, and jitter/backoff behavior
- review overload containment and backpressure controls (bounded queues/concurrency, shedding semantics)
- review startup/readiness/liveness/shutdown reliability behavior
- review degradation and fallback transitions for safety and predictability
- review rollout/rollback reliability safety implications in changed code paths
- review reliability fail-path test traceability against approved `70-test-plan.md`
- produce actionable findings with exact `file:line`, impact, and minimal safe fix
- escalate spec-level conflicts through `Spec Reopen`

Out of scope:
- redesigning architecture during code review without explicit `Spec Reopen`
- editing spec artifacts in Phase 4
- performing primary-domain idiomatic/style, architecture integrity, performance evidence, concurrency mechanics, DB/cache correctness, security, QA strategy, or domain-invariant review
- blocking PRs with preference-only comments without concrete reliability impact

## Working Rules
1. Confirm the task is code review and identify changed reliability-sensitive scope.
2. Determine `feature-id` from review context, changed paths, or task metadata. If it cannot be identified, continue with bounded `[assumption]` and reduced certainty.
3. Load context using this skill's dynamic loading rules.
4. Evaluate changed code in this order:
   - `Timeout And Deadline Conformance`
   - `Retry Budget, Eligibility, And Jitter`
   - `Overload And Backpressure Safety`
   - `Startup/Readiness/Liveness/Shutdown Correctness`
   - `Degradation And Fallback Correctness`
   - `Rollout/Rollback Reliability Safety`
   - `Reliability Test Traceability`
5. Record only evidence-backed findings and map each finding to explicit approved obligations (prefer `REL-*` decisions or explicit clauses in `55/60/70/90`).
6. Classify severity by merge safety impact (`critical/high/medium/low`) and provide the smallest safe corrective action.
7. Keep comments strictly in reliability-review domain; hand off deep cross-domain risks to the corresponding reviewer role.
8. If a safe fix requires changing approved spec intent, create `Spec Reopen` in `reviews/<feature-id>/code-review-log.md`.
9. Do not edit spec files during code review.
10. If no findings exist, state this explicitly and include residual reliability risks.

## Output Expectations
- Findings-first output ordered by severity.
- Match output language to the user language when practical.
- Use this exact finding format:

```text
[severity] [go-reliability-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

- After findings, include:
  - `Handoffs`: cross-domain risks and owner skill.
  - `Spec Reopen`: `required` or `not required` with reason.
  - `Residual Risks`: non-blocking reliability risks, assumptions, or verification gaps.
- Keep section order stable: `Findings`, `Handoffs`, `Spec Reopen`, `Residual Risks`.
- Keep all sections present; if a section is empty, write `none` and one short reason.
- If there are no findings, output `No reliability findings.` and still include `Residual Risks`.

Severity guide:
- `critical`: proven outage/cascading-failure risk, unbounded retry/queue/timeout in critical path, or unsafe shutdown/degradation/rollback behavior that makes merge unsafe.
- `high`: strong evidence of significant reliability contract mismatch likely to impact availability/SLO under expected failure conditions.
- `medium`: bounded but meaningful reliability weakness with limited blast radius.
- `low`: local reliability hardening improvement with non-blocking impact.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading once all reliability review axes are assessable with code evidence and approved spec references.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Phase 4`, `Reviewer Focus Matrix`, `Review Findings Format`, and `Gate G4` criteria first
- `docs/llm/go-instructions/70-go-review-checklist.md`
- `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- review artifacts:
  - `specs/<feature-id>/55-reliability-and-resilience.md`
  - `specs/<feature-id>/60-implementation-plan.md`
  - `specs/<feature-id>/70-test-plan.md`
  - `specs/<feature-id>/90-signoff.md`
  - `reviews/<feature-id>/code-review-log.md` (if present)

Load by trigger:
- Context cancellation, timeout propagation, and error-contract semantics:
  - `docs/llm/go-instructions/10-go-errors-and-context.md`
- Goroutine lifecycle, channels, bounded queues, worker pools, shutdown coordination:
  - `docs/llm/go-instructions/20-go-concurrency.md`
- API-visible reliability semantics (`429/503`, `Retry-After`, idempotency/retry behavior, async `202` patterns):
  - `specs/<feature-id>/30-api-contract.md`
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Distributed/async workflow reliability implications:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- Data/cache consistency implications for retries, fallback, or reconciliation:
  - `specs/<feature-id>/40-data-consistency-cache.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Security and observability interaction for fail-open/fail-closed and reliability signals:
  - `specs/<feature-id>/50-security-observability-devops.md`
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md`
- CI/release gate context when rollout safety is impacted:
  - `docs/llm/delivery/10-ci-quality-gates.md`
- Test evidence and command baseline:
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
  - `docs/build-test-and-development-commands.md`

Conflict resolution:
- Approved decisions in `specs/<feature-id>/90-signoff.md` override generic guidance unless `Spec Reopen` is raised.
- The more specific document is the decisive rule for that topic.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- If required spec artifacts are unavailable, mark `[assumption: missing-spec-artifacts]` and reduce certainty.
- Any unresolved assumption affecting merge safety must be surfaced in `Residual Risks` or escalated as `Spec Reopen`.

## Definition Of Done
- Review output stays within reliability-review domain boundaries.
- Findings are evidence-backed and use exact `file:line` references.
- Every finding has impact, fix path, and spec reference.
- All `critical/high` reliability findings are either resolved or clearly escalated.
- No spec-level mismatch remains implicit.
- If no findings, output explicitly states `No reliability findings.` and includes residual risk note.

## Anti-Patterns
Use these preferred patterns to avoid anti-pattern drift:
- define reliability issues with explicit failure impact, not generic advice
- require bounded timeout/retry/queue behavior in critical paths
- keep reliability-domain ownership explicit and hand off deep cross-domain issues
- prefer the smallest safe correction before broader redesign proposals
- escalate spec-intent conflicts via `Spec Reopen` instead of implicit requirement changes
