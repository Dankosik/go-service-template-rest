# Planning Session Readiness

## Behavior Change Thesis
When loaded for a planning pass with uncertain inputs, this file makes the model block or reopen upstream instead of planning from `spec.md` alone or inventing missing `design/` context inside `tasks.md`.

## When To Load
Load when required planning inputs are missing, stale, contradictory, or not yet checked before artifact writes.

## Decision Rubric
- Begin this non-trivial planning wrapper only from stable specification-review-approved `spec.md` plus required approved design artifacts. Direct-path work does not enter this wrapper; it has no planning-entry waiver.
- Treat missing or blocking specification review as a planning-entry blocker for non-trivial work.
- Treat an existing durable `workflow-plan.md` and any `ROUTING-PHASE-CONTROL`-triggered `workflow-plans/planning.md` as repairable planning inputs, not as optional chat memory; do not require or create either solely because planning is a distinct session.
- Check triggered conditional design artifacts before task breakdown; if sequencing, validation, rollout, data, contract, ownership, or dependency order depends on one, it must exist or planning blocks.
- If a missing decision would change implementation order, proof shape, ownership, compatibility, or rollout, record a reopen target instead of turning it into a task.
- Do not downgrade missing required inputs to `CONCERNS`; missing required inputs normally make readiness `FAIL` or planning blocked.

## Imitate
```markdown
phase_state: blocked.
handoff_readiness: ready.
Reason: `design/ownership-map.md` is missing, and task ordering depends on source-of-truth ownership.
Reopen target: go-code-ownership-design.
Writes performed: none.
```

Copy this shape: it names the missing input, why it matters to planning, and the upstream phase.

```markdown
Planning inputs confirmed:
- `spec.md`: artifact_expectation=expected, artifact_state=approved, record_validity=current
- core `design/`: artifact_expectation=expected, artifact_state=approved, record_validity=current
- conditional design artifacts: resolved to artifact_expectation=not_expected where no sequencing or rollout trigger exists
- `workflow-plans/planning.md`: artifact_expectation=expected, artifact_state=absent, repairable in this session because ROUTING-PHASE-CONTROL is satisfied
Next action: repair the triggered `workflow-plans/planning.md`, then produce expected `tasks.md`.
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
- Calling a missing `ROUTING-PHASE-CONTROL`-triggered `workflow-plans/planning.md` harmless, or creating it when phase control is not required.
- Forgetting that implementation already having started means missing planning artifacts require a planning reopen, not mid-implementation artifact invention.
