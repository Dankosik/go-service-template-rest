# Artifacts

Use when deciding whether a result must survive the current reasoning pass.

Persist only when another actor or phase must consume the result, work will
resume later, evidence must be refreshed or audited, implementation needs a
stable decision or ledger, or rollout has a real operational sequence.
Otherwise keep the result inline.

Task-local artifacts live under `specs/<task>/`:

| Trigger | Artifact | Owns |
| --- | --- | --- |
| New structured work completes requester interview | [`intent.md`](../interfaces/intent.md) | Requester meaning synthesized after interview |
| Behavior delta must survive | `spec.md` | Outcome, behavior delta, constraints, proof expectations |
| Implementation would choose mechanism or placement | `design/` | Selected system and ownership decisions |
| Several units/dependencies or durable execution state exist | `tasks.md`, `tasks/<ID>-<slug>.md` | Index order/status/results and each task's outcome/boundary/acceptance |
| Evidence must be reused or refreshed | `research/*.md` | Findings, limits, conflicts, decision effect |
| Deployment/migration/backfill has a real sequence | `rollout.md` | Operational gates, recovery, observables |
| Cross-session coordination cannot be recovered from those artifacts | `workflow-plan.md` | Current phase, active artifacts, blockers, next action |

Task artifacts preserve accepted behavior, design, and scope; current workflow
owners govern execution timing.

When an accepted instruction changes scope, completion criteria, or execution
policy, replace the affected active text in its owning artifact and consumed
task packets. Do not leave conflicting requirements active beneath a newer
addendum. Preserve unaffected decisions, effect authority, and valid evidence;
mark superseded records historical and remove them from the active read path.

Keep current outcome, exclusions, remaining work, and next action in their
existing owners. Link to historical evidence when needed instead of copying
its execution instructions into current state. Git retains superseded prose;
required operational and financial evidence retains its existing custody.

Reference stable OpenAPI, code, tests, generated sources, and external contracts
instead of copying them. Split an artifact only when the split creates a real
owner or materially improves review. Stop when the next actor can act, prove
the result, and identify the reopen owner without chat archaeology.

Do not persist a task ledger when Planning finds no executable implementation
unit. Record the no-implementation disposition in the owning phase result.
