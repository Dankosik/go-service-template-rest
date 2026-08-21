# Transition

Use when a result may move or reopen a macro phase, or cross an actor/session
boundary.

Move only when the current owner is `ready`, every triggered decision has a
disposition, required review permits movement, and the next owner can act
without inventing meaning, mechanism, ownership, proof strategy, or authority.
An untriggered phase may be `skipped` without loading or reviewing it. Ordinary
inline movement needs no receipt; a durable macro-phase boundary returns
[Transition Result V1](../interfaces/transition-result-v1.md).

Carry only the current result, authoritative locators, movement evidence,
candidate identity when relevant, proof boundary, exact blocker/next action,
and reopen condition. Package a next-session prompt through [Prompt
Composition](../../prompt-composition.md). Same-phase repair or context rollover
reports resume state without pretending movement occurred.

For a persisted Implementation ledger, the [Planning Ledger
Contract](../phases/planning/ledger-contract.md#acceptance-transition) owns the
immediate `Accepted` or `Blocked` transition. A fixed inline unit creates no
synthetic ledger transition.

Reopen only the smallest owner invalidated by current evidence and preserve
unaffected decisions and proof. Stop at an explicit phase boundary, unavailable
required external input, new authority boundary, or required durable handoff.
That stop applies to the current phase actor. A durable Orchestrator continues
through an authorized agent-owned handoff by opening the next owner, waiting for
its transition, and resuming the same unit without asking the user to confirm
technical routing.
