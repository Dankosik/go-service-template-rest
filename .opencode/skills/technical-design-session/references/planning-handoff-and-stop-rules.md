# Planning Handoff And Stop Rules

## Behavior Change Thesis
When loaded for closing a technical-design session, this file makes the model hand off to a later planning session or reopen target instead of drafting `tasks.md`, optional `plan.md`, code, tests, migrations, generated files, or review output once the design feels ready.

## When To Load
Load before claiming technical design is planning-ready, setting `Session boundary reached: yes`, or responding to a user request to keep going into planning or implementation.

## Decision Rubric
- A planning handoff may be ready only when required design artifacts are approved, triggered conditional artifacts are approved or explicitly not expected, and workflow files agree on blockers and next session.
- The handoff names what planning may consume: approved `spec.md`, required design artifacts, triggered conditional artifacts, accepted assumptions, unresolved trade-offs, and reopen conditions.
- The final action is a handoff or blocker update, not `tasks.md`, optional `plan.md`, implementation, tests, migrations, generation, or review.
- If a planning-critical question remains, route to `specification` or keep `technical-design` blocked; do not pass a TODO to planning.
- If the design is small but non-trivial, still stop at the recorded boundary unless an eligible upfront direct/local waiver already exists.

## Imitate
```markdown
Planning handoff: ready.
Planning may consume: approved `spec.md`; approved overview, component map, sequence, and ownership map; approved `design/data-model.md`; approved `design/contracts/`; `rollout.md` not expected.
Accepted assumptions: existing persisted state remains unchanged outside the new job table.
Next session starts with: planning.
Stop rule: do not write `tasks.md`, optional `plan.md`, code, tests, migrations, generated files, or review output in this session.
```

Copy this shape: it gives planning a concrete input list without beginning planning.

```markdown
Planning handoff: blocked.
Reason: sequence depends on an unresolved event durability decision in `spec.md`.
Reopen target: specification.
Next session starts with: specification.
Stop rule: do not draft planning tasks around the unresolved durability choice.
```

Copy this shape: it names the upstream decision and forbids the workaround.

```markdown
Planning handoff: not needed.
Reason: workflow control already records approved design and next session `planning`; no technical-design repair target is present.
Stop rule: do not rework design in this session.
```

Copy this shape: it avoids churn when the phase has already closed.

## Reject
```markdown
Design is ready, so I will now write `tasks.md`.
```

Failure: readiness is a handoff, not permission to cross phases.

```markdown
Planning can decide who owns invoice status.
```

Failure: source-of-truth ownership is a planning-critical design decision.

```markdown
Add TODO placeholders for contract generation and migration order, then mark the handoff ready.
```

Failure: placeholders hide missing triggered artifacts or upstream decisions.

## Agent Traps
- Treating "planning-ready" as "implementation-ready."
- Leaving `Next session starts with` implicit because the handoff paragraph is clear.
- Using `CONCERNS` language at design closeout to pass an unresolved spec contradiction downstream.
- Beginning contract generation because `design/contracts/` was approved.
