# Independent Implementation Review

Conditional branch of Implementation / Validation / Closeout for independently
falsifying one fixed acceptance unit. The implementation phase retains
correction, acceptance, integration, and completion authority.

## Read When

- Root-local execution or Worker integration has produced a fixed candidate that
  passes bounded root review and mapped validation.
- The shared [Review Independence](review-independence.md)
  trigger applies, or the user explicitly requires implementation review as an
  acceptance condition.

## Inputs

- The authoritative `tasks.md` path and exactly one grouped acceptance-unit ID
  or singleton task ID, or one fixed inline direct outcome.
- Cited accepted sources, the authoritative candidate location, current proof
  receipts, and irreproducible external evidence.

## Independence And Dispatch

Use one fresh, one-shot read-only lane against the fixed candidate. [Agent
Harness](../../agent-harness.md#read-only-lanes) chooses the ordinary or
critical role and harness-native clean-context mechanism. The implementation
actor and implementation Worker are not eligible reviewers, and a lane used for
one unit is not resumed for another task ID or unit.

For ledger review, resolve supplied task IDs against the authoritative ledger.
The dispatch must identify exactly one recorded singleton or grouped acceptance
unit. Otherwise reject the handoff without a phase verdict:

```text
REVIEW_HANDOFF_INVALID
received: <task or unit IDs>
recorded units: <matching singleton or grouped units, or none>
```

The root corrects the boundary and opens a fresh one-shot lane.
An inline direct outcome has no ledger IDs; its fixed accepted outcome and
bounded candidate are the complete review boundary.

The reviewer derives its result from the current contract, candidate,
production path, dependencies, retained scope, and claim-scoped proof rather
than an implementation summary. It may inspect wider candidate context and run
safe non-mutating checks, but returns a verdict only for the resolved unit. It
reuses existing proof receipts and runs only the missing or adversarial
falsifier required by its question.

## Verdict And Disposition

The lane answers one question: may the root accept this fixed unit? It returns
exactly one verdict:

- `PASS`: every accepted task postcondition and constraint is present on the
  real path, the retained delta is in scope, and current proof satisfies the
  implementation phase's [Stop Rule](../phases/implementation-validation-closeout.md#stop-rule).
- `FAIL`: a candidate-caused regression, accepted criterion violation, missing
  required surface, or remediable proof gap prevents acceptance. Return
  anchored findings and the smallest repair or reopen owner.
- `BLOCKED`: implementation may be complete, but required external or
  environmental proof is unavailable. Name the unverified claim, narrower
  evidence, and next proof owner.

The reviewer keeps the candidate fixed and edits or repairs nothing. The root
routes `FAIL` to local repair or the same Worker correction loop and records
`BLOCKED` as `implementation complete; verification incomplete`. A material
candidate or proof-precondition change invalidates the verdict; when review
remains triggered, the root opens a fresh lane only for the affected boundary.

## Completion Criterion

`PASS` returns the unit to root acceptance. The root then applies
[Acceptance-Unit Closure](../phases/implementation-validation-closeout.md#acceptance-unit-closure)
and its persisted transition through [Artifact Model](artifact-model.md#minimal-status).
When the unit contains the final unchecked task, the root also checks the
ledger's global `Completion` condition and every required task disposition.

`FAIL` remains in the phase-owned correction or reopen loop. `BLOCKED` leaves
the candidate unaccepted and closes only with the exact unverified claim,
narrower evidence, and next proof owner. An inline direct outcome has no ledger
transition and closes only under the implementation phase's Stop Rule.
