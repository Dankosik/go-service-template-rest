# Review

Use fresh independent review at the boundaries selected by this contract.

## Trigger

One fresh reviewer is required for a standalone Research macro result and each
fixed Specification, completed Technical Design, and Planning result before
movement. Test authoring is Implementation work and has no separate phase review.

Implementation tasks and subtask lanes have no review or self-review gate.
Return implemented code and continue the ready frontier immediately. Task
packets and protected-domain skills cannot reintroduce per-task review through
a generic risk trigger. Standalone review requests are separate work; a reviewer subtask inside an
implementation ledger cannot bypass this boundary.

Select independent review once for the final assembled delivery candidate when
the user requires it, materially changed behavior affects authorization, money,
data integrity, concurrency safety, or hard-to-reverse migration, or a material
correctness question remains uncertain or contested. Scope that review to the
changed outcome, including unit-local protected invariants and cross-unit
interactions; do not defer a unit risk and then omit it from final review.
Passing tests do not replace this final review or missing required proof.

Otherwise the delivery owner completes final validation without a separate
review report. Record the final disposition in the existing Completion result.
Resolve outstanding blocking findings before acceptance. Reuse current evidence
and recheck only repairs and invalidated reasoning; task count does not add
review cycles.

For a standalone Direct Work delivery, select review at its final boundary
using the same trigger. Pre-Implementation supporting work keeps its existing
phase method. Do not route ledger tasks through Direct Work to obtain an
intermediate self-review or reviewer.

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
close or new discriminating evidence advances the diagnosis. For repeated
findings without new evidence, apply [Parent-Owned
Recovery](transition.md#parent-owned-recovery)'s focused diagnosis and disposition;
do not restart whole-unit review by rotating reviewers. Use fresh review when
the original analysis is materially doubtful or the scope/freshness conditions
below fail. Recheck count alone does not require replacement; every blocking
finding still must close.

For non-Implementation reviews, allow at most one bounded delta recheck per
review result. If that still FAILs, one fresh reviewer inspects the repaired
candidate, or reopen if Outcome is invalid. Do not start a third cycle on the
same candidate. Parent-Owned Recovery still governs repeated findings without
new evidence; reviewer replacement does not restart that allowance.

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
| Planning result | [Task Review / Readiness](../phases/task-review-readiness.md) |
| Single-unit final delivery or explicit standalone implementation review | [Implementation Review](../phases/implementation-review.md) |
| Integrated implementation candidate | [Implementation Review](../phases/implementation-review.md#integrated-candidate) |

A phase-owned complementary panel replaces the default reviewer only when its
trigger partitions one fixed candidate into named non-overlapping lenses and
defines one synthesis threshold. Stop after one result for one fixed boundary.
