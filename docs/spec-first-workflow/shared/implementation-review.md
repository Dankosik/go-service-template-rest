# Independent Implementation Review

Semantic contract for independently falsifying one fixed implementation
acceptance unit. The implementation phase retains correction, acceptance,
integration, and completion authority.

## Read When

- A fixed candidate has passed acceptance-owner review and mapped validation,
  and [Review Independence](review-independence.md) triggers review.
- The user explicitly requires independent implementation review.

## Fixed Boundary

Bind exactly one recorded singleton or grouped acceptance unit from the
authoritative `tasks.md`, or one fixed inline direct outcome. Include accepted
sources, immutable candidate identity, current proof receipts, and
irreproducible external evidence.

Resolve supplied ledger IDs before review. An unresolved or multi-unit boundary
returns `REVIEW_HANDOFF_INVALID` through [Acceptance Review Result
V1](acceptance-review-result-v1.md) without a phase verdict. The acceptance
owner repairs the handoff and opens a fresh review.

## Falsification

Attempt to disprove the accepted postconditions and constraints on the real
production path, retained scope, dependencies, and claim-scoped proof. Derive
the result from current contracts and candidate evidence rather than the
implementation summary. Safe non-mutating checks may inspect wider context,
but the verdict covers only the fixed unit.

Keep the candidate unchanged and perform no repair. Apply the implementation
phase [Review And
Validate](../phases/implementation-validation-closeout.md#review-and-validate)
contract and the shared [Evidence Contract](evidence-contract.md). Reuse valid
receipts and run only the missing or adversarial falsifier needed by this
boundary.

## Verdict And Return

Return [Acceptance Review Result V1](acceptance-review-result-v1.md). `PASS`
returns to the acceptance owner for [Acceptance-Unit
Closure](acceptance-unit-closure.md). `FAIL` returns to phase-owned correction
or the narrowest reopen owner. `NEEDS_PARENT` returns through [Implementation
Obstacle Recovery](../phases/implementation-obstacle-recovery.md#bottom-up-resolution)
and is not itself a blocked transition.

A material candidate or proof-precondition change invalidates the verdict. If
review remains triggered, open a fresh review only for the affected fixed
boundary.
