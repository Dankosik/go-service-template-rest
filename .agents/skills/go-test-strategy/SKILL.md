---
name: go-test-strategy
description: "Test strategy: Use when accepted behavior needs risk scenarios, proof levels, deterministic oracles, or executable gates, or when changed tests and validation evidence need review. Own proof design and test conformance; Skip when behavior is unresolved, tests must be implemented, or completion claims need validation."
---

# Go Test Strategy

Load the [shared specialist contract](../specialist-contract.md). Keep scenario traceability, proof levels, fail-before discriminators, deterministic controls, stable oracles, fixtures, commands, cleanup proof, and residual gaps coherent.

## Choose The Branch

- **Decision** — select when proof policy is absent or changing. Load the [decision selector](references/decision/index.md) for one result-changing risk. Complete when every accepted obligation maps to a scenario, oracle, proving layer, command, fail-before signal, and reopen condition.
- **Review** — select when changed tests or validation evidence must conform to accepted proof policy. Load the [review selector](references/review/index.md) for the concrete false-pass or flake risk. Complete when every affected proof obligation is dispositioned as a finding or no finding with the smallest deterministic correction and focused command; missing proof policy stays in the decision branch.

Hand unresolved behavior to its domain skill, executable test code to `go-test-implementation`, and completion claims to `go-verification-before-completion`.
