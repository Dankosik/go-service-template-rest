# Adequacy Challenge And Stop Boundary

## Behavior Change Thesis
When loaded for symptom "I need to route the adequacy challenge, record the deterministic no-trigger audit, stop at the session boundary, or avoid a phase-control collision," this file makes the model keep the gate read-only and boundary-safe instead of treating short waits as failure, spawning research early, or creating a competing `workflow-plans/workflow-planning.md`.

## When To Load
Load this when the active uncertainty is challenge timing, challenge reconciliation, boundary status, or whether the session should refuse to create workflow-planning artifacts because another phase already owns control.

## Decision Rubric
- Run the formal challenge exactly when any `ADEQUACY-*` condition is true. Otherwise record the local deterministic matrix audit; do not create control files merely to give the challenger input.
- `agent_request=capability_only` has no adequacy or shape effect. `agent_request=substantive` affects shape through `FULL-AGENT-SUBSTANTIVE`, which then activates canonical adequacy evidence.
- The challenger may falsify the recorded route, including unsubstantiated full, but cannot classify, reclassify, edit, or approve it; the orchestrator owns any `TRANS-*` action.
- Reconcile blocking findings by editing workflow-control artifacts, recording accepted risk, or leaving the phase blocked; never let the challenger approve the plan.
- A short wait timeout is not a failed challenge when the result is required. Keep waiting unless it is clearly hung, superseded, canceled, or no longer needed.
- If the task is already in research or a later phase, or an approved phase file such as `workflow-plans/specification.md` already owns the control checkpoint, stop instead of creating `workflow-plans/workflow-planning.md` as a competing source.
- Set `session_boundary=reached` only after the master and every `ROUTING-PHASE-CONTROL`-triggered phase file agree on shape, research mode, artifact expectations, blockers, adequacy state, next session, and stop rule. When phase control is not required, the master alone carries that compact state.

## Imitate

Adequacy lane:

```markdown
Lane: A1
Role: `challenger-agent`
Owned question: Is the current workflow-control packet, including `workflow-plans/workflow-planning.md` only when `ROUTING-PHASE-CONTROL` requires it, sufficient for this task's recorded execution shape and research handoff?
Skill: `workflow-plan-adequacy-challenge`
Timing: after the draft master and every triggered phase-control file exist
Expected output: blocking and non-blocking workflow-control findings only
```

What to copy: the challenger checks handoff sufficiency; it does not design the feature.

Reconciled handoff:

```markdown
Adequacy procedural_gate_state: complete
Blocking findings: none open
Non-blocking findings: reconciled by the orchestrator in the current owning control artifact
session_boundary: reached
handoff_readiness: ready
Next session starts with: research, fan-out mode
Stop rule: do not spawn research lanes in this workflow-planning session
```

What to copy: "ready" is tied to reconciled findings and a hard stop.

Phase-control collision:

```markdown
phase_state: blocked
handoff_readiness: blocked
Reason: the active task already has approved pre-research control in `workflow-plans/specification.md`; creating `workflow-plans/workflow-planning.md` would create a competing source of truth.
Next action: resume from the approved current phase file or reopen workflow planning explicitly in a new session if the contract is wrong.
```

What to copy: refusal is a routing correction, not a failure to use the skill.

## Reject

```markdown
Adequacy procedural_gate_state: blocked after a short wait.
session_boundary: reached.
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
- Recording a shape-based skip instead of the local deterministic matrix audit when no `ADEQUACY-*` condition is true.
- Continuing into research because the lane table is written.
- Forcing this wrapper onto tasks whose current phase is already downstream.
