# Closeout Readiness Examples

## Behavior Change Thesis
When loaded for uncertainty about whether validation closeout may start, this file makes the model choose `proceed`, `skip`, or `reopen` before running proof commands instead of validating by momentum or creating missing process artifacts.

## When To Load
Load this when the session boundary is ambiguous: the user asks for closeout, but the current phase, required artifacts, or proof inputs may not actually be closeout-ready.

## Decision Rubric
- Proceed only when the implemented scope is already in the workspace, routing points to validation or closeout, expected validation artifacts already exist or were explicitly waived, and fresh proof can run without creating new workflow files.
- Skip the wrapper when the work is tiny/direct-path and inline validation is enough; do not manufacture a dedicated closeout session for ceremony.
- Reopen when implementation, review, or reconciliation is still active; required `tasks.md` or phase-control files are missing; or proving the claim would require new code, tests, migrations, or process artifacts.
- Treat stale CI, chat memory, or agent summaries as context only, never as readiness proof.

## Imitate

```markdown
Closeout readiness: proceed.
Claim: task done for the approved Phase 1 scope.
Routing: `workflow-plan.md` says current phase is `validation-phase-1`; the existing validation phase file is present.
Inputs: `spec.md`, `plan.md`, existing `tasks.md`, and `workflow-plans/validation-phase-1.md` list the same proof obligations.
Proof action: run fresh scoped package tests, API drift check, and migration validation now.
Boundary: no code, test, migration, or workflow-file creation in this session.
```

Copy the explicit readiness verdict and the boundary statement before choosing commands.

```markdown
Closeout readiness: not ready.
Claim is broader than the available artifacts: the user asks for task-wide completion, but `workflow-plan.md` still says `implementation-phase-2` is in progress.
Next action: stop validation and route the next session to `implementation-phase-2`; do not run closeout by momentum.
```

Copy the reopen-before-proof shape when the master workflow still points upstream.

## Reject

```markdown
Implementation is probably done. I will run tests and patch anything small that fails so we can still close it.
```

Fails because closeout cannot include repair work.

```markdown
The validation phase file is missing, so I will create `workflow-plans/validation-phase-1.md` and continue closeout.
```

Fails because validation is artifact-consuming and cannot invent missing phase files.

```markdown
Yesterday's CI was green, so the task is closeout-ready without another run.
```

Fails because stale output is not fresh proof and does not establish readiness.

## Agent Traps
- Treating "implementation seems done" as equivalent to validation routing.
- Creating missing `tasks.md` or `workflow-plans/validation-phase-<n>.md` because closeout needs somewhere to write.
- Running tests first and deciding session ownership afterward.
