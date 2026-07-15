---
name: go-implementation-ownership-review
description: "Use when changed Go may violate accepted package or file ownership, dependency direction, source-of-truth seams, or implementation boundaries; Own architecture-conformance defects in code placement and coupling; Skip when topology policy, Go semantics, local readability, or explicit whole-diff structural overbuild is primary."
---

# Go Implementation Ownership Review

Load the [shared specialist contract](../specialist-contract.md) for selection, evidence, return, and handoff. Review package/file responsibility, dependency direction, source of truth, seam ownership, and unjustified helpers. Load [the reference selector](references/index.md) only when its pressure changes the result, and another reference only for an independent pressure. Escalate placement or seam policy to `go-implementation-ownership-spec`; topology goes to `go-system-architecture-spec`. Return the owned decision or evidence-backed finding, forced consequence, and focused proof; stop rather than inventing another owner’s policy.
