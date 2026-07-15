---
name: go-test-review
description: "Use when a Go diff's tests and validation evidence need review for behavioral coverage, scenario traceability, assertion strength, determinism, and failure-path proof; Own test-quality defects; Skip when tests must be written, proof strategy must be designed, domain behavior is unresolved, or completion claims need final verification."
---

# Go Test Review

Load the [shared specialist contract](../specialist-contract.md) for selection, evidence, return, and handoff. Review scenario traceability, false-pass resistance, determinism, negative paths, fixtures, retired paths, and command fit. Load [the reference selector](references/index.md) only when its pressure changes the result, and another reference only for an independent pressure. Escalate changed proof strategy to `go-test-design` and missing behavior policy to its specification owner. Return the owned decision or evidence-backed finding, forced consequence, and focused proof; stop rather than inventing another owner’s policy.
