# Adequacy Challenge And Stop Boundary

## Behavior Change Thesis
When loaded for symptom "I need to route the adequacy challenge, record a skip, stop at the session boundary, or avoid a phase-control collision," this file makes the model keep the gate read-only and boundary-safe instead of treating short waits as failure, spawning research early, or creating a competing `workflow-plans/workflow-planning.md`.

## When To Load
Load this when the active uncertainty is challenge timing, challenge reconciliation, boundary status, or whether the session should refuse to create workflow-planning artifacts because another phase already owns control.

## Decision Rubric
- Direct path: skip the adequacy challenge only with a tiny/direct-path rationale; do not create workflow-control files just to give the challenger something to inspect.
- Lightweight local: skip only with an explicit waiver that says why the challenge would not reduce risk; otherwise run the read-only challenge after draft workflow artifacts exist.
- Full orchestrated or agent-backed: run one read-only challenger lane using only `workflow-plan-adequacy-challenge` after draft master and phase files exist.
- Reconcile blocking findings by editing workflow-control artifacts, recording accepted risk, or leaving the phase blocked; never let the challenger approve the plan.
- A short wait timeout is not a failed challenge when the result is required. Keep waiting unless it is clearly hung, superseded, canceled, or no longer needed.
- If the task is already in research or a later phase, or an approved phase file such as `workflow-plans/specification.md` already owns the control checkpoint, stop instead of creating `workflow-plans/workflow-planning.md` as a competing source.
- Mark `Session boundary reached: yes` only after master and phase file agree on shape, research mode, artifact expectations, blockers, adequacy status, next session, and stop rule.

## Imitate

Adequacy lane:

```markdown
Lane: A1
Role: `challenger-agent`
Owned question: Are `workflow-plan.md` and `workflow-plans/workflow-planning.md` sufficient for this task's recorded execution shape and research handoff?
Skill: `workflow-plan-adequacy-challenge`
Timing: after draft master and phase file exist
Expected output: blocking and non-blocking workflow-control findings only
```

What to copy: the challenger checks handoff sufficiency; it does not design the feature.

Reconciled handoff:

```markdown
Adequacy challenge status: complete
Blocking findings: none open
Non-blocking findings: recorded as accepted risk in `workflow-plans/workflow-planning.md`
Session boundary reached: yes
Ready for next session: yes
Next session starts with: research, fan-out mode
Stop rule: do not spawn research lanes in this workflow-planning session
```

What to copy: "ready" is tied to reconciled findings and a hard stop.

Phase-control collision:

```markdown
Phase status: blocked
Reason: the active task already has approved pre-research control in `workflow-plans/specification.md`; creating `workflow-plans/workflow-planning.md` would create a competing source of truth.
Next action: resume from the approved current phase file or reopen workflow planning explicitly in a new session if the contract is wrong.
```

What to copy: refusal is a routing correction, not a failure to use the skill.

## Reject

```markdown
Adequacy challenge status: timed out after a short wait, treated as failed.
Session boundary reached: yes.
Next session starts with: research anyway.
```

Failure: required subagent results cannot be abandoned after a short timeout.

```markdown
Adequacy challenger will approve the workflow plan and fix any missing lane rows directly.
```

Failure: the challenger is advisory and read-only; the orchestrator reconciles.

```markdown
Create `workflow-plans/workflow-planning.md` even though `workflow-plans/specification.md` is already approved, so this skill has its expected output.
```

Failure: creates a competing control artifact for the same checkpoint.

## Agent Traps
- Treating adequacy as a domain research or spec-clarification lane.
- Marking the session complete while blocking findings remain open.
- Recording a lightweight-local skip without explaining why challenge overhead is not buying safety.
- Continuing into research because the lane table is written.
- Forcing this wrapper onto tasks whose current phase is already downstream.
