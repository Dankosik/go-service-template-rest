# Task-Review Handoff Check

## Behavior Change Thesis
When loaded for a planning handoff that feels almost ready, this file makes the model self-check the completed `tasks.md` against the approved artifact chain and route gaps before task-review/readiness. It does not assign the final `PASS`, `CONCERNS`, `FAIL`, or `WAIVED` verdict; `docs/spec-first-workflow/phases/task-review-readiness.md` owns that gate.

## When To Load
Load when preparing completed `tasks.md` for the separate task-review/readiness phase.

## Decision Rubric
- Compare `tasks.md` to reviewed `spec.md`, specification-review obligations, compact or split design context, technical-design-review obligations, triggered test or rollout artifacts, named phase-control files, blocker resolution, and proof path.
- Leave task-review/readiness as `procedural_gate_state=pending`, `review_verdict=pending`, and `record_validity=current`; use `handoff_readiness=ready` only for the recorded task-review or actionable reopen session. Planning does not pre-decide the later lane disposition, verdict, or implementation authorization.
- If the ledger needs task coverage, ordering, proof, evidence, or handoff repair, keep planning open or blocked.
- If the gap belongs to an earlier phase, route to that phase instead of hiding it in task wording.
- Record the task-review/readiness handoff in the existing durable `workflow-plan.md`, in `workflow-plans/planning.md` only when `ROUTING-PHASE-CONTROL` requires it, and as a short reference in `tasks.md` when useful.
- Do not turn out-of-scope implications into blockers; record those as explicit concerns, proof obligations, or follow-up notes. In-scope target-state work belongs in the ledger or in a reopened earlier phase.

## Imitate
```markdown
Task-review/readiness procedural_gate_state: pending.
Task-review/readiness review_verdict: pending.
Task-review/readiness record_validity: current.
handoff_readiness: ready.
Next phase: task-review/readiness.
Handoff note: completed draft ledger is ready for coverage, ordering, proof, and handoff review.
Proof path: task-level proof is listed in `tasks.md`.
```

Copy this shape: planning names the review packet without approving it.

```markdown
Task-review/readiness procedural_gate_state: pending.
Task-review/readiness review_verdict: pending.
Task-review/readiness record_validity: current.
handoff_readiness: ready.
Task-review handoff concern: cache invalidation proof depends on first checkpoint integration evidence.
Carrying row: task T003 names the integration test and checkpoint gate.
Next phase: task-review/readiness.
```

Copy this shape: likely concerns are visible for the reviewer, but not accepted by planning.

```markdown
Task-review/readiness procedural_gate_state: blocked.
Task-review/readiness review_verdict: pending.
Planning phase_state: blocked.
session_boundary: reached.
handoff_readiness: ready.
Reopen target: system-integration-design.
Reason: task order depends on an unsettled backfill source-of-truth decision.
Readiness consequence: do not start task-review or implementation; start only the recorded system-integration-design repair session.
```

Copy this shape: upstream blockers stop planning before review.

## Reject
```markdown
Task-review/readiness review_verdict: <non-pending verdict assigned by planning>.
Risk: some validation risk remains.
```

Failure: planning is assigning readiness and the risk has no named carrying row.

```markdown
Task-review/readiness review_verdict: <non-pending verdict assigned by planning>.
Rationale: planning files are probably enough and the change is routine.
```

Failure: planning is assigning a waiver and the rationale is not eligible.

## Agent Traps
- Assigning readiness while `tasks.md` is missing for non-trivial work.
- Assigning readiness from a freshly written `tasks.md` instead of routing to task-review/readiness.
- Downgrading a missing required design artifact from `FAIL` to `CONCERNS`.
- Recording the gate only in chat.
- Letting `CONCERNS` carry unnamed risk that the implementation agent must rediscover.
