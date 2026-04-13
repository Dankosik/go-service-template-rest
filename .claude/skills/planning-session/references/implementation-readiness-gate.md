# Implementation Readiness Gate

## Behavior Change Thesis
When loaded for a handoff that feels almost ready, this file makes the model choose `PASS`, `CONCERNS`, `FAIL`, or `WAIVED` from concrete blockers and proof obligations instead of using optimistic `PASS`, vague `CONCERNS`, or convenience `WAIVED`.

## When To Load
Load when assigning or auditing implementation-readiness status.

## Decision Rubric
- `PASS`: all required spec, design, `tasks.md`, triggered test or rollout artifacts, any required named phase-control files, blocker resolution, and proof path are in place.
- `CONCERNS`: implementation may start only with named accepted risks and proof obligations that the next implementation task can satisfy without re-planning.
- `FAIL`: implementation must not start; name the earlier phase to reopen when a missing artifact, unresolved decision, or blocker could change correctness, ownership, rollout, sequencing, or validation.
- `WAIVED`: use only for tiny, direct-path, or prototype-scoped work with explicit rationale and scope; never use it to bypass normal non-trivial planning.
- Record the readiness status in `workflow-plan.md`, gate result and stop/handoff in `workflow-plans/planning.md`, and short reference in `tasks.md` when useful.

## Imitate
```markdown
Implementation readiness: PASS.
Gate result: implementation may start with T001 in a later session.
Proof path: task-level proof is listed in `tasks.md`.
```

Copy this shape: PASS is tied to named artifacts and a later-session entry point.

```markdown
Implementation readiness: CONCERNS.
Accepted risk: cache invalidation proof depends on first-phase integration evidence.
Proof obligation: task T003 must add and pass the named integration test before validation.
Gate result: implementation may start in the next session with this obligation visible.
```

Copy this shape: concerns are specific, accepted, and testable in the next phase.

```markdown
Implementation readiness: FAIL.
Reopen target: technical-design.
Reason: task order depends on an unsettled backfill source-of-truth decision.
Gate result: implementation must not start.
```

Copy this shape: FAIL routes upstream instead of pretending uncertainty is an implementation task.

## Reject
```markdown
Implementation readiness: CONCERNS.
Risk: some validation risk remains.
```

Failure: it has no named accepted risk and no proof obligation.

```markdown
Implementation readiness: WAIVED.
Rationale: planning files are probably enough and the change is routine.
```

Failure: waiver is not for routine non-trivial work.

## Agent Traps
- Passing readiness while `tasks.md` is missing for non-trivial work.
- Downgrading a missing required design artifact from `FAIL` to `CONCERNS`.
- Recording the gate only in chat.
- Letting `CONCERNS` carry unnamed risk that the implementation agent must rediscover.
