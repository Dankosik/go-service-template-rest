# Review

Use fresh independent review at the boundaries selected by this contract.

## Trigger

One fresh reviewer is required for a standalone Research macro result and each
fixed Specification, completed Technical Design, triggered Test Design, and
Planning result before movement. Their macro-phase boundaries are unchanged.

Implementation uses Lead self-review unless independent review is explicitly
required by the user or accepted task, materially changed behavior affects
authorization, money, data integrity, concurrency safety, or hard-to-reverse
migration, or a material correctness question remains uncertain, contested, or
inadequately covered by current proof. Name the specific risk or question in
the existing review brief. Passing tests do not waive review of materially
changed protected invariants; review does not replace missing mandatory proof.

Apply the same trigger to the integrated candidate before ledger `done`, using
cross-unit interactions and global Completion as its scope. Unit count alone
does not require another reviewer. Reuse unit evidence and review; inspect only
remaining integration questions or reasoning invalidated by assembly. If no
trigger applies, the acceptance owner records that disposition in the existing
result without a separate review report. Resolve any outstanding blocking
findings before acceptance; a changed review disposition cannot waive them.

Direct Work and supporting phase-internal work use root self-review unless the
boundary is high-impact, broad, hard to reverse or verify, protected-domain,
materially contested, or explicitly requested.

## Lifecycle

The owner fixes one candidate, its authoritative inputs, phase adapter, and
evidence boundary. The reviewer keeps that boundary read-only, attempts to
falsify it, and returns [Review Result V1](../interfaces/review-result-v1.md).
Review never owns repair, integration, acceptance, or movement.

Check newly adopted hard constraints against the [Evidence
Contract](evidence-contract.md), including their original source and scope.
For mechanical identity refreshes, use [Transition](transition.md)'s unchanged
semantic-scope rule instead of restarting an unaffected review.

Use a bounded delta recheck by the same reviewer when repair addresses only
that reviewer's anchored findings, leaves Outcome, Boundary, accepted inputs,
interfaces, and risk surface unchanged, and introduces no unrelated behavior or
writable owner. The reviewer remains read-only and does not prescribe or
perform the repair.

For Implementation, retain that reviewer across bounded repairs while findings
close or new discriminating evidence advances the diagnosis. A repeated cycle
with the same unresolved finding and no new evidence requires a fresh reviewer
through [Parent-Owned Recovery](transition.md#parent-owned-recovery), not another
recheck by the same reviewer. Use fresh review when the original analysis is
materially doubtful or the scope/freshness conditions below fail. Recheck count
alone does not require replacement; every blocking finding still must close.

For non-Implementation reviews, allow at most one bounded delta recheck per
review result. If that still FAILs, one fresh reviewer inspects the repaired
candidate, or reopen if Outcome is invalid. Do not start a third cycle on the
same candidate.

If the same-reviewer identity is unavailable, one fresh review covers the
bounded repair delta and invalidated proof, not the original unit, unless the
repair expanded the risk surface.

Use a fresh reviewer when repair changes the acceptance boundary, invalidates
unaffected review reasoning, introduces a new interface or behavior, or expands
the risk surface.

The phase adapter owns lenses and threshold, while the artifact, ledger, or
[Transition](transition.md) owner decides movement.

## Route

| Fixed boundary | Adapter |
| --- | --- |
| Standalone research synthesis | [Research](../phases/research.md#review) |
| Completed specification | [Specification Review](../phases/specification-review.md) |
| Technical and Go-ownership design | [Technical Design Review](../phases/technical-design-review.md) |
| Completed Test Design | [Test Design](../phases/test-design.md#review) |
| Planning result | [Task Review / Readiness](../phases/task-review-readiness.md) |
| Fixed implementation unit | [Implementation Review](../phases/implementation-review.md) |
| Integrated implementation candidate | [Implementation Review](../phases/implementation-review.md#integrated-candidate) |

A phase-owned complementary panel replaces the default reviewer only when its
trigger partitions one fixed candidate into named non-overlapping lenses and
defines one synthesis threshold. Stop after one result for one fixed boundary.
