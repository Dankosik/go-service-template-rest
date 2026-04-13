# Workflow Plan Completion Vs Reopen

## Behavior Change Thesis
When loaded for updating existing workflow routing after closeout proof, this file makes the model choose explicit phase status plus task or routing state, such as `complete` with `Task state: done` or `blocked` with `Routing state: reopened`, instead of leaving ambiguous "mostly done", `TBD`, or contradictory artifact status.

## When To Load
Load this when the proof result is known and the next task is to update existing `workflow-plan.md` or an existing `workflow-plans/validation-phase-<n>.md` with final closeout routing.

## Decision Rubric
- Completion routing requires all positive closeout claims to have fresh passing proof and all closeout artifacts to agree.
- Reopen routing requires the narrowest honest upstream target, the blocking proof gap, an explicit next-session start point, and a separate `Routing state` or `Task state` line rather than overloading `Phase status`.
- A validation phase file may be updated only if it already exists and the workflow uses it; otherwise record that it is not used or reopen planning if it was required and missing.
- Master `workflow-plan.md` must not say complete while `spec.md`, `tasks.md`, or validation phase notes say proof failed or remains missing.
- Avoid limbo states: no `mostly done`, `maybe`, `TBD`, or silent follow-up.

## Imitate

Completion:

```markdown
Current phase: validation-phase-1
Phase status: complete
spec.md status: Validation and Outcome refreshed from fresh proof in this session
tasks.md status: existing ledger updated for T001-T006 from fresh proof
workflow-plans/validation-phase-1.md status: complete
Blockers: none
Session boundary reached: yes
Ready for next session: no
Next session starts with: N/A
Next session context bundle: no next session; task is done
Task state: done
```

Copy the agreement shape: every relevant artifact has a status and no next session remains.

Reopen:

```markdown
Current phase: validation-phase-1
Phase status: blocked
spec.md status: Validation and Outcome refreshed with failing proof
tasks.md status: T003 remains unchecked because migration validation failed
workflow-plans/validation-phase-1.md status: blocked
Blockers: `make migrate-check` failed in this session
Session boundary reached: yes
Ready for next session: yes
Next session starts with: T003
Next session context bundle: `spec.md` for failed proof scope; `tasks.md` for unchecked T003
Task state: reopened
Routing state: reopen implementation at T003
```

Copy the reopen shape: failed proof, blocked phase status, separate reopen routing, and explicit next session target.

No dedicated validation phase:

```markdown
Validation phase file: not used by approved direct-path waiver
Routing: update existing `workflow-plan.md` and `spec.md` only; do not create `workflow-plans/validation-phase-1.md`.
```

Copy this when an approved direct-path waiver or workflow plan says no validation phase file is used.

## Reject

```markdown
Current phase: mostly done
Ready for next session: maybe
Next session starts with: TBD
Task state: complete enough
```

Fails because the next session and final state are not machine-actionable.

```markdown
Validation failed, but the workflow is done because all code has been written.
```

Fails because code completion does not override failed proof.

```markdown
Missing validation phase file created during closeout; status complete.
```

Fails because closeout cannot create missing workflow process artifacts.

## Agent Traps
- Updating the master workflow only and forgetting the existing validation phase file.
- Recording `Session boundary reached: yes` but leaving `Next session starts with: TBD`.
- Calling the workflow done while existing `tasks.md` still marks a failed proof item unchecked without a reopen route.
