# Triage And Next-Action Routing

## Behavior Change Thesis
When loaded after strong questions survive filtering, this file makes the model classify severity and choose a concrete resolution route by planning impact instead of overblocking everything or reflexively asking for more research.

## When To Load
Load this when you already have concrete challenge questions but are unsure whether they block planning, reopen one domain, defer, ask the user, or become an accepted risk.

## Decision Rubric
- `blocks_planning`: the answer can change scope, ownership, API contract, data shape, migration order, implementation sequence, or validation proof.
- `blocks_specific_domain`: the candidate path is mostly stable, but one specialist fact is missing, such as cache invalidation behavior, tenant isolation proof, mixed-version compatibility, rollback guardrail, or idempotency evidence.
- `non_blocking`: the issue is real but can be carried as design detail, validation detail, or explicit accepted risk without misleading task breakdown.
- `answer`: use when existing artifacts or repository evidence already contain enough information; do not reopen research as ritual.
- `re-research`: use when a factual, repository, runtime, or specialist claim must be verified; name the lane and the exact fact needed.
- `ask_user`: use only for external product, policy, compliance, launch, or risk-appetite choices that repo evidence cannot decide.
- `defer`: use when the point belongs in downstream design or validation and cannot change implementation order.
- `accept_risk`: use only when the risk is known, bounded, reversible enough for the task, and paired with a proof obligation.

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
- "`blocks_planning`: This feels risky."
  - Fails because severity must be tied to what planning would get wrong.
- "`re-research`: Look into this more."
  - Fails because it does not name the missing fact or specialist lane.
- "`ask_user`: Should Redis fallback be enough?"
  - Fails when the answer depends first on repository load, SLO, or failure-mode evidence.
- "`accept_risk`: This is probably fine for v1."
  - Fails because accepted risk needs blast radius, reversibility, and validation proof.

## Agent Traps
- Classifying by anxiety rather than by whether the answer changes planning.
- Sending every unresolved point to research when the orchestrator can answer from existing artifacts.
- Asking the user for repository facts or engineering evidence.
- Accepting risk without naming the invariant at risk and the proof obligation.
- Rerunning challenge automatically after any research; rerun only if a material decision changed or a major seam reopened.

## Validation Shape
Each final challenge item should make its route auditable: what changes, blocker level, next action, and, for `re-research`, the specialist lane plus the exact fact that would unblock planning.
