# Triage And Next-Action Routing

## Behavior Change Thesis
When loaded after strong questions survive filtering, this file makes the model classify severity and choose a concrete resolution route by planning impact instead of overblocking everything or reflexively asking for more research.

## When To Load
Load this when you already have concrete challenge questions but are unsure whether they block planning, reopen one domain, defer, ask the user, or become an accepted risk.

## The Move
Classify by whether the answer changes planning, not by anxiety:

- `blocks_planning`: the answer can change scope, ownership, API contract, data shape, migration order, implementation sequence, or validation proof.
- `blocks_specific_domain`: the candidate path is stable except one missing specialist fact — cache invalidation behavior, tenant isolation proof, mixed-version compatibility, rollback guardrail, idempotency evidence.
- `non_blocking`: real, but carried as design detail, validation detail, or explicit accepted risk without misleading task breakdown.

Then route: `answer` when existing artifacts or repository evidence already decide it — the orchestrator answers before reopening research; `re-research` when a factual claim must be verified, naming the lane and the exact fact; `ask_user` only for product, policy, compliance, launch, or risk-appetite choices repo evidence cannot decide — repository facts and engineering evidence never reach the user; `defer` when the point belongs downstream and cannot change implementation order; `accept_risk` only when the risk is known, bounded, reversible enough for the task, and paired with the invariant at risk and a proof obligation. Rerun challenge only when a material decision changed or a major seam reopened.

## Imitate
- "`blocks_planning` + `re-research`: If old clients may still send the previous payload, the API contract and task ordering can change; reopen API/delivery evidence for mixed-version behavior."
  - Copy the coupling of severity to a changed plan and a named evidence lane.
- "`blocks_specific_domain` + `re-research`: Only cache invalidation evidence is missing; reopen data/cache research, not the whole spec."
  - Copy the limited reopen scope.
- "`non_blocking` + `defer`: The log field name is unsettled, but the observability obligation and owner are clear; carry the exact field name into technical design."
  - Copy the refusal to block planning on polish.
- "`non_blocking` + `accept_risk`: Proceed without a canary only if the blast radius is one internal tenant, rollback is named, and validation checks fallback behavior before broad release."
  - Copy the requirement to state bounds and proof.

## Reject
- "`re-research`: Look into this more."
  - Names neither the missing fact nor the specialist lane.
- "`accept_risk`: This is probably fine for v1."
  - Accepted risk needs blast radius, reversibility, and validation proof.

## Validation Shape
Each final challenge item makes its route auditable: what changes, blocker level, next action, and, for `re-research`, the specialist lane plus the exact fact that would unblock planning.
