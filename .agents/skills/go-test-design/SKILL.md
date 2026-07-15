---
name: go-test-design
description: "Use when accepted behavior needs risk-based scenarios, proof levels, deterministic oracles, fail-before expectations, invariant traceability, or executable gates before coding; Own test strategy and proof design; Skip when behavior is unresolved, tests must be implemented, a diff needs test review, or completion claims need validation."
---

# Go Test Design

Load the [shared specialist contract](../specialist-contract.md) for selection, evidence, return, and handoff. Design a risk-based scenario matrix with independent oracles, fail-before discriminator, deterministic controls, commands, and residual gaps. Load [the reference selector](references/index.md) only when its pressure changes the result, and another reference only for an independent pressure. Escalate unresolved behavior to its specification owner and executable-test work to `go-test-implementation`. Return the owned decision or evidence-backed finding, forced consequence, and focused proof; stop rather than inventing another owner’s policy.
