# Acceptance Criteria And Proof Obligations

## Behavior Change Thesis
When loaded for vague acceptance or verification wording, this file makes the model state task-specific truths and matching proof commands instead of writing "looks good", "run tests", or optimistic readiness language.

## When To Load
Load this when acceptance criteria, planned verification, manual checks, implementation-readiness `CONCERNS`, or proof obligations feel vague or disconnected from the task ledger.

## Decision Rubric
- Acceptance criteria say what must be true; proof obligations say how the session will know.
- Tie acceptance to specification-review-approved `spec.md` plus required compact or split design surfaces and expected deferrals, not repository-wide quality slogans.
- Match proof scope to the changed surface: targeted package test, generator/drift check, diff read, manual artifact-boundary read, or `rtk git diff --check` as appropriate.
- `CONCERNS` is valid only with named accepted risks and proof obligations; `PASS` is valid only when the accepted target-state ledger has no hidden architecture, ownership, contract, sequencing, or rollout decision left to invent.
- Do not plan proof that requires an unapproved design, rollout, or compatibility decision.
- Out-of-scope implications can become explicit proof obligations or follow-up notes. In-scope target-state cleanup belongs in the ledger or in a reopened earlier phase, not in remembered-later notes.

## Imitate
```markdown
Acceptance Criteria:
- every changed surface named in `design/component-map.md` is either covered by a task or explicitly deferred in `tasks.md`
- `tasks.md` proof commands match the changed surfaces
- readiness is `PASS` or `CONCERNS` with named accepted risks and proof obligations
Planned Verification:
- targeted command for each changed package or artifact surface
- `rtk git diff --check`
- manual read for artifact-boundary drift
Review / Checkpoint:
- stop before implementation if any proof requires an unapproved design, rollout, or compatibility decision
```

Copy the separation between criteria and proof, especially the rule that a proof gap can block implementation instead of being hidden.

## Reject
```markdown
Acceptance: looks good.
Proof: run tests.
Readiness: should be fine.
```

This fails because it gives no task-specific condition, no command scope, and no readiness evidence.

## Validation Shape
- Docs-only or skill-only change: targeted diff/read checks plus `rtk git diff --check`; do not force a Go test unless runtime behavior changed.
- Generated artifact change: generator or drift command first, then targeted tests for consumers.
- Runtime package change: narrow package test first, broader command only when cross-package behavior or repo policy requires it.
- Accepted concern: name the residual risk, the proof owed during implementation or validation, and the condition that would reopen planning.

## Evidence Fields And Scenario Binding

- Bind behavior changes and bug fixes to proof-first or test-first tasking by default. If RED proof adds no value, record `Proof-first waiver: <reason>` on the task or checkpoint.
- When `test-plan.md` exists, reference the exact `TD-*` IDs, proof levels, pass/fail observables, fail-before expectations, quality gates, and reopen targets. Planning consumes those decisions; it does not invent them.
- Use `Command/read`, `Result`, `Key output/ref`, `Changed proof files`, and `Residual blocker/narrower claim` when the implementation session must update evidence. One narrow task may use a single evidence line.
- A task stays unchecked when proof is skipped, unavailable, stale, failing, or narrower than its claim. Record `Blocked:` or the narrower verified result instead.
- Generated and mirrored work needs owner-first change, regeneration/sync, and drift proof. Replacement work needs named negative proof for retired identifiers and retained-surface evidence when old artifacts remain intentionally.
