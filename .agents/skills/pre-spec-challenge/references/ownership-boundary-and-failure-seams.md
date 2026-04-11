# Ownership Boundary And Failure Seams

## Behavior Change Thesis
When loaded for ownership, actor, side-effect, or failure seams, this file makes the model challenge durable responsibility and recovery instead of asking vague ownership/auth questions or giving implementation advice.

## When To Load
Load this when the candidate path touches source-of-truth ownership, actor authority, destructive admin actions, cross-domain side effects, async handoffs, cache/state propagation, or failure semantics that would otherwise be decided during implementation.

## Decision Rubric
- Name the durable state transition first: created job, deactivated account, cached summary, deleted artifact, emitted side effect.
- Ask which component or actor owns the transition, recovery, and audit trail after partial success.
- Challenge actor authority only through a concrete action: deactivate, reactivate, revoke session, retry export, invalidate cache, clean up artifact.
- For async and side-effect flows, test the gap between "request accepted" and "side effect completed."
- For caches, test whether source of truth, key shape, invalidation, and staleness bounds survive failed invalidation or tenant collision.
- Keep the question only if the answer can change ownership, task split, API contract, rollback design, or validation proof.

## Imitate
- "Which component owns the durable state transition if the handler succeeds but the async side effect fails?"
  - Copy the partial-success shape; it forces an owner for recovery.
- "If cache invalidation fails after the DB commit, which source of truth wins and how is stale state bounded?"
  - Copy the source-of-truth framing instead of vague cache concern.
- "Which actor is allowed to reverse deactivation, and does the candidate path preserve auditability if support needs rollback?"
  - Copy the concrete actor/action/audit chain.
- "If an export artifact is written but job status update fails, what may the client observe and who cleans up the orphan?"
  - Copy the split between external object state and API state.

## Reject
- "Who owns this?"
  - Fails because it does not name the state or failure point being owned.
- "Is auth okay?"
  - Fails because actor authority must attach to a specific privileged action.
- "How do failures work?"
  - Fails because it asks for a design essay rather than a planning fork.
- "Add a worker reconciliation loop."
  - Fails because it answers the design instead of challenging whether recovery ownership is missing.

## Agent Traps
- Accepting "internal-only" as a reason to skip auditability, reversibility, or actor boundaries.
- Treating UUID secrecy as tenant authorization.
- Treating "let downstream integrations fail naturally" as a side-effect policy.
- Treating a manual DB fix as a recovery owner without trigger, authority, and proof.
