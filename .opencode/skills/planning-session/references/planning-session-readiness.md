# Planning Session Readiness

## Behavior Change Thesis
When loaded for a planning pass with uncertain inputs, this file makes the model block or reopen upstream instead of planning from `spec.md` alone or inventing missing `design/` context inside `tasks.md` or optional `plan.md`.

## When To Load
Load when required planning inputs are missing, stale, contradictory, or not yet checked before artifact writes.

## Decision Rubric
- Begin planning only from stable `spec.md` plus required approved design artifacts, unless an explicit eligible direct-path design-skip rationale already exists.
- Treat `workflow-plan.md` and `workflow-plans/planning.md` as repairable planning inputs, not as optional chat memory.
- Check triggered conditional design artifacts before task breakdown; if sequencing, validation, rollout, data, contract, ownership, or dependency order depends on one, it must exist or planning blocks.
- If a missing decision would change implementation order, proof shape, ownership, compatibility, or rollout, record a reopen target instead of turning it into a task.
- Do not downgrade missing required inputs to `CONCERNS`; missing required inputs normally make readiness `FAIL` or planning blocked.

## Imitate
```markdown
Planning status: blocked.
Reason: `design/ownership-map.md` is missing, and task ordering depends on source-of-truth ownership.
Reopen target: technical-design.
Writes performed: none.
```

Copy this shape: it names the missing input, why it matters to planning, and the upstream phase.

```markdown
Planning inputs confirmed:
- `spec.md`: approved
- core `design/`: approved
- conditional design artifacts: none triggered by sequencing or rollout
- `workflow-plans/planning.md`: missing but repairable in this session
Next action: repair `workflow-plans/planning.md`, then produce expected `tasks.md`.
```

Copy this shape: it distinguishes a repairable planning-control gap from a blocking decision/design gap.

## Reject
```markdown
`spec.md` has enough context, so create `tasks.md` now and add any missing ownership details as task notes.
```

Failure: it makes `tasks.md` a replacement for missing design authority.

```markdown
Planning can proceed with CONCERNS because `design/data-model.md` is missing but the migration tasks look straightforward.
```

Failure: a missing triggered data artifact changes ordering and validation, so the handoff should block or reopen.

## Agent Traps
- Treating "the spec looks detailed" as a substitute for the required design bundle.
- Creating a tidy task ledger that hides unresolved ownership, rollout, or contract decisions.
- Calling a missing `workflow-plans/planning.md` harmless instead of repairing it during planning.
- Forgetting that implementation already having started means missing planning artifacts require a planning reopen, not mid-implementation artifact invention.
