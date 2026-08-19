# Codex Orchestration

## Read When

Read immediately before creating, waiting on, correcting, handing off, or
cleaning up a known Codex App task. The [Codex adapter](../codex.md) and
installed callable schemas remain authoritative.

Vendor authority: [Worktrees and
Handoff](https://learn.chatgpt.com/docs/environments/git-worktrees), [long-running
work](https://learn.chatgpt.com/docs/long-running-work), and
[skills](https://learn.chatgpt.com/docs/build-skills).

## Launch

- Bind one Ledger Orchestrator only when the installed App can create and
  inspect saved-project tasks and operate Goals. Otherwise use structured
  one-unit execution when it can close the outcome, or record the missing
  carrier.
- Re-read the canonical ledger after every transition. Start one Local Lead for
  a serial unit, or one Worktree Lead for each currently ready member of a
  ledger-proven wave. Create no projectless or cloud fallback.
- Give every create a unique `dispatch_scope` from ledger revision, unit, and
  attempt. When the create schema cannot accept the parent-selected model and
  effort directly, bootstrap with the role and scope only, require exactly
  `READY_FOR_DISPATCH`, then send one technical follow-up with the selected
  supported fields.
- Set `startingState` only when the accepted base requires it. Before a Worker
  writes, verify its actual Worktree identity and exact frozen base.

## Identity, Wait, And Correction

Decode each native response once and copy identity from the installed schema.
Retain `threadId`, `hostId`, and the latest wait cursor; address later waits,
messages, and Handoff by identity, never title. Wait on relevant tasks together
and emit no conclusion from an unchanged timeout. Correct the same task without
model or effort overrides.

A Local Lead creates one acceptance Goal. A Worktree Lead completes its
candidate Goal, returns `HANDOFF_READY`, and creates a separate Local acceptance
Goal only after successful Handoff. The two Goals never overlap.

## Worktree Fan-In

Load [Implementation
Handoff](../../spec-first-workflow/shared/implementation-handoff.md) before this
transition.

Before Handoff, re-read Local HEAD, status, and attributed dirt and compare them
with the recorded precondition. Invoke Handoff once for the same Lead and
candidate, retain its operation identity and revision, and wait through native
status. Move at most one Lead into the Local integration checkout at a time. An
ambiguous outcome preserves the candidate in its original carrier and is never
retried blindly.

## Terminal Task Cleanup

After a child reaches native terminality, verify its canonical result and
candidate safety. Unpin and archive it only when no resume or recovery route
needs its identity. Keep Workers reachable through the unit's final receipt or
blocker so same-Worker correction remains possible.
