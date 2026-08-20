# Implementation

Close one accepted outcome with the smallest reliable workflow and proof that
matches the completion claim.

## Read When

- Structured or orchestrated work enters Implementation.
- Direct work reaches a non-obvious integration, deployment, independent
  review, or blocked-closeout boundary.

## Ownership

Direct work keeps one root-owned inline outcome. For a persisted ledger, the
Ledger Orchestrator selects ready acceptance units and assigns each to a fresh
Acceptance-Unit Lead; independent units may run concurrently.

The Lead owns one unit end to end: understanding the accepted intent, choosing
the execution strategy, integrating any delegated result, validating the real
path, resolving recoverable problems, and recording one accepted result or
precise blocker. Delegation never transfers unit ownership.

## Execute

Inspect the accepted references, current repository, canonical and generated
owners, and relevant proof. Then choose the simplest reliable route. The Lead
may:

- implement directly when the unit is small, tightly couples reasoning and
  editing, or delegation would cost more context than it saves;
- delegate a bounded implementation, investigation, verification, or review;
- run genuinely independent work concurrently; or
- change the route when integration evidence makes another approach faster or
  safer.

Prefer delegation when a clear boundary saves time, cost, or context. Keep
integration and acceptance with the Lead. Parallel work must not compete over
the same files, mutable resources, or changing assumptions. Record a short
working dependency note only when the relationship is not obvious; no formal
execution map is required.

Trace the observable through its real entry point, callers, siblings, owning
code, tests, and generated/manual boundary. Make the smallest complete change,
preserve unrelated work, and fix a shared cause once at its earliest valid
owner.

For delegated work, use the [Subagent Brief
Template](../../subagent-brief-template.md). The harness adapter carries model,
effort, isolation, native identity, and worktree details as structured controls.
Do not repeat them in prose when the tool already guarantees them. Use a fresh
reviewer rather than an ordinary delegated lane when independence is material.

## Recovery

A delegated agent resolves ordinary in-scope problems and returns either its
result or the facts and boundary it could not cross. The Lead then chooses the
best next action: fix directly, continue with the same agent while its context
is useful, start a fresh agent, change the split, gather evidence, or repair the
smallest invalid upstream source of truth and resume the same unit. An upstream
reopen is an action, not a separate implementation role; preserve unaffected
decisions and revalidate only what changed.

Do not repeat an unchanged failed route. Stop only for unavailable business
meaning or external input, new authority, an irreversible choice, a native
capability failure with no safe fallback, or an exhausted in-scope recovery
path. Preserve any useful candidate and name the reopen owner and condition.

## Review And Proof

Self-review the bounded candidate against every accepted criterion and
triggered risk. Use repository [Validation
Routing](../../validation-routing.md) for the smallest proof that can falsify
the outcome, then apply the shared [Evidence
Contract](../shared/evidence-contract.md) to every readiness or completion
claim.

After the candidate passes Lead review and mapped validation, apply shared
[Review](../shared/review.md). When its [trigger](../shared/review.md#trigger)
applies, use a fresh reviewer that does not repair the candidate. The Lead
decides whether to fix directly, reuse a useful agent context, or change the
route; a material candidate change receives fresh review when the trigger still
applies.

Before deployment or remote proof, apply [Deployment And Remote-Proof
Preflight](../shared/deployment-proof-preflight.md).

## Acceptance And Continuation

Accept only when the fixed unit satisfies its postcondition, important
constraints, mapped proof, and any triggered fresh review. For ledger work,
apply [Acceptance-Unit Closure](../shared/acceptance-unit-closure.md) and record
one canonical `Accepted:` receipt or `Blocked:` record before selecting later
work. A delegated result, worktree candidate, or handoff releases no dependency
by itself.

Across a session or checkout boundary, carry only the unit, authoritative
references, candidate identity, current proof, attributed dirt, and exact
blocker or next action. The selected harness adapter owns native task creation,
handoff, resume, waiting, and recovery.

## Stop Rule

Return the changed outcome, proof actually run, and remaining gap or reopen
owner. Close only when every accepted behavior has a terminal disposition, the
complete bounded change and required synchronization are present, unrelated
work is excluded, and the Evidence Contract permits the completion claim.
