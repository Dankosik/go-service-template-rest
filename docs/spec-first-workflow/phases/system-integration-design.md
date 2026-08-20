# System / Integration Design

Use when Implementation would otherwise choose a runtime boundary, source of
truth, material crossing, failure/recovery behavior, or rollout mechanism. Own
one behaviorally equivalent system mechanism and its material flows.

## Inputs

Consume the ready Specification, current runtime/generated/provider/consumer
authority, repository architecture when a boundary changes, accepted risks and
proof obligations, and decision-changing Research.

## Method

1. Derive material drivers, hard constraints, workload/critical-path evidence,
   failure and rollout obligations, and acceptance boundaries. Give each driver
   a consequence that can admit or reject a mechanism.
2. Treat the current system as evidence. Reuse, simplify, replace, or delete its
   machinery according to the drivers.
3. When evidence leaves a real fork, compare viable same-level substitutes
   against the same drivers and select one; otherwise record what collapses the
   fork. Keep adopted elements of rejected substitutes explicit.
4. Trace every affected actor/trigger to caller-visible completion or durable
   finality through [Material Flow](../rubrics/material-flow.md).
5. Record only boundaries, contracts, authorities, state/effects, failure and
   recovery decisions, operational consequences, and proof inputs that
   Implementation must not invent.

## Conditional Methods

- changed runtime/system boundary or consistency owner -> `go-system-architecture`;
- client-visible REST representation/wiring -> `go-api-contract`;
- trust, secrets, authorization, signing, or byte-sensitive verification -> `go-security`;
- scale-sensitive path or accepted numeric budget -> `go-performance`;
- durable delivery, replay, compensation, or reconciliation -> `go-distributed`;
- deployment, migration, mixed versions, or managed dependency -> [Release
  Closure](../rubrics/release-closure.md);
- non-mechanical Go placement -> [Go Code / Ownership Design](go-code-ownership-design.md).

A hash/signature-sensitive shape also fixes exact bytes, algorithm, and one
deterministic non-secret golden vector. Current official evidence owns external
platform behavior.

## Output

Return the selected mechanism, driver/alternative dispositions, material-flow
record, bounded assumptions, measurable acceptance/proof boundaries, and
reopen conditions. Distinguish retained, replaced, and removed surfaces. Persist
`design/overview.md` or one focused design file only through
[Artifacts](../shared/artifacts.md).

## Review

After System and any triggered Go Ownership output are fixed, use root
self-review. Load [Technical Design Review](technical-design-review.md) only
when shared [Review](../shared/review.md) triggers.

## Exit And Reopen

Exit when every material flow is traceable, one coherent necessary mechanism
satisfies all triggered drivers, and placement cannot reopen a system decision.
Reopen Specification for observable ambiguity, Research for missing
decision-changing evidence, or the named external owner for a required input.
