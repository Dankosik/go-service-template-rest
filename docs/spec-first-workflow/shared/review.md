# Review

Use independent review when fresh context can materially improve confidence in
one fixed artifact or implementation unit.

## Trigger

Open one fresh reviewer when the boundary is high-impact, broad,
hard-to-reverse, hard to verify, protected-domain, materially contested, or
explicitly requested. Ordinary work uses root self-review. Re-evaluate only
after a material candidate or risk change.

## Lifecycle

The artifact or acceptance owner fixes the candidate, brief, authoritative
inputs, and evidence boundary. The reviewer independently falsifies that
boundary, keeps it unchanged, and returns [Review Result
V1](../interfaces/review-result-v1.md) using the phase adapter's lenses and
threshold. Review never takes ownership of repair, integration, acceptance, or
movement.

Use the shared [Finding Envelope](review-findings.md). `PASS` moves;
`CONCERNS` moves only when every concern has a proof/risk owner, observable, and
reopen condition and leaves no downstream semantic choice; otherwise the result
is `FAIL`. The owner repairs or reopens the smallest decision. A material
mutation invalidates affected findings and proof; when the trigger still
applies, review that affected fixed boundary in fresh context. Reuse unaffected
findings.

## Route

| Fixed boundary | Adapter |
| --- | --- |
| Standalone research synthesis | [Research](../phases/research.md#review) |
| Completed specification | [Specification Review](../phases/specification-review.md) |
| Technical and Go-ownership design | [Technical Design Review](../phases/technical-design-review.md) |
| Non-obvious test design | [Test Design](../phases/test-design.md#review) |
| Executable ledger | [Task Review / Readiness](../phases/task-review-readiness.md) |
| Fixed implementation unit | [Implementation Review](#implementation-review) |

A phase-owned complementary panel replaces the default reviewer only when its
trigger partitions one fixed candidate into named non-overlapping lenses and
defines one synthesis threshold.

## Implementation Review

Run after one fixed unit or inline outcome passes owner self-review and mapped
validation. Give the reviewer its accepted sources, immutable candidate
identity when crossing a boundary, current proof, and irreproducible external
evidence.

The reviewer tries to disprove the postcondition and important constraints on
the real path, retained scope, dependencies, and claim-scoped proof. Reuse valid
receipts and run only a missing or adversarial falsifier. Use the Implementation
verdicts in [Review Result V1](../interfaces/review-result-v1.md): `PASS`
returns for acceptance, `FAIL` returns anchored candidate-caused findings, and
`NEEDS_PARENT` names proof or action outside reviewer authority.

## Stop Rule

Stop after one evidence-bounded result for one fixed boundary. A standalone
review remains read-only and does not repair or move its candidate.
