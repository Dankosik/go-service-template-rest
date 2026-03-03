---
name: go-systematic-debugging
description: "Debug Go service bugs and failing tests with a root-cause-first workflow. Use when handling a bug report, flaky/failing test, build or integration failure, or production incident before proposing fixes. Skip when the task is feature specification, normal greenfield implementation, or domain-scoped code review without an active defect."
---

# Go Systematic Debugging

## Purpose
Produce reproducible, evidence-backed root-cause analysis for defects in this Go service template before any fix is proposed. Success means the issue is reproduced, root cause is identified at source, fix scope is minimal and contract-safe, and verification evidence is explicit.

## Scope And Boundaries
In scope:
- debug failing tests, bugs, regressions, build failures, integration failures, and runtime incidents
- establish deterministic reproduction and isolate failing boundary/layer
- collect minimal diagnostic evidence and map it to concrete hypotheses
- implement and verify the smallest safe fix after root cause is confirmed
- enforce spec-first escalation when fix changes approved contract or architecture intent
- document debugging evidence in a way reviewers can validate

Out of scope:
- feature design/spec authoring as a primary task
- broad refactors while debugging a scoped defect
- changing API/data/security/reliability semantics without spec clarification or spec reopen
- domain-scoped review outputs that belong to `*-review` skills
- shipping a "best guess" fix without reproducible evidence

## Hard Skills
### Systematic Debugging Core Instructions

#### Mission
- Find root cause before code changes.
- Keep fixes minimal, reversible, and aligned with approved spec/contracts.
- Prevent recurrence with explicit regression proof and layered safeguards.

#### Default Posture
- Evidence over intuition: no fix without reproducible failure signal.
- One hypothesis at a time: avoid bundled speculative changes.
- Source-level correction: fix where bad state originates, not only where it crashes.
- Contract safety first: never silently shift API/data/security/reliability behavior.
- Minimal blast radius: avoid opportunistic refactors in defect fix mode.

#### Reproducibility And Baseline Competency
- Capture exact failing command and environment before proposing fixes.
- Use the smallest deterministic reproducer first, then expand scope:
  - targeted: `go test ./... -run '<TestNameRegex>' -count=1`
  - package-level: `go test ./...`
  - repository-level: `make test`
- For flaky behavior, rerun with repetition and consistent seed/inputs where possible.
- Distinguish three states explicitly:
  - deterministic failure;
  - flaky/intermittent failure;
  - cannot reproduce (insufficient evidence).

#### Evidence Collection And Boundary Tracing Competency
- Trace the failing path across service boundaries explicitly:
  - transport (`internal/infra/http` and generated API adapters)
  - use-case/application (`internal/app`)
  - domain contracts (`internal/domain`)
  - infrastructure adapters (`internal/infra/*`)
  - external systems (DB/cache/queue/network).
- Add temporary diagnostics only where they reduce ambiguity.
- Keep diagnostics safe-by-default:
  - no secret/token leakage;
  - no high-cardinality production labels;
  - bounded scope and easy cleanup.
- Prefer deterministic capture:
  - exact input payload/fixture,
  - exact failing stack or wrapped error chain,
  - exact request/dependency boundary where invariant first breaks.

#### Single-Hypothesis Experiment Competency
- Form one concrete hypothesis: `I think <cause> because <evidence>`.
- Validate with a minimal experiment that changes exactly one variable.
- If experiment fails, reject hypothesis and return to evidence-gathering.
- Do not stack fixes from multiple hypotheses in a single iteration.

#### Spec-First Escalation Competency
- If debugging reveals mismatch between code and approved spec intent:
  - during implementation (Phase 3): create `Spec Clarification Request` and pause affected fix path;
  - during review (Phase 4): create `Spec Reopen` record in `reviews/<feature-id>/code-review-log.md`.
- During `Spec Freeze`, do not invent new architecture/API/data/security/reliability behavior in debug mode.
- If fix requires contract-level behavior change, escalate before implementing final fix.

#### Go Runtime And Concurrency Diagnostics Competency
- Validate wrapped error chains using `errors.Is`/`errors.As`; never rely on string matching.
- Preserve cancellation semantics (`context.Canceled`, `context.DeadlineExceeded`) when diagnosing timeout paths.
- For concurrency-sensitive failures:
  - run race evidence (`make test-race` or `go test -race ./...`),
  - verify goroutine completion/cancellation path,
  - check for blocked channel send/receive and shared-state races.
- Avoid introducing goroutines in debug patches unless strictly required by root cause.

#### Flaky Test Stabilization Competency
- Replace sleep-based timing guesses with condition-based waiting.
- Use polling with explicit timeout and diagnostics on timeout.
- Keep test determinism explicit:
  - controlled time/randomness,
  - isolated fixtures,
  - explicit cleanup (`t.Cleanup`).
- Do not close a flake by only increasing timeout without proving race/timing root cause.

#### Defense-In-Depth Remediation Competency
- After fixing source root cause, evaluate guardrails at each relevant layer:
  - boundary validation (decode/input limits/semantic checks),
  - use-case invariant checks,
  - infrastructure safety guards,
  - diagnostic hooks for future triage.
- Add only guardrails justified by the discovered failure mode.
- Keep additional checks cheap and explicit; avoid hidden magic.

#### Verification And Regression Proof Competency
- Require explicit RED/GREEN proof for the reproduced defect:
  - failing reproduction recorded;
  - minimal fix implemented;
  - reproduction now passes.
- Run minimal command set proving no obvious regressions for changed scope:
  - baseline: `make test`
  - concurrency-sensitive: `make test-race`
  - API/runtime contract impact: `make openapi-check`
  - static baseline when relevant: `make lint`, `go vet ./...`
- No completion claim without fresh command evidence.
- Before declaring fix/readiness complete, apply `go-verification-before-completion` for claim-to-proof alignment.

#### Evidence Threshold Competency
- Each debugging conclusion must include:
  - failing symptom and deterministic reproducer,
  - boundary where invariant first failed,
  - accepted/rejected hypotheses,
  - root-cause statement,
  - minimal fix scope,
  - post-fix verification commands and outcomes.
- If root cause remains uncertain, state it explicitly and list next evidence step.

#### Review Blockers For This Skill
- proposing or implementing fixes before reproducible evidence is captured
- multi-fix speculative patches in one iteration
- root cause described only as symptom location
- no failing test/reproducer for a fixable defect
- flaky test "fix" done only by timeout inflation
- contract-level behavior drift introduced without spec escalation
- completion claims without fresh verification command output

## Working Rules
1. Classify defect type and impacted boundary (runtime bug, test flake, integration, build, performance).
2. Capture the exact failing command, inputs, and observable symptom.
3. Reproduce deterministically; if not deterministic, gather data to classify flake pattern.
4. Trace data/control flow backward to the first broken invariant.
5. Write one explicit hypothesis and one minimal experiment.
6. Run experiment and record accept/reject result.
7. Repeat steps 4-6 until root cause is source-level, not symptom-level.
8. Decide whether fix is contract-safe under current spec/gate state.
9. If not contract-safe, escalate (`Spec Clarification Request` or `Spec Reopen`) before final fix.
10. Implement minimal fix and add a focused regression test/reproducer.
11. Run required validation commands for changed scope.
12. Return structured debugging report with evidence and residual risks.

## Output Expectations
- Match output language to the user language when practical.
- Use this section order:

```text
Symptom
Reproduction
Evidence
Hypotheses
Root Cause
Fix Scope
Verification
Escalations
Residual Risks
```

- `Reproduction` must include exact command(s).
- `Verification` must include executed command(s) and pass/fail outcome.
- `Escalations` must explicitly state `none` or list required spec escalation items.
- If root cause is not yet proven, output must end with a concrete next experiment.

## Definition Of Done
- Defect is reproduced (or flake pattern is explicitly characterized with evidence).
- Root cause is identified at source-level with supporting evidence.
- Minimal fix is implemented or clearly blocked by spec escalation.
- Regression proof exists (failing before, passing after).
- Required validation commands for changed scope were executed and reported.
- Residual risks and assumptions are explicit.

## Anti-Patterns
- "quick fix first, investigate later"
- fixing the crash site without tracing origin
- combining several speculative changes in one commit
- relying on log verbosity instead of targeted evidence
- adding permanent debug noise after incident without guardrails/cleanup
- changing spec-level behavior silently under debugging pressure
- declaring success without command evidence

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when reproduction strategy, root-cause path, constraints, and verification commands are unambiguous for this defect.

Always load:
- `docs/build-test-and-development-commands.md`
- `docs/llm/go-instructions/10-go-errors-and-context.md`
- `docs/llm/go-instructions/40-go-testing-and-quality.md`

Load by trigger:
- Concurrency/race/deadlock/leak symptoms:
  - `docs/llm/go-instructions/20-go-concurrency.md`
- API behavior mismatch, boundary validation, idempotency/retry semantics:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- SQL/cache/migration behavior mismatch:
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Reliability/degradation/timeout policy symptoms:
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Observability/debug-surface and telemetry debugging constraints:
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
- Defect appears in active spec-first feature work:
  - `docs/spec-first-workflow.md`
  - `specs/<feature-id>/15-domain-invariants-and-acceptance.md`
  - `specs/<feature-id>/30-api-contract.md`
  - `specs/<feature-id>/40-data-consistency-cache.md`
  - `specs/<feature-id>/55-reliability-and-resilience.md`
  - `specs/<feature-id>/65-coder-detailed-plan.md`
  - `specs/<feature-id>/60-implementation-plan.md`
  - `specs/<feature-id>/70-test-plan.md`
  - `specs/<feature-id>/80-open-questions.md`
  - `specs/<feature-id>/90-signoff.md`

Companion references in this skill folder:
- `references/root-cause-tracing-go.md`
- `references/defense-in-depth-go.md`
- `references/condition-based-waiting-go.md`

Conflict resolution:
- Prefer the most specific artifact for the failing boundary.
- If equal specificity conflicts, preserve approved spec intent and escalate.

Unknowns:
- If critical facts are missing, proceed with bounded `[assumption]` and state how to validate it.
