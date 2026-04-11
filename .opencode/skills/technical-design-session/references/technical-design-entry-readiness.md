# Technical Design Entry Readiness

## Behavior Change Thesis
When loaded for a technical-design request with uncertain phase or spec status, this file makes the model block, reopen, or narrow the write surface instead of treating user momentum, a detailed draft spec, or an obvious implementation path as permission to start design.

## When To Load
Load before starting or resuming technical design when `spec.md`, current phase, allowed writes, or user-requested phase mixing is unclear.

## Decision Rubric
- Start only when `spec.md` is approved or explicitly planning-stable enough for design, and any required clarification gate status is recorded.
- Confirm master `workflow-plan.md` and `workflow-plans/technical-design.md` agree that the current session owns `technical-design`, or that a reopen target intentionally points back here.
- If workflow control says the next session starts with `planning` and approved design already exists, stop and hand off instead of reworking design by momentum.
- If the user asks for `plan.md`, `tasks.md`, code, tests, migrations, generation, or review in the same request, keep this session to design writes and record the later phase as the next session.
- If `spec.md` contains a planning-critical contradiction, route back to `specification`; do not solve it by inventing design authority.

## Imitate
```markdown
Entry readiness: pass.
Evidence: `spec.md` approved; clarification gate resolved; current phase is `technical-design`.
Allowed writes: `design/`, `workflow-plan.md`, and `workflow-plans/technical-design.md` only.
Next action: build or repair the design bundle, then stop at the planning handoff.
```

Copy this shape: it ties entry permission to artifacts and names the exact write boundary.

```markdown
Entry readiness: blocked.
Reason: `spec.md` still disagrees on durable completion semantics.
Reopen target: specification.
Writes performed: workflow-control blocker only; no design bundle.
```

Copy this shape: it blocks for the missing decision instead of using design to hide a contradiction.

```markdown
Entry readiness: no-op handoff.
Reason: design artifacts are approved and workflow control says the next session starts with `planning`.
Stop rule: do not reopen technical design unless a recorded blocker or stale artifact requires repair.
```

Copy this shape: it avoids redoing completed design work.

## Reject
```markdown
The draft spec is detailed enough, so I will write the design and clean up the contradiction in the ownership map.
```

Failure: it lets design become a substitute for specification approval.

```markdown
The design direction is obvious, so I will produce `plan.md` after the design bundle.
```

Failure: it crosses the technical-design session boundary.

## Agent Traps
- Trusting the user's phrase "we are ready" without checking workflow-control artifacts.
- Treating a missing `workflow-plans/technical-design.md` as harmless chat context instead of a repairable phase-control gap.
- Calling a phase "blocked" in the final message but not recording the blocker in workflow control.
- Reading conditional artifact pressure before confirming that technical design is actually allowed to start.
