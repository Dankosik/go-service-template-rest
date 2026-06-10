# Task-Review Handoff Check

## Behavior Change Thesis
When loaded for a planning handoff that feels almost ready, this file makes the model self-check the completed `tasks.md` against the approved artifact chain and route gaps before task-review/readiness. It does not assign the final `PASS`, `CONCERNS`, `FAIL`, or `WAIVED` verdict; `docs/spec-first-workflow/phases/task-review-readiness.md` owns that gate.

## When To Load
Load when preparing completed `tasks.md` for the separate task-review/readiness phase.

## Decision Rubric
- Compare `tasks.md` to reviewed `spec.md`, specification-review obligations, compact or split design context, technical-design-review obligations, triggered test or rollout artifacts, named phase-control files, blocker resolution, and proof path.
- Leave `Task ledger review`, `Implementation readiness`, `Ledger-review fan-out`, and `Ledger-review fan-out rationale` as `pending_task_review`.
- If the ledger needs task coverage, ordering, proof, evidence, or handoff repair, keep planning open or blocked.
- If the gap belongs to an earlier phase, route to that phase instead of hiding it in task wording.
- Record the task-review/readiness handoff in `workflow-plan.md`, stop/handoff in `workflow-plans/planning.md`, and a short reference in `tasks.md` when useful.
- Do not turn out-of-scope implications into blockers; record those as explicit concerns, proof obligations, or follow-up notes. In-scope target-state work belongs in the ledger or in a reopened earlier phase.

## Imitate
```markdown
Task ledger review: pending_task_review.
Implementation readiness: pending_task_review.
Next phase: task-review/readiness.
Handoff note: completed draft ledger is ready for coverage, ordering, proof, and handoff review.
Proof path: task-level proof is listed in `tasks.md`.
```

Copy this shape: planning names the review packet without approving it.

```markdown
Task ledger review: pending_task_review.
Implementation readiness: pending_task_review.
Task-review handoff concern: cache invalidation proof depends on first checkpoint integration evidence.
Carrying row: task T003 names the integration test and checkpoint gate.
Next phase: task-review/readiness.
```

Copy this shape: likely concerns are visible for the reviewer, but not accepted by planning.

```markdown
Task ledger review: pending_task_review.
Implementation readiness: pending_task_review.
Reopen target: system-integration-design.
Reason: task order depends on an unsettled backfill source-of-truth decision.
Gate result: planning blocked; do not start task-review or implementation.
```

Copy this shape: upstream blockers stop planning before review.

## Reject
```markdown
Implementation readiness: <non-pending verdict>.
Risk: some validation risk remains.
```

Failure: planning is assigning readiness and the risk has no named carrying row.

```markdown
Implementation readiness: <non-pending verdict>.
Rationale: planning files are probably enough and the change is routine.
```

Failure: planning is assigning a waiver and the rationale is not eligible.

## Agent Traps
- Assigning readiness while `tasks.md` is missing for non-trivial work.
- Assigning readiness from a freshly written `tasks.md` instead of routing to task-review/readiness.
- Downgrading a missing required design artifact from `FAIL` to `CONCERNS`.
- Recording the gate only in chat.
- Letting `CONCERNS` carry unnamed risk that the implementation agent must rediscover.
