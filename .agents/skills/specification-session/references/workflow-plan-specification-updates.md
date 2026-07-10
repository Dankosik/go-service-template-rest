# Workflow Plan Specification Updates

## Behavior Change Thesis
When loaded while writing or repairing `workflow-plan.md` and `workflow-plans/specification.md`, this file makes the model choose a clean master-vs-phase split instead of the likely mistake of turning the phase file into a second `spec.md`, putting implementation order in the master plan, or leaving next-session routing only in chat.

## When To Load
Load this when specification-only work must update workflow routing, phase status, artifact status, blockers, formal clarification `procedural_gate_state`, inline `risk_challenge_outcome`, or next-session handoff.

## Decision Rubric
- `workflow-plan.md` records cross-phase state: current phase, typed artifact/gate state, blockers, session boundary, handoff readiness, and next-session route.
- When `ROUTING-PHASE-CONTROL` is satisfied, `workflow-plans/specification.md` records phase-local details: handoff readiness, input sources, challenge lane, resolution, phase state, completion marker, stop rule, next action, blockers, and parallelizable work. Otherwise compact state remains in `spec.md` or the master.
- Every triggered control artifact must agree that authoring leaves `spec.md` `review_ready`, `draft`, or `blocked`; specification authoring never marks it approved.
- Downstream artifact status may be recorded as missing or expected; do not create downstream artifacts.
- Blockers must name the missing decision and why it matters, not just "needs more info."

## Imitate
Review-ready specification in `workflow-plan.md`:

```text
Current phase: specification
phase_state: complete
session_boundary: reached
handoff_readiness: ready
Next session starts with: specification-review
Phase workflow plans: specification artifact_state=complete, record_validity=current
Artifacts: `spec.md` artifact_expectation=expected, artifact_state=review_ready, record_validity=current; `design/` and `tasks.md` remain artifact_state=absent under their recorded expectations
Lean Risk Challenge when applicable: risk_challenge_outcome=PASS|CONCERNS
Formal clarification when triggered: procedural_gate_state=complete, record_validity=current, findings reconciled
Blockers: none
```

Review-ready specification in triggered `workflow-plans/specification.md`:

```text
phase_state: complete
handoff_readiness: ready
Input sources used: workflow-plan.md, research summary, candidate decisions, existing spec.md
Lean Risk Challenge when applicable: risk_challenge_outcome=PASS|CONCERNS
Formal clarification when triggered: procedural_gate_state=complete; record_validity=current
Completion marker: spec.md review_ready and master routing updated
Stop rule: stop before technical design, planning, tests, or implementation
Next action: begin specification-review in a new session
Parallelizable work: none in this phase
```

Copy the split: master owns cross-phase routing; phase-local owns how the specification checkpoint completed.

## Reject
Second spec:

```text
workflow-plans/specification.md
Decisions: [full product and API decision record]
```

This fails because `spec.md` is the canonical decision artifact.

Chat-only handoff instead of artifact-backed handoff:

```text
I'll mention in the final response that `system-integration-design` is next when separate design depth is triggered.
```

This fails when it replaces triggered workflow updates, because future sessions resume from durable artifacts, not chat memory. A final response may still render a recommended next-session prompt, but only after `spec.md`, the current master when used, and any `ROUTING-PHASE-CONTROL`-triggered phase file record the routing state, start point, and context bundle. Do not write the full ready-to-paste prompt into those files.

## Agent Traps
- Marking `handoff_readiness=ready` while `spec.md` is draft or the clarification gate is blocked.
- Putting implementation task order in `workflow-plan.md`.
- Letting master and phase-local files disagree after resolving a blocker.
- Recording downstream artifacts as "created" or "approved" when they remain missing by design.
