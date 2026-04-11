# Failed Proof And Reopen Handling

## Behavior Change Thesis
When loaded for failed, missing, stale, skipped, or too-narrow proof, this file makes the model record a blocker and the narrowest reopen target instead of fixing code, creating missing artifacts, or converting failure into "complete with caveats."

## When To Load
Load this as a challenge rubric when any required proof does not support the current positive closeout claim.

## Decision Rubric
- Name the failed, missing, stale, skipped, or too-narrow proof and why it blocks the claim now.
- Pick the narrowest reopen target: implementation for wrong behavior/tests, review for unresolved review blockers, planning or design/specification for missing acceptance criteria or phase artifacts.
- Update closeout artifacts to `not verified`, `blocked`, or `reopened`; do not balance failure against other green checks.
- Stop after routing the reopen. Do not implement fixes, author tests, create missing process files, or silently continue into the reopened phase.
- If a command was skipped for cost or environment reasons, report that as unverified unless the approved plan allowed an alternate proof path.

## Imitate

```markdown
Claim: task done.
Scope: repository-wide task closeout.
Verification Commands:
- `go test ./... -count=1`
- `make openapi-check`
Observed Result:
- `go test ./... -count=1` failed in `internal/export`.
- `make openapi-check` was not run because the first required proof already blocks closeout and the next session must reopen implementation.
Conclusion: not verified.
Next Action: reopen `implementation-phase-1` to fix the failing export package test, then rerun validation.
Boundary: no code changes in this closeout session.
```

Copy the pattern for a failed command: observed failure, not verified, reopen target, and boundary.

```markdown
Claim: validation-phase-1 complete.
Scope: T001-T004 in existing `tasks.md`.
Observed Result: required `tasks.md` is missing even though `workflow-plan.md` says the non-trivial task uses a ledger.
Conclusion: not verified.
Next Action: reopen `planning` to repair the missing task ledger and phase routing. Do not create `tasks.md` during validation.
```

Copy the upstream reopen target when the proof gap is a missing required planning artifact rather than a code failure.

## Reject

```markdown
One test failed. I fixed the code and reran it during closeout, so the task is complete.
```

Fails because closeout cannot perform implementation repair.

```markdown
The required validation file is missing. I created it and marked it complete.
```

Fails because closeout cannot create required process artifacts.

```markdown
OpenAPI drift failed, but the implementation is otherwise done, so Outcome: complete with minor follow-up.
```

Fails because failed required proof blocks completion.

```markdown
The command was skipped because it is slow. Outcome: ready.
```

Fails because skipped required proof cannot support readiness.

## Agent Traps
- Treating a failure as "unrelated-looking" without routing it through a recorded proof gap.
- Choosing a weaker substitute command after a required command fails or is skipped, then claiming the original obligation passed.
- Continuing into the reopened implementation phase in the same closeout session.
