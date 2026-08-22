# Review

Use fresh independent review at the boundaries selected by this contract.

## Trigger

One fresh reviewer is required for a standalone Research macro result and each
fixed Specification, completed Technical Design, triggered Test Design,
Planning result, and Implementation acceptance unit before movement or
acceptance.

Direct Work and supporting phase-internal work use root self-review unless the
boundary is high-impact, broad, hard to reverse or verify, protected-domain,
materially contested, or explicitly requested.

## Lifecycle

The owner fixes one candidate, its authoritative inputs, phase adapter, and
evidence boundary. The triggering owner then dispatches one fresh reviewer
context, applies the result, and remains until `PASS` or a true external
blocker. Required review stays inside the active macro phase. The reviewer
keeps that boundary read-only, attempts to falsify it, and returns [Review
Result V1](../interfaces/review-result-v1.md). Review never owns repair,
integration, acceptance, or movement.

A material candidate mutation invalidates affected findings and proof. When the
trigger still applies, review the changed boundary in fresh context; reuse only
unaffected findings. The phase adapter owns lenses and threshold, while the
artifact, ledger, or [Transition](transition.md) owner decides movement.

## Route

| Fixed boundary | Adapter |
| --- | --- |
| Standalone research synthesis | [Research](../phases/research.md#review) |
| Completed specification | [Specification Review](../phases/specification-review.md) |
| Technical and Go-ownership design | [Technical Design Review](../phases/technical-design-review.md) |
| Completed Test Design | [Test Design](../phases/test-design.md#review) |
| Planning result | [Task Review / Readiness](../phases/task-review-readiness.md) |
| Fixed implementation unit | [Implementation Review](../phases/implementation-review.md) |

A phase-owned complementary panel replaces the default reviewer only when its
trigger partitions one fixed candidate into named non-overlapping lenses and
defines one synthesis threshold. Stop after one result for one fixed boundary.
