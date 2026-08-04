---
name: go-test-strategy
description: "Test strategy: Use for risk scenarios, proof levels, deterministic oracles, gates, or evidence review. Own proof design; Skip unresolved behavior, test implementation, or completion validation."
---

# Go Test Strategy

Proof is designed around **falsifiers**: for every obligation, the scenario that fails on the wrong behavior, the deterministic controls that make it repeatable, and the independent oracle that cannot be fooled by the code under test.

`risk scenario -> fail-before discriminator -> deterministic controls -> independent oracle -> proving layer -> gate and reopen condition`

A test earns existence through a failure it can catch; determinism is designed with controls and fixtures rather than hoped for; and exact source text is an oracle only when the text itself is the accepted external artifact — string presence otherwise proves nothing about behavior.

Load the [shared specialist contract](../specialist-contract.md). Reconstruct every accepted proof obligation from approved behavior, design/test handoffs, affected contract, state, trust and lifecycle boundaries, and current proof surfaces. Build one falsifier for each obligation from a scenario, fail-before discriminator, deterministic control and fixtures, independent oracle, proving layer, command, cleanup proof, and reopen condition.

## Choose The Branch

- **Decision** — select when proof policy is absent or changing. Load the [decision selector](references/decision/index.md) for one result-changing risk. Return one shared Decision disposition per obligation; completion requires a falsifier and reopen condition for every obligation needing proof.
- **Review** — select when changed tests or validation evidence must conform to accepted proof policy. Load the [review selector](references/review/index.md) for the concrete false-pass or flake risk. Return one shared finding-envelope disposition per obligation.

Hand unresolved behavior to its domain skill, executable test code to `go-test-implementation`, and completion claims to `go-verification-before-completion`.
