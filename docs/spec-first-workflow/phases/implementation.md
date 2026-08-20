# Implementation

Use when one accepted implementation unit is ready and authorized. Own that
fixed unit through integration, proof, required review, and acceptance.

Before editing, choose the cheapest ownership shape that preserves one coherent
unit. Apply `go-coder` directly when the Lead already holds the needed context,
the change stays in one owner with bounded proof, and handoff costs more than
the edit. Delegate a fixed `worker-agent` lane when separate context,
specialized work, or independent parallel execution reduces total work;
dispatch independent ready lanes concurrently and keep coupled files,
resources, and assumptions in one lane. The Lead retains decisions,
integration, proof, review, and acceptance.

Select only domain skills exposed by the changed surface. Use [Validation
Routing](../../validation-routing.md) and the [Evidence
Contract](../shared/evidence-contract.md) for claims, and bind one fixed
candidate through shared [Review](../shared/review.md) before acceptance.

Before deployment or costly remote proof, apply [Deployment And Remote-Proof
Preflight](../shared/deployment-proof-preflight.md).

For a persisted ledger, keep the unit fixed until its `Accepted` or `Blocked`
result is recorded through the [Planning Ledger
Contract](planning/ledger-contract.md). A delegated result or handoff is not
acceptance.

Done when the real path satisfies the unit's postcondition, constraints, mapped
proof, and review. Otherwise return the exact blocker or `implementation
complete; verification incomplete`. Reopen only the smallest accepted decision
invalidated by evidence; use [Transition](../shared/transition.md) across a real
session boundary.
