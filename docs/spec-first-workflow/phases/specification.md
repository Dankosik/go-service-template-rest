# Specification

Use when structured work lacks a ready behavior delta. Own what must be true
before selecting a mechanism.

Consume the requester-meaning authority selected by [Phase
Selection](../../spec-first-workflow.md#phase-selection), decision-changing
Research, current runtime, generated and consumer authority, stable decisions,
and current findings. Disposition every material requester-meaning item as accepted
behavior, deliberately unchanged behavior, a non-goal, or an upstream open
decision; never silently replace requester meaning. Reconstruct affected actors
and surfaces; record changed, removed, deliberately unchanged, and non-goal
behavior. Apply [Material
Rule](../rubrics/material-rule.md) to each materially affected rule so scope,
policy, invariants, compatibility, truth/finality, failure, replay/recovery, and
success meaning cannot diverge across two reasonable implementations.

Load only a method whose pressure changes observable meaning:

- client-visible REST semantics -> `go-api-contract`;
- business transitions or violations -> `go-domain-invariant`;
- persistence or cache truth -> `go-data-architecture`;
- trust, authorization, isolation, or sensitive data -> `go-security`.

Outcome: Check whether all requirements could pass while the accepted
user outcome still fails; resolve any such gap.

Necessity: Trace each added requirement to accepted intent or a mandatory
constraint. Treat research recommendations as evidence, not accepted
requirements.

Composition: Check interacting rules together through a representative
user-visible scenario; resolve conflicts in terminology, precedence,
and outcomes.

Return a compact behavioral contract and reference unchanged code, contracts,
tests, mockups, or evidence. Persist `spec.md` only through
[Artifacts](../shared/artifacts.md). Apply [Specification
Review](specification-review.md) through shared [Review](../shared/review.md).

Ready when every material requester-meaning item and divergence has one grounded
disposition and the next owner need not invent product meaning. Reopen Intake
when intent changes or is incomplete, Research for evidence, or the named policy
owner for its missing decision.
