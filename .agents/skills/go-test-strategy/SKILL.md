---
name: go-test-strategy
description: "Test strategy: Use when accepted behavior needs risk scenarios, proof levels, deterministic oracles, or executable gates, or when changed tests and validation evidence need review. Own proof design and test conformance; Skip when behavior is unresolved, tests must be implemented, or completion claims need validation."
---

# Go Test Strategy

Load the [shared specialist contract](../specialist-contract.md). Reconstruct every accepted proof obligation from approved behavior, design/test handoffs, affected contract, state, trust and lifecycle boundaries, and current proof surfaces. Build one falsifier for each obligation from a scenario, fail-before discriminator, deterministic control and fixtures, independent oracle, proving layer, command, cleanup proof, and reopen condition.

## Choose The Branch

- **Decision** — select when proof policy is absent or changing. Load the [decision selector](references/decision/index.md) for one result-changing risk. Return one shared Decision disposition per obligation; completion requires a falsifier and reopen condition for every obligation needing proof.
- **Review** — select when changed tests or validation evidence must conform to accepted proof policy. Load the [review selector](references/review/index.md) for the concrete false-pass or flake risk. Return one shared finding-envelope disposition per obligation, naming any outside boundary or proof blocker with the smallest deterministic correction and focused command. Missing proof policy returns to the named Proof Decision owner.

Hand unresolved behavior to its domain skill, executable test code to `go-test-implementation`, and completion claims to `go-verification-before-completion`.
