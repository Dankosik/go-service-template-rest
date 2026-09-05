---
name: go-test-strategy
description: "Falsification design. Use before test code when the wrong behavior, deterministic controls, independent oracle, proving layer, or reopen condition is non-obvious or disputed."
metadata:
  invocation: model
  kind: method
---

# Go Test Strategy

Design **falsifiers** for every obligation needing proof: a scenario rejecting
wrong behavior, deterministic controls for repeatability, and an independent
oracle the code under test cannot fool.

`risk scenario -> behavioral discriminator -> deterministic controls -> independent oracle -> proving layer -> gate and reopen condition`

Each test must be capable of catching a failure. Design determinism with
controls and fixtures.
Exact source text is an oracle only when it is the accepted external artifact;
otherwise string presence proves no behavior.

Load the [shared specialist contract](../../contracts/specialist-contract.md).
For every accepted behavior at risk needing proof, build
`ProofObligation{wrong_behavior,
discriminator, controls, fixture, oracle, layer, command, cleanup, reopen}` from
approved behavior, handoffs, affected contract, state, trust and lifecycle
boundaries, and current proof surfaces.

Reachability: Ground each scenario in a supported use, plausible failure,
or adversarial path. Preserve the data relationships and event ordering
that make the risk real. Before expanding a fixture shared across producers or
consumers, trace one minimal example from source to expected result: source
cardinality, entity ownership, and absent or partial values must match the
accepted contract at each boundary. Reuse sufficient existing examples; resolve
an unclear relationship or oracle through its smallest owner while retaining
unaffected scenarios.

Stability: Choose assertions that survive behavior-preserving refactoring.
Assert internal structure or interactions only when they carry an
accepted contract or resource bound.

Discriminator: Name the wrong behavior each scenario rejects. Distinguish
a planned discriminator from an observed pre-fix failure; missing code
or a build failure is not behavioral evidence.

## Choose The Branch

- **Decision** — load the [decision selector](references/decision/index.md) for
  one result-changing risk. Return one shared Decision disposition per
  obligation; completion requires a falsifier and reopen condition for every
  obligation needing proof.
- **Review** — load the [review selector](references/review/index.md) for the
  concrete false-pass or flake risk. Return one shared finding-envelope
  disposition per obligation.

Complete only when every obligation has a disposition under
[Test Plan V1](../../../docs/spec-first-workflow/interfaces/test-plan-v1.md), and
each obligation needing proof has a falsifier for the named wrong behavior,
an independent oracle, deterministic controls, and a reopen condition.
