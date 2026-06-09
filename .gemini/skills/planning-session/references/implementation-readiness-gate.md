# Implementation Readiness Gate

## Behavior Change Thesis
When loaded for a handoff that feels almost ready, this file makes the model review the completed `tasks.md` against the approved artifact chain and choose `PASS`, `CONCERNS`, `FAIL`, or `WAIVED` from concrete blockers and proof obligations instead of using optimistic `PASS`, vague `CONCERNS`, or convenience `WAIVED`.

## When To Load
Load when reviewing completed `tasks.md` or assigning/auditing implementation-readiness status.

## Decision Rubric
- First compare `tasks.md` to reviewed `spec.md`, specification-review obligations, compact or split design context, technical-design-review obligations, triggered test or rollout artifacts, named phase-control files, blocker resolution, and proof path. A written ledger is still draft until this review passes.
- `PASS`: the accepted target-state implementation ledger matches that artifact chain and can start without inventing hidden architecture, ownership, contract, sequencing, or rollout decisions.
- `CONCERNS`: implementation may start only with named accepted risks and proof obligations that the implementation ledger can satisfy without re-planning.
- `FAIL`: implementation must not start; name `planning` for ledger repair, `technical design review` for missing or unresolved review gates, `technical design` for missing ownership/sequence/rollout/validation context, or `specification` for scope/behavior/contract contradictions.
- `WAIVED`: use only for tiny, direct-path, or prototype-scoped work with explicit rationale and scope; never use it to bypass normal non-trivial planning.
- Record the task-ledger review and readiness status in `workflow-plan.md`, gate result and stop/handoff in `workflow-plans/planning.md`, and short reference in `tasks.md` when useful.
- Do not turn out-of-scope implications into blockers; record those as explicit concerns, proof obligations, or follow-up notes. In-scope target-state work belongs in the ledger or in a reopened earlier phase.

## Imitate
```markdown
Task ledger review: PASS.
Implementation readiness: PASS.
Gate result: implementation may start with T001 in a later session.
Proof path: task-level proof is listed in `tasks.md`.
```

Copy this shape: PASS is tied to named artifacts and a later-session entry point.

```markdown
Task ledger review: CONCERNS.
Implementation readiness: CONCERNS.
Accepted risk: cache invalidation proof depends on first-phase integration evidence.
Proof obligation: task T003 must add and pass the named integration test before validation.
Gate result: implementation may start in the next session with this obligation visible.
```

Copy this shape: concerns are specific, accepted, and testable in the next phase.

```markdown
Task ledger review: FAIL.
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
- Passing readiness from a freshly written `tasks.md` without checking it against `spec.md`, required design context, and review obligations.
- Downgrading a missing required design artifact from `FAIL` to `CONCERNS`.
- Recording the gate only in chat.
- Letting `CONCERNS` carry unnamed risk that the implementation agent must rediscover.
