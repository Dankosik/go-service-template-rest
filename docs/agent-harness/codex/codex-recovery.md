# Codex Recovery

## Read When

Read only to reconcile unknown create or Handoff state, terminalize a known
Lead, or resume an upstream reopen. Load [Implementation Obstacle
Recovery](../../spec-first-workflow/phases/implementation-obstacle-recovery.md)
for role authority and the applicable Handoff owner before changing task state.

## Upstream Reopen Return

Keep a blocked Lead, candidate, Goal, and native identities pinned. Route one
fresh Local `UPSTREAM_REOPEN_LEAD` for the smallest owning phase, verify its
review-cleared artifact revisions, then resume the original Lead Goal when a
documented control supports it. If current native evidence proves that Goal
non-resumable, create one replacement Local Lead with the same unit and
candidate plus a new attempt scope. The replacement becomes sole owner only
after validating candidate identity. Unknown state never qualifies.

## Unknown Native State

A pending client identity is not a `threadId`. Reconcile an unknown create
through decoded native lists, documented resolvers, then the narrow App-owned
session receipts by creator and exact `dispatch_scope`. Confirm any candidate
identity through native read or wait before use. Zero or multiple exact matches
record `UNKNOWN_CREATE`; never redispatch automatically.

If a known Lead terminates without a canonical transition, send the compact
[terminalization prompt](../../spec-first-workflow/shared/implementation-handoff.md#known-lead-terminalization)
once to that same task. If the transition remains absent, block dependants; do
not replace the Lead except after the proven non-resumable upstream-reopen case
above. Native state, the canonical ledger, and Git are the complete recovery
set; add no scheduler file or database.

When terminal task creation has no uniquely recoverable task identity, record
`Blocked: UNKNOWN_CREATE for <scope>; unverified: whether one native task
exists; evidence: <no identity, or client identity without one recoverable
thread identity>; next proof owner: native task reconciliation; candidate:
none`. Apply the ledger rollup rule and do not redispatch automatically. A
pending client identity alone is not an unknown outcome.

For an unresolved Handoff, record `Blocked: unknown Handoff outcome for
<scope>; unverified: whether the known Lead moved to Local; evidence: <known
task and available operation/revision or missing response>; next proof owner:
native Handoff reconciliation; candidate: <fixed commit/tree or none>`. Invoke
Handoff again only after native state proves the retry safe.
