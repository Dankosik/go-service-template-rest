# Specification

Use when structured work lacks a ready behavior delta. Own what must be true
before selecting a mechanism.

## Inputs

Consume the accepted brief, decision-changing Research, current
runtime/generated/consumer authority, stable existing decisions, and any current
spec findings.

## Method

1. Reconstruct affected actors and observable surfaces from the accepted outcome
   and current authority, not from the candidate spec alone.
2. Record changed, removed, deliberately unchanged, and non-goal behavior. Close
   scope, policy, invariants, compatibility, source-of-truth semantics, failure,
   replay/recovery/finality outcomes, proof expectations, and bounded risks only
   where they can change meaning.
3. For each rule, test whether two reasonable implementations could satisfy it
   yet differ observably. Load [Material Rule](../rubrics/material-rule.md) only
   when that divergence is not already closed.
4. Ground normative choices in the accepted outcome or named owner and factual
   claims in current evidence. Missing user/external policy blocks its owner;
   Design may choose only behaviorally equivalent realizations.

## Conditional Methods

- client-visible REST semantics -> `go-api-contract`;
- business transitions or violation behavior -> `go-domain-invariant`;
- persistence or cache truth semantics -> `go-data-architecture`;
- trust, authorization, isolation, or sensitive-data policy -> `go-security`.

Apply only the method whose pressure changes observable meaning.

## Output

Return a compact behavioral contract: outcome/problem when durable, scope and
non-goals, behavior/contract delta, invariants and edge cases, decisions and
authorities, success/proof expectations, and risks/assumptions/reopen
conditions. Reference unchanged OpenAPI, code, tests, mockups, or external
contracts instead of copying them. Persist `spec.md` only through
[Artifacts](../shared/artifacts.md).

## Review

When shared [Review](../shared/review.md) triggers, load [Specification
Review](specification-review.md).

## Exit And Reopen

Exit when every material divergence has one grounded disposition and Design or
Planning can act without inventing product meaning. Reopen Intake for unresolved
intent, Research for decision-changing evidence, or the named specialist/user
owner for its missing decision. A blocked spec never enters Implementation.
