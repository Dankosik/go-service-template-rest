# Transition

Use when a result may move or reopen a macro phase, or cross an actor/session
boundary.

Move only when the current owner is `ready`, every triggered decision has a
disposition, required review permits movement, and the next owner can act
without inventing meaning, mechanism, ownership, proof strategy, or authority.
An untriggered phase may be `skipped` without loading or reviewing it. Ordinary
inline movement needs no structured receipt.

At a durable macro-phase boundary return:

```text
status: ready | blocked | skipped
owner: <phase>
result: <artifact path or inline result>
review: <Review Result V1 locator or inline result; none for blocked/skipped>
movement_evidence: <why the next owner may act, or none>
reopen_owner: <owner or none>
next_owner: <owner or none>
```

Across a real actor, session, or macro-phase boundary carry only the current
result, authoritative locators, movement evidence, candidate identity when
relevant, proof boundary, exact blocker/next action, and reopen condition.
Package a next-session prompt through [Prompt
Composition](../../prompt-composition.md). Same-phase repair or context rollover
reports resume state without pretending movement occurred.

For a persisted Implementation ledger, the [Planning Ledger
Contract](../phases/planning/ledger-contract.md#acceptance-transition) owns the
immediate `Accepted` or `Blocked` transition. A fixed inline unit creates no
synthetic ledger transition.

Reopen only the smallest owner invalidated by current evidence and preserve
unaffected decisions and proof. Stop at an explicit named phase boundary,
unavailable required external input, new authority boundary, or required
durable handoff.
