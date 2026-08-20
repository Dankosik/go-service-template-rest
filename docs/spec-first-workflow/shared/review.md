# Review

Use independent review when fresh context materially improves confidence in a
fixed artifact or implementation acceptance unit.

## Trigger

Open one fresh reviewer when the fixed boundary is high-impact, broad,
hard-to-reverse, hard to verify, protected-domain, materially contested, or
explicitly requested. Ordinary work uses root self-review. Re-evaluate only
after a material candidate or risk change.

## Route

| Fixed boundary | Review owner |
| --- | --- |
| Standalone research synthesis | [Research Review](../phases/research.md#review) |
| Completed specification | [Specification Review](../phases/specification-review.md) |
| Technical and Go-ownership design | [Technical Design Review](../phases/technical-design-review.md) |
| Non-obvious test design | [Test Design Review](../phases/test-design.md#review) |
| Executable ledger | [Task Review / Readiness](../phases/task-review-readiness.md) |
| Fixed implementation unit | [Implementation Review](#implementation-review) |

Phase-owned complementary panels remain owned by their phase. The artifact
owner repairs findings and receives the verdict; review never takes ownership
of implementation, integration, acceptance, or completion.

## Implementation Review

Run only after the Lead has fixed one recorded singleton or grouped acceptance
unit, or one direct inline outcome, and that candidate has passed mapped
validation. Give the fresh reviewer the accepted sources, immutable candidate
identity, current proof receipts, and irreproducible external evidence.

The reviewer independently tries to disprove the postconditions and important
constraints on the real path, retained scope, dependencies, and claim-scoped
proof. Derive the result from current contracts and candidate evidence rather
than the implementation summary. Keep the candidate unchanged and perform no
repair. Reuse valid receipts and run only a missing or adversarial falsifier
needed by this boundary.

Return [Acceptance Review Result V1](acceptance-review-result-v1.md). An
unresolved or multi-unit boundary returns `REVIEW_HANDOFF_INVALID` without a
verdict. `PASS` returns to the Lead for acceptance; `FAIL` returns anchored
findings for Lead-owned correction or the narrowest upstream repair;
`NEEDS_PARENT` names the unavailable proof or action and its owner rather than
becoming a blocker by itself.

## Stop Rule

Stop after one evidence-bounded verdict for one fixed boundary. A material
candidate or proof-precondition change invalidates the verdict; when the trigger
still applies, review the affected fixed boundary in a fresh context.
