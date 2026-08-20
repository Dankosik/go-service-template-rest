# Artifacts

Use when deciding whether a result must survive the current reasoning pass.

Persist only when another actor or phase must consume the result, work will
resume later, evidence must be refreshed or audited, implementation needs a
stable decision or ledger, or rollout has a real operational sequence.
Otherwise keep the result inline.

Task-local artifacts live under `specs/<task>/`:

| Trigger | Artifact | Owns |
| --- | --- | --- |
| Behavior delta must survive | `spec.md` | Outcome, behavior delta, constraints, proof expectations |
| Implementation would choose mechanism or placement | `design/` | Selected system and ownership decisions |
| Proof needs a scenario matrix | `test-plan.md` | Proof obligations, observables, levels, gaps |
| Several units/dependencies or durable execution state exist | `tasks.md` | Order, dependencies, proof placement, progress |
| Evidence must be reused or refreshed | `research/*.md` | Findings, limits, conflicts, decision effect |
| Deployment/migration/backfill has a real sequence | `rollout.md` | Operational gates, recovery, observables |
| Cross-session coordination cannot be recovered from those artifacts | `workflow-plan.md` | Current phase, active artifacts, blockers, next action |

Reference stable OpenAPI, code, tests, generated sources, and external contracts
instead of copying them. Split an artifact only when the split creates a real
owner or materially improves review. Stop when the next actor can act, prove
the result, and identify the reopen owner without chat archaeology.
