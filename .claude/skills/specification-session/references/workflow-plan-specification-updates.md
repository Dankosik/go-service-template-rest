# Workflow Plan Specification Updates

## Behavior Change Thesis
When loaded while writing or repairing `workflow-plan.md` and `workflow-plans/specification.md`, this file makes the model choose a clean master-vs-phase split instead of the likely mistake of turning the phase file into a second `spec.md`, putting implementation order in the master plan, or leaving next-session routing only in chat.

## When To Load
Load this when specification-only work must update workflow routing, phase status, artifact status, blockers, clarification gate status, or next-session handoff.

## Decision Rubric
- `workflow-plan.md` records cross-phase state: current phase, artifact status, blockers, clarification gate status, session boundary, readiness, and next-session route.
- `workflow-plans/specification.md` records phase-local details: readiness, input sources, challenge lane, resolution, phase status, completion marker, stop rule, next action, blockers, and parallelizable work.
- The two files must agree on whether `spec.md` is approved, draft, or blocked.
- Downstream artifact status may be recorded as missing or expected; do not create downstream artifacts.
- Blockers must name the missing decision and why it matters, not just "needs more info."

## Imitate
Approved specification in `workflow-plan.md`:

```text
Current phase: specification
Current phase status: complete
Session boundary reached: yes
Ready for next session: yes
Next session starts with: technical-design
Phase workflow plans: specification complete
Artifacts: spec.md approved; design/ missing; tasks.md missing
Clarification gate: complete and reconciled
Blockers: none
```

Approved specification in `workflow-plans/specification.md`:

```text
Phase status: complete
Readiness outcome: spec-ready
Input sources used: workflow-plan.md, research summary, candidate decisions, existing spec.md
Clarification challenge: complete and reconciled
Completion marker: spec.md approved and master routing updated
Stop rule: stop before technical design, planning, tests, or implementation
Next action: begin technical-design in a new session
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

Chat-only handoff:

```text
I'll mention in the final response that technical design is next.
```

This fails because future sessions resume from workflow artifacts, not chat memory.

## Agent Traps
- Marking `Ready for next session: yes` while `spec.md` is draft or the clarification gate is blocked.
- Putting implementation task order in `workflow-plan.md`.
- Letting master and phase-local files disagree after resolving a blocker.
- Recording downstream artifacts as "created" or "approved" when they remain missing by design.
