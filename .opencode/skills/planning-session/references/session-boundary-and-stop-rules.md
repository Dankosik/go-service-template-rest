# Session Boundary And Stop Rules

## Behavior Change Thesis
When loaded for planning closeout, this file makes the model stop with a named next session or reopen target instead of starting implementation, editing upstream artifacts, or declaring completion with an incomplete handoff.

## When To Load
Load when closing a planning session or resolving whether the phase is complete, blocked, reopened, or still in progress.

## Decision Rubric
- Mark `Session boundary reached: yes` only after planning artifacts, workflow-control artifacts, pending task-review/readiness handoff, and required adequacy challenge packet or skip rationale agree.
- Set `Ready for next session: yes` when planning is complete enough for `task-review/readiness`, or when an eligible direct-path waiver explicitly removes that gate.
- If planning is blocked, the boundary is not reached for task review; next session starts with the named reopen target.
- The final planning action is a handoff update, not a code, review, validation, rollout, closeout, `spec.md`, or `design/` action.
- If the user asks to keep going into implementation, repeat the recorded handoff and stop unless an eligible upfront direct/lean waiver already exists.

## Imitate
```markdown
Planning phase complete.
Session boundary reached: yes.
Ready for next session: yes.
Next session starts with: task-review/readiness.
Stop rule: do not perform code, test, migration, review, validation, rollout execution, or closeout work in this planning session.
```

Copy this shape: it closes the phase and names the next phase without entering it.

```markdown
Planning phase blocked.
Session boundary reached: no.
Ready for next session: no.
Next session starts with: go-code-ownership-design.
Stop rule: do not create implementation tasks that depend on the missing ownership decision.
```

Copy this shape: a blocked stop names the reopen target and the forbidden shortcut.

```markdown
Planning phase complete with proof obligations.
Session boundary reached: yes.
Ready for next session: yes.
Next session starts with: task-review/readiness.
Stop rule: the next session reviews whether the proof obligations are mapped well enough for implementation.
```

Copy this shape: planning can cross the boundary only because the risk and proof mapping is visible for review.

## Reject
```markdown
Planning complete. Beginning T001 now.
```

Failure: it crosses into implementation after the planning boundary.

```markdown
Session boundary reached: yes.
Implementation readiness: <non-pending verdict>.
Next session starts with: T001.
```

Failure: planning should not assign readiness, and implementation cannot start from a failed gate.

## Agent Traps
- Leaving `Next session starts with` blank because the chat already says what to do.
- Saying "continue implementation" when no implementation session has started.
- Clearing a planning blocker by editing `spec.md` or `design/` in the same session.
- Treating unresolved adequacy challenge findings as a final closeout detail.
