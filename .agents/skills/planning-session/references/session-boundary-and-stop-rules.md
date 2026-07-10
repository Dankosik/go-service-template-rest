# Session Boundary And Stop Rules

## Behavior Change Thesis
When loaded for planning closeout, this file makes the model stop with a named next session or reopen target instead of starting implementation, editing upstream artifacts, or declaring completion with an incomplete handoff.

## When To Load
Load when closing a planning session or resolving whether the phase is complete, blocked, reopened, or still in progress.

## Decision Rubric
- Set `session_boundary=reached` only after planning artifacts, triggered workflow-control artifacts, the pending task-review/readiness handoff, and either the required adequacy challenge packet or the local deterministic no-trigger audit agree.
- Set `handoff_readiness=ready` when planning is complete enough for a new `task-review/readiness` session. Direct-path work omits untriggered planning rather than waiving a distinct phase after planning has started.
- If planning is blocked, the boundary is not reached for task review; when the named reopen session can start, set `session_boundary=reached` and `handoff_readiness=ready` for that reopen target.
- The final planning action is a handoff update, not a code, review, validation, rollout, closeout, `spec.md`, or `design/` action.
- If the user asks to keep going into implementation, repeat the recorded handoff and stop; `ROUTING-NO-COLLAPSE` does not grant a direct/lean phase-collapse waiver.

## Imitate
```markdown
Planning phase complete.
phase_state: complete.
session_boundary: reached.
handoff_readiness: ready.
Next session starts with: task-review/readiness.
Stop rule: do not perform code, test, migration, review, validation, rollout execution, or closeout work in this planning session.
```

Copy this shape: it closes the phase and names the next phase without entering it.

```markdown
Planning phase blocked.
phase_state: blocked.
session_boundary: reached.
handoff_readiness: ready.
Next session starts with: go-code-ownership-design.
Stop rule: do not create implementation tasks that depend on the missing ownership decision.
```

Copy this shape: a blocked stop names the reopen target and the forbidden shortcut.

```markdown
Planning phase complete with proof obligations.
phase_state: complete.
session_boundary: reached.
handoff_readiness: ready.
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
session_boundary: reached.
Task-review/readiness review_verdict: <non-pending verdict assigned by planning>.
Next session starts with: T001.
```

Failure: planning should not assign readiness, and implementation cannot start from a failed gate.

## Agent Traps
- Leaving `Next session starts with` blank because the chat already says what to do.
- Saying "continue implementation" when no implementation session has started.
- Clearing a planning blocker by editing `spec.md` or `design/` in the same session.
- Treating unresolved adequacy challenge findings as a final closeout detail.
