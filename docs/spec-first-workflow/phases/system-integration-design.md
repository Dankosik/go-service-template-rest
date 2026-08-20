# System / Integration Design

Use when Implementation would otherwise choose a runtime boundary, source of
truth, material crossing, failure/recovery behavior, or rollout mechanism. Own
one behaviorally equivalent system mechanism and its material flows.

Consume the ready Specification, current runtime/generated/provider/consumer
authority, relevant repository architecture, accepted risks and proof
obligations, and decision-changing Research. Derive material drivers and hard
constraints, compare viable same-level substitutes only while a real fork
survives, and trace each affected actor or trigger to caller-visible completion
or durable finality through [Material Flow](../rubrics/material-flow.md).

Load only methods exposed by the decision:

- runtime boundary, consistency, or truth owner -> `go-system-architecture`;
- client-visible REST wiring -> `go-api-contract`;
- trust, secrets, authorization, signing, or exact bytes -> `go-security`;
- numeric scale or budget -> `go-performance`;
- durable delivery, replay, compensation, or reconciliation -> `go-distributed`;
- migration, mixed versions, or managed dependency -> [Release
  Closure](../rubrics/release-closure.md);
- non-mechanical Go placement -> [Go Code / Ownership
  Design](go-code-ownership-design.md).

Use [Read-Only Delegation](../shared/read-only-delegation.md) only for
independent material domain questions. The root validates and synthesizes their
results. A signature-sensitive shape fixes exact bytes, algorithm, and one
deterministic non-secret vector.

Return the selected mechanism, driver and alternative dispositions, material
flows, affected authorities and contracts, measurable proof boundaries, and
reopen conditions. Persist only through [Artifacts](../shared/artifacts.md),
then apply [Technical Design Review](technical-design-review.md) through shared
[Review](../shared/review.md).

Ready when every material flow and owner is closed without downstream invention.
Reopen Specification for behavior, Research for evidence, or the named external
owner for a required input.
