# Acceptance Criteria And Proof Obligations

## Behavior Change Thesis
When loaded for vague acceptance or verification wording, this file makes the model state task-specific truths and matching proof commands instead of writing "looks good", "run tests", or optimistic readiness language.

## When To Load
Load this when acceptance criteria, planned verification, manual checks, implementation-readiness `CONCERNS`, or proof obligations feel vague or disconnected from the task ledger.

## Decision Rubric
- Acceptance criteria say what must be true; proof obligations say how the session will know.
- Tie acceptance to approved `spec.md + design/` surfaces and expected deferrals, not repository-wide quality slogans.
- Match proof scope to the changed surface: targeted package test, generator/drift check, diff read, manual artifact-boundary read, or `rtk git diff --check` as appropriate.
- `CONCERNS` is valid only with named accepted risks and proof obligations; `PASS` is valid only when no planning-critical blocker remains.
- Do not plan proof that requires an unapproved design, rollout, or compatibility decision.

## Imitate
```markdown
Acceptance Criteria:
- every changed surface named in `design/component-map.md` is either updated or explicitly deferred in `plan.md`
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
