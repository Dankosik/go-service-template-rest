---
name: go-test-strategy
description: "Falsification design. Use before test code when the wrong behavior, deterministic controls, independent oracle, proving layer, or reopen condition is non-obvious or disputed."
metadata:
  invocation: model
  kind: method
---

# Go Test Strategy

Proof is designed around **falsifiers**: for every obligation, the scenario that fails on the wrong behavior, the deterministic controls that make it repeatable, and the independent oracle that cannot be fooled by the code under test.

`risk scenario -> fail-before discriminator -> deterministic controls -> independent oracle -> proving layer -> gate and reopen condition`

A test earns existence through a failure it can catch; determinism is designed with controls and fixtures rather than hoped for; and exact source text is an oracle only when the text itself is the accepted external artifact — string presence otherwise proves nothing about behavior.

Load the [shared specialist contract](../../contracts/specialist-contract.md).
For every accepted behavior at risk, build `ProofObligation{wrong_behavior,
fail_before, controls, fixture, oracle, layer, command, cleanup, reopen}` from
approved behavior, handoffs, affected contract, state, trust and lifecycle
boundaries, and current proof surfaces.

## Choose The Branch

- **Decision** — load the [decision selector](references/decision/index.md) for
  one result-changing risk. Return one shared Decision disposition per
  obligation; completion requires a falsifier and reopen condition for every
  obligation needing proof.
- **Review** — load the [review selector](references/review/index.md) for the
  concrete false-pass or flake risk. Return one shared finding-envelope
  disposition per obligation.

Complete only when every obligation has a falsifier that would fail before the
repair, an independent oracle, deterministic controls, and a reopen condition.
