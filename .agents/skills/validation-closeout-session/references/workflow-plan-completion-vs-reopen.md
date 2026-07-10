# Named Validation Phase Completion Vs Reopen

## Behavior Change Thesis
When loaded for updating a ledger-named validation phase after closeout proof, this file makes the model choose explicit typed phase, artifact, and handoff state plus task or routing state instead of updating master `workflow-plan.md` by habit or leaving ambiguous "mostly done", `TBD`, or contradictory state.

## When To Load
Load this when the proof result is known and approved `tasks.md` explicitly names an existing `workflow-plans/validation-phase-<n>.md` that must record final closeout routing. Do not load it merely because `workflow-plan.md` exists.

## Decision Rubric
- Completion routing requires all positive closeout claims to have fresh passing proof and all closeout artifacts to agree.
- Reopen routing requires the narrowest honest upstream target, the blocking proof gap, an explicit next-session start point, and separate task/reopen fields rather than overloading `phase_state`.
- A validation phase file may be updated only if it already exists and approved `tasks.md` names it; otherwise record that it is not used in `tasks.md`/`spec.md` closeout or reopen planning if it was required and missing.
- Master `workflow-plan.md` is not a closeout surface after approved `tasks.md`; do not update it to mirror completion or reopen state.
- Avoid limbo states: no `mostly done`, `maybe`, `TBD`, or silent follow-up.

## Imitate

Completion:

```markdown
Current phase: validation-phase-1
phase_state: complete
spec.md artifact_state: complete; Validation and Outcome refreshed from fresh proof in this session
tasks.md artifact_state: complete; existing ledger updated for T001-T006 from fresh proof
workflow-plans/validation-phase-1.md artifact_state: complete
Blockers: none
session_boundary: reached
handoff_readiness: not_ready
Next session starts with: N/A
Next session context bundle: no next session; task is done
tasks.md progress: every required ledger item is complete with fresh proof
```

Copy the agreement shape: every ledger-owned closeout artifact has a status and no next session remains.

Reopen:

```markdown
Current phase: validation-phase-1
phase_state: blocked
spec.md artifact_state: blocked; Validation and Outcome refreshed with failing proof
tasks.md artifact_state: blocked; T003 remains unchecked because migration validation failed
workflow-plans/validation-phase-1.md artifact_state: blocked
Blockers: `make migrate-check` failed in this session
session_boundary: reached
handoff_readiness: ready
Next session starts with: T003
Next session context bundle: `spec.md` for failed proof scope; `tasks.md` for unchecked T003
tasks.md blocker: T003 remains incomplete
reopen_target: implementation at T003
```

Copy the reopen shape: failed proof, blocked phase status, separate reopen routing, and explicit next session target.

No dedicated validation phase for current-session direct work:

```markdown
Validation phase file: not used by the current SHAPE-DIRECT route
Routing: record fresh proof in the current `direct_state_envelope`; do not create or update `workflow-plan.md`, `spec.md`, `tasks.md`, or `workflow-plans/validation-phase-1.md`.
```

For an approved ledger that does not name a validation phase file, update only ledger-owned `tasks.md` progress and required `spec.md` closeout fields; do not update `workflow-plan.md` or create the phase file.

## Reject

```markdown
Current phase: mostly done
handoff_readiness: maybe
Next session starts with: TBD
tasks.md progress: complete enough
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
- Recording `session_boundary=reached` but leaving `Next session starts with: TBD`.
- Calling the workflow done while existing `tasks.md` still marks a failed proof item unchecked without a reopen route.
