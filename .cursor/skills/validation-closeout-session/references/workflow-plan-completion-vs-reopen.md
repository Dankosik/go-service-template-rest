# Named Validation Phase Completion Vs Reopen

## Behavior Change Thesis
When loaded for updating a ledger-named validation phase after closeout proof, this file makes the model choose explicit phase status plus task or routing state, such as `complete` with `Task state: done` or `blocked` with `Routing state: reopened`, instead of updating master `workflow-plan.md` by habit or leaving ambiguous "mostly done", `TBD`, or contradictory artifact status.

## When To Load
Load this when the proof result is known and approved `tasks.md` explicitly names an existing `workflow-plans/validation-phase-<n>.md` that must record final closeout routing. Do not load it merely because `workflow-plan.md` exists.

## Decision Rubric
- Completion routing requires all positive closeout claims to have fresh passing proof and all closeout artifacts to agree.
- Reopen routing requires the narrowest honest upstream target, the blocking proof gap, an explicit next-session start point, and a separate `Routing state` or `Task state` line rather than overloading `Phase status`.
- A validation phase file may be updated only if it already exists and approved `tasks.md` names it; otherwise record that it is not used in `tasks.md`/`spec.md` closeout or reopen planning if it was required and missing.
- Master `workflow-plan.md` is not a closeout surface after approved `tasks.md`; do not update it to mirror completion or reopen state.
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

Copy the agreement shape: every ledger-owned closeout artifact has a status and no next session remains.

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
Routing: update `spec.md` and existing `tasks.md` only; do not update `workflow-plan.md` and do not create `workflow-plans/validation-phase-1.md`.
```

Copy this when an approved direct-path waiver or approved ledger says no validation phase file is used.

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
- Updating the master workflow plan after `tasks.md` approval instead of the ledger-owned closeout artifacts.
- Recording `Session boundary reached: yes` but leaving `Next session starts with: TBD`.
- Calling the workflow done while existing `tasks.md` still marks a failed proof item unchecked without a reopen route.
