# Ownership Boundary And Failure Seams

## Behavior Change Thesis
When loaded for ownership, actor, side-effect, or failure seams, this file makes the model challenge durable responsibility and recovery instead of asking vague ownership/auth questions or giving implementation advice.

## When To Load
Load this when the candidate path touches source-of-truth ownership, actor authority, destructive admin actions, cross-domain side effects, async handoffs, cache/state propagation, or failure semantics that would otherwise be decided during implementation.

## The Move
Name the durable state transition first — created job, deactivated account, cached summary, deleted artifact, emitted side effect — then ask which component or actor owns the transition, recovery, and audit trail after partial success. Attach actor authority to a concrete privileged action (deactivate, reactivate, revoke session, retry export, invalidate cache, clean up artifact); "internal-only" waives none of auditability, reversibility, or actor boundaries, and UUID secrecy is not tenant authorization. For async and side-effect flows, test the gap between "request accepted" and "side effect completed" — a manual DB fix counts as a recovery owner only with trigger, authority, and proof, and "let downstream integrations fail naturally" is a policy to challenge, not a default. For caches, test whether source of truth, key shape, invalidation, and staleness bounds survive failed invalidation or tenant collision. Keep the question only if the answer can change ownership, task split, API contract, rollback design, or validation proof.

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
  - Does not name the state or failure point being owned.
- "Add a worker reconciliation loop."
  - Answers the design instead of challenging whether recovery ownership is missing.
