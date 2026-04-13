# Spec Validation And Outcome Updates

## Behavior Change Thesis
When loaded for editing `spec.md` closeout sections, this file makes the model write a proof-shaped `Validation` record and honest `Outcome` instead of vague "tests pass" prose or completion language that outruns the evidence.

## When To Load
Load this after fresh proof was run or a proof gap was discovered, when the next step is to update only the task-local `spec.md` `Validation` and `Outcome` sections.

## Decision Rubric
- `Validation` records the claim, scope, commands actually run, observed result, conclusion, and next action.
- `Outcome` says only what the fresh proof supports: complete, blocked, reopened, or partially verified with explicit limits.
- Failed, skipped, stale, or too-narrow proof produces `not verified` plus a reopen target, not a softened success outcome.
- Do not rewrite `Decisions` or `design/` during validation; record a reopen if closeout exposes a real upstream gap.
- Do not paste chat summaries or old output as if they were current command evidence.

## Imitate

```markdown
## Validation

Claim: Phase 1 is complete for tenant export job API and migration scope.
Scope: approved Phase 1 surfaces in existing `tasks.md`.
Verification Commands:
- `go test ./internal/httpapi/export ./internal/export -count=1`
- `make openapi-check`
- `make migrate-check`
Observed Result: all three commands passed in this session.
Conclusion: verified for Phase 1 scope.
Next Action: close `validation-phase-1` and mark the workflow complete unless later planned phases remain.

## Outcome

Phase 1 closed with fresh proof for API behavior, generated contract drift, and migration validation. No implementation work was performed during closeout.
```

Copy the concrete shape: claim, scope, exact commands, observed result, conclusion, next action, and bounded outcome.

```markdown
## Validation

Claim: task done.
Scope: repository-wide task closeout.
Verification Commands:
- `go test ./... -count=1`
- `make openapi-check`
Observed Result: `go test ./... -count=1` failed in `internal/export`.
Conclusion: not verified.
Next Action: reopen implementation at T003 to address the failing export package test, then return to validation.

## Outcome

Closeout blocked. Fresh proof failed, so the task is reopened to implementation at T003.
```

Copy the failure shape when proof fails: the spec records the failed result and routes reopening without repair work.

## Reject

```markdown
## Validation

Tests looked fine.

## Outcome

Done.
```

Fails because it omits claim, scope, command evidence, observed result, and next action.

```markdown
## Validation

`go test ./...` failed, but only in an unrelated-looking area.

## Outcome

Task complete except for a minor follow-up.
```

Fails because a failed required proof cannot coexist with task completion wording.

```markdown
## Outcome

Implementation completed and spec decisions were adjusted during validation.
```

Fails because validation closeout cannot smuggle in new spec decisions.

## Agent Traps
- Writing a narrative outcome but forgetting the proof table shape.
- Converting "one command failed" into "complete with caveats."
- Updating broader spec sections because validation exposed a missed decision.
