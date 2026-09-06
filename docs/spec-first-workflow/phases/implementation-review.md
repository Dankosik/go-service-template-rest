# Implementation Review

Use when shared [Review](../shared/review.md) routes one fixed implementation
unit, inline outcome, or integrated candidate. This adapter owns only
implementation falsifiers and verdicts.

## Final Delivery

Review one fixed delivery candidate. It may contain the entire task ledger;
multiple implemented tasks do not require separate handoffs or verdicts.

Try to disprove the postcondition and important constraints on the real path,
retained scope, dependencies, and claim-scoped proof. Consume the accepted
sources, fixed candidate identity when crossing a boundary, current proof, and
irreproducible external evidence. Reuse valid receipts and run only a missing or
adversarial falsifier.

When named checks are still running on the fixed candidate, inspect the code
and supported traces without duplicating those checks. Pending execution alone
is not a defect or unavailable proof. Receive the results before a final PASS;
report discovered defects promptly. Failed or unavailable required proof keeps
its existing failure or NEEDS_PARENT path. If repair changes the candidate,
apply shared Review's delta and freshness rules.

An intermediate task or lane handoff never triggers this adapter. Review a
single unit here only as final delivery or under an explicit standalone review
request; use the integrated-candidate section for multi-task final delivery.

Classify each surviving finding using [Review Result
V1](../interfaces/review-result-v1.md). A blocking finding must identify the
exact accepted claim it falsifies, the candidate anchor, a reproducible or
mechanically checkable failure, and the smallest repair boundary. Style
preferences, alternative architecture, naming improvements, speculative future
risks, and unrelated cleanup are `FOLLOW_UP` unless they falsify the current
Outcome, Boundary, constraint, or final validation claim. A `FOLLOW_UP` cannot fail
the current task.

For a structure `TASK_DEFECT`, name the current constraint violated and the
concrete unnecessary responsibility, dependency, duplication, or retained path.
Apply the accepted simplicity and dependency constraints, including
[Engineering](../../../AGENTS.md#engineering). A shorter equivalent design alone
does not establish a violation. An alternative organization that preserves
current responsibilities and constraints is `FOLLOW_UP`.

`PASS` returns the candidate for acceptance. `FAIL` returns anchored
candidate-caused findings. `NEEDS_PARENT` names proof or action outside reviewer
authority. A missing fixed delivery boundary returns `REVIEW_HANDOFF_INVALID`
through [Review Result V1](../interfaces/review-result-v1.md).

## Delta recheck

When Review selects a bounded delta recheck, falsify only the anchored findings
and their invalidated proof. Do not reopen unaffected reasoning.

## Integrated candidate

Use when shared Review selects an integrated candidate. Falsify
the complete changed outcome: deferred unit-local protected invariants,
whole-spec coverage, cross-unit compatibility, assembly, and global Completion.
Implemented units have no prior review verdict to reuse. Reuse only actual
still-current evidence and review; do not restart unaffected closed findings.
Each surviving finding names the smallest affected task owner for repair. Use
`INTEGRATION_DEFECT` for a seam or assembly failure. Reopen Planning only when
no existing owner can repair it without changing accepted behavior or scope.
Keep repair and invalidated checks inside final validation; do not recreate
individual task acceptance or review cycles. Preserve unaffected implementation
and current proof.
`PASS` returns the integrated candidate for ledger completion.
