---
name: go-systematic-debugging
description: "Debug Go service bugs, flaky tests, build failures, and incidents with a reproducible, root-cause-first process and evidence-backed verification."
---

# Go Systematic Debugging

## Purpose
Find and verify the real root cause of defects before proposing or finalizing a fix.

## Scope
- debug failing tests, bugs, regressions, build failures, integration failures, and runtime incidents
- establish deterministic reproduction or characterize flake behavior
- collect minimal but sufficient evidence and convert it into concrete hypotheses
- implement and verify the smallest safe fix once root cause is confirmed
- document debugging evidence so reviewers can validate the conclusion

## Boundaries
Do not:
- treat feature design or broad refactoring as the main task
- change API, data, security, reliability, or architecture semantics silently under defect pressure
- ship a “best guess” fix without reproducible evidence
- combine several speculative fixes into one debugging iteration

## Core Defaults
- Evidence over intuition.
- One hypothesis at a time.
- Fix the source of bad state, not only the crash site.
- Keep fixes minimal, reversible, and aligned with approved behavior.
- Preserve a small blast radius and avoid opportunistic refactors while debugging.

## Expertise

### Reproducibility And Baseline
- Capture the exact failing command, inputs, and environment before proposing fixes.
- Start with the smallest deterministic reproducer, then expand scope only as needed.
- Distinguish explicitly between:
  - deterministic failure
  - flaky or intermittent failure
  - cannot reproduce yet

### Evidence Collection And Boundary Tracing
- Trace the failing path across boundaries:
  - transport
  - application or use-case layer
  - domain layer
  - infrastructure adapters
  - external systems
- Add temporary diagnostics only where they reduce ambiguity.
- Keep diagnostics safe and bounded:
  - no secrets or token leakage
  - no unbounded cardinality
  - easy cleanup
- Prefer deterministic capture:
  - exact input payload or fixture
  - exact failing stack or error chain
  - exact boundary where invariant first breaks

### Single-Hypothesis Experiment
- State one hypothesis clearly: `I think <cause> because <evidence>`.
- Test it with the smallest experiment that changes one variable.
- If the experiment fails, reject the hypothesis and return to evidence gathering.
- Do not stack fixes from multiple hypotheses in one pass.

### Contract And Design Escalation
- If the defect exposes a mismatch between current code and approved design or contract intent, escalate before finalizing a fix that changes behavior materially.
- Do not invent new API, data, security, or reliability behavior in debug mode.
- If the bug can only be fixed safely through a contract or design change, make that dependency explicit.

### Go Runtime And Concurrency Diagnostics
- Use `errors.Is` and `errors.As` for wrapped error chains; do not rely on string matching.
- Preserve `context.Canceled` and `context.DeadlineExceeded` semantics while tracing timeout paths.
- For concurrency-sensitive failures:
  - collect race evidence
  - verify goroutine completion and cancellation paths
  - check blocked channel operations and shared-state access
- Avoid introducing new goroutines in debug patches unless the root cause really requires them.

### Flaky Test Stabilization
- Replace sleep-based timing guesses with condition-based waiting.
- Use polling with explicit timeout and useful timeout diagnostics.
- Keep time, randomness, fixtures, and cleanup controlled and explicit.
- Do not “fix” a flake only by inflating a timeout unless timing itself is the behavior under test.

### Defense-In-Depth Remediation
- After fixing the source root cause, add only the guardrails justified by the actual failure mode.
- Evaluate relevant layers:
  - boundary validation
  - use-case or domain invariant checks
  - infrastructure safety constraints
  - diagnostics useful for future triage
- Do not add unrelated hardening that increases complexity without reducing the defect class risk.

### Verification And Regression Proof
- Require explicit RED/GREEN proof:
  - failing reproduction recorded
  - minimal fix applied
  - reproduction now passes
- Run the smallest command set that honestly supports the claim.
- Include race, contract, lint, build, or migration checks when the changed scope actually requires them.
- Do not claim completion without fresh command evidence.

## Debugging Quality Bar
Each debugging conclusion should make the following explicit:
- failing symptom and reproducer
- boundary where the first invariant failed
- accepted and rejected hypotheses
- root-cause statement
- minimal fix scope
- verification commands and outcomes
- residual risk or next evidence step if still uncertain

## Deliverable Shape
Return debugging work in this order:
- `Symptom`
- `Reproduction`
- `Evidence`
- `Hypotheses`
- `Root Cause`
- `Fix Scope`
- `Verification`
- `Escalations`
- `Residual Risks`

If root cause is not proven yet, end with the next concrete experiment.

## Escalate When
Escalate if:
- a fix is being proposed before reproducible evidence exists
- root cause is described only as the symptom location
- the necessary fix would materially change approved contract or design behavior
- several speculative changes are being bundled together
- a flake is being “fixed” only by timeout inflation
- no fresh regression proof exists for a claimed fix
